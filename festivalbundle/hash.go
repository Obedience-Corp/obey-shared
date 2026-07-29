package festivalbundle

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PayloadContentID returns the SPEC §7 content hash for an on-disk bundle root
// that contains a payload/ directory. Result form: "sha256:" + 64 lowercase hex.
func PayloadContentID(ctx context.Context, bundleRoot string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	root, err := filepath.Abs(bundleRoot)
	if err != nil {
		return "", err
	}
	payload := filepath.Join(root, "payload")
	st, err := os.Stat(payload)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return "", fmt.Errorf("festivalbundle: payload is not a directory: %s", payload)
	}

	type entry struct {
		rel  string
		path string
	}
	var entries []entry

	err = filepath.WalkDir(payload, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		// Detect symlink via Lstat before following.
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: %s", ErrSymlinkRejected, path)
		}
		if d.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		// SPEC: paths relative to bundle root with / separators.
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "../") || rel == ".." {
			return fmt.Errorf("festivalbundle: path escapes bundle root: %s", rel)
		}
		entries = append(entries, entry{rel: rel, path: path})
		return nil
	})
	if err != nil {
		return "", err
	}

	// Sort by UTF-8 byte order of path strings.
	sort.Slice(entries, func(i, j int) bool {
		return bytes.Compare([]byte(entries[i].rel), []byte(entries[j].rel)) < 0
	})

	h := sha256.New()
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		data, err := os.ReadFile(e.path)
		if err != nil {
			return "", err
		}
		// path + \n + decimal size + \n + raw bytes
		if _, err := h.Write([]byte(e.rel)); err != nil {
			return "", err
		}
		if _, err := h.Write([]byte{'\n'}); err != nil {
			return "", err
		}
		size := fmt.Sprintf("%d", len(data))
		if _, err := h.Write([]byte(size)); err != nil {
			return "", err
		}
		if _, err := h.Write([]byte{'\n'}); err != nil {
			return "", err
		}
		if _, err := h.Write(data); err != nil {
			return "", err
		}
	}

	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
