package updatecheck

import "time"

// shouldAmbientNotify applies D3: notify when an update is available and
// enough time has passed since the last notice. A release not yet announced
// waits minGap; the release already announced is repeated every repeatGap for
// as long as the install stays behind, because one line after one command is
// easy to miss. A last notice stamped in the future (clock moved back) does
// not hold either gap open.
func shouldAmbientNotify(c *Cache, local string, now time.Time, minGap, repeatGap time.Duration) bool {
	if c == nil || !UpdateAvailable(local, c.LatestVersion) {
		return false
	}
	gap := minGap
	if NormalizeVersion(c.LastNotifiedVersion) == c.LatestVersion {
		gap = repeatGap
	}
	last := c.LastNotifiedAt
	if !last.IsZero() && !last.After(now) && now.Sub(last) < gap {
		return false
	}
	return true
}

func formatNotice(local, latest string, hint UpgradeHint) Notice {
	return Notice{
		Message: "Festival suite update available (" + NormalizeVersion(local) + " → " + NormalizeVersion(latest) + ")",
		Command: hint.Text,
		Kind:    hint.Kind,
	}
}
