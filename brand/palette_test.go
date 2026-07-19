package brand

import "testing"

func TestParseMode(t *testing.T) {
	tests := []struct {
		value string
		want  Mode
	}{
		{value: "dark", want: ModeDark},
		{value: "light", want: ModeLight},
		{value: "high-contrast", want: ModeHighContrast},
		{value: "plain", want: ModePlain},
		{value: "", want: ModeAdaptive},
		{value: "neon", want: ModeAdaptive},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := ParseMode(tt.value); got != tt.want {
				t.Fatalf("ParseMode(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestResolveModes(t *testing.T) {
	interactiveDark := Capabilities{IsTTY: true, ColorDepth: ColorTrueColor, DarkBackground: true, BackgroundKnown: true}
	interactiveLight := Capabilities{IsTTY: true, ColorDepth: ColorANSI256, BackgroundKnown: true}

	tests := []struct {
		name  string
		mode  Mode
		caps  Capabilities
		want  Mode
		color string
	}{
		{name: "dark", mode: ModeDark, caps: interactiveDark, want: ModeDark, color: "#F2721C"},
		{name: "adaptive dark", mode: ModeAdaptive, caps: interactiveDark, want: ModeDark, color: "#F2721C"},
		{name: "adaptive light", mode: ModeAdaptive, caps: interactiveLight, want: ModeLight, color: "#C2410C"},
		{name: "adaptive unknown background", mode: ModeAdaptive, caps: Capabilities{IsTTY: true}, want: ModeDark, color: "#F2721C"},
		{name: "high contrast", mode: ModeHighContrast, caps: interactiveDark, want: ModeHighContrast, color: "#FF8A4C"},
		{name: "pipe forces plain", mode: ModeDark, caps: Capabilities{ColorDepth: ColorTrueColor}, want: ModePlain},
		{name: "no color forces plain", mode: ModeDark, caps: Capabilities{IsTTY: true, ColorDepth: ColorTrueColor, NoColor: true}, want: ModePlain},
		{name: "ci forces plain", mode: ModeLight, caps: Capabilities{IsTTY: true, ColorDepth: ColorTrueColor, ContinuousIntegration: true}, want: ModePlain},
		{name: "dumb terminal forces plain", mode: ModeDark, caps: Capabilities{IsTTY: true, ColorDepth: ColorTrueColor, DumbTerminal: true}, want: ModePlain},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Resolve(tt.mode, tt.caps)
			if got.Mode != tt.want {
				t.Fatalf("Resolve(...).Mode = %q, want %q", got.Mode, tt.want)
			}
			if tt.color != "" && got.Accent != tt.color {
				t.Fatalf("Resolve(...).Accent = %q, want %q", got.Accent, tt.color)
			}
			if got.ColorEnabled != (tt.want != ModePlain) {
				t.Fatalf("Resolve(...).ColorEnabled = %t for mode %q", got.ColorEnabled, tt.want)
			}
		})
	}
}

func TestPlainPaletteHasNoColorTokens(t *testing.T) {
	p := Resolve(ModePlain, Capabilities{IsTTY: true, ColorDepth: ColorTrueColor})
	if p.ColorEnabled {
		t.Fatal("plain palette should disable color")
	}
	if p.SurfaceBase != "" || p.TextPrimary != "" || p.Accent != "" || p.StatusError != "" {
		t.Fatalf("plain palette contains color tokens: %#v", p)
	}
}

func TestEnvironmentCapabilities(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CI", "1")
	t.Setenv("TERM", "dumb")
	t.Setenv("OBEY_REDUCED_MOTION", "")
	t.Setenv("FESTIVAL_REDUCED_MOTION", "yes")

	caps := EnvironmentCapabilities(true, ColorANSI256)
	if !caps.NoColor || !caps.ContinuousIntegration || !caps.DumbTerminal || !caps.ReducedMotion {
		t.Fatalf("environment capabilities missed policy: %#v", caps)
	}
}

func TestReducedMotionSharedName(t *testing.T) {
	t.Setenv("OBEY_REDUCED_MOTION", "on")
	t.Setenv("FESTIVAL_REDUCED_MOTION", "")
	if !ReducedMotion() {
		t.Fatal("OBEY_REDUCED_MOTION should enable reduced motion")
	}
}
