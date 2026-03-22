// Package camputil provides canonical campaign root detection for all Obey CLI tools.
package camputil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// CampaignDir is the marker directory that identifies a campaign root.
	CampaignDir = ".campaign"

	// CampaignConfigFile is the name of the campaign configuration file.
	CampaignConfigFile = "campaign.yaml"

	// EnvCampaignRoot is the environment variable that overrides campaign detection.
	EnvCampaignRoot = "CAMP_ROOT"
)

// ErrNoCampaign is returned when no campaign root can be found.
var ErrNoCampaign = errors.New("not inside a campaign directory")

// FindCampaignRoot locates the campaign root directory.
//
// Priority:
//  1. CAMP_ROOT env var (if set and valid — must contain .campaign/)
//  2. Walk up from startDir looking for .campaign/ marker
//
// Resolves symlinks for path consistency across platforms.
func FindCampaignRoot(ctx context.Context, startDir string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	// Check CAMP_ROOT env var first.
	if root := os.Getenv(EnvCampaignRoot); root != "" {
		resolved, err := filepath.EvalSymlinks(root)
		if err != nil {
			return "", fmt.Errorf("CAMP_ROOT %q: %w", root, err)
		}
		resolved, err = filepath.Abs(resolved)
		if err != nil {
			return "", fmt.Errorf("CAMP_ROOT absolute path: %w", err)
		}
		info, err := os.Stat(filepath.Join(resolved, CampaignDir))
		if err != nil || !info.IsDir() {
			return "", fmt.Errorf("CAMP_ROOT %q does not contain a %s/ directory", root, CampaignDir)
		}
		return resolved, nil
	}

	// Resolve symlinks and get absolute path for consistent walk.
	dir, err := filepath.EvalSymlinks(startDir)
	if err != nil {
		return "", fmt.Errorf("resolve start path: %w", err)
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("absolute path: %w", err)
	}

	// Walk up directory tree looking for .campaign/ marker.
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}

		info, err := os.Stat(filepath.Join(dir, CampaignDir))
		if err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("%w\nHint: Run 'camp init' to create a campaign, or set CAMP_ROOT", ErrNoCampaign)
		}
		dir = parent
	}
}
