package festivalbundle

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Unbundle extracts a .festival ZIP into destDir.
func Unbundle(ctx context.Context, festivalPath, destDir string, opts UnbundleOptions) (*Info, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	festivalPath, err := filepath.Abs(festivalPath)
	if err != nil {
		return nil, err
	}
	destDir, err = filepath.Abs(destDir)
	if err != nil {
		return nil, err
	}

	if st, err := os.Stat(destDir); err == nil {
		if st.IsDir() {
			entries, err := os.ReadDir(destDir)
			if err != nil {
				return nil, err
			}
			if len(entries) > 0 && !opts.Force {
				return nil, ErrDestNotEmpty
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, err
	}

	tmp, err := os.MkdirTemp("", "festival-unbundle-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	bundleRoot := filepath.Join(tmp, "bundle")
	if err := os.MkdirAll(bundleRoot, 0o755); err != nil {
		return nil, err
	}

	if err := extractZip(ctx, festivalPath, bundleRoot); err != nil {
		return nil, err
	}

	infoPath := filepath.Join(bundleRoot, "info.json")
	payload := filepath.Join(bundleRoot, "payload")
	raw, err := os.ReadFile(infoPath)
	if err != nil {
		return nil, fmt.Errorf("%w: missing info.json", ErrInvalidInfo)
	}
	var info Info
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInfo, err)
	}
	if info.FormatVersion == 0 || info.Kind == "" || info.Bundle.ID == "" {
		return nil, fmt.Errorf("%w: missing required fields", ErrInvalidInfo)
	}

	if !opts.SkipVerify {
		computed, err := PayloadContentID(ctx, bundleRoot)
		if err != nil {
			return nil, err
		}
		if computed != info.Bundle.ID {
			return nil, fmt.Errorf("%w: expected %s computed %s", ErrHashMismatch, info.Bundle.ID, computed)
		}
	}

	if err := copyTreeAll(ctx, payload, destDir); err != nil {
		return nil, err
	}

	if opts.WriteReceivedRecord {
		if err := WriteReceivedRecord(destDir, &info, festivalPath); err != nil {
			return &info, fmt.Errorf("festivalbundle: unbundle ok but received record failed: %w", err)
		}
	}
	return &info, nil
}

// Verify recomputes the payload content hash of a .festival and compares it to bundle.id.
func Verify(ctx context.Context, festivalPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	festivalPath, err := filepath.Abs(festivalPath)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "festival-verify-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if err := extractZip(ctx, festivalPath, tmp); err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(tmp, "info.json"))
	if err != nil {
		return fmt.Errorf("%w: missing info.json", ErrInvalidInfo)
	}
	var info Info
	if err := json.Unmarshal(raw, &info); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInfo, err)
	}
	computed, err := PayloadContentID(ctx, tmp)
	if err != nil {
		return err
	}
	if computed != info.Bundle.ID {
		return fmt.Errorf("%w: expected %s computed %s", ErrHashMismatch, info.Bundle.ID, computed)
	}
	return nil
}

func extractZip(ctx context.Context, zipPath, dest string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	dest, err = filepath.Abs(dest)
	if err != nil {
		return err
	}

	for _, f := range r.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := f.Name
		if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "..") {
			return fmt.Errorf("%w: %s", ErrUnsafeZipPath, name)
		}
		// Clean and ensure under dest
		target := filepath.Join(dest, filepath.FromSlash(name))
		rel, err := filepath.Rel(dest, target)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("%w: %s", ErrUnsafeZipPath, name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			_ = rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		_ = out.Close()
		_ = rc.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

func copyTreeAll(ctx context.Context, src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: %s", ErrSymlinkRejected, path)
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target)
	})
}

// ReadInfo opens a .festival ZIP and returns its info.json.
func ReadInfo(ctx context.Context, festivalPath string) (*Info, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r, err := zip.OpenReader(festivalPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name == "info.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			var info Info
			if err := json.NewDecoder(rc).Decode(&info); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidInfo, err)
			}
			return &info, nil
		}
	}
	return nil, fmt.Errorf("%w: missing info.json", ErrInvalidInfo)
}
