package buildutil

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Obedience-Corp/obey-shared/buildutil/ui"
)

// doBuild runs go vet and go build on all packages, then builds the main binary.
func doBuild(cfg BuildConfig, verbose bool) error {
	ui.Section("Building " + cfg.SectionName)

	packages, err := discoverPackages()
	if err != nil {
		return fmt.Errorf("failed to discover packages: %w", err)
	}

	if verbose {
		fmt.Printf("Found %d packages\n", len(packages))
	}

	results := make([]packageResult, 0, len(packages))
	total := len(packages)

	for i, pkg := range packages {
		shortName := strings.TrimPrefix(pkg, "./")
		if shortName == "." {
			shortName = "root"
		}

		result := packageResult{Package: shortName}

		// Vet
		ui.Progress(i+1, total, fmt.Sprintf("Vetting %s", shortName))
		start := time.Now()
		cmd := exec.Command("go", "vet", pkg)
		if verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		result.VetPass = cmd.Run() == nil
		result.VetTime = time.Since(start)

		// Build
		ui.Progress(i+1, total, fmt.Sprintf("Building %s", shortName))
		start = time.Now()
		cmd = exec.Command("go", "build", "-o", os.DevNull, pkg)
		if verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		result.BuildPass = cmd.Run() == nil
		result.BuildTime = time.Since(start)

		results = append(results, result)
	}

	// Build main binary
	ui.Progress(total, total, "Building main binary")
	start := time.Now()

	if err := os.MkdirAll("bin", 0o755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	buildArgs := []string{"build"}
	buildArgs = append(buildArgs, buildTagsArgs(cfg)...)

	// Add ldflags if configured
	if cfg.LDFlags != nil {
		if ldflags := cfg.LDFlags(); ldflags != "" {
			buildArgs = append(buildArgs, "-ldflags", ldflags)
		}
	}

	outputPath := fmt.Sprintf("bin/%s", cfg.BinaryName)
	buildArgs = append(buildArgs, "-o", outputPath, cfg.MainPath)

	cmd := exec.Command("go", buildArgs...)
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	mainBuildSuccess := cmd.Run() == nil
	mainBuildTime := time.Since(start)

	ui.ClearProgress()

	results = append(results, packageResult{
		Package:   fmt.Sprintf("bin/%s", cfg.BinaryName),
		VetPass:   true,
		BuildPass: mainBuildSuccess,
		BuildTime: mainBuildTime,
	})

	// Calculate totals
	var totalTime time.Duration
	for _, r := range results {
		totalTime += r.VetTime + r.BuildTime
	}

	// Display summary - only show packages with errors
	rows := [][]string{}
	hasFailures := false

	for _, r := range results {
		if !r.VetPass || !r.BuildPass {
			hasFailures = true

			vetStatus := "✓"
			if !r.VetPass {
				vetStatus = "✗"
			}
			if ui.ColourEnabled() {
				if r.VetPass {
					vetStatus = ui.Green + vetStatus + ui.Reset
				} else {
					vetStatus = ui.Red + vetStatus + ui.Reset
				}
			}

			buildStatus := "✓"
			if !r.BuildPass {
				buildStatus = "✗"
			}
			if ui.ColourEnabled() {
				if r.BuildPass {
					buildStatus = ui.Green + buildStatus + ui.Reset
				} else {
					buildStatus = ui.Red + buildStatus + ui.Reset
				}
			}

			rows = append(rows, []string{
				r.Package,
				fmt.Sprintf("%s %.2fs", vetStatus, r.VetTime.Seconds()),
				fmt.Sprintf("%s %.2fs", buildStatus, r.BuildTime.Seconds()),
			})
		}
	}

	if hasFailures {
		rows = append([][]string{{"Package", "Vet", "Build"}}, rows...)
	}

	var title string
	if hasFailures {
		title = "Build Failures"
	} else {
		title = "Build Complete - No Errors"
	}

	ui.SummaryCard(title, rows, fmt.Sprintf("%.2fs", totalTime.Seconds()), !hasFailures)

	if hasFailures {
		failedPackages := []string{}
		for _, r := range results {
			if !r.VetPass || !r.BuildPass {
				failedPackages = append(failedPackages, r.Package)
			}
		}
		return fmt.Errorf("build failed for packages: %s", strings.Join(failedPackages, ", "))
	}

	return nil
}

// doBuildOnly builds the main binary without running go vet (fast user installation).
func doBuildOnly(cfg BuildConfig, verbose bool) error {
	ui.Section("Building " + cfg.SectionName)

	if err := os.MkdirAll("bin", 0o755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	ui.Task("Building", cfg.BinaryName+" binary")

	buildArgs := []string{"build"}
	buildArgs = append(buildArgs, buildTagsArgs(cfg)...)

	if cfg.LDFlags != nil {
		if ldflags := cfg.LDFlags(); ldflags != "" {
			buildArgs = append(buildArgs, "-ldflags", ldflags)
		}
	}

	outputPath := fmt.Sprintf("bin/%s", cfg.BinaryName)
	buildArgs = append(buildArgs, "-o", outputPath, cfg.MainPath)

	cmd := exec.Command("go", buildArgs...)
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		ui.TaskFail()
		return fmt.Errorf("failed to build %s binary: %w", cfg.BinaryName, err)
	}

	ui.TaskPass()

	ui.SummaryCard(
		"Build Complete - No Errors",
		[][]string{
			{"Task", "Status"},
			{"Binary Build", ui.Green + "✓ Complete" + ui.Reset},
		},
		"< 5s",
		true,
	)

	return nil
}

// packageResult tracks build results for a package.
type packageResult struct {
	Package   string
	VetPass   bool
	BuildPass bool
	VetTime   time.Duration
	BuildTime time.Duration
}
