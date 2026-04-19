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
	// CampaignMetadataDir is the marker directory that identifies a campaign root.
	CampaignMetadataDir = ".campaign"

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
// Linked project directories are also supported through `.camp` markers that
// resolve back to registered campaign roots.
func FindCampaignRoot(ctx context.Context, startDir string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	if envRoot, ok := detectCampaignRootOverride(); ok {
		return envRoot, nil
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

	var err error
	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	if root, err := detectCampaignByWalking(ctx, dir); err == nil {
		return root, nil
	} else if !errors.Is(err, ErrNotInCampaign) {
		return "", err
	}

	realDir, err := filepath.EvalSymlinks(dir)
	if err == nil && realDir != dir {
		if root, walkErr := detectCampaignByWalking(ctx, realDir); walkErr == nil {
			return root, nil
		} else if !errors.Is(walkErr, ErrNotInCampaign) {
			return "", walkErr
		}
	}

	return "", ErrNotInCampaign
}

func detectCampaignRootOverride() (string, bool) {
	envRoot := os.Getenv(EnvCampaignRoot)
	if envRoot == "" {
		return "", false
	}

	if absRoot, err := filepath.Abs(envRoot); err == nil {
		envRoot = absRoot
	}
	if resolvedRoot, err := filepath.EvalSymlinks(envRoot); err == nil {
		envRoot = resolvedRoot
	}
	if !IsCampaignRoot(envRoot) {
		return "", false
	}

	return envRoot, true
}

func detectCampaignByWalking(ctx context.Context, startDir string) (string, error) {
	dir := startDir

	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		campaignPath := filepath.Join(dir, CampaignMetadataDir)
		info, err := os.Stat(campaignPath)
		if err == nil && info.IsDir() {
			if resolved, resolveErr := filepath.EvalSymlinks(dir); resolveErr == nil {
				return resolved, nil
			}
			return dir, nil
		}

		markerPath := filepath.Join(dir, linkMarkerFile)
		marker, markerErr := readLinkMarkerFile(markerPath)
		if markerErr == nil {
			if root, ok, resolveErr := resolveMarkerCampaignRoot(ctx, marker); resolveErr != nil {
				return "", resolveErr
			} else if ok {
				return root, nil
			}
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

func resolveMarkerCampaignRoot(ctx context.Context, marker *linkMarker) (string, bool, error) {
	if marker == nil {
		return "", false, nil
	}

	if activeID := marker.effectiveCampaignID(); activeID != "" {
		root, found, err := lookupRegisteredCampaignRoot(ctx, activeID)
		if err != nil {
			return "", false, err
		}
		if found && IsCampaignRoot(root) {
			return root, true, nil
		}
	}

	// Legacy fallback for pre-v2 markers that persisted campaign roots.
	if marker.CampaignRoot != "" {
		root, err := filepath.Abs(marker.CampaignRoot)
		if err != nil {
			return "", false, err
		}
		if resolved, err := filepath.EvalSymlinks(root); err == nil {
			root = resolved
		}
		if IsCampaignRoot(root) {
			return root, true, nil
		}
	}

	return "", false, nil
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
	campaignPath := filepath.Join(dir, CampaignMetadataDir)
	info, err := os.Stat(campaignPath)
	return err == nil && info.IsDir()
}

// CampaignMetadataPath returns the path to the .campaign/ directory for a given root.
func CampaignMetadataPath(root string) string {
	return filepath.Join(root, CampaignMetadataDir)
}

// CampaignMetadataSubPath returns the path to a file or directory nested under the
// .campaign/ directory of the given root. It is equivalent to
// filepath.Join(root, CampaignMetadataDir, parts...), expressed as a single call so
// consumers do not have to spell the nesting pattern at every call site.
// Passing no parts returns the same value as CampaignMetadataPath(root).
//
// Parts are not validated: because this is a thin wrapper around filepath.Join,
// sufficient ".." segments in parts can escape the .campaign/ directory (and even
// the campaign root itself). Callers are responsible for ensuring no component
// escapes the campaign boundary when parts originate from untrusted input.
func CampaignMetadataSubPath(root string, parts ...string) string {
	if len(parts) == 0 {
		return CampaignMetadataPath(root)
	}
	return filepath.Join(append([]string{root, CampaignMetadataDir}, parts...)...)
}
