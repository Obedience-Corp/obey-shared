// Package buildutil provides a configurable build/test dashboard for Go projects.
//
// Each consuming project creates a thin main.go wrapper that imports this package
// and calls Run with a BuildConfig. This preserves the go run ./internal/buildutil
// invocation pattern while sharing all task and UI logic.
package buildutil

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Obedience-Corp/obey-shared/buildutil/ui"
)

// BuildConfig defines project-specific build parameters.
type BuildConfig struct {
	// BinaryName is the output binary name (e.g., "obey", "camp", "fest").
	BinaryName string

	// MainPath is the Go package path to build (e.g., "./cmd/obey").
	MainPath string

	// SectionName used in UI headings (e.g., "Obey CLI", "Camp CLI").
	SectionName string

	// BuildTags applied to go build/vet (e.g., []string{"dev"}).
	// Merged with BUILD_TAGS environment variable.
	BuildTags []string

	// LDFlags returns the -ldflags string for the main binary build.
	// If nil, no ldflags are applied. Used for version injection.
	LDFlags func() string

	// IntegrationTestDir is the base directory for integration tests.
	// Defaults to "tests/integration" if empty.
	IntegrationTestDir string

	// IntegrationBuildEnv provides extra env vars for cross-compiling
	// the Linux binary used in Docker integration tests.
	// If nil, uses default CGO+zig cross-compilation.
	IntegrationBuildEnv func() []string

	// CleanPatterns overrides the default list of glob patterns to clean.
	// If nil, uses the built-in default list.
	CleanPatterns []string
}

// Run is the main entry point. Pass os.Args[1:] and a BuildConfig.
func Run(args []string, cfg BuildConfig) {
	fs := flag.NewFlagSet("buildutil", flag.ExitOnError)
	var noColor bool
	var verbose bool
	fs.BoolVar(&noColor, "no-color", false, "disable ANSI colours")
	fs.BoolVar(&verbose, "v", false, "verbose output")
	_ = fs.Parse(args)

	ui.Init(noColor)

	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "usage: buildutil <build|build-only|test|integration|clean|all>\n")
		os.Exit(1)
	}

	cmd := fs.Arg(0)
	startTime := time.Now()

	if ui.ColourEnabled() {
		fmt.Print(ui.HideCursor)
		defer fmt.Print(ui.ShowCursor)
	}

	var err error

	switch cmd {
	case "build":
		err = doBuild(cfg, verbose)
	case "build-only":
		err = doBuildOnly(cfg, verbose)
	case "test":
		err = doTest(cfg, verbose)
	case "integration":
		err = doIntegration(cfg, verbose)
	case "clean":
		err = doClean(cfg, verbose)
	case "all":
		err = doAll(cfg, verbose, startTime)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmd)
		os.Exit(1)
	}

	if err != nil {
		if ui.ColourEnabled() {
			fmt.Printf("\n%s✗ Error: %v%s\n", ui.Red, err, ui.Reset)
		} else {
			fmt.Printf("\nError: %v\n", err)
		}
		os.Exit(1)
	}
}

func doAll(cfg BuildConfig, verbose bool, startTime time.Time) error {
	var errors []error

	fmt.Println("\n🧹 Cleaning...")
	if cleanErr := doClean(cfg, verbose); cleanErr != nil {
		errors = append(errors, fmt.Errorf("clean failed: %w", cleanErr))
	}

	fmt.Println("\n🔨 Building...")
	if buildErr := doBuild(cfg, verbose); buildErr != nil {
		return fmt.Errorf("stopping due to build failure: %w", buildErr)
	}

	fmt.Println("\n🧪 Testing...")
	if testErr := doTest(cfg, verbose); testErr != nil {
		errors = append(errors, fmt.Errorf("tests failed: %w", testErr))
	}

	fmt.Println("\n🔗 Integration Testing...")
	if integrationErr := doIntegration(cfg, verbose); integrationErr != nil {
		errors = append(errors, fmt.Errorf("integration tests failed: %w", integrationErr))
	}

	if len(errors) > 0 {
		return fmt.Errorf("%d tasks failed", len(errors))
	}

	totalTime := time.Since(startTime)
	cleanStatus := "✓ Complete"
	buildStatus := "✓ Complete"
	testStatus := "✓ Complete"
	integrationStatus := "✓ Complete"

	if ui.ColourEnabled() {
		cleanStatus = ui.Green + cleanStatus + ui.Reset
		buildStatus = ui.Green + buildStatus + ui.Reset
		testStatus = ui.Green + testStatus + ui.Reset
		integrationStatus = ui.Green + integrationStatus + ui.Reset
	}

	rows := [][]string{
		{"Task", "Status"},
		{"Clean", cleanStatus},
		{"Build", buildStatus},
		{"Test", testStatus},
		{"Integration", integrationStatus},
	}
	ui.SummaryCard("All Tasks Complete", rows, fmt.Sprintf("%.2fs", totalTime.Seconds()), true)

	return nil
}

// buildTagsArgs returns the build tag arguments for go commands,
// merging config tags with the BUILD_TAGS environment variable.
func buildTagsArgs(cfg BuildConfig) []string {
	var tags []string
	tags = append(tags, cfg.BuildTags...)

	if envTags := strings.TrimSpace(os.Getenv("BUILD_TAGS")); envTags != "" {
		tags = append(tags, strings.Split(envTags, ",")...)
	}

	if len(tags) == 0 {
		return nil
	}
	return []string{"-tags", strings.Join(tags, ",")}
}

// integrationTestDir returns the configured or default integration test directory.
func integrationTestDir(cfg BuildConfig) string {
	if cfg.IntegrationTestDir != "" {
		return cfg.IntegrationTestDir
	}
	return "tests/integration"
}
