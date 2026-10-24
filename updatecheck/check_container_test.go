//go:build container_fs

package updatecheck

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func explicit(t *testing.T, path string, srv *releaseServer, now time.Time) Options {
	t.Helper()
	t.Setenv("PATH", "")
	return Options{CachePath: path, Fetcher: srv.fetcher(), Now: at(now), ExecutablePath: absentExe}
}

func TestCheckFreshCacheMakesNoNetworkCalls(t *testing.T) {
	path := cachePath(t)
	seed(t, path, cacheWith("0.3.24", t0))
	srv := newReleaseServer(t, serveTag("v0.3.30"))

	res, err := Check(context.Background(), "v0.3.23", explicit(t, path, srv, t0.Add(time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	srv.requireHits(t, 0)
	want := Result{
		CurrentVersion:  "0.3.23",
		LatestVersion:   "0.3.24",
		LatestTag:       "v0.3.24",
		HTMLURL:         releaseTagPagePrefix + "v0.3.24",
		Source:          Source,
		Channel:         ChannelStable,
		UpdateAvailable: true,
		UpgradeCommand:  DefaultReleasePage,
		UpgradeKind:     HintSee,
		CheckedAt:       t0,
		FromCache:       true,
	}
	if res != want {
		t.Errorf("Check =\n%+v\nwant\n%+v", res, want)
	}
}

func TestCheckRefreshesStaleCacheWithoutTouchingNotifyState(t *testing.T) {
	path := cachePath(t)
	c := cacheWith("0.3.24", t0)
	markNotified(c, t0)
	seed(t, path, c)
	srv := newReleaseServer(t, serveTag("v0.3.25"))
	now := t0.Add(DefaultCacheTTL)

	res, err := Check(context.Background(), "v0.3.23", explicit(t, path, srv, now))
	if err != nil {
		t.Fatal(err)
	}
	srv.requireHits(t, 1)
	if res.LatestVersion != "0.3.25" || res.FromCache || res.RefreshErr != nil {
		t.Errorf("Check = %+v", res)
	}
	got := mustLoad(t, path)
	if got.LatestVersion != "0.3.25" || got.LastNotifiedVersion != "0.3.24" || !got.LastNotifiedAt.Equal(t0) {
		t.Errorf("cache = %+v", got)
	}
}

func TestCheckForceRefresh(t *testing.T) {
	path := cachePath(t)
	seed(t, path, cacheWith("0.3.24", t0))
	srv := newReleaseServer(t, serveTag("v0.3.25"))
	opts := explicit(t, path, srv, t0)
	opts.ForceRefresh = true

	res, err := Check(context.Background(), "v0.3.25", opts)
	if err != nil {
		t.Fatal(err)
	}
	srv.requireHits(t, 1)
	if res.UpdateAvailable || res.UpgradeCommand != "" || res.UpgradeKind != "" || res.LatestVersion != "0.3.25" {
		t.Errorf("up-to-date Check = %+v", res)
	}
}

func TestCheckReportsRefreshFailure(t *testing.T) {
	srv := newReleaseServer(t, serveStatus(http.StatusNotFound))

	path := cachePath(t)
	seed(t, path, cacheWith("0.3.24", t0))
	res, err := Check(context.Background(), "v0.3.23", explicit(t, path, srv, t0.Add(48*time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(res.RefreshErr, ErrReleaseStatus) || !res.FromCache || res.LatestVersion != "0.3.24" || !res.UpdateAvailable {
		t.Errorf("stale-cache failure = %+v", res)
	}
	if c := mustLoad(t, path); !c.LastAttemptAt.IsZero() {
		t.Errorf("Check recorded an ambient retry attempt: %+v", c)
	}

	empty := cachePath(t)
	res, err = Check(context.Background(), "v0.3.23", explicit(t, empty, srv, t0))
	if err != nil {
		t.Fatal(err)
	}
	if res.RefreshErr == nil || res.FromCache || res.LatestVersion != "" || res.UpdateAvailable {
		t.Errorf("no-cache failure = %+v", res)
	}
	if _, err := os.Stat(filepath.Dir(empty)); !os.IsNotExist(err) {
		t.Errorf("failed Check created the cache dir: %v", err)
	}
}

// The explicit path ignores the ambient suppress matrix.
func TestCheckIgnoresAmbientSuppress(t *testing.T) {
	t.Setenv("CI", "true")
	t.Setenv(EnvNoUpdateCheck, "1")
	srv := newReleaseServer(t, serveTag("v0.3.24"))
	opts := explicit(t, cachePath(t), srv, t0)
	opts.Suppress = SuppressInput{JSONMode: true}

	res, err := Check(context.Background(), "v0.3.23", opts)
	if err != nil || !res.UpdateAvailable {
		t.Errorf("Check = %+v, %v", res, err)
	}
	srv.requireHits(t, 1)
}

func TestCheckDevVersionStillReportsLatest(t *testing.T) {
	srv := newReleaseServer(t, serveTag("v0.3.24"))
	res, err := Check(context.Background(), "dev", explicit(t, cachePath(t), srv, t0))
	if err != nil {
		t.Fatal(err)
	}
	if !res.DevOrNonRelease || res.UpdateAvailable || res.LatestVersion != "0.3.24" {
		t.Errorf("Check(dev) = %+v", res)
	}
}

func TestResultJSONShape(t *testing.T) {
	srv := newReleaseServer(t, serveTag("v0.3.24"))
	res, err := Check(context.Background(), "v0.3.23", explicit(t, cachePath(t), srv, t0))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"current_version", "latest_version", "latest_tag", "html_url", "source", "channel", "update_available", "upgrade_command", "upgrade_kind", "checked_at", "from_cache"} {
		if _, ok := got[k]; !ok {
			t.Errorf("Result JSON missing %q: %s", k, raw)
		}
	}
	if _, ok := got["RefreshErr"]; ok {
		t.Errorf("RefreshErr serialized: %s", raw)
	}
}
