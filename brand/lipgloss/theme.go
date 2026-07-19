// Package lipgloss adapts brand's renderer-neutral palette for Charm-based
// terminal user interfaces.
package lipgloss

import (
	"strings"

	"github.com/Obedience-Corp/obey-shared/brand"
	charmgloss "github.com/charmbracelet/lipgloss"
)

// Styles holds semantic Lip Gloss styles for shared Obedience Corp TUIs.
type Styles struct {
	Title      charmgloss.Style
	Subtitle   charmgloss.Style
	Muted      charmgloss.Style
	Fire       charmgloss.Style
	FireCore   charmgloss.Style
	FireTip    charmgloss.Style
	Selected   charmgloss.Style
	Normal     charmgloss.Style
	OK         charmgloss.Style
	Warn       charmgloss.Style
	Err        charmgloss.Style
	Border     charmgloss.Style
	Header     charmgloss.Style
	Footer     charmgloss.Style
	StatusOK   charmgloss.Style
	StatusWarn charmgloss.Style
	StatusFail charmgloss.Style
	Tagline    charmgloss.Style
}

// New creates semantic styles from a resolved shared palette.
func New(palette brand.Palette) Styles {
	return Styles{
		Title:      textStyle(palette.TextPrimary, true),
		Subtitle:   textStyle(palette.TextMuted, false),
		Muted:      textStyle(palette.TextMuted, false),
		Fire:       textStyle(palette.Accent, true),
		FireCore:   textStyle(palette.AccentStrong, false),
		FireTip:    textStyle(palette.AccentHighlight, false),
		Selected:   textStyle(palette.Accent, true).PaddingLeft(1),
		Normal:     textStyle(palette.TextPrimary, false).PaddingLeft(1),
		OK:         textStyle(palette.StatusSuccess, false),
		Warn:       textStyle(palette.StatusWarning, false),
		Err:        textStyle(palette.StatusError, false),
		Border:     textStyle(palette.Border, false),
		Header:     textStyle(palette.TextPrimary, false),
		Footer:     textStyle(palette.TextMuted, false),
		StatusOK:   textStyle(palette.StatusSuccess, false),
		StatusWarn: textStyle(palette.StatusWarning, false),
		StatusFail: textStyle(palette.StatusError, false),
		Tagline:    textStyle(palette.TextMuted, false).Italic(palette.ColorEnabled),
	}
}

// Default creates the dark fire theme for an interactive true-color TUI.
// Consumers that need runtime fallback behavior should call brand.Resolve
// with their capabilities and pass the result to New.
func Default() Styles {
	return New(brand.Resolve(brand.ModeDark, brand.Capabilities{
		IsTTY:      true,
		ColorDepth: brand.ColorTrueColor,
	}))
}

// Rule draws a bounded horizontal fire rule.
func Rule(width int, styles Styles) string {
	if width < 1 {
		width = 1
	}
	if width > 300 {
		width = 300
	}
	return styles.Border.MaxWidth(width).Render(strings.Repeat("─", width))
}

func textStyle(color string, bold bool) charmgloss.Style {
	style := charmgloss.NewStyle()
	if color != "" {
		style = style.Foreground(charmgloss.Color(color))
	}
	if bold {
		style = style.Bold(true)
	}
	return style
}
