package festivalbundle_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Obedience-Corp/obey-shared/festivalbundle"
)

func TestRewrite_inRootUnchanged(t *testing.T) {
	packRoot := t.TempDir()
	// pack root tree
	_ = os.WriteFile(filepath.Join(packRoot, "note.md"), []byte("![local](media/pixel.txt)\n![web](https://example.com/x.png)\n"), 0o644)
	_ = os.MkdirAll(filepath.Join(packRoot, "media"), 0o755)
	_ = os.WriteFile(filepath.Join(packRoot, "media", "pixel.txt"), []byte("px"), 0o644)

	payload := t.TempDir()
	copyTree(t, packRoot, payload)

	warns, err := festivalbundle.RewritePayloadLinks(context.Background(), payload, packRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	_ = warns
	text := readFile(t, filepath.Join(payload, "note.md"))
	if !strings.Contains(text, "![local](media/pixel.txt)") {
		t.Fatalf("in-root link rewritten: %s", text)
	}
	if !strings.Contains(text, "https://example.com/x.png") {
		t.Fatalf("URL changed: %s", text)
	}
	if _, err := os.Stat(filepath.Join(payload, ".artifacts")); !os.IsNotExist(err) {
		t.Fatal("did not expect .artifacts for in-root only")
	}
}

func TestRewrite_outOfRootVendored(t *testing.T) {
	base := t.TempDir()
	packRoot := filepath.Join(base, "unit")
	outside := filepath.Join(base, "shared")
	_ = os.MkdirAll(packRoot, 0o755)
	_ = os.MkdirAll(outside, 0o755)
	_ = os.WriteFile(filepath.Join(outside, "logo.txt"), []byte("OUTSIDE-ASSET\n"), 0o644)
	_ = os.WriteFile(filepath.Join(packRoot, "local.txt"), []byte("local\n"), 0o644)
	note := "![local](local.txt)\n![out](../shared/logo.txt)\n[docs](https://textbundle.org/spec/)\n"
	_ = os.WriteFile(filepath.Join(packRoot, "note.md"), []byte(note), 0o644)

	payload := t.TempDir()
	copyTree(t, packRoot, payload)

	warns, err := festivalbundle.RewritePayloadLinks(context.Background(), payload, packRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	_ = warns

	text := readFile(t, filepath.Join(payload, "note.md"))
	if strings.Contains(text, "../shared/logo.txt") {
		t.Fatalf("out-of-root path not rewritten: %s", text)
	}
	if !strings.Contains(text, ".artifacts/") {
		t.Fatalf("expected .artifacts link: %s", text)
	}
	if strings.Contains(text, "/var/") || strings.Contains(text, "Temp") {
		t.Fatalf("host temp path leaked into link: %s", text)
	}
	if !strings.Contains(text, "![local](local.txt)") {
		t.Fatalf("in-root local link changed: %s", text)
	}
	if !strings.Contains(text, "https://textbundle.org/spec/") {
		t.Fatalf("URL changed: %s", text)
	}

	arts, err := os.ReadDir(filepath.Join(payload, ".artifacts"))
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 1 {
		t.Fatalf("want 1 artifact, got %d", len(arts))
	}
	body := readFile(t, filepath.Join(payload, ".artifacts", arts[0].Name()))
	if !strings.Contains(body, "OUTSIDE-ASSET") {
		t.Fatalf("artifact content: %q", body)
	}
}

func TestRewrite_absolutePathVendored(t *testing.T) {
	packRoot := t.TempDir()
	outsideFile := filepath.Join(t.TempDir(), "abs-asset.txt")
	if err := os.WriteFile(outsideFile, []byte("ABS\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	note := "![a](" + outsideFile + ")\n"
	if err := os.WriteFile(filepath.Join(packRoot, "note.md"), []byte(note), 0o644); err != nil {
		t.Fatal(err)
	}
	payload := t.TempDir()
	copyTree(t, packRoot, payload)

	if _, err := festivalbundle.RewritePayloadLinks(context.Background(), payload, packRoot, false); err != nil {
		t.Fatal(err)
	}
	text := readFile(t, filepath.Join(payload, "note.md"))
	if strings.Contains(text, outsideFile) {
		t.Fatalf("absolute path not rewritten: %s", text)
	}
	if !strings.Contains(text, ".artifacts/") {
		t.Fatalf("expected artifacts link: %s", text)
	}
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
