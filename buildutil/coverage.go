package buildutil

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Obedience-Corp/obey-shared/buildutil/ui"
)

const (
	coverageDir     = "coverage"
	coverageProfile = "coverage/coverage.out"
	coverageHTML    = "coverage/coverage.html"
)

// doCoverage runs `go test -coverprofile` across all test packages and
// renders an HTML report. Integration test directories are excluded.
func doCoverage(cfg BuildConfig, verbose bool) error {
	ui.Section("Coverage " + cfg.SectionName)

	packages, err := discoverTestPackages()
	if err != nil {
		return fmt.Errorf("failed to discover test packages: %w", err)
	}
	if len(packages) == 0 {
		ui.SummaryCard("No Test Packages", [][]string{{"Result", "None"}}, "< 1s", true)
		return nil
	}

	if err := os.MkdirAll(coverageDir, 0o755); err != nil {
		return fmt.Errorf("failed to create coverage directory: %w", err)
	}

	ui.Task("Running", fmt.Sprintf("go test -coverprofile on %d packages", len(packages)))

	args := []string{"test", "-count=1", "-coverprofile=" + coverageProfile}
	args = append(args, buildTagsArgs(cfg)...)
	args = append(args, packages...)

	start := time.Now()
	cmd := exec.Command("go", args...)
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	runErr := cmd.Run()
	duration := time.Since(start)

	if runErr != nil {
		ui.TaskFail()
		return fmt.Errorf("coverage run failed: %w", runErr)
	}
	ui.TaskPass()

	pct, pctErr := totalCoveragePercent()

	ui.Task("Rendering", "HTML report")
	htmlCmd := exec.Command("go", "tool", "cover", "-html="+coverageProfile, "-o", coverageHTML)
	if verbose {
		htmlCmd.Stdout = os.Stdout
		htmlCmd.Stderr = os.Stderr
	}
	if err := htmlCmd.Run(); err != nil {
		ui.TaskFail()
		return fmt.Errorf("failed to render HTML coverage: %w", err)
	}
	ui.TaskPass()

	pctDisplay := "n/a"
	if pctErr == nil {
		pctDisplay = fmt.Sprintf("%.1f%%", pct)
		if ui.ColourEnabled() {
			pctDisplay = ui.Green + pctDisplay + ui.Reset
		}
	}

	rows := [][]string{
		{"Metric", "Value"},
		{"Total coverage", pctDisplay},
		{"Profile", coverageProfile},
		{"HTML report", coverageHTML},
	}

	ui.SummaryCard("Coverage Complete", rows, fmt.Sprintf("%.2fs", duration.Seconds()), true)
	return nil
}

// totalCoveragePercent shells out to `go tool cover -func` to obtain
// the total coverage percentage from the generated profile.
func totalCoveragePercent() (float64, error) {
	cmd := exec.Command("go", "tool", "cover", "-func="+coverageProfile)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		if !strings.HasPrefix(line, "total:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pctStr := strings.TrimSuffix(fields[len(fields)-1], "%")
		var pct float64
		if _, err := fmt.Sscanf(pctStr, "%f", &pct); err != nil {
			return 0, err
		}
		return pct, nil
	}
	return 0, fmt.Errorf("total line not found in coverage output")
}
