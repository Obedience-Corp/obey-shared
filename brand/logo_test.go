package brand

import (
	"reflect"
	"testing"
)

func TestFlameFramesAreBoundedAndDeterministic(t *testing.T) {
	first := FlameFrame(0)
	if first.Height() != 6 {
		t.Fatalf("flame height = %d, want 6", first.Height())
	}
	if first.Width() > 9 {
		t.Fatalf("flame width = %d, want <= 9", first.Width())
	}
	if got := FlameFrame(len(flameFrames)); !reflect.DeepEqual(got.Lines(), first.Lines()) {
		t.Fatalf("wrapped frame differs: got %v want %v", got.Lines(), first.Lines())
	}
	if got := FlameFrame(-1); !reflect.DeepEqual(got.Lines(), first.Lines()) {
		t.Fatalf("negative frame differs: got %v want %v", got.Lines(), first.Lines())
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
