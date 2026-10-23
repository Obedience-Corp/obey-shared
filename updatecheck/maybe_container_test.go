//go:build container_fs

package updatecheck

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const absentExe = "/updatecheck-test-absent/bin/camp"

func ambient(t *testing.T, path string, srv *releaseServer, now time.Time) Options {
	t.Helper()
	t.Setenv("PATH", "")
	return Options{CachePath: path, Fetcher: srv.fetcher(), Suppress: tty, Now: at(now), ExecutablePath: absentExe}
}

func requireNotice(t *testing.T, n *Notice, latest string) {
	t.Helper()
	if n == nil {
		t.Fatalf("no notice, want one for %s", latest)
	}
	want := Notice{
		Message: "Festival suite update available (0.3.23 → " + latest + ")",
		Command: DefaultReleasePage,
		Kind:    HintSee,
	}
	if *n != want {
		t.Fatalf("notice = %+v, want %+v", *n, want)
	}
}

func requireNoNotice(t *testing.T, n *Notice) {
	t.Helper()
	if n != nil {
		t.Fatalf("unexpected notice %+v", *n)
	}
}

func TestMaybeNotifyFirstRunFetchesPersistsAndNotifiesOnce(t *testing.T) {
	path := cachePath(t)
	srv := newReleaseServer(t, serveTag("v0.3.24"))
	opts := ambient(t, path, srv, t0)

	requireNotice(t, MaybeNotify(context.Background(), "v0.3.23", opts), "0.3.24")
	srv.requireHits(t, 1)
	c := mustLoad(t, path)
	if c.LastNotifiedVersion != "0.3.24" || !c.LastNotifiedAt.Equal(t0) || !c.CheckedAt.Equal(t0) {
		t.Errorf("persisted state = %+v", c)
	}

	requireNoNotice(t, MaybeNotify(context.Background(), "v0.3.23", opts))
	srv.requireHits(t, 1)
}

func TestFreshCacheMakesNoNetworkCalls(t *testing.T) {
	path := cachePath(t)
	seed(t, path, cacheWith("0.3.24", t0))
	srv := newReleaseServer(t, serveTag("v0.3.30"))

	now := t0.Add(DefaultCacheTTL - time.Second)
	requireNotice(t, MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, srv, now)), "0.3.24")
	srv.requireHits(t, 0)
}

func TestStaleCacheRefreshes(t *testing.T) {
	path := cachePath(t)
	seed(t, path, cacheWith("0.3.24", t0))
	srv := newReleaseServer(t, serveTag("v0.3.25"))

	now := t0.Add(DefaultCacheTTL)
	requireNotice(t, MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, srv, now)), "0.3.25")
	srv.requireHits(t, 1)
	if c := mustLoad(t, path); c.LatestVersion != "0.3.25" || !c.CheckedAt.Equal(now) {
		t.Errorf("cache = %+v", c)
	}
}

func TestFutureCheckedAtRefreshes(t *testing.T) {
	path := cachePath(t)
	seed(t, path, cacheWith("0.3.24", t0.Add(48*time.Hour)))
	srv := newReleaseServer(t, serveTag("v0.3.25"))

	requireNotice(t, MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, srv, t0)), "0.3.25")
	srv.requireHits(t, 1)
}

// D4: a failed refresh keeps the prior cache, still notifies from it, and
// returns within the ambient budget.
func TestFailedRefreshFailsOpenOnPriorCache(t *testing.T) {
	tests := map[string]func(t *testing.T) *releaseServer{
		"404":       func(t *testing.T) *releaseServer { return newReleaseServer(t, serveStatus(http.StatusNotFound)) },
		"malformed": func(t *testing.T) *releaseServer { return newReleaseServer(t, serveBody("{oops")) },
		"bad tag":   func(t *testing.T) *releaseServer { return newReleaseServer(t, serveTag("v9.9.9\x1b[2J")) },
		"timeout":   newBlockingServer,
	}
	for name, newServer := range tests {
		t.Run(name, func(t *testing.T) {
			path := cachePath(t)
			seed(t, path, cacheWith("0.3.24", t0))
			srv := newServer(t)
			now := t0.Add(48 * time.Hour)

			start := time.Now()
			n := MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, srv, now))
			if elapsed := time.Since(start); elapsed > AmbientTimeout+300*time.Millisecond {
				t.Errorf("MaybeNotify took %v, budget %v", elapsed, AmbientTimeout)
			}
			requireNotice(t, n, "0.3.24")
			srv.requireHits(t, 1)
			c := mustLoad(t, path)
			if c.LatestVersion != "0.3.24" || !c.CheckedAt.Equal(t0) || !c.LastAttemptAt.Equal(now) {
				t.Errorf("cache after failure = %+v", c)
			}
		})
	}
}

// With no cache yet, a link too slow for the budget must not cost the timeout
// on every command.
func TestFailedFirstFetchBacksOff(t *testing.T) {
	path := cachePath(t)
	srv := newBlockingServer(t)

	requireNoNotice(t, MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, srv, t0)))
	srv.requireHits(t, 1)
	c := mustLoad(t, path)
	if hasLatest(c) || !c.LastAttemptAt.Equal(t0) || !c.CheckedAt.IsZero() {
		t.Fatalf("attempt record = %+v", c)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "checked_at") || strings.Contains(string(raw), "0001-01-01") {
		t.Errorf("attempt record carries a zero checked_at: %s", raw)
	}

	start := time.Now()
	for _, offset := range []time.Duration{time.Second, time.Minute, 10 * time.Minute, 30 * time.Minute, DefaultRetryBackoff - time.Second} {
		requireNoNotice(t, MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, srv, t0.Add(offset))))
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Errorf("5 backed-off calls took %v", elapsed)
	}
	srv.requireHits(t, 1)

	srv.serve(serveTag("v0.3.24"))
	recovered := t0.Add(DefaultRetryBackoff)
	requireNotice(t, MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, srv, recovered)), "0.3.24")
	srv.requireHits(t, 2)
	c = mustLoad(t, path)
	if c.LatestVersion != "0.3.24" || !c.CheckedAt.Equal(recovered) || !c.LastAttemptAt.IsZero() ||
		c.LastNotifiedVersion != "0.3.24" || !c.LastNotifiedAt.Equal(recovered) {
		t.Fatalf("record after recovery = %+v", c)
	}

	// D3 on the recovered record: the same version stays quiet from the fresh
	// cache, and a newer one after the gap notifies.
	requireNoNotice(t, MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, srv, recovered.Add(time.Hour))))
	srv.requireHits(t, 2)
	srv.serve(serveTag("v0.3.25"))
	requireNotice(t, MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, srv, recovered.Add(DefaultCacheTTL))), "0.3.25")
	srv.requireHits(t, 3)
}

func TestRetryBackoffAfterFailedRefresh(t *testing.T) {
	path := cachePath(t)
	seed(t, path, cacheWith("0.3.24", t0))
	srv := newReleaseServer(t, serveStatus(http.StatusInternalServerError))
	failedAt := t0.Add(48 * time.Hour)

	MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, srv, failedAt))
	srv.requireHits(t, 1)
	MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, srv, failedAt.Add(DefaultRetryBackoff-time.Second)))
	srv.requireHits(t, 1)

	srv.serve(serveTag("v0.3.25"))
	retryAt := failedAt.Add(DefaultRetryBackoff)
	MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, srv, retryAt))
	srv.requireHits(t, 2)
	if c := mustLoad(t, path); c.LatestVersion != "0.3.25" || !c.LastAttemptAt.IsZero() || !c.CheckedAt.Equal(retryAt) {
		t.Errorf("cache after recovery = %+v", c)
	}
}

// D3 end to end with a 1h TTL so every step after the first refreshes.
func TestDisplayPolicyAcrossInvocations(t *testing.T) {
	path := cachePath(t)
	srv := newReleaseServer(t, serveTag("v0.3.24"))
	step := func(offset time.Duration) *Notice {
		opts := ambient(t, path, srv, t0.Add(offset))
		opts.CacheTTL = time.Hour
		return MaybeNotify(context.Background(), "v0.3.23", opts)
	}

	requireNotice(t, step(0), "0.3.24")
	requireNoNotice(t, step(30*time.Minute))
	requireNoNotice(t, step(7*time.Hour))

	srv.serve(serveTag("v0.3.25"))
	requireNotice(t, step(9*time.Hour), "0.3.25")

	srv.serve(serveTag("v0.3.26"))
	requireNoNotice(t, step(11*time.Hour))
	requireNotice(t, step(15*time.Hour), "0.3.26")
}

// D3: one nudge per version per machine, whichever CLI shows it.
func TestNotifyStateIsSharedAcrossCLIs(t *testing.T) {
	path := cachePath(t)
	srv := newReleaseServer(t, serveTag("v0.3.24"))
	camp := ambient(t, path, srv, t0)
	fest := camp
	fest.ExecutablePath = "/updatecheck-test-absent/bin/fest"
	fest.Now = at(t0.Add(time.Minute))

	requireNotice(t, MaybeNotify(context.Background(), "v0.3.23", camp), "0.3.24")
	requireNoNotice(t, MaybeNotify(context.Background(), "v0.3.23", fest))
}

func TestPersistFailureShowsNoNotice(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	writeRaw(t, blocker, "x")
	srv := newReleaseServer(t, serveTag("v0.3.24"))

	requireNoNotice(t, MaybeNotify(context.Background(), "v0.3.23", ambient(t, filepath.Join(blocker, "suite.json"), srv, t0)))
	srv.requireHits(t, 1)
}

func TestCorruptCacheRecovers(t *testing.T) {
	path := cachePath(t)
	writeRaw(t, path, "\x00garbage")
	down := newReleaseServer(t, serveStatus(http.StatusBadGateway))
	requireNoNotice(t, MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, down, t0)))

	up := newReleaseServer(t, serveTag("v0.3.24"))
	requireNotice(t, MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, up, t0.Add(DefaultRetryBackoff))), "0.3.24")
	mustLoad(t, path)
}

func TestTamperedCacheNeverReachesNotice(t *testing.T) {
	path := cachePath(t)
	writeRaw(t, path, `{"schema_version":"`+SchemaVersion+`","checked_at":"2026-10-23T11:00:00Z","latest_tag":"v9.9.9\u001b[2J","latest_version":"9.9.9\u001b[2J"}`)
	down := newReleaseServer(t, serveStatus(http.StatusNotFound))
	requireNoNotice(t, MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, down, t0)))

	up := newReleaseServer(t, serveTag("v0.3.24"))
	n := MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, up, t0.Add(DefaultRetryBackoff)))
	requireNotice(t, n, "0.3.24")
	var buf bytes.Buffer
	n.Render(&buf)
	if strings.ContainsRune(buf.String(), 0x1b) {
		t.Errorf("rendered notice carries an escape: %q", buf.String())
	}
}

func TestDefaultCachePathFollowsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	want := filepath.Join(home, ".obey", "update-check", "suite.json")
	if got := DefaultCachePath(); got != want {
		t.Fatalf("DefaultCachePath = %q, want %q", got, want)
	}
	srv := newReleaseServer(t, serveTag("v0.3.24"))
	opts := ambient(t, "", srv, t0)
	requireNotice(t, MaybeNotify(context.Background(), "v0.3.23", opts), "0.3.24")
	mustLoad(t, want)

	for _, bad := range []string{"", "relative/home"} {
		t.Setenv("HOME", bad)
		if got := DefaultCachePath(); got != "" {
			t.Errorf("HOME=%q: DefaultCachePath = %q, want empty", bad, got)
		}
		requireNoNotice(t, MaybeNotify(context.Background(), "v0.3.23", opts))
	}
	srv.requireHits(t, 1)
}

// S3: with an explicit CachePath nothing lands in HOME or the working
// directory, across success, failure, and notify paths.
func TestEveryWriteDerivesFromCachePath(t *testing.T) {
	home, cwd := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(cwd)
	path := cachePath(t)
	srv := newReleaseServer(t, serveTag("v0.3.24"))

	MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, srv, t0))
	srv.serve(serveStatus(http.StatusNotFound))
	MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, srv, t0.Add(48*time.Hour)))
	if _, err := Check(context.Background(), "v0.3.23", Options{CachePath: path, Fetcher: srv.fetcher(), ForceRefresh: true}); err != nil {
		t.Fatal(err)
	}

	if names := entries(t, home); len(names) != 0 {
		t.Errorf("HOME gained %v", names)
	}
	if names := entries(t, cwd); len(names) != 0 {
		t.Errorf("working directory gained %v", names)
	}
	if names := entries(t, filepath.Dir(path)); !slices.Equal(names, []string{"suite.json"}) {
		t.Errorf("cache dir holds %v, want only suite.json", names)
	}
}

func TestGatedCallsCreateNothing(t *testing.T) {
	path := cachePath(t)
	srv := newReleaseServer(t, serveTag("v0.3.24"))
	opts := ambient(t, path, srv, t0)

	MaybeNotify(context.Background(), "", opts)
	if _, err := Check(context.Background(), "", opts); err != nil {
		t.Fatal(err)
	}
	suppressed := opts
	suppressed.Suppress = SuppressInput{StderrIsTTY: true, LookupEnv: envOf(EnvNoUpdateCheck, "1")}
	MaybeNotify(context.Background(), "v0.3.23", suppressed)

	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Errorf("cache dir created by a gated call: %v", err)
	}
	srv.requireHits(t, 0)

	seed(t, path, cacheWith("0.3.24", t0))
	before, _ := os.ReadFile(path)
	requireNoNotice(t, MaybeNotify(context.Background(), "", opts))
	requireNoNotice(t, MaybeNotify(context.Background(), "v0.3.23", suppressed))
	if after, _ := os.ReadFile(path); !bytes.Equal(before, after) {
		t.Errorf("gated call rewrote the cache:\n%s\n->\n%s", before, after)
	}
}

// Ctrl-C during the refresh is the caller giving up, not the network failing,
// so it must not start a retry backoff.
func TestCallerCancelRecordsNoAttempt(t *testing.T) {
	path := cachePath(t)
	seed(t, path, cacheWith("0.3.24", t0))
	srv := newBlockingServer(t)
	now := t0.Add(48 * time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	requireNoNotice(t, MaybeNotify(ctx, "v0.3.23", ambient(t, path, srv, now)))
	if c := mustLoad(t, path); !c.LastAttemptAt.IsZero() {
		t.Fatalf("cancel recorded an attempt: %+v", c)
	}

	srv.serve(serveTag("v0.3.25"))
	requireNotice(t, MaybeNotify(context.Background(), "v0.3.23", ambient(t, path, srv, now)), "0.3.25")
	srv.requireHits(t, 2)
}
