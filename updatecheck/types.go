package updatecheck

import (
	"fmt"
	"io"
	"time"
)

const (
	// SchemaVersion identifies the on-disk cache contract.
	SchemaVersion = "festival-suite-update-check/v1alpha1"

	// Source identifies the suite version source of truth.
	Source = "github:Obedience-Corp/festival"

	// ChannelStable is the only channel in v1.
	ChannelStable = "stable"

	// EnvNoUpdateCheck disables the ambient path when set to any value other
	// than "", "0", "false", "no", or "off".
	EnvNoUpdateCheck = "FESTIVAL_NO_UPDATE_CHECK"

	// DefaultCacheTTL is how long a successful remote check stays fresh.
	DefaultCacheTTL = 24 * time.Hour

	// DefaultNotifyMinGap is the minimum time between ambient notices.
	DefaultNotifyMinGap = 6 * time.Hour

	// DefaultRetryBackoff is how long the ambient path waits after a failed
	// refresh before trying the network again, when a cached release exists.
	DefaultRetryBackoff = time.Hour

	// AmbientTimeout bounds the network refresh on the ambient path.
	AmbientTimeout = 500 * time.Millisecond

	// ExplicitTimeout bounds the network refresh on the Check path.
	ExplicitTimeout = 5 * time.Second

	// DefaultReleaseURL is the GitHub API endpoint for the stable latest release.
	DefaultReleaseURL = "https://api.github.com/repos/Obedience-Corp/festival/releases/latest"

	// DefaultReleasePage is the human-facing latest release page.
	DefaultReleasePage = "https://github.com/Obedience-Corp/festival/releases/latest"

	releaseTagPagePrefix = "https://github.com/Obedience-Corp/festival/releases/tag/"
)

const (
	// HintRun marks an UpgradeHint whose Text is a shell command.
	HintRun = "run"

	// HintSee marks an UpgradeHint whose Text is a URL.
	HintSee = "see"

	// FestivalUpdateCommand is the hint when a festival hub binary is present.
	FestivalUpdateCommand = "festival update"

	// BrewUpgradeCommand is the hint for an obvious Homebrew install.
	BrewUpgradeCommand = "brew upgrade --cask festival"
)

// Cache is the persisted state in suite.json, shared by every suite CLI on the
// machine.
type Cache struct {
	SchemaVersion       string    `json:"schema_version"`
	Source              string    `json:"source"`
	Channel             string    `json:"channel"`
	CheckedAt           time.Time `json:"checked_at"`
	LatestTag           string    `json:"latest_tag"`
	LatestVersion       string    `json:"latest_version"`
	HTMLURL             string    `json:"html_url,omitempty"`
	LastNotifiedAt      time.Time `json:"last_notified_at,omitzero"`
	LastNotifiedVersion string    `json:"last_notified_version,omitempty"`
	LastAttemptAt       time.Time `json:"last_attempt_at,omitzero"`
}

// Release is a validated stable suite release.
type Release struct {
	TagName string
	Version string
	HTMLURL string
}

// Result is the outcome of Check.
type Result struct {
	CurrentVersion  string    `json:"current_version"`
	LatestVersion   string    `json:"latest_version,omitempty"`
	LatestTag       string    `json:"latest_tag,omitempty"`
	HTMLURL         string    `json:"html_url,omitempty"`
	Source          string    `json:"source"`
	Channel         string    `json:"channel"`
	UpdateAvailable bool      `json:"update_available"`
	UpgradeCommand  string    `json:"upgrade_command,omitempty"`
	UpgradeKind     string    `json:"upgrade_kind,omitempty"`
	CheckedAt       time.Time `json:"checked_at,omitzero"`
	FromCache       bool      `json:"from_cache"`
	DevOrNonRelease bool      `json:"dev_or_non_release,omitempty"`

	// RefreshErr is the network refresh failure, if one was attempted and
	// failed. Latest fields may still be populated from the cache.
	RefreshErr error `json:"-"`
}

// Notice is an ambient advisory for the caller to render on stderr.
type Notice struct {
	// Message is one line, for example
	// "Festival suite update available (0.3.23 → 0.3.24)".
	Message string
	// Command is the action: a command for HintRun, a URL for HintSee.
	Command string
	// Kind is HintRun or HintSee and labels the action line.
	Kind string
}

// Render writes the notice as two plain-text lines.
func (n Notice) Render(w io.Writer) {
	_, _ = fmt.Fprintf(w, "⚠ notice: %s\n  %s: %s\n", n.Message, n.Kind, n.Command)
}

// UpgradeHint is how the user should upgrade.
type UpgradeHint struct {
	// Kind is HintRun or HintSee.
	Kind string
	// Text is the command or URL.
	Text string
}
