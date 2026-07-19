package brand

import (
	"reflect"
	"testing"

	"github.com/mattn/go-runewidth"
)

func TestFlameFramesAreBoundedAndDeterministic(t *testing.T) {
	first := FlameFrame(0)
	if first.Height() != FlameMaxHeight {
		t.Fatalf("flame height = %d, want %d", first.Height(), FlameMaxHeight)
	}
	if first.Width() != FlameMaxWidth {
		t.Fatalf("flame width = %d, want %d", first.Width(), FlameMaxWidth)
	}
	if got := FlameFrame(len(flameFrames)); !reflect.DeepEqual(got.Lines(), first.Lines()) {
		t.Fatalf("wrapped frame differs: got %v want %v", got.Lines(), first.Lines())
	}
	if got := FlameFrame(-1); !reflect.DeepEqual(got.Lines(), first.Lines()) {
		t.Fatalf("negative frame differs: got %v want %v", got.Lines(), first.Lines())
	}
}

func TestFlameFramesShareFixedGeometry(t *testing.T) {
	for i := range flameFrames {
		frame := FlameFrame(i)
		if frame.Width() != FlameMaxWidth {
			t.Fatalf("frame %d width = %d, want %d (lines=%q)", i, frame.Width(), FlameMaxWidth, frame.Lines())
		}
		if frame.Height() != FlameMaxHeight {
			t.Fatalf("frame %d height = %d, want %d", i, frame.Height(), FlameMaxHeight)
		}
		for j, line := range frame.Lines() {
			if got := runewidth.StringWidth(line); got != FlameMaxWidth {
				t.Fatalf("frame %d line %d cell width = %d, want %d (%q)", i, j, got, FlameMaxWidth, line)
			}
		}
	}
}

func TestFrameLinesAreCopied(t *testing.T) {
	lines := FlameFrame(0).Lines()
	lines[0] = "mutated"
	if FlameFrame(0).Lines()[0] == "mutated" {
		t.Fatal("mutating returned frame lines changed canonical data")
	}
}

func TestLogoVariants(t *testing.T) {
	if got := StaticFor(VariantCompactMark).String(); got != CompactMark {
		t.Fatalf("compact mark = %q, want %q", got, CompactMark)
	}
	if got := StaticFor(VariantWordmark).String(); got != Wordmark {
		t.Fatalf("wordmark = %q, want %q", got, Wordmark)
	}
	small := SmallFlame()
	if small.Height() != 2 || small.Width() != 3 {
		t.Fatalf("small flame dimensions = %dx%d, want 3x2", small.Width(), small.Height())
	}
	if got := FrameFor("unknown", 0); got.Height() != 0 {
		t.Fatalf("unknown variant returned %v", got.Lines())
	}
}

func TestFrameWidthUsesCellWidth(t *testing.T) {
	// Full-width ideograph is two terminal cells; rune count would report 1.
	wide := newFrame("全")
	if got := wide.Width(); got != 2 {
		t.Fatalf("Width() = %d, want 2 for full-width glyph", got)
	}
	// Compact mark stays a single cell.
	if got := StaticFor(VariantCompactMark).Width(); got != 1 {
		t.Fatalf("compact mark Width() = %d, want 1", got)
	}
}

func TestFrameForCapabilitiesHonorsReducedMotion(t *testing.T) {
	static := StaticFlame()
	animated := FlameFrame(3)
	if reflect.DeepEqual(static.Lines(), animated.Lines()) {
		t.Fatal("fixture assumes frame 3 differs from static frame 0")
	}

	reduced := Capabilities{IsTTY: true, ColorDepth: ColorTrueColor, ReducedMotion: true}
	if got := FlameFrameFor(3, reduced); !reflect.DeepEqual(got.Lines(), static.Lines()) {
		t.Fatalf("reduced motion returned animated frame: got %v want %v", got.Lines(), static.Lines())
	}

	plain := Capabilities{IsTTY: true, ColorDepth: ColorTrueColor, ContinuousIntegration: true}
	if got := FrameForCapabilities(VariantFlame, 3, plain); !reflect.DeepEqual(got.Lines(), static.Lines()) {
		t.Fatalf("plain policy returned animated frame: got %v want %v", got.Lines(), static.Lines())
	}

	interactive := Capabilities{IsTTY: true, ColorDepth: ColorTrueColor}
	if got := FlameFrameFor(3, interactive); !reflect.DeepEqual(got.Lines(), animated.Lines()) {
		t.Fatalf("interactive motion returned wrong frame: got %v want %v", got.Lines(), animated.Lines())
	}
}
