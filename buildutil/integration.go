package buildutil

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Obedience-Corp/obey-shared/buildutil/ui"
)

// doIntegration runs integration tests with Docker support and real-time progress.
func doIntegration(cfg BuildConfig, verbose bool) error {
	ui.Section("Running Integration Tests")

	// Clean up orphaned test containers
	ui.Task("Cleaning", "orphaned test containers")
	cleanCmd := exec.Command("docker", "container", "prune", "-f", "--filter", "label=org.testcontainers=true")
	if err := cleanCmd.Run(); err != nil {
		ui.TaskFail()
	} else {
		ui.TaskPass()
	}

	// Build Linux binary for Docker-based integration tests
	ui.Task("Building", "Linux binary for Docker tests")
	if err := os.MkdirAll("bin/linux", 0o755); err != nil {
		ui.TaskFail()
		return fmt.Errorf("failed to create bin/linux directory: %w", err)
	}

	buildArgs := []string{"build"}
	buildArgs = append(buildArgs, buildTagsArgs(cfg)...)
	buildArgs = append(buildArgs, "-ldflags", "-s -w -extldflags '-static'")
	buildArgs = append(buildArgs, "-o", fmt.Sprintf("bin/linux/%s", cfg.BinaryName), cfg.MainPath)

	cmd := exec.Command("go", buildArgs...)

	// Use custom build env if provided, otherwise default CGO+zig
	if cfg.IntegrationBuildEnv != nil {
		cmd.Env = append(os.Environ(), cfg.IntegrationBuildEnv()...)
	} else {
		// Default: CGO + zig cross-compilation for Linux
		zigTarget := "aarch64-linux-musl"
		if runtime.GOARCH == "amd64" {
			zigTarget = "x86_64-linux-musl"
		}
		cc := fmt.Sprintf("zig cc -target %s", zigTarget)

		if _, err := exec.LookPath("zig"); err != nil {
			ui.TaskFail()
			return fmt.Errorf("zig not found in PATH (required for CGO cross-compilation to Linux): %w", err)
		}

		cmd.Env = append(os.Environ(),
			"CGO_ENABLED=1",
			"GOOS=linux",
			"GOARCH="+runtime.GOARCH,
			"CC="+cc,
		)
	}

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		ui.TaskFail()
		return fmt.Errorf("failed to build Linux binary: %w", err)
	}
	ui.TaskPass()

	suites, err := discoverIntegrationSuites(cfg)
	if err != nil {
		return fmt.Errorf("failed to discover integration test suites: %w", err)
	}

	if len(suites) == 0 {
		ui.Status("No integration tests found", true)
		return nil
	}

	if verbose {
		fmt.Printf("Found %d integration test suites\n", len(suites))
	}

	results := make([]integrationResult, 0, len(suites))
	total := len(suites)
	failures := 0

	testDir := integrationTestDir(cfg)

	for i, suite := range suites {
		name := strings.TrimPrefix(suite, testDir+"/")
		if name == suite {
			name = testDir
		}

		start := time.Now()

		var pass bool
		var testsPassed, testsFailed int
		var failedTests []string

		if verbose {
			cmd := exec.Command("go", "test", "-v", "-parallel", "4", "-tags", "integration", "-timeout", "2m", "./"+suite)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			ui.Progress(i+1, total, fmt.Sprintf("Testing %s", name))
			pass = cmd.Run() == nil
		} else {
			cmd := exec.Command("go", "test", "-json", "-count=1", "-parallel", "4", "-tags", "integration", "-timeout", "2m", "./"+suite)
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				return fmt.Errorf("failed to create stdout pipe: %w", err)
			}

			if err := cmd.Start(); err != nil {
				return fmt.Errorf("failed to start test: %w", err)
			}

			var currentTest string
			var currentOutput string
			var mu sync.Mutex

			spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			spinnerIdx := 0

			fmt.Println("  → Starting...")
			fmt.Printf("[%d/%d] ⠋ Starting... 0s", i+1, total)

			done := make(chan bool)
			go func() {
				ticker := time.NewTicker(100 * time.Millisecond)
				defer ticker.Stop()
				for {
					select {
					case <-done:
						return
					case <-ticker.C:
						mu.Lock()
						elapsed := time.Since(start).Seconds()
						testName := currentTest
						output := currentOutput
						passed := testsPassed
						failed := testsFailed
						mu.Unlock()

						spinner := spinnerChars[spinnerIdx%len(spinnerChars)]
						spinnerIdx++

						status := fmt.Sprintf("%d✓", passed)
						if failed > 0 {
							status += fmt.Sprintf(" %d✗", failed)
						}

						var progressLine string
						if testName != "" {
							progressLine = fmt.Sprintf("%s %s (%s) %.0fs", spinner, testName, status, elapsed)
						} else {
							progressLine = fmt.Sprintf("%s Starting... %.0fs", spinner, elapsed)
						}

						if output == "" {
							output = "waiting for output..."
						}
						ui.ProgressWithOutput(i+1, total, output, progressLine)
					}
				}
			}()

			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				var event integrationTestEvent
				if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
					continue
				}

				mu.Lock()
				if event.Action == "output" && event.Output != "" {
					trimmed := strings.TrimSpace(event.Output)
					if strings.HasPrefix(trimmed, "=== RUN") {
						currentOutput = strings.TrimPrefix(trimmed, "=== RUN   ")
					} else if trimmed != "" && !strings.HasPrefix(trimmed, "---") && !strings.HasPrefix(trimmed, "PASS") && !strings.HasPrefix(trimmed, "FAIL") {
						currentOutput = trimmed
					}
				}

				if event.Test != "" {
					switch event.Action {
					case "run":
						if !strings.Contains(event.Test, "/") {
							currentTest = event.Test
						}
					case "pass":
						if !strings.Contains(event.Test, "/") {
							testsPassed++
						}
					case "fail":
						failedTests = append(failedTests, event.Test)
						if !strings.Contains(event.Test, "/") {
							testsFailed++
						}
					}
				}
				mu.Unlock()
			}

			close(done)
			waitErr := cmd.Wait()
			pass = testsFailed == 0 && waitErr == nil
			ui.ClearProgressWithOutput()
		}

		duration := time.Since(start)

		results = append(results, integrationResult{
			Suite:       name,
			Pass:        pass,
			Duration:    duration,
			TestsPassed: testsPassed,
			TestsFailed: testsFailed,
			FailedTests: failedTests,
		})

		if !pass {
			failures++
		}
	}

	ui.ClearProgress()

	// Calculate totals
	var totalTime time.Duration
	totalTestsPassed := 0
	totalTestsFailed := 0
	for _, r := range results {
		totalTime += r.Duration
		totalTestsPassed += r.TestsPassed
		totalTestsFailed += r.TestsFailed
	}
	totalTests := totalTestsPassed + totalTestsFailed

	// Display summary
	rows := [][]string{}
	hasFailures := failures > 0

	for _, r := range results {
		if !r.Pass && len(r.FailedTests) > 0 {
			for _, testName := range r.FailedTests {
				status := "✗ FAILED"
				if ui.ColourEnabled() {
					status = ui.Red + status + ui.Reset
				}
				rows = append(rows, []string{
					testName,
					status,
					"",
				})
			}
		}
	}

	if hasFailures && len(rows) > 0 {
		rows = append([][]string{{"Failed Test", "Status", ""}}, rows...)
	}

	totalStatus := fmt.Sprintf("%d/%d tests passed", totalTestsPassed, totalTests)
	if ui.ColourEnabled() {
		if totalTestsFailed > 0 {
			totalStatus = ui.Red + totalStatus + ui.Reset
		} else {
			totalStatus = ui.Green + totalStatus + ui.Reset
		}
	}

	rows = append(rows, []string{
		fmt.Sprintf("%d suites", len(results)),
		totalStatus,
		fmt.Sprintf("%.2fs", totalTime.Seconds()),
	})

	success := failures == 0

	successMsg := fmt.Sprintf("✓ ALL %d TESTS PASSED", totalTestsPassed)
	failMsg := fmt.Sprintf("✗ %d/%d TESTS FAILED", totalTestsFailed, totalTests)

	ui.SummaryCardWithStatus("Integration Test Summary", rows, fmt.Sprintf("%.2fs", totalTime.Seconds()), success, successMsg, failMsg)

	if failures > 0 {
		return fmt.Errorf("%d integration test suites failed", failures)
	}

	return nil
}

// integrationTestEvent represents a go test -json output line.
type integrationTestEvent struct {
	Action  string `json:"Action"`
	Package string `json:"Package"`
	Test    string `json:"Test"`
	Output  string `json:"Output"`
}

// integrationResult tracks integration test results.
type integrationResult struct {
	Suite       string
	Pass        bool
	Duration    time.Duration
	TestsPassed int
	TestsFailed int
	FailedTests []string
}
