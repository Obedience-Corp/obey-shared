// Package mdrender provides TTY-aware markdown rendering for CLI tools.
//
// When stdout is a TTY, markdown is rendered with syntax highlighting, word
// wrapping, and styled headings via glamour. When stdout is piped, in CI, or
// NO_COLOR is set, the raw markdown is returned unchanged so agents and
// downstream tools receive unmodified content.
package mdrender

import (
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"golang.org/x/term"
)

// Option configures the markdown renderer.
type Option func(*config)

type config struct {
	width    int    // 0 = auto-detect from terminal
	style    string // "auto", "dark", "light"
	forceTTY *bool  // nil = auto-detect; true/false = override
	writer   io.Writer
}

// WithWidth overrides automatic terminal width detection.
func WithWidth(width int) Option {
	return func(c *config) { c.width = width }
}

// WithStyle forces a specific glamour style. Accepted values: "dark", "light", "auto".
// Default is "auto" which detects terminal background color.
func WithStyle(style string) Option {
	return func(c *config) { c.style = style }
}

// WithForceTTY forces TTY-mode rendering regardless of actual terminal state.
// Useful for TUI contexts or testing.
func WithForceTTY(force bool) Option {
	return func(c *config) { c.forceTTY = &force }
}

// WithForceRaw forces raw markdown passthrough regardless of terminal state.
// Useful when the caller knows the consumer is an agent.
func WithForceRaw(force bool) Option {
	return func(c *config) {
		f := !force
		c.forceTTY = &f
	}
}

// WithWriter sets the writer whose file descriptor is used for TTY detection
// instead of os.Stdout.
func WithWriter(w io.Writer) Option {
	return func(c *config) { c.writer = w }
}

// Render renders markdown content with TTY awareness.
//
// When the output target is a TTY, the content is rendered through glamour
// with syntax highlighting, styled headings, and word wrapping. When the
// output is piped, in CI, or NO_COLOR is set, the raw markdown is returned
// unchanged.
func Render(markdown string, opts ...Option) string {
	if markdown == "" {
		return ""
	}

	cfg := config{style: "auto"}
	for _, opt := range opts {
		opt(&cfg)
	}

	if !shouldRender(cfg) {
		return markdown
	}

	width := cfg.width
	if width <= 0 {
		width = detectWidth(cfg.writer)
	}

	rendered, err := renderGlamour(markdown, width, cfg.style)
	if err != nil {
		return markdown
	}
	return strings.TrimRight(rendered, "\n")
}

func shouldRender(cfg config) bool {
	if cfg.forceTTY != nil {
		return *cfg.forceTTY
	}

	if os.Getenv("NO_COLOR") != "" || os.Getenv("CI") != "" {
		return false
	}

	return isTTY(cfg.writer)
}

func isTTY(w io.Writer) bool {
	if w != nil {
		if f, ok := w.(*os.File); ok {
			return term.IsTerminal(int(f.Fd()))
		}
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func detectWidth(w io.Writer) int {
	fd := int(os.Stdout.Fd())
	if w != nil {
		if f, ok := w.(*os.File); ok {
			fd = int(f.Fd())
		}
	}

	width, _, err := term.GetSize(fd)
	if err != nil || width <= 0 {
		return 80
	}
	return width
}

func renderGlamour(markdown string, width int, style string) (string, error) {
	glamourOpts := []glamour.TermRendererOption{
		glamour.WithWordWrap(width - 4),
	}

	switch style {
	case "dark":
		glamourOpts = append(glamourOpts, glamour.WithStyles(styles.DarkStyleConfig))
	case "light":
		glamourOpts = append(glamourOpts, glamour.WithStyles(styles.LightStyleConfig))
	default:
		glamourOpts = append(glamourOpts, glamour.WithAutoStyle())
	}

	renderer, err := glamour.NewTermRenderer(glamourOpts...)
	if err != nil {
		return "", err
	}

	return renderer.Render(markdown)
}
