package mdrender

import (
	"strings"
	"testing"
	"time"
)

const testMarkdown = `# Hello World

This is a **bold** paragraph with some ` + "`inline code`" + `.

## Code Example

` + "```go" + `
func main() {
    fmt.Println("hello")
}
` + "```" + `

- item one
- item two
`

func TestRender_RawPassthrough(t *testing.T) {
	result := Render(testMarkdown, WithForceRaw(true))
	if result != testMarkdown {
		t.Errorf("expected raw passthrough, got transformed output")
	}
}

func TestRender_TTYProducesStyledOutput(t *testing.T) {
	result := Render(testMarkdown, WithForceTTY(true), WithWidth(80), WithStyle("dark"))
	if result == testMarkdown {
		t.Errorf("expected glamour-rendered output, got raw markdown")
	}
	// Glamour wraps text in ANSI sequences, so check for escape codes.
	if !strings.Contains(result, "\x1b[") {
		t.Errorf("expected ANSI escape sequences in styled output")
	}
}

func TestRender_EmptyInput(t *testing.T) {
	result := Render("", WithForceTTY(true))
	if result != "" {
		t.Errorf("expected empty string for empty input, got %q", result)
	}
}

func TestRender_WidthControl(t *testing.T) {
	narrow := Render(testMarkdown, WithForceTTY(true), WithWidth(40))
	wide := Render(testMarkdown, WithForceTTY(true), WithWidth(120))
	if narrow == wide {
		t.Errorf("expected different output for different widths")
	}
}

func TestRender_StyleOverride(t *testing.T) {
	dark := Render(testMarkdown, WithForceTTY(true), WithWidth(80), WithStyle("dark"))
	light := Render(testMarkdown, WithForceTTY(true), WithWidth(80), WithStyle("light"))
	if dark == "" || light == "" {
		t.Errorf("expected non-empty output for both styles")
	}
}

// TestRender_AutoStyleCompletesWithoutBlocking reaches renderGlamour's default
// "auto" style (no WithStyle override) and asserts the call returns promptly.
//
// Coverage note: under go test / CI, os.Stdout is not a TTY, so glamour's
// auto path selects NoTTYStyleConfig and does not call
// termenv.HasDarkBackground() / OSC 11. This test is therefore a hang
// tripwire for the auto branch under non-TTY stdout — not a regression for
// the original mute-pty freeze (which needs a real unresponsive TTY on the
// process stdout). WithForceTTY only affects mdrender's own TTY gating; it
// does not make glamour treat stdout as a TTY. The 2s bound is well under
// termenv's ~5s OSCTimeout so a future always-on blocking query on this
// path still fails the suite quickly instead of hanging indefinitely.
func TestRender_AutoStyleCompletesWithoutBlocking(t *testing.T) {
	done := make(chan string, 1)
	go func() {
		done <- Render(testMarkdown, WithForceTTY(true), WithWidth(80))
	}()

	select {
	case result := <-done:
		if result == testMarkdown {
			t.Errorf("expected glamour-rendered output for auto style, got raw markdown passthrough")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Render with default auto style did not return within 2s under non-TTY stdout (auto branch hang tripwire)")
	}
}

func TestRender_MalformedMarkdown(t *testing.T) {
	malformed := "# Unclosed\n```\nno closing fence\n## Another heading"
	result := Render(malformed, WithForceTTY(true), WithWidth(80))
	if result == "" {
		t.Errorf("expected non-empty output for malformed markdown")
	}
}

func TestRender_NoTrailingNewlines(t *testing.T) {
	result := Render(testMarkdown, WithForceTTY(true), WithWidth(80))
	if strings.HasSuffix(result, "\n") {
		t.Errorf("expected no trailing newlines in rendered output")
	}
}

func TestRender_NOCOLORRespected(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	result := Render(testMarkdown)
	if result != testMarkdown {
		t.Errorf("expected raw passthrough when NO_COLOR is set")
	}
}

func TestRender_CIRespected(t *testing.T) {
	t.Setenv("CI", "true")
	result := Render(testMarkdown)
	if result != testMarkdown {
		t.Errorf("expected raw passthrough when CI is set")
	}
}
