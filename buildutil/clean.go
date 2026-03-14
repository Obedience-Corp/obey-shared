package buildutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Obedience-Corp/obey-shared/buildutil/ui"
)

var defaultCleanPatterns = []string{
	"bin/",
	"*.test",
	"*.exe",
	"coverage.out",
	"coverage_*.out",
	".test-*",
	"*.disabled",
	"*.old",
	"*.wip",
	"*.backup",
	"*.tmp",
	"*.bak",
	"*~",
	".test-results.tmp",
	".test-timing.tmp",
	"coverage.html",
}

// doClean removes build artifacts.
func doClean(cfg BuildConfig, verbose bool) error {
	ui.Section("Cleaning Build Artifacts")

	artifacts := cfg.CleanPatterns
	if artifacts == nil {
		artifacts = defaultCleanPatterns
	}

	total := len(artifacts)
	removed := 0

	for i, pattern := range artifacts {
		ui.Progress(i+1, total, fmt.Sprintf("Removing %s", pattern))

		if strings.Contains(pattern, "*") {
			cmd := exec.Command("sh", "-c", fmt.Sprintf("rm -rf %s 2>/dev/null || true", pattern))
			if verbose {
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
			}
			_ = cmd.Run()
			removed++
		} else {
			if err := os.RemoveAll(pattern); err == nil {
				removed++
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	ui.ClearProgress()

	// Clean up .test binaries in subdirectories
	_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() && (info.Name() == "vendor" || info.Name() == ".git") {
			return filepath.SkipDir
		}

		if strings.HasSuffix(info.Name(), ".test") {
			_ = os.Remove(path)
			removed++
		}

		return nil
	})

	// Clean up orphaned test containers
	ui.Task("Cleaning", "orphaned Docker test containers")
	dockerCmd := exec.Command("docker", "container", "prune", "-f", "--filter", "label=org.testcontainers=true")
	if verbose {
		dockerCmd.Stdout = os.Stdout
		dockerCmd.Stderr = os.Stderr
	}
	if err := dockerCmd.Run(); err != nil {
		if verbose {
			fmt.Printf("Note: Docker cleanup skipped (docker not available)\n")
		}
	}
	ui.TaskPass()

	removeStatus := fmt.Sprintf("✓ %d items removed", removed)
	cleanStatus := "✓ Complete"

	if ui.ColourEnabled() {
		removeStatus = ui.Green + removeStatus + ui.Reset
		cleanStatus = ui.Green + cleanStatus + ui.Reset
	}

	rows := [][]string{
		{"Action", "Status"},
		{"Remove build artifacts", removeStatus},
		{"Clean workspace", cleanStatus},
	}

	ui.SummaryCardWithStatus("Clean Summary", rows, "< 1s", true, "✓ CLEAN SUCCESSFUL", "✗ CLEAN FAILED")

	return nil
}
