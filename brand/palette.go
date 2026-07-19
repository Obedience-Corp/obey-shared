package brand

// Mode selects the palette expression used by a consumer.
type Mode string

const (
	ModeAdaptive     Mode = "adaptive"
	ModeDark         Mode = "dark"
	ModeLight        Mode = "light"
	ModeHighContrast Mode = "high-contrast"
	ModePlain        Mode = "plain"
)

// ParseMode converts a user-facing mode value to a supported mode. Unknown
// values intentionally fall back to adaptive behavior.
func ParseMode(value string) Mode {
	switch Mode(value) {
	case ModeDark, ModeLight, ModeHighContrast, ModePlain:
		return Mode(value)
	default:
		return ModeAdaptive
	}
}

// ColorDepth describes the color capability available to a renderer.
// ColorUnknown means that the consumer has not determined a specific depth;
// it does not disable color when the output is otherwise interactive.
//
// Depth-aware token approximation is deliberately deferred: Resolve returns
// semantic hex tokens for truecolor/dark/light/high-contrast roles regardless
// of ColorANSI16 or ColorANSI256. Adapters (termenv profile, Lip Gloss, or a
// future brand/ansi package) own mapping those tokens onto the available depth.
// Only ColorNone participates in ForcesPlain(). Consumers that need to assert
// profile selection should inspect ColorDepth directly rather than expecting
// Resolve to rewrite Accent/Status hex values.
type ColorDepth uint8

const (
	ColorUnknown ColorDepth = iota
	ColorNone
	ColorANSI16
	ColorANSI256
	ColorTrueColor
)

// Capabilities describes the output environment without requiring the brand
// package to query a terminal or own a renderer lifecycle.
type Capabilities struct {
	IsTTY                 bool
	ColorDepth            ColorDepth
	DarkBackground        bool
	BackgroundKnown       bool
	Width                 int
	ReducedMotion         bool
	NoColor               bool
	ContinuousIntegration bool
	DumbTerminal          bool
}

// ForcesPlain reports whether output policy requires the renderer-neutral
// plain palette. It covers pipes, NO_COLOR, CI, TERM=dumb, and an explicit
// absence of color support.
func (c Capabilities) ForcesPlain() bool {
	return !c.IsTTY || c.ColorDepth == ColorNone || c.NoColor ||
		c.ContinuousIntegration || c.DumbTerminal
}

// AllowMotion reports whether decorative animation is permitted.
// Motion is disabled when ReducedMotion is set or when ForcesPlain() applies
// (pipes, CI, NO_COLOR, dumb terminals, no color support).
//
// Logo helpers such as FrameForCapabilities and FlameFrameFor consult this
// method so adopters do not re-implement the policy. Raw FrameFor/FlameFrame
// remain available for callers that already resolved motion themselves.
func (c Capabilities) AllowMotion() bool {
	return !c.ReducedMotion && !c.ForcesPlain()
}

// Palette contains semantic colors shared by Obedience Corp terminal UIs.
// Consumers should use roles such as Accent or StatusError instead of
// depending on individual raw color values.
type Palette struct {
	Mode         Mode
	ColorEnabled bool

	SurfaceBase     string
	SurfaceRaised   string
	TextPrimary     string
	TextMuted       string
	Accent          string
	AccentStrong    string
	AccentHighlight string
	AccentSubtle    string
	StatusSuccess   string
	StatusWarning   string
	StatusError     string
	Focus           string
	Border          string
}

// Resolve returns a palette for mode and capabilities. Explicit light, dark,
// and high-contrast modes are preserved for interactive output. Adaptive mode
// uses a known light background when available and otherwise defaults to the
// fire theme's dark expression.
//
// Runtime output policy can still force ModePlain. This keeps non-interactive
// output deterministic even when a product's configured mode is colorful.
func Resolve(mode Mode, capabilities Capabilities) Palette {
	if mode == "" {
		mode = ModeAdaptive
	}
	if mode != ModeDark && mode != ModeLight && mode != ModeHighContrast && mode != ModePlain && mode != ModeAdaptive {
		mode = ModeAdaptive
	}
	if mode != ModePlain && capabilities.ForcesPlain() {
		mode = ModePlain
	}
	if mode == ModeAdaptive {
		mode = ModeDark
		if capabilities.BackgroundKnown && !capabilities.DarkBackground {
			mode = ModeLight
		}
	}

	switch mode {
	case ModeLight:
		return lightPalette()
	case ModeHighContrast:
		return highContrastPalette()
	case ModePlain:
		return plainPalette()
	default:
		return darkPalette()
	}
}

func darkPalette() Palette {
	return Palette{
		Mode:            ModeDark,
		ColorEnabled:    true,
		SurfaceBase:     "#0D0D11",
		SurfaceRaised:   "#16161A",
		TextPrimary:     "#F8F8F2",
		TextMuted:       "#A1A1AA",
		Accent:          "#F2721C",
		AccentStrong:    "#EA5513",
		AccentHighlight: "#FFB347",
		AccentSubtle:    "#C2410C",
		StatusSuccess:   "#50FA7B",
		StatusWarning:   "#FEBC2E",
		StatusError:     "#FF5F57",
		Focus:           "#F2721C",
		Border:          "#C2410C",
	}
}

func lightPalette() Palette {
	return Palette{
		Mode:            ModeLight,
		ColorEnabled:    true,
		SurfaceBase:     "#FFFFFF",
		SurfaceRaised:   "#F4F4F5",
		TextPrimary:     "#18181B",
		TextMuted:       "#52525B",
		Accent:          "#C2410C",
		AccentStrong:    "#9A3412",
		AccentHighlight: "#B45309",
		AccentSubtle:    "#7C2D12",
		StatusSuccess:   "#166534",
		StatusWarning:   "#854D0E",
		StatusError:     "#B91C1C",
		Focus:           "#9A3412",
		Border:          "#71717A",
	}
}

func highContrastPalette() Palette {
	return Palette{
		Mode:            ModeHighContrast,
		ColorEnabled:    true,
		SurfaceBase:     "#000000",
		SurfaceRaised:   "#111111",
		TextPrimary:     "#FFFFFF",
		TextMuted:       "#E4E4E7",
		Accent:          "#FF8A4C",
		AccentStrong:    "#FFB347",
		AccentHighlight: "#FFD166",
		AccentSubtle:    "#FF8A4C",
		StatusSuccess:   "#7CFF9B",
		StatusWarning:   "#FFE08A",
		StatusError:     "#FF8080",
		Focus:           "#FFFFFF",
		Border:          "#FFFFFF",
	}
}

func plainPalette() Palette {
	return Palette{Mode: ModePlain}
}
