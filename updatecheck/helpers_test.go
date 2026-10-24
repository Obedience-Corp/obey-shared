package updatecheck

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

type releaseServer struct {
	*httptest.Server
	hits    atomic.Int32
	handler atomic.Value
}

func newReleaseServer(t *testing.T, h http.HandlerFunc) *releaseServer {
	t.Helper()
	rs := &releaseServer{}
	rs.handler.Store(h)
	rs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rs.hits.Add(1)
		rs.handler.Load().(http.HandlerFunc)(w, r)
	}))
	t.Cleanup(rs.Close)
	return rs
}

func (rs *releaseServer) fetcher() *Fetcher { return &Fetcher{URL: rs.URL} }

func (rs *releaseServer) requireHits(t *testing.T, want int32) {
	t.Helper()
	if got := rs.hits.Load(); got != want {
		t.Fatalf("release API hits = %d, want %d", got, want)
	}
}

// newBlockingServer holds every request until the client gives up or the test
// ends. The release cleanup is registered after Close so it runs first.
func newBlockingServer(t *testing.T) *releaseServer {
	t.Helper()
	release := make(chan struct{})
	rs := newReleaseServer(t, func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	})
	t.Cleanup(func() { close(release) })
	return rs
}

func serveTag(tag string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"tag_name": tag,
			"html_url": "https://evil.example/\x1b[2J",
		})
	}
}

func serveStatus(code int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) }
}

func serveBody(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }
}

func cacheWith(latest string, checkedAt time.Time) *Cache {
	rel := releaseFromTag("v" + latest)
	return applyRelease(&Cache{}, rel, checkedAt)
}
