package updatecheck

import (
	"strings"
	"testing"
)

func TestIsReleaseVersion(t *testing.T) {
	valid := []string{
		"0.3.24", "v0.3.24", "v1.0.0-rc.1", "1.0.0-alpha.beta.1", "v10.20.30",
		"1.0.0-0", "1.0.0-x-y-z", "1.0.0-0a",
	}
	invalid := []string{
		"", "dev", "unknown", "v", "1", "1.2", "v1.2", "1.2.3.4", "01.2.3", "1.02.3",
		"1.2.03", "1.0.0-01", "1.2.3+build", "v1.2.3+sha.abc", " v1.2.3", "v1.2.3 ",
		"v1.2.3\n", "vv1.2.3", "V1.2.3", "1.2.3-", "1.2.3-rc..1", "1.2.3-rc.$",
		"v1.2.3\x1b[31m", "v1.2.3-\x1b[2J", "1.2.3-é",
		"1.2.3-" + strings.Repeat("a", maxVersionLen),
	}
	for _, v := range valid {
		if !IsReleaseVersion(v) {
			t.Errorf("IsReleaseVersion(%q) = false, want true", v)
		}
	}
	for _, v := range invalid {
		if IsReleaseVersion(v) {
			t.Errorf("IsReleaseVersion(%q) = true, want false", v)
		}
	}
}

func TestUpdateAvailable(t *testing.T) {
	tests := []struct {
		local, remote string
		want          bool
	}{
		{"v0.3.23", "v0.3.24", true},
		{"0.3.23", "v0.3.24", true},
		{"v0.3.24", "v0.3.24", false},
		{"v0.3.24", "0.3.24", false},
		{"v0.3.25", "v0.3.24", false},
		{"v0.9.0", "v0.10.0", true},
		{"v0.10.0", "v0.9.0", false},
		{"v1.9.9", "v2.0.0", true},
		{"v0.3.24-rc.1", "v0.3.24", true},
		{"v0.3.24", "v0.3.25-rc.1", true},
		{"v0.3.25-rc.1", "v0.3.24", false},
		{"v99999999999999999999999.0.0", "v100000000000000000000000.0.0", true},
		{"", "v0.3.24", false},
		{"dev", "v0.3.24", false},
		{"v0.3.23", "", false},
		{"v0.3.23", "latest", false},
		{"v0.3.23", "v0.3.24+build", false},
	}
	for _, tt := range tests {
		if got := UpdateAvailable(tt.local, tt.remote); got != tt.want {
			t.Errorf("UpdateAvailable(%q, %q) = %v, want %v", tt.local, tt.remote, got, tt.want)
		}
	}
}

// The ordering example from semver 2.0.0 section 11.
func TestPrereleasePrecedenceFollowsSemver(t *testing.T) {
	ordered := []string{
		"1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.beta", "1.0.0-beta",
		"1.0.0-beta.2", "1.0.0-beta.11", "1.0.0-rc.1", "1.0.0",
	}
	for i := range ordered {
		for j := range ordered {
			if got, want := UpdateAvailable(ordered[i], ordered[j]), i < j; got != want {
				t.Errorf("UpdateAvailable(%q, %q) = %v, want %v", ordered[i], ordered[j], got, want)
			}
		}
	}
}

func TestNormalizeVersionStripsOneV(t *testing.T) {
	for in, want := range map[string]string{"v0.3.24": "0.3.24", "0.3.24": "0.3.24", "vv1": "v1", "": ""} {
		if got := NormalizeVersion(in); got != want {
			t.Errorf("NormalizeVersion(%q) = %q, want %q", in, got, want)
		}
	}
}
