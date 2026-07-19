package lipgloss

import (
	"strings"
	"testing"

	"github.com/Obedience-Corp/obey-shared/brand"
	charmgloss "github.com/charmbracelet/lipgloss"
)

func TestFlameScaleIsBounded(t *testing.T) {
	styles := New(brand.Resolve(brand.ModePlain, brand.Capabilities{IsTTY: true, ColorDepth: brand.ColorTrueColor}))
	for _, scale := range []float64{-1, 0, 0.5, 1, 2} {
		out := Flame(0, scale, styles)
		if out == "" {
			t.Fatalf("Flame(..., %v) returned empty output", scale)
		}
		if width := charmgloss.Width(out); width > 9 {
			t.Fatalf("Flame(..., %v) width = %d, want <= 9", scale, width)
		}
	}
}

func TestPlainStylesDoNotEmitANSI(t *testing.T) {
	styles := New(brand.Resolve(brand.ModePlain, brand.Capabilities{IsTTY: true, ColorDepth: brand.ColorTrueColor}))
	for name, out := range map[string]string{
		"flame":       Flame(0, 1, styles),
		"wordmark":    Wordmark(styles),
		"small flame": SmallFlame(styles),
		"celebrate":   Celebrate(0, styles),
	} {
		if strings.Contains(out, "\x1b[") {
			t.Fatalf("%s emitted ANSI in plain mode: %q", name, out)
		}
	}
}

func TestSparkFieldHandlesNegativeTicks(t *testing.T) {
	styles := New(brand.Resolve(brand.ModePlain, brand.Capabilities{IsTTY: true, ColorDepth: brand.ColorTrueColor}))
	out := (SparkField{Width: 12, Height: 3, Tick: -4}).Render(styles)
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("spark line count = %d, want 3", len(lines))
	}
	for i, line := range lines {
		if got := charmgloss.Width(line); got != 12 {
			t.Fatalf("spark line %d width = %d, want 12", i, got)
		}
	}
}

func TestRuleIsCapped(t *testing.T) {
	styles := New(brand.Resolve(brand.ModePlain, brand.Capabilities{IsTTY: true, ColorDepth: brand.ColorTrueColor}))
	if got := charmgloss.Width(Rule(0, styles)); got != 1 {
		t.Fatalf("zero-width rule = %d, want 1", got)
	}
	if got := charmgloss.Width(Rule(500, styles)); got != 300 {
		t.Fatalf("wide rule = %d, want 300", got)
	}
}
