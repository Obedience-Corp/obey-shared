package festivalbundle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Markdown image/link: ![alt](url) or [text](url)
var markdownLinkRE = regexp.MustCompile(`(!?\[[^\]]*\]\()([^)]+)(\))`)

var scanSuffixes = map[string]struct{}{
	".md": {}, ".markdown": {}, ".txt": {}, ".yaml": {}, ".yml": {},
	".json": {}, ".html": {}, ".htm": {},
}

// RewritePayloadLinks scans payload text files and vendors out-of-root file
// links into payload/.artifacts/, rewriting those links to portable relatives.
//
// packRoot is the original source directory used to classify in-root vs
// out-of-root. payloadDir is the working payload tree (copy of source).
// Relative links are resolved as if documents still lived under packRoot at
// the same relative paths.
//
// Returns warning messages (e.g. missing targets when not strict).
func RewritePayloadLinks(ctx context.Context, payloadDir, packRoot string, strict bool) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	payloadDir, err := filepath.Abs(payloadDir)
	if err != nil {
		return nil, err
	}
	packRoot, err = filepath.Abs(packRoot)
	if err != nil {
		return nil, err
	}

	var warnings []string
	artifactsDir := filepath.Join(payloadDir, ".artifacts")
	vendored := map[string]string{} // abs source path -> artifact abs path in payload

	var docs []string
	err = filepath.WalkDir(payloadDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".artifacts" {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: %s", ErrSymlinkRejected, path)
		}
		ext := strings.ToLower(filepath.Ext(path))
		if _, ok := scanSuffixes[ext]; !ok {
			return nil
		}
		docs = append(docs, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, doc := range docs {
		if err := ctx.Err(); err != nil {
			return warnings, err
		}
		relDoc, err := filepath.Rel(payloadDir, doc)
		if err != nil {
			return warnings, err
		}
		sourceDoc := filepath.Join(packRoot, relDoc)

		raw, err := os.ReadFile(doc)
		if err != nil {
			return warnings, err
		}
		// Skip non-UTF8-ish binary
		if !isMostlyText(raw) {
			warnings = append(warnings, fmt.Sprintf("skip non-text scan: %s", relDoc))
			continue
		}
		text := string(raw)

		newText := markdownLinkRE.ReplaceAllStringFunc(text, func(match string) string {
			sub := markdownLinkRE.FindStringSubmatch(match)
			if len(sub) != 4 {
				return match
			}
			prefix, rawURL, suffix := sub[1], strings.TrimSpace(sub[2]), sub[3]

			resolved, isFile := resolveFileTarget(rawURL, sourceDoc)
			if !isFile {
				return match // URL or non-file
			}

			st, err := os.Stat(resolved)
			if err != nil || !st.Mode().IsRegular() {
				msg := fmt.Sprintf("missing link target %q in %s", rawURL, relDoc)
				if strict {
					// Capture via panic-free path: set outer error through closure is hard;
					// collect as warning and fail after if strict.
					warnings = append(warnings, "STRICT:"+msg)
				} else {
					warnings = append(warnings, msg)
				}
				return match
			}
			resolved, err = filepath.EvalSymlinks(resolved)
			if err != nil {
				resolved, _ = filepath.Abs(resolved)
			} else {
				resolved, _ = filepath.Abs(resolved)
			}

			if isUnder(resolved, packRoot) {
				// In-root: preserve original link text and file location.
				return match
			}

			// Out-of-root: vendor into .artifacts/
			key := resolved
			artPath, ok := vendored[key]
			if !ok {
				if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
					warnings = append(warnings, err.Error())
					return match
				}
				name := artifactName(resolved)
				dest := filepath.Join(artifactsDir, name)
				if same, _ := sameFileContent(dest, resolved); !same {
					if err := copyFile(resolved, dest); err != nil {
						warnings = append(warnings, err.Error())
						return match
					}
				}
				// Collision with different content: hash the path into the name.
				if st, err := os.Stat(dest); err == nil && st.Mode().IsRegular() {
					if ok, _ := sameFileContent(dest, resolved); !ok {
						sum := sha256.Sum256([]byte(key))
						name = fmt.Sprintf("%s-%s%s",
							strings.TrimSuffix(filepath.Base(resolved), filepath.Ext(resolved)),
							hex.EncodeToString(sum[:6]),
							filepath.Ext(resolved),
						)
						dest = filepath.Join(artifactsDir, name)
						if err := copyFile(resolved, dest); err != nil {
							warnings = append(warnings, err.Error())
							return match
						}
					}
				}
				vendored[key] = dest
				artPath = dest
			}

			relLink, err := relLinkWithinTree(filepath.Dir(doc), artPath, payloadDir)
			if err != nil {
				warnings = append(warnings, err.Error())
				return match
			}
			return prefix + relLink + suffix
		})

		if newText != text {
			if err := os.WriteFile(doc, []byte(newText), 0o644); err != nil {
				return warnings, err
			}
		}
	}

	if strict {
		for _, w := range warnings {
			if strings.HasPrefix(w, "STRICT:") {
				return warnings, fmt.Errorf("festivalbundle: %s", strings.TrimPrefix(w, "STRICT:"))
			}
		}
	}
	return warnings, nil
}

func isMostlyText(b []byte) bool {
	if len(b) == 0 {
		return true
	}
	// Reject if contains NUL
	for _, c := range b {
		if c == 0 {
			return false
		}
	}
	return true
}

func resolveFileTarget(target, baseFile string) (abs string, isFile bool) {
	t := strings.TrimSpace(target)
	t = strings.Trim(t, "<>")
	t = strings.TrimSpace(t)
	if t == "" || strings.HasPrefix(t, "#") {
		return "", false
	}
	if hasNetworkOrNonFileScheme(t) {
		return "", false
	}
	if strings.HasPrefix(strings.ToLower(t), "file:") {
		u, err := url.Parse(t)
		if err != nil {
			return "", false
		}
		path, err := url.PathUnescape(u.Path)
		if err != nil {
			path = u.Path
		}
		abs, err = filepath.Abs(path)
		if err != nil {
			return "", false
		}
		return abs, true
	}
	if filepath.IsAbs(t) {
		abs, err := filepath.Abs(t)
		if err != nil {
			return "", false
		}
		return abs, true
	}
	// Relative to source document location
	joined := filepath.Join(filepath.Dir(baseFile), filepath.FromSlash(t))
	abs, err := filepath.Abs(joined)
	if err != nil {
		return "", false
	}
	return abs, true
}

func hasNetworkOrNonFileScheme(target string) bool {
	t := strings.TrimSpace(target)
	if strings.HasPrefix(t, "//") {
		return true
	}
	lower := strings.ToLower(t)
	if strings.HasPrefix(lower, "file:") {
		return false
	}
	// scheme: present
	if i := strings.Index(t, ":"); i > 0 {
		scheme := strings.ToLower(t[:i])
		switch scheme {
		case "http", "https", "mailto", "data", "ftp":
			return true
		}
		// other schemes (except bare Windows drive letters like C:\)
		if len(scheme) > 1 && !strings.Contains(scheme, `\`) {
			// Windows drive: single letter
			if len(scheme) == 1 && scheme[0] >= 'a' && scheme[0] <= 'z' {
				return false
			}
			return true
		}
	}
	return false
}

func isUnder(path, root string) bool {
	path = filepath.Clean(path)
	root = filepath.Clean(root)
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	// macOS /var vs /private/var: normalize via EvalSymlinks when possible.
	if r, err := filepath.EvalSymlinks(root); err == nil {
		root = r
	}
	if p, err := filepath.EvalSymlinks(path); err == nil {
		path = p
	} else {
		// path may not exist yet; eval parent
		if p, err := filepath.EvalSymlinks(filepath.Dir(path)); err == nil {
			path = filepath.Join(p, filepath.Base(path))
		}
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func artifactName(src string) string {
	data, err := os.ReadFile(src)
	if err != nil {
		data = []byte(src)
	}
	sum := sha256.Sum256(data)
	base := filepath.Base(src)
	base = strings.ReplaceAll(base, "..", "_")
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	return fmt.Sprintf("%s-%s%s", stem, hex.EncodeToString(sum[:6]), ext)
}

func relLinkWithinTree(fromDir, toFile, treeRoot string) (string, error) {
	fromDir, err := filepath.Abs(fromDir)
	if err != nil {
		return "", err
	}
	toFile, err = filepath.Abs(toFile)
	if err != nil {
		return "", err
	}
	treeRoot, err = filepath.Abs(treeRoot)
	if err != nil {
		return "", err
	}
	fromRel, err := filepath.Rel(treeRoot, fromDir)
	if err != nil {
		return "", err
	}
	toRel, err := filepath.Rel(treeRoot, toFile)
	if err != nil {
		return "", err
	}
	// depth under tree root
	fromRel = filepath.ToSlash(fromRel)
	toRel = filepath.ToSlash(toRel)
	if fromRel == "." {
		return toRel, nil
	}
	parts := strings.Split(fromRel, "/")
	depth := len(parts)
	if fromRel == "" {
		depth = 0
	}
	prefix := strings.Repeat("../", depth)
	return prefix + toRel, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func sameFileContent(a, b string) (bool, error) {
	da, err := os.ReadFile(a)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	db, err := os.ReadFile(b)
	if err != nil {
		return false, err
	}
	return string(da) == string(db), nil
}
