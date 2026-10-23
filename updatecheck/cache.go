package updatecheck

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

const maxCacheBytes = 64 << 10

// ErrCorruptCache means suite.json is not a regular file, is too large, is
// not JSON, or carries another schema version. It never includes file content.
var ErrCorruptCache = errors.New("updatecheck: corrupt or unsupported cache")

// LoadCache reads suite.json at path. An empty path or a missing file returns
// (nil, nil). suite.json is a shared writable file, so a release that fails
// validation is dropped and HTMLURL is rederived from the validated tag.
func LoadCache(path string) (*Cache, error) {
	if path == "" {
		return nil, nil
	}
	fi, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("updatecheck: stat cache: %w", err)
	}
	if !fi.Mode().IsRegular() {
		return nil, ErrCorruptCache
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("updatecheck: open cache: %w", err)
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxCacheBytes+1))
	if err != nil {
		return nil, fmt.Errorf("updatecheck: read cache: %w", err)
	}
	if len(data) > maxCacheBytes {
		return nil, ErrCorruptCache
	}
	var c Cache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, ErrCorruptCache
	}
	if c.SchemaVersion != SchemaVersion {
		return nil, ErrCorruptCache
	}
	sanitizeCache(&c)
	return &c, nil
}

func sanitizeCache(c *Cache) {
	c.Source = Source
	c.Channel = ChannelStable
	if isStableVersion(c.LatestTag) && NormalizeVersion(c.LatestTag) == c.LatestVersion {
		c.HTMLURL = releaseTagPagePrefix + c.LatestTag
	} else {
		c.LatestTag, c.LatestVersion, c.HTMLURL = "", "", ""
		c.CheckedAt = time.Time{}
	}
	if !IsReleaseVersion(c.LastNotifiedVersion) {
		c.LastNotifiedVersion = ""
	}
}

// saveCache writes c to path through a temp file in the same directory and a
// rename, so concurrent readers see the old or the new file, never a partial
// one. Concurrent writers: last rename wins.
func saveCache(path string, c *Cache) error {
	c.SchemaVersion, c.Source, c.Channel = SchemaVersion, Source, ChannelStable
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("updatecheck: encode cache: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("updatecheck: create cache dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("updatecheck: create temp cache: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("updatecheck: write temp cache: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("updatecheck: close temp cache: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("updatecheck: replace cache: %w", err)
	}
	return nil
}

func hasLatest(c *Cache) bool {
	return c != nil && c.LatestVersion != ""
}

func cacheFresh(c *Cache, now time.Time, ttl time.Duration) bool {
	return hasLatest(c) && within(c.CheckedAt, now, ttl)
}

func recentAttempt(c *Cache, now time.Time, backoff time.Duration) bool {
	return c != nil && within(c.LastAttemptAt, now, backoff)
}

// within reports whether t is set, not in the future, and less than d before now.
func within(t, now time.Time, d time.Duration) bool {
	return !t.IsZero() && !t.After(now) && now.Sub(t) < d
}

func applyRelease(c *Cache, rel release, now time.Time) *Cache {
	if c == nil {
		c = &Cache{}
	}
	c.CheckedAt = now.UTC()
	c.LatestTag = rel.TagName
	c.LatestVersion = rel.Version
	c.HTMLURL = rel.HTMLURL
	c.LastAttemptAt = time.Time{}
	return c
}

func markNotified(c *Cache, now time.Time) {
	c.LastNotifiedAt = now.UTC()
	c.LastNotifiedVersion = c.LatestVersion
}
