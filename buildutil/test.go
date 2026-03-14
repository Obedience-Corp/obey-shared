package buildutil

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Obedience-Corp/obey-shared/buildutil/ui"
)

// doTest runs go test on all packages with JSON parsing and visual dashboard.
func doTest(cfg BuildConfig, verbose bool) error {
	ui.Section("Testing " + cfg.SectionName)

	packages, err := discoverTestPackages()
	if err != nil {
		return fmt.Errorf("failed to discover test packages: %w", err)
	}

	if verbose {
		fmt.Printf("Found %d packages with tests\n", len(packages))
	}

	results := make([]testResult, 0, len(packages))
	total := len(packages)
	pkgFailures := 0

	for i, pkg := range packages {
		shortName := strings.TrimPrefix(pkg, "./")
		if shortName == "." {
			shortName = "root"
		}

		ui.Progress(i+1, total, fmt.Sprintf("Testing %s", shortName))

		start := time.Now()

		cmd := exec.Command("go", "test", "-json", "-count=1", "-short", "-timeout", "30s", pkg)
		output, _ := cmd.Output()
		duration := time.Since(start)

		testsPassed, testsFailed := parseTestOutput(output, verbose)
		pass := testsFailed == 0

		results = append(results, testResult{
			Package:     shortName,
			Pass:        pass,
			Duration:    duration,
			TestsPassed: testsPassed,
			TestsFailed: testsFailed,
		})

		if !pass {
			pkgFailures++
		}
	}

	ui.ClearProgress()

	// Calculate totals
	var totalTime time.Duration
	totalTestsPassed := 0
	totalTestsFailed := 0
	pkgsPassed := 0

	for _, r := range results {
		totalTime += r.Duration
		totalTestsPassed += r.TestsPassed
		totalTestsFailed += r.TestsFailed
		if r.Pass {
			pkgsPassed++
		}
	}

	// Display summary - only show packages with failures
	rows := [][]string{}
	hasFailures := pkgFailures > 0

	for _, r := range results {
		if !r.Pass {
			status := fmt.Sprintf("✗ %d failed", r.TestsFailed)
			if ui.ColourEnabled() {
				status = ui.Red + status + ui.Reset
			}

			rows = append(rows, []string{
				r.Package,
				status,
				fmt.Sprintf("%.2fs", r.Duration.Seconds()),
			})
		}
	}

	if hasFailures {
		rows = append([][]string{{"Package", "Status", "Time"}}, rows...)
	}

	totalTests := totalTestsPassed + totalTestsFailed
	totalStatus := fmt.Sprintf("%d/%d tests passed", totalTestsPassed, totalTests)
	if ui.ColourEnabled() {
		if totalTestsFailed > 0 {
			totalStatus = ui.Red + totalStatus + ui.Reset
		} else {
			totalStatus = ui.Green + totalStatus + ui.Reset
		}
	}

	rows = append(rows, []string{
		fmt.Sprintf("%d packages", len(results)),
		totalStatus,
		fmt.Sprintf("%.2fs", totalTime.Seconds()),
	})

	success := pkgFailures == 0
	var title string
	if hasFailures {
		title = "Test Failures"
	} else {
		title = "Tests Complete - All Passed"
	}

	successMsg := fmt.Sprintf("✓ ALL %d TESTS PASSED", totalTestsPassed)
	failMsg := fmt.Sprintf("✗ %d/%d TESTS FAILED", totalTestsFailed, totalTests)

	ui.SummaryCardWithStatus(title, rows, fmt.Sprintf("%.2fs", totalTime.Seconds()), success, successMsg, failMsg)

	if pkgFailures > 0 {
		return fmt.Errorf("%d packages had test failures (%d tests failed)", pkgFailures, totalTestsFailed)
	}

	return nil
}

// parseTestOutput parses go test -json output and returns pass/fail counts.
func parseTestOutput(output []byte, verbose bool) (passed, failed int) {
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event testEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		// Only count actual test results (not package-level or sub-tests)
		if event.Test != "" && !strings.Contains(event.Test, "/") {
			switch event.Action {
			case "pass":
				passed++
			case "fail":
				failed++
				if verbose {
					fmt.Printf("  FAIL: %s\n", event.Test)
				}
			}
		}
	}

	return passed, failed
}

// testEvent represents a single line of go test -json output.
type testEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
	Output  string  `json:"Output"`
}

// testResult tracks test results for a package.
type testResult struct {
	Package     string
	Pass        bool
	Duration    time.Duration
	TestsPassed int
	TestsFailed int
}
