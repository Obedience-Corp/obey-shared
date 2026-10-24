//go:build container_fs

package updatecheck

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func cachePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "obey", "update-check", "suite.json")
}

func writeRaw(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func seed(t *testing.T, path string, c *Cache) {
	t.Helper()
	if err := saveCache(path, c); err != nil {
		t.Fatalf("seed cache: %v", err)
	}
}

func mustLoad(t *testing.T, path string) *Cache {
	t.Helper()
	c, err := LoadCache(path)
	if err != nil {
		t.Fatalf("LoadCache: %v", err)
	}
	if c == nil {
		t.Fatal("LoadCache: no cache")
	}
	return c
}

func entries(t *testing.T, dir string) []string {
	t.Helper()
	des, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, de := range des {
		names = append(names, de.Name())
	}
	return names
}

func TestSaveCacheRoundTripsAndLeavesNoTempFiles(t *testing.T) {
	path := cachePath(t)
	first := cacheWith("0.3.24", t0)
	markNotified(first, t0)
	seed(t, path, first)
	seed(t, path, cacheWith("0.3.25", t0.Add(time.Hour)))

	got := mustLoad(t, path)
	if got.LatestVersion != "0.3.25" || got.LatestTag != "v0.3.25" || !got.CheckedAt.Equal(t0.Add(time.Hour)) {
		t.Errorf("cache = %+v", got)
	}
	if got.SchemaVersion != SchemaVersion || got.Source != Source || got.Channel != ChannelStable {
		t.Errorf("schema fields = %q %q %q", got.SchemaVersion, got.Source, got.Channel)
	}
	if names := entries(t, filepath.Dir(path)); !slices.Equal(names, []string{"suite.json"}) {
		t.Errorf("cache dir holds %v, want only suite.json", names)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]any
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"schema_version", "source", "channel", "checked_at", "latest_tag", "latest_version", "html_url"} {
		if _, ok := keys[k]; !ok {
			t.Errorf("suite.json missing %q: %s", k, raw)
		}
	}
	if _, ok := keys["last_attempt_at"]; ok {
		t.Errorf("zero last_attempt_at serialized: %s", raw)
	}
}

// Readers racing writers must always see a whole file.
func TestSaveCacheIsAtomicUnderConcurrency(t *testing.T) {
	path := cachePath(t)
	seed(t, path, cacheWith("0.3.0", t0))

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		failures []error
		stop     = make(chan struct{})
	)
	record := func(err error) {
		mu.Lock()
		failures = append(failures, err)
		mu.Unlock()
	}
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				c, err := LoadCache(path)
				if err != nil || !hasLatest(c) {
					record(fmt.Errorf("partial read: cache=%+v err=%v", c, err))
				}
			}
		}()
	}
	var writers sync.WaitGroup
	for w := range 8 {
		writers.Add(1)
		go func() {
			defer writers.Done()
			for i := range 50 {
				if err := saveCache(path, cacheWith(fmt.Sprintf("%d.%d.0", w+1, i), t0)); err != nil {
					record(err)
				}
			}
		}()
	}
	writers.Wait()
	close(stop)
	wg.Wait()

	if len(failures) > 0 {
		t.Fatalf("%d failures, first: %v", len(failures), failures[0])
	}
	mustLoad(t, path)
	if names := entries(t, filepath.Dir(path)); !slices.Equal(names, []string{"suite.json"}) {
		t.Errorf("cache dir holds %v, want only suite.json", names)
	}
}

func TestLoadCacheMissingIsEmpty(t *testing.T) {
	for _, p := range []string{"", cachePath(t)} {
		c, err := LoadCache(p)
		if c != nil || err != nil {
			t.Errorf("LoadCache(%q) = %+v, %v; want nil, nil", p, c, err)
		}
	}
}

func TestLoadCacheRejectsCorruptFiles(t *testing.T) {
	tests := map[string]func(t *testing.T, path string){
		"garbage":      func(t *testing.T, p string) { writeRaw(t, p, "\x00\x01 not json") },
		"truncated":    func(t *testing.T, p string) { writeRaw(t, p, `{"schema_version":"`+SchemaVersion+`","latest`) },
		"other schema": func(t *testing.T, p string) { writeRaw(t, p, `{"schema_version":"v9","latest_tag":"v1.0.0"}`) },
		"no schema":    func(t *testing.T, p string) { writeRaw(t, p, `{"latest_tag":"v1.0.0","latest_version":"1.0.0"}`) },
		"oversized":    func(t *testing.T, p string) { writeRaw(t, p, strings.Repeat(" ", maxCacheBytes+1)) },
		"directory": func(t *testing.T, p string) {
			if err := os.MkdirAll(p, 0o755); err != nil {
				t.Fatal(err)
			}
		},
		"fifo": func(t *testing.T, p string) {
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := syscall.Mkfifo(p, 0o644); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, setup := range tests {
		t.Run(name, func(t *testing.T) {
			path := cachePath(t)
			setup(t, path)
			done := make(chan error, 1)
			go func() {
				_, err := LoadCache(path)
				done <- err
			}()
			select {
			case err := <-done:
				if !errors.Is(err, ErrCorruptCache) || err.Error() != ErrCorruptCache.Error() {
					t.Errorf("err = %q, want ErrCorruptCache with no file content", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("LoadCache blocked")
			}
		})
	}
}

// suite.json is shared and writable, so its strings get the same validation
// as server strings.
func TestLoadCacheDropsUnvalidatedRelease(t *testing.T) {
	tests := map[string]string{
		"escape in version":   `"latest_tag":"v0.3.99\u001b[2J","latest_version":"0.3.99\u001b[2J"`,
		"tag/version differ":  `"latest_tag":"v0.3.24","latest_version":"0.4.0"`,
		"non-semver tag":      `"latest_tag":"latest","latest_version":"latest"`,
		"version without tag": `"latest_version":"0.3.24"`,
		"prerelease tag":      `"latest_tag":"v0.3.25-rc.1","latest_version":"0.3.25-rc.1"`,
	}
	for name, fields := range tests {
		t.Run(name, func(t *testing.T) {
			path := cachePath(t)
			writeRaw(t, path, `{"schema_version":"`+SchemaVersion+`","checked_at":"2026-10-23T11:00:00Z",`+fields+`,"html_url":"javascript:alert(1)","last_notified_version":"\u001b[31m"}`)
			c := mustLoad(t, path)
			if c.LatestVersion != "" || c.LatestTag != "" || c.HTMLURL != "" || !c.CheckedAt.IsZero() {
				t.Errorf("unvalidated release kept: %+v", c)
			}
			if c.LastNotifiedVersion != "" {
				t.Errorf("LastNotifiedVersion = %q", c.LastNotifiedVersion)
			}
		})
	}

	path := cachePath(t)
	writeRaw(t, path, `{"schema_version":"`+SchemaVersion+`","source":"evil","channel":"next","checked_at":"2026-10-23T11:00:00Z","latest_tag":"v0.3.24","latest_version":"0.3.24","html_url":"https://evil.example/"}`)
	c := mustLoad(t, path)
	if c.HTMLURL != releaseTagPagePrefix+"v0.3.24" || c.Source != Source || c.Channel != ChannelStable {
		t.Errorf("valid cache not normalized: %+v", c)
	}
}
