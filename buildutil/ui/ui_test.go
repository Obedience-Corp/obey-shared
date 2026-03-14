package ui

import (
	"testing"
)

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "hello", "hello"},
		{"empty", "", ""},
		{"red text", Red + "error" + Reset, "error"},
		{"bold green", Bold + Green + "ok" + Reset, "ok"},
		{"multiple colors", Red + "a" + Green + "b" + Yellow + "c" + Reset, "abc"},
		{"cursor hide/show", HideCursor + "text" + ShowCursor, "text"},
		{"clear line", ClearLine + "fresh", "fresh"},
		{"move up", MoveUp + "moved", "moved"},
		{"mixed all", Cyan + "┌" + Reset + Bold + Red + "err" + Reset + ClearLine, "┌err"},
		{"no false positives", "normal [31m text", "normal [31m text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripANSI(tt.input)
			if got != tt.want {
				t.Errorf("StripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestVisualLength(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"plain ascii", "hello", 5},
		{"empty", "", 0},
		{"with ansi", Red + "error" + Reset, 5},
		{"emoji", "🎪", 2},
		{"text with emoji", "ok 🎪", 5},
		{"unicode basic", "café", 4},
		{"multiple ansi", Bold + Green + "✓" + Reset + " ok", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VisualLength(tt.input)
			if got != tt.want {
				t.Errorf("VisualLength(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestCenter(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		width int
		want  string
	}{
		{"basic", "hi", 10, "    hi    "},
		{"exact width", "hello", 5, "hello"},
		{"wider than width", "hello world", 5, "hello world"},
		{"odd padding", "ab", 7, "  ab   "},
		{"single char", "x", 5, "  x  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Center(tt.text, tt.width)
			if got != tt.want {
				t.Errorf("Center(%q, %d) = %q, want %q", tt.text, tt.width, got, tt.want)
			}
		})
	}
}

func TestInit_NoColor(t *testing.T) {
	// Force no-color via flag
	Init(true)
	if ColourEnabled() {
		t.Error("ColourEnabled() should be false when Init(true)")
	}
}

func TestInit_NoColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	Init(false)
	if ColourEnabled() {
		t.Error("ColourEnabled() should be false when NO_COLOR is set")
	}
}

func TestInit_CIEnv(t *testing.T) {
	t.Setenv("CI", "true")
	Init(false)
	if ColourEnabled() {
		t.Error("ColourEnabled() should be false when CI is set")
	}
}

func TestTermWidth_Default(t *testing.T) {
	w := TermWidth()
	if w <= 0 {
		t.Errorf("TermWidth() = %d, want > 0", w)
	}
}
