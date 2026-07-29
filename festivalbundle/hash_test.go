package festivalbundle_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Obedience-Corp/obey-shared/festivalbundle"
)

func TestPayloadContentID_emptyPayloadFile(t *testing.T) {
	root := t.TempDir()
	payload := filepath.Join(root, "payload")
	if err := os.MkdirAll(payload, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(payload, "empty.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	id, err := festivalbundle.PayloadContentID(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(id, "sha256:") || len(id) != len("sha256:")+64 {
		t.Fatalf("unexpected id form: %q", id)
	}

	// Same tree twice → stable id.
	id2, err := festivalbundle.PayloadContentID(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if id != id2 {
		t.Fatalf("hash not stable: %q vs %q", id, id2)
	}
}

func TestPayloadContentID_orderIndependentOfWalk(t *testing.T) {
	// Two files; hash must sort by path bytes, not creation order.
	root := t.TempDir()
	payload := filepath.Join(root, "payload")
	if err := os.MkdirAll(filepath.Join(payload, "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(payload, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Create b before a on disk; sorted feed still a then b.
	if err := os.WriteFile(filepath.Join(payload, "b", "x.txt"), []byte("bx"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(payload, "a", "x.txt"), []byte("ax"), 0o644); err != nil {
		t.Fatal(err)
	}

	id, err := festivalbundle.PayloadContentID(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}

	// Rebuild with opposite write order should match.
	root2 := t.TempDir()
	p2 := filepath.Join(root2, "payload")
	_ = os.MkdirAll(filepath.Join(p2, "a"), 0o755)
	_ = os.MkdirAll(filepath.Join(p2, "b"), 0o755)
	_ = os.WriteFile(filepath.Join(p2, "a", "x.txt"), []byte("ax"), 0o644)
	_ = os.WriteFile(filepath.Join(p2, "b", "x.txt"), []byte("bx"), 0o644)
	id2, err := festivalbundle.PayloadContentID(context.Background(), root2)
	if err != nil {
		t.Fatal(err)
	}
	if id != id2 {
		t.Fatalf("order affected hash:\n  %s\n  %s", id, id2)
	}
}

func TestPayloadContentID_infoJSONNotIncluded(t *testing.T) {
	root := t.TempDir()
	payload := filepath.Join(root, "payload")
	if err := os.MkdirAll(payload, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(payload, "note.md"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	id1, err := festivalbundle.PayloadContentID(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "info.json"), []byte(`{"format_version":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	id2, err := festivalbundle.PayloadContentID(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 {
		t.Fatalf("info.json affected hash: %q vs %q", id1, id2)
	}
}

func TestPayloadContentID_matchesPythonOracle(t *testing.T) {
	// Compare against the design workitem prototype when available on disk.
	candidates := []string{
		// From projects/obey-shared when tests run with that cwd
		filepath.Join("..", "..", "workflow", "design", "festival-portable-text-bundle", "prototype"),
		// From festivalbundle package dir
		filepath.Join("..", "..", "..", "workflow", "design", "festival-portable-text-bundle", "prototype"),
	}
	// Also try absolute from env CAMP_ROOT / detect
	if wd, err := os.Getwd(); err == nil {
		// module root when go test ./festivalbundle
		candidates = append(candidates,
			filepath.Join(wd, "..", "..", "workflow", "design", "festival-portable-text-bundle", "prototype"),
			filepath.Join(wd, "..", "..", "..", "workflow", "design", "festival-portable-text-bundle", "prototype"),
		)
	}

	var protoRoot string
	for _, c := range candidates {
		if st, err := os.Stat(filepath.Join(c, "festival_bundle", "hashutil.py")); err == nil && !st.IsDir() {
			protoRoot, _ = filepath.Abs(c)
			break
		}
	}
	if protoRoot == "" {
		t.Skip("python prototype not found (run tests from campaign checkout)")
	}

	// Build a tiny bundle tree and hash with both implementations.
	root := t.TempDir()
	payload := filepath.Join(root, "payload")
	_ = os.MkdirAll(filepath.Join(payload, "sub"), 0o755)
	_ = os.WriteFile(filepath.Join(payload, "a.md"), []byte("alpha\n"), 0o644)
	_ = os.WriteFile(filepath.Join(payload, "sub", "b.txt"), []byte("beta"), 0o644)

	goID, err := festivalbundle.PayloadContentID(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}

	// Python one-liner using prototype hashutil
	script := `
import sys
sys.path.insert(0, sys.argv[1])
from pathlib import Path
from festival_bundle.hashutil import payload_content_id
print(payload_content_id(Path(sys.argv[2])))
`
	cmd := exec.Command("python3", "-c", script, protoRoot, root)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("python oracle failed: %v\n%s", err, out)
	}
	pyID := strings.TrimSpace(string(out))
	if goID != pyID {
		t.Fatalf("Go/Python hash mismatch:\n  go: %s\n  py: %s", goID, pyID)
	}
}

func TestPayloadContentID_rejectsSymlink(t *testing.T) {
	root := t.TempDir()
	payload := filepath.Join(root, "payload")
	if err := os.MkdirAll(payload, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(payload, "real.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(payload, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks not supported on this platform")
	}
	_, err := festivalbundle.PayloadContentID(context.Background(), root)
	if !errors.Is(err, festivalbundle.ErrSymlinkRejected) {
		t.Fatalf("want ErrSymlinkRejected, got %v", err)
	}
}

func TestPayloadContentID_contextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := festivalbundle.PayloadContentID(ctx, t.TempDir())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want canceled, got %v", err)
	}
}
