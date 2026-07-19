package brand

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// LogoVariant identifies a bounded shared logo primitive.
type LogoVariant string

const (
	VariantCompactMark LogoVariant = "compact-mark"
	VariantWordmark    LogoVariant = "wordmark"
	VariantFlame       LogoVariant = "flame"
)

const (
	// CompactMark is a single-cell mark selected for predictable terminal width.
	CompactMark = "▲"
	// Wordmark is the shared Festival wordmark.
	Wordmark = "FESTIVAL"

	// FlameMaxWidth is the fixed terminal cell width of every flame frame.
	// All flame lines are padded to this width so animation does not reflow layout.
	FlameMaxWidth = 9
	// FlameMaxHeight is the fixed line count of every full flame frame.
	FlameMaxHeight = 6
)

// Frame contains one bounded logo frame. Lines are copied on construction and
// when returned so consumers cannot mutate the canonical frame data.
type Frame struct {
	lines []string
}

// Lines returns a copy of the frame lines.
func (f Frame) Lines() []string {
	return append([]string(nil), f.lines...)
}

// Width returns the maximum terminal cell width of the frame.
// Cell width uses go-runewidth so full-width and emoji glyphs are measured
// correctly; rune count alone is not used.
func (f Frame) Width() int {
	width := 0
	for _, line := range f.lines {
		if n := runewidth.StringWidth(line); n > width {
			width = n
		}
	}
	return width
}

// Height returns the number of lines in the frame.
func (f Frame) Height() int {
	return len(f.lines)
}

// String joins the frame lines with newlines.
func (f Frame) String() string {
	return strings.Join(f.lines, "\n")
}

// FrameFor returns a deterministic frame for a logo variant. Only the flame
// variant animates; compact mark and wordmark variants are stable primitives.
// Negative frame values select the first flame frame.
//
// This helper does not consult reduced-motion policy. Prefer
// FrameForCapabilities when the caller has resolved Capabilities.
func FrameFor(variant LogoVariant, frame int) Frame {
	switch variant {
	case VariantCompactMark:
		return newFrame(CompactMark)
	case VariantWordmark:
		return newFrame(Wordmark)
	case VariantFlame:
		return flameFrame(frame)
	default:
		return Frame{}
	}
}

// FrameForCapabilities returns a logo frame that respects motion policy.
// When caps.AllowMotion() is false, the static frame is returned and frame
// index is ignored.
func FrameForCapabilities(variant LogoVariant, frame int, caps Capabilities) Frame {
	if !caps.AllowMotion() {
		return StaticFor(variant)
	}
	return FrameFor(variant, frame)
}

// StaticFor returns the non-animated frame for a logo variant.
func StaticFor(variant LogoVariant) Frame {
	return FrameFor(variant, 0)
}

// FlameFrame returns a deterministic animated flame frame.
// Prefer FlameFrameFor when reduced-motion or plain policy should apply.
func FlameFrame(frame int) Frame {
	return FrameFor(VariantFlame, frame)
}

// FlameFrameFor returns a flame frame that respects motion policy.
func FlameFrameFor(frame int, caps Capabilities) Frame {
	return FrameForCapabilities(VariantFlame, frame, caps)
}

// StaticFlame returns the first full flame frame for plain or reduced-motion
// output.
func StaticFlame() Frame {
	return StaticFor(VariantFlame)
}

// SmallFlame returns the compact two-line flame used in tight headers.
func SmallFlame() Frame {
	return newFrame(")^(", ")#(")
}

func newFrame(lines ...string) Frame {
	return Frame{lines: append([]string(nil), lines...)}
}

// newPaddedFrame pads each line with trailing spaces to a fixed cell width so
// animated frames share stable geometry.
func newPaddedFrame(width int, lines ...string) Frame {
	padded := make([]string, len(lines))
	for i, line := range lines {
		w := runewidth.StringWidth(line)
		if w < width {
			padded[i] = line + strings.Repeat(" ", width-w)
		} else {
			padded[i] = line
		}
	}
	return Frame{lines: padded}
}

var flameFrames = []Frame{
	newPaddedFrame(FlameMaxWidth,
		"    .  ",
		"   )(  ",
		"  )##( ",
		" )####(",
		")######(",
		" `####' ",
	),
	newPaddedFrame(FlameMaxWidth,
		"   .   ",
		"   )(  ",
		"  )##( ",
		" )####(",
		")######(",
		" `####' ",
	),
	newPaddedFrame(FlameMaxWidth,
		"  . .  ",
		"  )#(  ",
		" )###( ",
		")#####(",
		")######(",
		" `####' ",
	),
	newPaddedFrame(FlameMaxWidth,
		"   .*  ",
		"  )##( ",
		" )####(",
		")######(",
		")#######(",
		" `#####' ",
	),
	newPaddedFrame(FlameMaxWidth,
		"  * .  ",
		"  )#(  ",
		" )###( ",
		")#####(",
		")######(",
		" `####' ",
	),
	newPaddedFrame(FlameMaxWidth,
		"   .   ",
		"  )#(  ",
		" )###( ",
		")#####(",
		")######(",
		" `####' ",
	),
}

func flameFrame(frame int) Frame {
	if frame < 0 {
		frame = 0
	}
	return flameFrames[frame%len(flameFrames)]
}
