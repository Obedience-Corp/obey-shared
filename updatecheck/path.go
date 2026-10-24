package updatecheck

import (
	"os"
	"path/filepath"
)

// DefaultCachePath returns ~/.obey/update-check/suite.json, or "" when the
// user home directory cannot be resolved to an absolute path. An empty cache
// path disables all cache I/O.
func DefaultCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil || !filepath.IsAbs(home) {
		return ""
	}
	return filepath.Join(home, ".obey", "update-check", "suite.json")
}
