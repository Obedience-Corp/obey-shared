package updatecheck

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const maxReleaseBody = 1 << 20

var (
	// ErrReleaseStatus wraps a non-200 response from the release API.
	ErrReleaseStatus = errors.New("updatecheck: unexpected release API status")

	// ErrInvalidRelease means the release API body was not JSON, was marked
	// prerelease or draft, or its tag_name was not a stable X.Y.Z version.
	// Server content is never included.
	ErrInvalidRelease = errors.New("updatecheck: release API returned an invalid release")
)

// Fetcher loads the stable latest suite release.
type Fetcher struct {
	// URL defaults to DefaultReleaseURL.
	URL string
	// HTTPClient defaults to a client bounded by the caller's timeout.
	HTTPClient *http.Client
	// UserAgent defaults to "obey-shared-updatecheck".
	UserAgent string
}

// fetchLatestStable returns the stable latest release. It sends no
// credentials, so a stale token in the environment cannot turn every check
// into a 401.
func (f *Fetcher) fetchLatestStable(ctx context.Context) (release, error) {
	endpoint := f.URL
	if endpoint == "" {
		endpoint = DefaultReleaseURL
	}
	client := f.HTTPClient
	if client == nil {
		client = budgetClient(ExplicitTimeout)
	}
	ua := f.UserAgent
	if ua == "" {
		ua = "obey-shared-updatecheck"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return release{}, fmt.Errorf("updatecheck: build release request: %w", err)
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return release{}, newFetchError(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return release{}, fmt.Errorf("%w: %d", ErrReleaseStatus, resp.StatusCode)
	}

	var raw struct {
		TagName    string `json:"tag_name"`
		Prerelease bool   `json:"prerelease"`
		Draft      bool   `json:"draft"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxReleaseBody)).Decode(&raw); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return release{}, newFetchError(ctxErr)
		}
		return release{}, ErrInvalidRelease
	}
	if raw.Prerelease || raw.Draft || !isStableVersion(raw.TagName) {
		return release{}, ErrInvalidRelease
	}
	return releaseFromTag(raw.TagName), nil
}

// fetchError has fixed text, because transport errors can carry
// server-chosen strings (a redirect URL, certificate names). Unwrap keeps
// errors.Is and errors.As working on the cause.
type fetchError struct {
	kind string
	err  error
}

func (e *fetchError) Error() string { return "updatecheck: fetch latest release: " + e.kind }

func (e *fetchError) Unwrap() error { return e.err }

func newFetchError(err error) error {
	var (
		dnsErr  *net.DNSError
		certErr *tls.CertificateVerificationError
		netErr  net.Error
	)
	kind := "network error"
	switch {
	case errors.Is(err, context.Canceled):
		kind = "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		kind = "timed out"
	case errors.As(err, &dnsErr):
		kind = "DNS lookup failed"
	case errors.As(err, &certErr):
		kind = "TLS certificate rejected"
	case errors.As(err, &netErr) && netErr.Timeout():
		kind = "timed out"
	}
	return &fetchError{kind: kind, err: err}
}

// budgetClient follows redirects only within the original scheme and host
// (a renamed repo), so a redirect cannot steer the request or its error text
// to a server-chosen host.
func budgetClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			first := via[0].URL
			if len(via) >= 5 || req.URL.Scheme != first.Scheme || req.URL.Host != first.Host {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}

func releaseFromTag(tag string) release {
	return release{
		TagName: tag,
		Version: NormalizeVersion(tag),
		HTMLURL: releaseTagPagePrefix + tag,
	}
}
