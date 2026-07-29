package festivalbundle

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var defaultExcludeDirNames = map[string]struct{}{
	".git": {}, ".bundles": {}, "__pycache__": {}, "node_modules": {},
	".DS_Store": {},
}

// Pack packs sourceDir into a compressed .festival ZIP at outputPath.
func Pack(ctx context.Context, sourceDir, outputPath string, opts PackOptions) (*Info, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(opts.Kind) == "" {
		return nil, fmt.Errorf("%w: kind is required", ErrInvalidInfo)
	}

	sourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(sourceDir)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("festivalbundle: source is not a directory: %s", sourceDir)
	}

	outputPath, err = filepath.Abs(outputPath)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(strings.ToLower(outputPath), ".festival") {
		outputPath += ".festival"
	}

	name := opts.Name
	if name == "" {
		name = filepath.Base(sourceDir)
	}
	now := time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)

	tmp, err := os.MkdirTemp("", "festival-pack-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	bundleRoot := filepath.Join(tmp, "bundle")
	payload := filepath.Join(bundleRoot, "payload")
	if err := os.MkdirAll(payload, 0o755); err != nil {
		return nil, err
	}
	if err := copyTreeFiltered(ctx, sourceDir, payload, opts.ExcludePatterns); err != nil {
		return nil, err
	}

	if _, err := RewritePayloadLinks(ctx, payload, sourceDir, opts.Strict); err != nil {
		return nil, err
	}

	// Ensure at least one file
	hasFile := false
	_ = filepath.WalkDir(payload, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		hasFile = true
		return fs.SkipAll
	})
	if !hasFile {
		return nil, ErrEmptyPayload
	}

	bundleID, err := PayloadContentID(ctx, bundleRoot)
	if err != nil {
		return nil, err
	}

	info := &Info{
		FormatVersion: FormatVersion,
		Kind:          opts.Kind,
		Bundle: BundleMeta{
			ID:        bundleID,
			HashAlg:   HashAlgSHA256,
			Name:      name,
			CreatedAt: now,
			PackedAt:  now,
			Summary:   opts.Summary,
			Creator:   opts.Creator,
			Tags:      opts.Tags,
		},
		From:    opts.From,
		Subject: opts.Subject,
		App:     opts.App,
	}

	infoPath := filepath.Join(bundleRoot, "info.json")
	raw, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return nil, err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(infoPath, raw, 0o644); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return nil, err
	}
	if err := writeZip(ctx, bundleRoot, outputPath); err != nil {
		return nil, err
	}

	if opts.WriteSentRecord {
		if err := WriteSentRecord(sourceDir, info, outputPath); err != nil {
			return info, fmt.Errorf("festivalbundle: pack ok but sent record failed: %w", err)
		}
	}
	return info, nil
}

func copyTreeFiltered(ctx context.Context, src, dst string, extraExclude []string) error {
	extra := map[string]struct{}{}
	for _, p := range extraExclude {
		extra[p] = struct{}{}
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
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
		// Exclude by path component
		for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
			if _, ok := defaultExcludeDirNames[part]; ok {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if _, ok := extra[part]; ok {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
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

func writeZip(ctx context.Context, bundleRoot, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	err = filepath.WalkDir(bundleRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(bundleRoot, path)
		if err != nil {
			return err
		}
		arc := filepath.ToSlash(rel)
		w, err := zw.Create(arc)
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(w, in)
		_ = in.Close()
		return copyErr
	})
	if err != nil {
		_ = zw.Close()
		return err
	}
	return zw.Close()
}
