package updatecheck

import (
	"cmp"
	"regexp"
	"strings"
)

const maxVersionLen = 64

var semverPattern = regexp.MustCompile(
	`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)` +
		`(?:-((?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)` +
		`(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*))?$`)

type semver struct {
	core [3]string
	pre  []string
}

func parseVersion(v string) (semver, bool) {
	if len(v) > maxVersionLen {
		return semver{}, false
	}
	m := semverPattern.FindStringSubmatch(v)
	if m == nil {
		return semver{}, false
	}
	s := semver{core: [3]string{m[1], m[2], m[3]}}
	if m[4] != "" {
		s.pre = strings.Split(m[4], ".")
	}
	return s, true
}

func isStableVersion(v string) bool {
	s, ok := parseVersion(v)
	return ok && len(s.pre) == 0
}

// NormalizeVersion strips one leading "v".
func NormalizeVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

// IsReleaseVersion reports whether v is X.Y.Z with an optional leading "v"
// and an optional prerelease suffix. Build metadata, whitespace, and anything
// else are rejected.
func IsReleaseVersion(v string) bool {
	_, ok := parseVersion(v)
	return ok
}

// UpdateAvailable reports whether remote is a newer release than local. Either
// side failing IsReleaseVersion yields false.
func UpdateAvailable(local, remote string) bool {
	l, lok := parseVersion(local)
	r, rok := parseVersion(remote)
	return lok && rok && compareSemver(l, r) < 0
}

func compareSemver(a, b semver) int {
	for i := range a.core {
		if c := compareNumeric(a.core[i], b.core[i]); c != 0 {
			return c
		}
	}
	return comparePrerelease(a.pre, b.pre)
}

// compareNumeric orders digit strings that have no leading zeros, so length
// decides first and no integer parse can overflow.
func compareNumeric(a, b string) int {
	if c := cmp.Compare(len(a), len(b)); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}

func comparePrerelease(a, b []string) int {
	switch {
	case len(a) == 0 && len(b) == 0:
		return 0
	case len(a) == 0:
		return 1
	case len(b) == 0:
		return -1
	}
	for i := 0; i < min(len(a), len(b)); i++ {
		aNum, bNum := isNumeric(a[i]), isNumeric(b[i])
		var c int
		switch {
		case aNum && bNum:
			c = compareNumeric(a[i], b[i])
		case aNum:
			c = -1
		case bNum:
			c = 1
		default:
			c = strings.Compare(a[i], b[i])
		}
		if c != 0 {
			return c
		}
	}
	return cmp.Compare(len(a), len(b))
}

func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}
