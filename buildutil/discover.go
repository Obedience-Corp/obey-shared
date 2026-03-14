package buildutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// discoverPackages finds all Go packages in the project, filtering out
// vendor, testdata, integration tests, benchmarks, and e2e directories.
func discoverPackages() ([]string, error) {
	cmd := exec.Command("go", "list", "./...")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	module := getModuleName()

	var packages []string
	for _, line := range lines {
		if line != "" &&
			!strings.Contains(line, "/vendor/") &&
			!strings.Contains(line, "/testdata") &&
			!strings.Contains(line, "/tests/integration") &&
			!strings.HasSuffix(line, "/tests/integration") &&
			!strings.Contains(line, "_test") &&
			!strings.HasSuffix(line, "/test/benchmark") &&
			!strings.HasSuffix(line, "/test/e2e") {
			if module != "" && strings.HasPrefix(line, module) {
				relativePath := strings.TrimPrefix(line, module)
				if relativePath == "" {
					packages = append(packages, ".")
				} else if strings.HasPrefix(relativePath, "/") {
					packages = append(packages, "."+relativePath)
				}
			}
		}
	}

	return packages, nil
}

// getModuleName reads the module name from go.mod.
func getModuleName() string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

// discoverTestPackages finds all packages that have test files,
// excluding integration test directories. Uses a single go list call
// instead of per-package subprocess invocations.
func discoverTestPackages() ([]string, error) {
	cmd := exec.Command("go", "list", "-f", `{{if .TestGoFiles}}{{.ImportPath}}{{end}}`, "./...")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	module := getModuleName()
	var testPackages []string

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		if strings.Contains(line, "/tests/integration") {
			continue
		}
		// Convert to relative path
		if module != "" && strings.HasPrefix(line, module) {
			rel := strings.TrimPrefix(line, module)
			if rel == "" {
				testPackages = append(testPackages, ".")
			} else if strings.HasPrefix(rel, "/") {
				testPackages = append(testPackages, "."+rel)
			}
		}
	}

	return testPackages, nil
}

// discoverIntegrationSuites finds all integration test directories
// under the configured integration test directory.
func discoverIntegrationSuites(cfg BuildConfig) ([]string, error) {
	dir := integrationTestDir(cfg)
	var suites []string

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return suites, nil
	}

	// Check if the directory itself has test files (flat structure)
	matches, _ := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if len(matches) > 0 {
		suites = append(suites, dir)
	}

	// Walk for subdirectories with tests
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() && path != dir {
			subMatches, _ := filepath.Glob(filepath.Join(path, "*_test.go"))
			if len(subMatches) > 0 {
				suites = append(suites, path)
			}
		}

		return nil
	})

	return suites, err
}
