package lipgloss

import (
	"math"
	"strings"

	"github.com/Obedience-Corp/obey-shared/brand"
)

// SmallFlame renders a compact two-line fire mark for reduced motion or tight
// headers.
func SmallFlame(styles Styles) string {
	frame := brand.SmallFlame()
	lines := frame.Lines()
	return styles.FireTip.Render(lines[0]) + "\n" + styles.Fire.Render(lines[1])
}

// Flame renders a deterministic breathing flame at frame index. heightScale
// is clamped to 0..1 and controls how many bottom lines are shown.
func Flame(frame int, heightScale float64, styles Styles) string {
	if heightScale < 0 {
		heightScale = 0
	}
	if heightScale > 1 {
		heightScale = 1
	}

	lines := brand.FlameFrame(frame).Lines()
	lineCount := 2 + int(float64(len(lines)-2)*heightScale)
	if lineCount > len(lines) {
		lineCount = len(lines)
	}
	start := len(lines) - lineCount

	var rendered strings.Builder
	for i := start; i < len(lines); i++ {
		relative := float64(i-start) / float64(max(lineCount-1, 1))
		style := styles.FireCore
		switch {
		case relative < 0.35:
			style = styles.FireTip
		case relative < 0.7:
			style = styles.Fire
		}
		rendered.WriteString(style.Render(lines[i]))
		if i < len(lines)-1 {
			rendered.WriteByte('\n')
		}
	}
	return rendered.String()
}

// StaticFlame renders the full non-animated flame.
func StaticFlame(styles Styles) string {
	return Flame(0, 1, styles)
}

// Wordmark renders the shared Festival wordmark.
func Wordmark(styles Styles) string {
	return styles.Fire.Render(brand.Wordmark)
}

// SparkField describes a deterministic field of drifting embers. Tick
// ownership and cancellation remain with the consuming TUI.
type SparkField struct {
	Width  int
	Height int
	Tick   int
}

// Render draws the spark field without querying terminal state or starting
// background work.
func (field SparkField) Render(styles Styles) string {
	if field.Width < 4 || field.Height < 1 {
		return ""
	}

	lines := make([]string, 0, field.Height)
	for y := 0; y < field.Height; y++ {
		row := make([]rune, field.Width)
		for x := range row {
			row[x] = ' '
		}
		for i := 0; i < 3; i++ {
			phase := field.Tick*2 + y*7 + i*13
			x := int(math.Abs(float64(phase*3+y))) % field.Width
			glyphs := []rune{'.', '·', '*', '°'}
			row[x] = glyphs[positiveMod(phase+i, len(glyphs))]
		}
		line := string(row)
		if (y+field.Tick)%2 == 0 {
			lines = append(lines, styles.FireTip.Render(line))
		} else {
			lines = append(lines, styles.FireCore.Render(line))
		}
	}
	return strings.Join(lines, "\n")
}

// Celebrate renders a short deterministic spark burst and status line.
func Celebrate(frame int, styles Styles) string {
	patterns := []string{
		"  *  .  *  .  *  ",
		" .  *  °  *  .  ",
		"*  °  *  .  *  °",
		"  .  *  .  *  . ",
	}
	if frame < 0 {
		frame = 0
	}
	return styles.FireTip.Render(patterns[frame%len(patterns)]) + "\n" + styles.OK.Render("  ready — the fire is lit")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func positiveMod(value, divisor int) int {
	result := value % divisor
	if result < 0 {
		return result + divisor
	}
	return result
}
