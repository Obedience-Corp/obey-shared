// Package updatecheck tells camp and fest users when a newer Festival suite
// release exists.
//
// The version source of truth is the stable latest release of
// Obedience-Corp/festival on GitHub. Callers pass their suite version (the
// version.Bundle ldflag, for example "v0.3.24"), never their component
// version. An empty suite version marks a non-suite build, and the package then
// does nothing at all: no cache read, no network, no output.
//
// MaybeNotify is the ambient path. It is cache-first and fails open. It
// refreshes within AmbientTimeout, at most once per DefaultCacheTTL after a
// success and once per DefaultRetryBackoff after a failure, and returns a
// Notice at most once per new release per machine with DefaultNotifyMinGap
// between notices. Check is the explicit "version --check" path.
//
// The package never installs or upgrades anything. The upgrade hint is computed
// locally; the only server-derived string that can reach a terminal is a
// release version that passed strict semver validation.
//
// Design: workflow/design/.dungeon/completed/2026-07-25/suite-update-notifications
// in the obey-campaign camp (WI-2f3133).
package updatecheck
