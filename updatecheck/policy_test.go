package updatecheck

import (
	"bytes"
	"testing"
	"time"
)

func notified(c *Cache, version string, when time.Time) *Cache {
	c.LastNotifiedVersion = version
	c.LastNotifiedAt = when
	return c
}

func TestShouldAmbientNotifyD3(t *testing.T) {
	const gap = DefaultNotifyMinGap
	const repeat = DefaultNotifyRepeatGap
	tests := []struct {
		name  string
		cache *Cache
		local string
		now   time.Time
		want  bool
	}{
		{"new version, never notified", cacheWith("0.3.24", t0), "v0.3.23", t0, true},
		{"same version within gap", notified(cacheWith("0.3.24", t0), "0.3.24", t0), "v0.3.23", t0.Add(time.Hour), false},
		{"same version after the new-release gap", notified(cacheWith("0.3.24", t0), "0.3.24", t0), "v0.3.23", t0.Add(gap), false},
		{"same version just before the repeat gap", notified(cacheWith("0.3.24", t0), "0.3.24", t0), "v0.3.23", t0.Add(repeat - time.Second), false},
		{"same version at the repeat gap", notified(cacheWith("0.3.24", t0), "0.3.24", t0), "v0.3.23", t0.Add(repeat), true},
		{"same version a week later", notified(cacheWith("0.3.24", t0), "0.3.24", t0), "v0.3.23", t0.Add(7 * 24 * time.Hour), true},
		{"same version, last notice in the future", notified(cacheWith("0.3.24", t0), "0.3.24", t0.Add(48*time.Hour)), "v0.3.23", t0, true},
		{"same version after updating", notified(cacheWith("0.3.24", t0), "0.3.24", t0), "v0.3.24", t0.Add(7 * 24 * time.Hour), false},
		{"newer version after gap", notified(cacheWith("0.3.25", t0), "0.3.24", t0), "v0.3.23", t0.Add(gap), true},
		{"newer version within gap", notified(cacheWith("0.3.25", t0), "0.3.24", t0), "v0.3.23", t0.Add(gap - time.Second), false},
		{"last notice in the future", notified(cacheWith("0.3.25", t0), "0.3.24", t0.Add(48*time.Hour)), "v0.3.23", t0, true},
		{"up to date", cacheWith("0.3.24", t0), "v0.3.24", t0, false},
		{"local ahead", cacheWith("0.3.24", t0), "v0.3.30", t0, false},
		{"no cache", nil, "v0.3.23", t0, false},
		{"no latest", &Cache{}, "v0.3.23", t0, false},
		{"empty local", cacheWith("0.3.24", t0), "", t0, false},
		{"dev local", cacheWith("0.3.24", t0), "dev", t0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldAmbientNotify(tt.cache, tt.local, tt.now, gap, repeat); got != tt.want {
				t.Errorf("shouldAmbientNotify = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoticeRendersTwoPlainLines(t *testing.T) {
	n := formatNotice("v0.3.23", "0.3.24", UpgradeHint{Kind: HintRun, Text: FestivalUpdateCommand})
	var buf bytes.Buffer
	n.Render(&buf)
	want := "⚠ notice: Festival suite update available (0.3.23 → 0.3.24)\n  run: festival update\n"
	if buf.String() != want {
		t.Errorf("Render =\n%q\nwant\n%q", buf.String(), want)
	}

	see := formatNotice("v0.3.23", "0.3.24", UpgradeHint{Kind: HintSee, Text: DefaultReleasePage})
	buf.Reset()
	see.Render(&buf)
	want = "⚠ notice: Festival suite update available (0.3.23 → 0.3.24)\n  see: " + DefaultReleasePage + "\n"
	if buf.String() != want {
		t.Errorf("Render =\n%q\nwant\n%q", buf.String(), want)
	}
}
