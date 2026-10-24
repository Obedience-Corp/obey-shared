package updatecheck

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchParsesTagAndIgnoresServerURL(t *testing.T) {
	var gotUA, gotAccept, gotAuth string
	srv := newReleaseServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotUA, gotAccept, gotAuth = r.UserAgent(), r.Header.Get("Accept"), r.Header.Get("Authorization")
		serveTag("v0.3.24")(w, r)
	})
	t.Setenv("GH_TOKEN", "host-token-must-not-leak")
	t.Setenv("GITHUB_TOKEN", "host-token-must-not-leak")

	rel, err := srv.fetcher().fetchLatestStable(context.Background())
	if err != nil {
		t.Fatalf("fetchLatestStable: %v", err)
	}
	want := release{
		TagName: "v0.3.24",
		Version: "0.3.24",
		HTMLURL: "https://github.com/Obedience-Corp/festival/releases/tag/v0.3.24",
	}
	if rel != want {
		t.Errorf("release = %+v, want %+v", rel, want)
	}
	if gotAuth != "" {
		t.Errorf("Authorization = %q, want none", gotAuth)
	}
	if gotUA == "" || gotAccept != "application/vnd.github+json" {
		t.Errorf("User-Agent = %q, Accept = %q", gotUA, gotAccept)
	}
}

func TestFetchRejectsBadResponses(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    error
	}{
		{"404", serveStatus(http.StatusNotFound), ErrReleaseStatus},
		{"rate limited", serveStatus(http.StatusForbidden), ErrReleaseStatus},
		{"500", serveStatus(http.StatusInternalServerError), ErrReleaseStatus},
		{"malformed", serveBody("{not json \x1b[2J"), ErrInvalidRelease},
		{"html", serveBody("<html>captive portal</html>"), ErrInvalidRelease},
		{"empty body", serveBody(""), ErrInvalidRelease},
		{"missing tag", serveBody(`{"html_url":"x"}`), ErrInvalidRelease},
		{"non-semver tag", serveTag("latest"), ErrInvalidRelease},
		{"two-part tag", serveTag("v0.3"), ErrInvalidRelease},
		{"build metadata", serveTag("v0.3.24+abc"), ErrInvalidRelease},
		{"escape sequence", serveTag("v0.3.24\x1b[31m"), ErrInvalidRelease},
		{"whitespace", serveTag(" v0.3.24"), ErrInvalidRelease},
		{"prerelease", serveBody(`{"tag_name":"v0.3.25-rc.1","prerelease":true}`), ErrInvalidRelease},
		{"draft", serveBody(`{"tag_name":"v0.3.25","draft":true}`), ErrInvalidRelease},
		{"prerelease tag marked stable", serveBody(`{"tag_name":"v0.3.25-rc.1","prerelease":false}`), ErrInvalidRelease},
		{"oversized", serveBody(`{"tag_name":"` + strings.Repeat("9", maxReleaseBody) + `"}`), ErrInvalidRelease},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newReleaseServer(t, tt.handler)
			_, err := srv.fetcher().fetchLatestStable(context.Background())
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if strings.Contains(err.Error(), "\x1b") || strings.Contains(err.Error(), "captive") {
				t.Errorf("error carries server content: %q", err)
			}
		})
	}
}

func TestFetchWithBudgetStopsAtAmbientTimeout(t *testing.T) {
	srv := newBlockingServer(t)
	start := time.Now()
	_, err := fetchWithBudget(context.Background(), Options{Fetcher: srv.fetcher()}, AmbientTimeout)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("blocked server returned a release")
	}
	if elapsed > AmbientTimeout+300*time.Millisecond {
		t.Errorf("returned after %v, budget is %v", elapsed, AmbientTimeout)
	}
	if err.Error() != "updatecheck: fetch latest release: timed out" {
		t.Errorf("err = %q", err)
	}
}

// A caller-supplied client without a timeout is still bounded by the budget.
func TestFetchWithBudgetBoundsCallerClient(t *testing.T) {
	srv := newBlockingServer(t)
	f := &Fetcher{URL: srv.URL, HTTPClient: &http.Client{}}
	start := time.Now()
	if _, err := fetchWithBudget(context.Background(), Options{Fetcher: f}, AmbientTimeout); err == nil {
		t.Fatal("blocked server returned a release")
	}
	if elapsed := time.Since(start); elapsed > AmbientTimeout+300*time.Millisecond {
		t.Errorf("returned after %v, budget is %v", elapsed, AmbientTimeout)
	}
}

func TestFetchHonorsCallerCancellation(t *testing.T) {
	srv := newBlockingServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	_, err := fetchWithBudget(ctx, Options{Fetcher: srv.fetcher()}, ExplicitTimeout)
	if !errors.Is(err, context.Canceled) || err.Error() != "updatecheck: fetch latest release: canceled" {
		t.Fatalf("err = %v, want context.Canceled with fixed text", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("returned after %v despite cancellation", elapsed)
	}
}

// A renamed repo answers with a same-host redirect, which is followed.
func TestFetchFollowsSameHostRedirect(t *testing.T) {
	srv := newReleaseServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/moved" {
			http.Redirect(w, r, "/moved", http.StatusMovedPermanently)
			return
		}
		serveTag("v0.3.24")(w, r)
	})
	rel, err := srv.fetcher().fetchLatestStable(context.Background())
	if err != nil || rel.Version != "0.3.24" {
		t.Fatalf("release = %+v, err = %v", rel, err)
	}
}

// A redirect elsewhere is not followed, and server-chosen URL text never
// reaches the error.
func TestFetchKeepsRedirectTextOutOfErrors(t *testing.T) {
	cross := newReleaseServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1:1/SERVER-TEXT", http.StatusFound)
	})
	_, err := cross.fetcher().fetchLatestStable(context.Background())
	if !errors.Is(err, ErrReleaseStatus) || strings.Contains(err.Error(), "SERVER-TEXT") {
		t.Errorf("cross-host redirect: err = %v", err)
	}

	same := newReleaseServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/SERVER-TEXT", http.StatusFound)
			return
		}
		conn, _, hijackErr := w.(http.Hijacker).Hijack()
		if hijackErr == nil {
			_ = conn.Close()
		}
	})
	_, err = same.fetcher().fetchLatestStable(context.Background())
	if err == nil || strings.Contains(err.Error(), "SERVER-TEXT") {
		t.Errorf("same-host redirect then dropped connection: err = %v", err)
	}
}

// Certificate names are server-chosen text, so a TLS failure reports a fixed
// category while errors.As still reaches the cause.
func TestFetchTLSFailureHasFixedText(t *testing.T) {
	srv := httptest.NewUnstartedServer(serveTag("v0.3.24"))
	srv.Config.ErrorLog = log.New(io.Discard, "", 0)
	srv.StartTLS()
	t.Cleanup(srv.Close)
	_, err := (&Fetcher{URL: srv.URL}).fetchLatestStable(context.Background())
	var certErr *tls.CertificateVerificationError
	if !errors.As(err, &certErr) {
		t.Fatalf("err = %v, want a certificate verification error", err)
	}
	if err.Error() != "updatecheck: fetch latest release: TLS certificate rejected" {
		t.Errorf("err = %q", err)
	}
}
