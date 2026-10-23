package updatecheck

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// unwritableCache cannot be created on any Unix host because /dev/null is not
// a directory, so a broken gate fails these tests without touching the host.
const unwritableCache = "/dev/null/updatecheck/suite.json"

type probes struct {
	envReads atomic.Int32
	nowCalls atomic.Int32
}

func (p *probes) opts(srv *releaseServer, in SuppressInput) Options {
	in.LookupEnv = func(string) string { p.envReads.Add(1); return "" }
	return Options{
		CachePath: unwritableCache,
		Fetcher:   srv.fetcher(),
		Suppress:  in,
		Now:       func() time.Time { p.nowCalls.Add(1); return t0 },
	}
}

// S2: an empty suite version is a non-suite build. Nothing runs: no env
// reads, no clock, no cache, no network. The clock probe stands in for the
// cache read because both entry points read the clock before the cache.
func TestEmptySuiteVersionDoesNothing(t *testing.T) {
	srv := newReleaseServer(t, serveTag("v0.3.24"))
	var p probes
	opts := p.opts(srv, SuppressInput{StderrIsTTY: true})

	if n := MaybeNotify(context.Background(), "", opts); n != nil {
		t.Errorf("MaybeNotify returned %+v", n)
	}
	res, err := Check(context.Background(), "", opts)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	want := Result{Source: Source, Channel: ChannelStable, DevOrNonRelease: true}
	if res != want {
		t.Errorf("Check = %+v, want %+v", res, want)
	}
	srv.requireHits(t, 0)
	if p.envReads.Load() != 0 || p.nowCalls.Load() != 0 {
		t.Errorf("env reads = %d, clock reads = %d, want 0", p.envReads.Load(), p.nowCalls.Load())
	}
}

func TestNonReleaseSuiteVersionSkipsAmbient(t *testing.T) {
	srv := newReleaseServer(t, serveTag("v0.3.24"))
	var p probes
	for _, v := range []string{"dev", "unknown", "0.3", "v0.3.24+dirty"} {
		if n := MaybeNotify(context.Background(), v, p.opts(srv, SuppressInput{StderrIsTTY: true})); n != nil {
			t.Errorf("MaybeNotify(%q) returned %+v", v, n)
		}
	}
	srv.requireHits(t, 0)
	if p.envReads.Load() != 0 || p.nowCalls.Load() != 0 {
		t.Errorf("env reads = %d, clock reads = %d, want 0", p.envReads.Load(), p.nowCalls.Load())
	}
}

func TestSuppressedAmbientDoesNoIO(t *testing.T) {
	srv := newReleaseServer(t, serveTag("v0.3.24"))
	inputs := map[string]SuppressInput{
		"non-tty": {},
		"json":    {StderrIsTTY: true, JSONMode: true},
		"agent":   {StderrIsTTY: true, AgentMode: true},
	}
	for name, in := range inputs {
		var p probes
		if n := MaybeNotify(context.Background(), "v0.3.23", p.opts(srv, in)); n != nil {
			t.Errorf("%s: MaybeNotify returned %+v", name, n)
		}
		if p.nowCalls.Load() != 0 {
			t.Errorf("%s: clock read %d times, want 0", name, p.nowCalls.Load())
		}
	}

	var p probes
	opts := p.opts(srv, SuppressInput{StderrIsTTY: true})
	opts.Suppress.LookupEnv = envOf("CI", "true")
	if n := MaybeNotify(context.Background(), "v0.3.23", opts); n != nil {
		t.Errorf("CI: MaybeNotify returned %+v", n)
	}
	if p.nowCalls.Load() != 0 {
		t.Errorf("CI: clock read %d times, want 0", p.nowCalls.Load())
	}
	srv.requireHits(t, 0)
}

func TestCancelledContextSkipsAmbient(t *testing.T) {
	srv := newReleaseServer(t, serveTag("v0.3.24"))
	var p probes
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if n := MaybeNotify(ctx, "v0.3.23", p.opts(srv, SuppressInput{StderrIsTTY: true})); n != nil {
		t.Errorf("MaybeNotify returned %+v", n)
	}
	if _, err := Check(ctx, "v0.3.23", p.opts(srv, SuppressInput{})); err == nil {
		t.Error("Check ignored a cancelled context")
	}
	srv.requireHits(t, 0)
}

// When persisting notify state fails, no notice is shown, so a read-only home
// cannot produce a notice on every command.
func TestNoNoticeWhenNotifyStateCannotPersist(t *testing.T) {
	srv := newReleaseServer(t, serveTag("v0.3.24"))
	var p probes
	if n := MaybeNotify(context.Background(), "v0.3.23", p.opts(srv, SuppressInput{StderrIsTTY: true})); n != nil {
		t.Errorf("MaybeNotify returned %+v with an unwritable cache", n)
	}
	srv.requireHits(t, 1)
}
