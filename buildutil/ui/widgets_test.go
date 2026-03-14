package ui

import (
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close write pipe: %v", err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close read pipe: %v", err)
	}

	return string(out)
}

func TestSummaryCardWithStatus_ConsistentLineWidths(t *testing.T) {
	t.Setenv("COLUMNS", "80")
	Init(true) // Disable ANSI output for deterministic assertions.

	rows := [][]string{
		{"Failed Test", "Status", ""},
		{"TestGRPC_SessionMessageStreaming", "✗ FAILED", ""},
		{"TestGRPC_SessionCampaignIsolation", "✗ FAILED", ""},
		{"4 suites", "28/35 tests passed", "93.54s"},
	}

	out := captureStdout(t, func() {
		SummaryCardWithStatus("Integration Test Summary", rows, "93.54s", false, "✓ OK", "✗ FAILED")
	})

	lines := strings.Split(out, "\n")
	expectedWidth := -1
	checked := 0

	for _, line := range lines {
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "╔") &&
			!strings.HasPrefix(line, "╠") &&
			!strings.HasPrefix(line, "║") &&
			!strings.HasPrefix(line, "╚") {
			continue
		}

		width := VisualLength(line)
		if expectedWidth == -1 {
			expectedWidth = width
		}
		if width != expectedWidth {
			t.Fatalf("line width mismatch: got %d want %d line=%q", width, expectedWidth, line)
		}
		checked++
	}

	if checked == 0 {
		t.Fatal("no summary card lines were captured")
	}
}
