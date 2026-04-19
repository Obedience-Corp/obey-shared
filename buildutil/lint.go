package buildutil

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Obedience-Corp/obey-shared/buildutil/ui"
)

// doLint runs go vet on all packages and renders a dashboard of results.
func doLint(cfg BuildConfig, verbose bool) error {
	ui.Section("Linting " + cfg.SectionName)

	packages, err := discoverPackages()
	if err != nil {
		return fmt.Errorf("failed to discover packages: %w", err)
	}

	if verbose {
		fmt.Printf("Found %d packages\n", len(packages))
	}

	type lintResult struct {
		Package  string
		Pass     bool
		Duration time.Duration
		Output   string
	}

	results := make([]lintResult, 0, len(packages))
	total := len(packages)
	failures := 0

	vetArgs := []string{"vet"}
	vetArgs = append(vetArgs, buildTagsArgs(cfg)...)

	for i, pkg := range packages {
		shortName := strings.TrimPrefix(pkg, "./")
		if shortName == "." {
			shortName = "root"
		}

		ui.Progress(i+1, total, fmt.Sprintf("Vetting %s", shortName))

		args := append([]string{}, vetArgs...)
		args = append(args, pkg)

		start := time.Now()
		cmd := exec.Command("go", args...)
		out, runErr := cmd.CombinedOutput()
		duration := time.Since(start)

		pass := runErr == nil
		if !pass {
			failures++
		}

		results = append(results, lintResult{
			Package:  shortName,
			Pass:     pass,
			Duration: duration,
			Output:   string(out),
		})

		if verbose && !pass {
			fmt.Fprintf(os.Stderr, "--- vet failures in %s ---\n%s\n", shortName, out)
		}
	}

	ui.ClearProgress()

	var totalTime time.Duration
	for _, r := range results {
		totalTime += r.Duration
	}

	rows := [][]string{}
	for _, r := range results {
		if r.Pass {
			continue
		}
		status := "✗ vet failed"
		if ui.ColourEnabled() {
			status = ui.Red + status + ui.Reset
		}
		rows = append(rows, []string{
			r.Package,
			status,
			fmt.Sprintf("%.2fs", r.Duration.Seconds()),
		})
	}

	if failures > 0 {
		rows = append([][]string{{"Package", "Status", "Time"}}, rows...)
	}

	summary := fmt.Sprintf("%d/%d packages clean", total-failures, total)
	if ui.ColourEnabled() {
		if failures > 0 {
			summary = ui.Red + summary + ui.Reset
		} else {
			summary = ui.Green + summary + ui.Reset
		}
	}
	rows = append(rows, []string{
		fmt.Sprintf("%d packages", total),
		summary,
		fmt.Sprintf("%.2fs", totalTime.Seconds()),
	})

	success := failures == 0
	title := "Lint Complete - No Issues"
	if !success {
		title = "Lint Failures"
	}
	ui.SummaryCardWithStatus(title, rows, fmt.Sprintf("%.2fs", totalTime.Seconds()), success,
		fmt.Sprintf("✓ ALL %d PACKAGES CLEAN", total),
		fmt.Sprintf("✗ %d/%d PACKAGES WITH VET ISSUES", failures, total))

	if !success {
		if !verbose {
			for _, r := range results {
				if !r.Pass {
					fmt.Fprintf(os.Stderr, "--- vet failures in %s ---\n%s\n", r.Package, r.Output)
				}
			}
		}
		return fmt.Errorf("%d packages had vet issues", failures)
	}

	return nil
}
