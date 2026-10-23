package updatecheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const maxReleaseBody = 1 << 20

var (
	// ErrReleaseStatus wraps a non-200 response from the release API.
	ErrReleaseStatus = errors.New("updatecheck: unexpected release API status")

	// ErrInvalidRelease means the release API body was not JSON, was marked
	// prerelease or draft, or its tag_name was not a release version. Server
	// content is never included.
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
	url := f.URL
	if url == "" {
		url = DefaultReleaseURL
	}
	client := f.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: ExplicitTimeout}
	}
	ua := f.UserAgent
	if ua == "" {
		ua = "obey-shared-updatecheck"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return release{}, fmt.Errorf("updatecheck: build release request: %w", err)
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return release{}, fmt.Errorf("updatecheck: fetch latest release: %w", err)
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
			return release{}, fmt.Errorf("updatecheck: read latest release: %w", ctxErr)
		}
		return release{}, ErrInvalidRelease
	}
	if raw.Prerelease || raw.Draft || !IsReleaseVersion(raw.TagName) {
		return release{}, ErrInvalidRelease
	}
	return releaseFromTag(raw.TagName), nil
}

func releaseFromTag(tag string) release {
	return release{
		TagName: tag,
		Version: NormalizeVersion(tag),
		HTMLURL: releaseTagPagePrefix + tag,
	}
}
