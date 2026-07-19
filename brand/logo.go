package brand

import "strings"

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

// Width returns the maximum rune width of the frame.
func (f Frame) Width() int {
	width := 0
	for _, line := range f.lines {
		if n := len([]rune(line)); n > width {
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

// StaticFor returns the non-animated frame for a logo variant.
func StaticFor(variant LogoVariant) Frame {
	return FrameFor(variant, 0)
}

// FlameFrame returns a deterministic animated flame frame.
func FlameFrame(frame int) Frame {
	return FrameFor(VariantFlame, frame)
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

var flameFrames = []Frame{
	newFrame(
		"    .  ",
		"   )(  ",
		"  )##( ",
		" )####(",
		")######(",
		" `####' ",
	),
	newFrame(
		"   .   ",
		"   )(  ",
		"  )##( ",
		" )####(",
		")######(",
		" `####' ",
	),
	newFrame(
		"  . .  ",
		"  )#(  ",
		" )###( ",
		")#####(",
		")######(",
		" `####' ",
	),
	newFrame(
		"   .*  ",
		"  )##( ",
		" )####(",
		")######(",
		")#######(",
		" `#####' ",
	),
	newFrame(
		"  * .  ",
		"  )#(  ",
		" )###( ",
		")#####(",
		")######(",
		" `####' ",
	),
	newFrame(
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
