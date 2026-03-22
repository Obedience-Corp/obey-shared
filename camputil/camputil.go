// Package camputil provides canonical campaign root detection for all Obey CLI tools.
package camputil

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// DefaultDetectTimeout is the maximum time to spend detecting a campaign root.
// This prevents hanging on slow network filesystems.
const DefaultDetectTimeout = 5 * time.Second

const (
	// CampaignDir is the marker directory that identifies a campaign root.
	CampaignDir = ".campaign"

	// CampaignConfigFile is the name of the campaign configuration file.
	CampaignConfigFile = "campaign.yaml"

	// EnvCampaignRoot is the environment variable that overrides campaign detection.
	EnvCampaignRoot = "CAMP_ROOT"
)

// ErrNotInCampaign is returned when the current directory is not inside a campaign.
var ErrNotInCampaign = errors.New("not inside a campaign directory\n" +
	"Hint: Run 'camp init' to create a campaign, or navigate to an existing one")

// FindCampaignRoot locates the campaign root by walking up from startDir.
// Returns the directory containing .campaign/, not .campaign/ itself.
// If startDir is empty, uses the current working directory.
// If CAMP_ROOT is set and points to a valid campaign root, it is returned.
// If CAMP_ROOT is set but invalid, detection falls back to walk-up search.
func FindCampaignRoot(ctx context.Context, startDir string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// Check for environment variable override.
	if envRoot := os.Getenv(EnvCampaignRoot); envRoot != "" {
		resolved, err := filepath.EvalSymlinks(envRoot)
		if err == nil {
			resolved, err = filepath.Abs(resolved)
		}
		if err == nil {
			campaignPath := filepath.Join(resolved, CampaignDir)
			if info, statErr := os.Stat(campaignPath); statErr == nil && info.IsDir() {
				return resolved, nil
			}
		}
		// If env var is set but invalid, continue with detection.
	}

	// Start from given directory or cwd.
	dir := startDir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	// Resolve to absolute path and follow symlinks.
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	// Walk up directory tree.
	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		campaignPath := filepath.Join(dir, CampaignDir)
		info, err := os.Stat(campaignPath)
		if err == nil && info.IsDir() {
			return dir, nil
		}

		// Handle permission errors gracefully — just keep walking up.
		// We may not have permission to read a parent directory but
		// might still find the campaign root higher up.
		if err != nil && !os.IsNotExist(err) && !os.IsPermission(err) {
			return "", err
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotInCampaign
		}
		dir = parent
	}
}

// FindWithTimeout detects campaign root with a timeout using a background context.
// If the filesystem is slow (e.g., network drives), detection will
// be aborted after the default timeout to prevent hanging.
// Callers with an existing context should use FindCampaignRoot directly.
func FindWithTimeout(startDir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultDetectTimeout)
	defer cancel()
	return FindCampaignRoot(ctx, startDir)
}

// FindFromCwdWithTimeout detects from the current working directory with a timeout
// using a background context. Callers with an existing context should use
// FindCampaignRoot or FindFromCwd directly.
func FindFromCwdWithTimeout() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultDetectTimeout)
	defer cancel()
	return FindCampaignRoot(ctx, "")
}

// FindFromCwd is a convenience function that detects from current working directory.
func FindFromCwd(ctx context.Context) (string, error) {
	return FindCampaignRoot(ctx, "")
}

// IsCampaignRoot checks if the given directory is a campaign root (contains .campaign/).
func IsCampaignRoot(dir string) bool {
	campaignPath := filepath.Join(dir, CampaignDir)
	info, err := os.Stat(campaignPath)
	return err == nil && info.IsDir()
}

// CampaignPath returns the path to the .campaign/ directory for a given root.
func CampaignPath(root string) string {
	return filepath.Join(root, CampaignDir)
}
