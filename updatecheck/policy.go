package updatecheck

import "time"

// shouldAmbientNotify applies D3: notify when an update is available, the
// latest version was never notified, and at least minGap has passed since the
// last notice. A last notice stamped in the future (clock moved back) does not
// hold the gap open.
func shouldAmbientNotify(c *Cache, local string, now time.Time, minGap time.Duration) bool {
	if c == nil || !UpdateAvailable(local, c.LatestVersion) {
		return false
	}
	if NormalizeVersion(c.LastNotifiedVersion) == c.LatestVersion {
		return false
	}
	last := c.LastNotifiedAt
	if !last.IsZero() && !last.After(now) && now.Sub(last) < minGap {
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
