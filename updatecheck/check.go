package updatecheck

import (
	"context"
	"time"
)

// Options configure Check and MaybeNotify. Production callers set Suppress
// (MaybeNotify only) and leave the rest zero.
type Options struct {
	// CachePath is suite.json. Empty means DefaultCachePath(). Every file the
	// package creates (the cache, its directory, its temp file) derives from it.
	CachePath string

	// ExecutablePath is the running binary, used for the upgrade hint.
	// Empty means os.Executable().
	ExecutablePath string

	// Suppress is the caller's process context for MaybeNotify.
	Suppress SuppressInput

	// ForceRefresh makes Check fetch even when the cache is fresh.
	ForceRefresh bool

	// Fetcher defaults to a Fetcher for DefaultReleaseURL.
	Fetcher *Fetcher

	// Now defaults to time.Now.
	Now func() time.Time

	// CacheTTL defaults to DefaultCacheTTL.
	CacheTTL time.Duration

	// NotifyMinGap defaults to DefaultNotifyMinGap.
	NotifyMinGap time.Duration

	// RetryBackoff defaults to DefaultRetryBackoff.
	RetryBackoff time.Duration
}

// MaybeNotify is the ambient path. It returns a Notice the caller should
// render on stderr, or nil. It never writes to any writer and never fails.
//
// It does no I/O at all when suiteVersion is not a release version (empty
// means a non-suite build) or when Suppress says so. Otherwise it reads the
// cache, refreshes within AmbientTimeout if the cache is older than CacheTTL
// and no refresh failed within RetryBackoff, and applies the D3 display
// policy. A caller that cancels during the refresh gets no notice and no
// write. The notice is returned only after the notified state is persisted,
// so a read-only home cannot cause a notice on every command.
func MaybeNotify(ctx context.Context, suiteVersion string, opts Options) *Notice {
	if !IsReleaseVersion(suiteVersion) || AmbientSuppressed(opts.Suppress) || ctx.Err() != nil {
		return nil
	}
	opts = normalizeOptions(opts)
	if opts.CachePath == "" {
		return nil
	}
	now := opts.Now()
	cache, err := LoadCache(opts.CachePath)
	if err != nil {
		cache = nil
	}

	cache, dirty := refreshAmbient(ctx, cache, now, opts)
	if ctx.Err() != nil {
		return nil
	}
	notify := shouldAmbientNotify(cache, suiteVersion, now, opts.NotifyMinGap)
	if notify {
		markNotified(cache, now)
	}
	if !dirty && !notify {
		return nil
	}
	if err := saveCache(opts.CachePath, cache); err != nil || !notify {
		return nil
	}
	n := formatNotice(suiteVersion, cache.LatestVersion, ResolveUpgradeHint(opts.ExecutablePath))
	return &n
}

// refreshAmbient fetches when the cache is stale. A failure records the
// attempt, with or without a cached release, so a slow or offline link pays
// the timeout at most once per RetryBackoff. An attempt-only record carries
// no release.
func refreshAmbient(ctx context.Context, cache *Cache, now time.Time, opts Options) (*Cache, bool) {
	if cacheFresh(cache, now, opts.CacheTTL) || recentAttempt(cache, now, opts.RetryBackoff) {
		return cache, false
	}
	rel, err := fetchWithBudget(ctx, opts, AmbientTimeout)
	if err == nil {
		return applyRelease(cache, rel, now), true
	}
	if cache == nil {
		cache = &Cache{}
	}
	cache.LastAttemptAt = now.UTC()
	return cache, true
}

// Check is the explicit "version --check" path. It refreshes within
// ExplicitTimeout when the cache is stale or ForceRefresh is set, ignores the
// ambient suppress and display policy, and never writes notified state.
// Network failures are reported in Result.RefreshErr, not as an error.
//
// An empty suiteVersion returns a DevOrNonRelease result without any I/O.
// The only error is the context's.
func Check(ctx context.Context, suiteVersion string, opts Options) (Result, error) {
	if suiteVersion == "" {
		return Result{Source: Source, Channel: ChannelStable, DevOrNonRelease: true}, nil
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	opts = normalizeOptions(opts)
	now := opts.Now()

	var cache *Cache
	if opts.CachePath != "" {
		if c, err := LoadCache(opts.CachePath); err == nil {
			cache = c
		}
	}

	refreshed := false
	var refreshErr error
	if opts.ForceRefresh || !cacheFresh(cache, now, opts.CacheTTL) {
		rel, err := fetchWithBudget(ctx, opts, ExplicitTimeout)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Result{}, ctxErr
		}
		if err != nil {
			refreshErr = err
		} else {
			cache = applyRelease(cache, rel, now)
			refreshed = true
			if opts.CachePath != "" {
				_ = saveCache(opts.CachePath, cache)
			}
		}
	}

	res := resultFrom(suiteVersion, cache, opts)
	res.FromCache = hasLatest(cache) && !refreshed
	res.RefreshErr = refreshErr
	return res, nil
}

func resultFrom(local string, cache *Cache, opts Options) Result {
	res := Result{
		CurrentVersion:  NormalizeVersion(local),
		Source:          Source,
		Channel:         ChannelStable,
		DevOrNonRelease: !IsReleaseVersion(local),
	}
	if hasLatest(cache) {
		res.LatestVersion = cache.LatestVersion
		res.LatestTag = cache.LatestTag
		res.HTMLURL = releaseTagPagePrefix + cache.LatestTag
		res.CheckedAt = cache.CheckedAt
	}
	res.UpdateAvailable = UpdateAvailable(local, res.LatestVersion)
	if res.UpdateAvailable {
		hint := ResolveUpgradeHint(opts.ExecutablePath)
		res.UpgradeKind = hint.Kind
		res.UpgradeCommand = hint.Text
	}
	return res
}

func normalizeOptions(opts Options) Options {
	if opts.CachePath == "" {
		opts.CachePath = DefaultCachePath()
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.CacheTTL <= 0 {
		opts.CacheTTL = DefaultCacheTTL
	}
	if opts.NotifyMinGap <= 0 {
		opts.NotifyMinGap = DefaultNotifyMinGap
	}
	if opts.RetryBackoff <= 0 {
		opts.RetryBackoff = DefaultRetryBackoff
	}
	if opts.Fetcher == nil {
		opts.Fetcher = &Fetcher{}
	}
	return opts
}

func fetchWithBudget(ctx context.Context, opts Options, timeout time.Duration) (release, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	f := *opts.Fetcher
	if f.HTTPClient == nil {
		f.HTTPClient = budgetClient(timeout)
	}
	return f.fetchLatestStable(ctx)
}
