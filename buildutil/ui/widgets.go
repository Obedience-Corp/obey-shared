package ui

import (
	"fmt"
	"strings"
	"sync"
)

var (
	screenMutex sync.Mutex
	progressMsg string
)

// Section renders a boxed heading with rounded borders
func Section(title string) {
	screenMutex.Lock()
	defer screenMutex.Unlock()

	width := TermWidth()

	bar := strings.Repeat("─", width-2)

	fmt.Println()
	if ColourEnabled() {
		fmt.Printf("%s┌%s┐%s\n", Cyan, bar, Reset)
		fmt.Printf("%s│%s%s%s│%s\n", Cyan, Reset, Center(Bold+"🎪 "+title+Reset, width-2), Cyan, Reset)
		fmt.Printf("%s└%s┘%s\n", Cyan, bar, Reset)
	} else {
		fmt.Printf("┌%s┐\n", bar)
		fmt.Printf("│%s│\n", Center("🎪 "+title, width-2))
		fmt.Printf("└%s┘\n", bar)
	}
}

// Progress redraws an in-place progress bar
func Progress(curr, total int, msg string) {
	screenMutex.Lock()
	defer screenMutex.Unlock()

	if ColourEnabled() {
		fmt.Printf("\r%s[%d/%d] %s", ClearLine, curr, total, msg)
	} else {
		fmt.Printf("\r[%d/%d] %s", curr, total, msg)
	}

	progressMsg = msg
}

// Status prints a status indicator with checkmark or X
func Status(label string, ok bool) {
	screenMutex.Lock()
	defer screenMutex.Unlock()

	symbol := "✗"
	color := Red
	if ok {
		symbol = "✓"
		color = Green
	}

	if ColourEnabled() {
		fmt.Printf("  %s%s%s %s\n", color, symbol, Reset, label)
	} else {
		fmt.Printf("  %s %s\n", symbol, label)
	}
}

// ClearProgress clears any in-progress output
func ClearProgress() {
	screenMutex.Lock()
	defer screenMutex.Unlock()

	if progressMsg != "" {
		if ColourEnabled() {
			fmt.Printf("\r%s\n", ClearLine)
		} else {
			fmt.Println()
		}
		progressMsg = ""
	}
}

// ProgressWithOutput updates two lines: output line above, progress line below
func ProgressWithOutput(curr, total int, output, progress string) {
	screenMutex.Lock()
	defer screenMutex.Unlock()

	if ColourEnabled() {
		width := TermWidth()
		truncatedOutput := strings.TrimSpace(output)
		maxLen := width - 6
		if len(truncatedOutput) > maxLen {
			truncatedOutput = truncatedOutput[:maxLen-3] + "..."
		}

		fmt.Printf("\r%s%s  → %s\n\r%s[%d/%d] %s",
			MoveUp, ClearLine, truncatedOutput,
			ClearLine, curr, total, progress)
	} else {
		fmt.Printf("\r[%d/%d] %s", curr, total, progress)
	}

	progressMsg = progress
}

// ClearProgressWithOutput clears the two-line output display
func ClearProgressWithOutput() {
	screenMutex.Lock()
	defer screenMutex.Unlock()

	if ColourEnabled() {
		fmt.Printf("%s%s\n%s\n", MoveUp, ClearLine, ClearLine)
	} else {
		fmt.Println()
	}
	progressMsg = ""
}

// Task displays a task in progress
func Task(action, description string) {
	screenMutex.Lock()
	defer screenMutex.Unlock()

	if ColourEnabled() {
		fmt.Printf("  %s[%s]%s %s... ", Yellow, action, Reset, description)
	} else {
		fmt.Printf("  [%s] %s... ", action, description)
	}
}

// TaskPass marks the current task as passed
func TaskPass() {
	if ColourEnabled() {
		fmt.Printf("%s✓%s\n", Green, Reset)
	} else {
		fmt.Println("✓")
	}
}

// TaskFail marks the current task as failed
func TaskFail() {
	if ColourEnabled() {
		fmt.Printf("%s✗%s\n", Red, Reset)
	} else {
		fmt.Println("✗")
	}
}

// Success prints a success message
func Success(msg string) {
	screenMutex.Lock()
	defer screenMutex.Unlock()

	if ColourEnabled() {
		fmt.Printf("%s✓ %s%s\n", Green, msg, Reset)
	} else {
		fmt.Printf("✓ %s\n", msg)
	}
}

// Warning prints a warning message
func Warning(msg string) {
	screenMutex.Lock()
	defer screenMutex.Unlock()

	if ColourEnabled() {
		fmt.Printf("%s⚠ %s%s\n", Yellow, msg, Reset)
	} else {
		fmt.Printf("⚠ %s\n", msg)
	}
}

// SummaryCard displays a final status card
func SummaryCard(title string, rows [][]string, totalTime string, success bool) {
	SummaryCardWithStatus(title, rows, totalTime, success, "", "")
}

// SummaryCardWithStatus displays a final status card with custom status messages
func SummaryCardWithStatus(title string, rows [][]string, totalTime string, success bool, successMsg, failMsg string) {
	screenMutex.Lock()
	defer screenMutex.Unlock()

	width := TermWidth()

	bar := strings.Repeat("═", width-2)

	// Top border
	fmt.Println()
	if ColourEnabled() {
		fmt.Printf("%s╔%s╗%s\n", Cyan, bar, Reset)
		fmt.Printf("%s║%s%s%s║%s\n", Cyan, Reset, Center(Bold+"🎪 "+title+Reset, width-2), Cyan, Reset)
		fmt.Printf("%s╠%s╣%s\n", Cyan, bar, Reset)
	} else {
		fmt.Printf("╔%s╗\n", bar)
		fmt.Printf("║%s║\n", Center("🎪 "+title, width-2))
		fmt.Printf("╠%s╣\n", bar)
	}

	// Calculate column widths
	if len(rows) > 0 {
		colWidths := make([]int, len(rows[0]))
		for _, row := range rows {
			for i, cell := range row {
				if cellLen := VisualLength(cell); cellLen > colWidths[i] {
					colWidths[i] = cellLen
				}
			}
		}

		// Render rows
		for i, row := range rows {
			if ColourEnabled() {
				fmt.Printf("%s║%s ", Cyan, Reset)
			} else {
				fmt.Print("║ ")
			}

			totalColWidth := 0
			for _, cw := range colWidths {
				totalColWidth += cw
			}

			spaceBetweenCols := (len(colWidths) - 1) * 2
			contentWidth := totalColWidth + spaceBetweenCols

			availableSpace := width - 4
			extraSpace := availableSpace - contentWidth
			if extraSpace > 0 && len(colWidths) > 0 {
				colWidths[len(colWidths)-1] += extraSpace
			}

			for j, cell := range row {
				fmt.Print(cell)
				padding := colWidths[j] - VisualLength(cell)
				if j < len(row)-1 {
					padding += 2
				}
				fmt.Print(strings.Repeat(" ", padding))
			}

			if ColourEnabled() {
				fmt.Printf(" %s║%s\n", Cyan, Reset)
			} else {
				fmt.Println(" ║")
			}

			if i == 0 {
				if ColourEnabled() {
					fmt.Printf("%s╠%s╣%s\n", Cyan, bar, Reset)
				} else {
					fmt.Printf("╠%s╣\n", bar)
				}
			}
		}
	}

	// Time summary
	if ColourEnabled() {
		fmt.Printf("%s╠%s╣%s\n", Cyan, bar, Reset)
		fmt.Printf("%s║%s%s%s║%s\n", Cyan, Reset, Center("Total Time: "+totalTime, width-2), Cyan, Reset)
	} else {
		fmt.Printf("╠%s╣\n", bar)
		fmt.Printf("║%s║\n", Center("Total Time: "+totalTime, width-2))
	}

	// Status
	if ColourEnabled() {
		fmt.Printf("%s╠%s╣%s\n", Cyan, bar, Reset)
	} else {
		fmt.Printf("╠%s╣\n", bar)
	}

	var statusText string
	if success {
		if successMsg != "" {
			statusText = successMsg
		} else {
			statusText = "✓ BUILD SUCCESSFUL"
		}
	} else {
		if failMsg != "" {
			statusText = failMsg
		} else {
			statusText = "✗ BUILD FAILED"
		}
	}

	if ColourEnabled() {
		statusColor := Green
		if !success {
			statusColor = Red
		}
		statusLine := Center(Bold+statusColor+statusText+Reset, width-2)
		fmt.Printf("%s║%s%s%s║%s\n", Cyan, Reset, statusLine, Cyan, Reset)
	} else {
		fmt.Printf("║%s║\n", Center(statusText, width-2))
	}

	// Bottom border
	if ColourEnabled() {
		fmt.Printf("%s╚%s╝%s\n\n", Cyan, bar, Reset)
	} else {
		fmt.Printf("╚%s╝\n\n", bar)
	}
}
