package mdrender

import (
	"strings"
	"testing"
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
