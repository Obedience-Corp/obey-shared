package updatecheck

import "testing"

func TestIsHomebrewPath(t *testing.T) {
	tests := map[string]bool{
		"/opt/homebrew/Caskroom/festival/0.3.24/camp":    true,
		"/opt/homebrew/bin/camp":                         true,
		"/usr/local/Caskroom/festival/0.3.24/camp":       true,
		"/usr/local/Cellar/festival/0.3.24/bin/fest":     true,
		"/home/linuxbrew/.linuxbrew/Caskroom/f/1/camp":   true,
		"/usr/local/bin/camp":                            false,
		"/Users/me/src/homebrew/camp":                    false,
		"/Users/me/.obey/installer/bin/camp":             false,
		"/home/me/.local/bin/camp":                       false,
		"/usr/lib/node_modules/@obedience-corp/bin/camp": false,
		"": false,
	}
	for p, want := range tests {
		if got := isHomebrewPath(p); got != want {
			t.Errorf("isHomebrewPath(%q) = %v, want %v", p, got, want)
		}
	}
}

// These paths do not exist and PATH is empty, so only the path heuristics
// decide; the sibling and PATH cases need real files and live in the
// container_fs lane.
func TestResolveUpgradeHintFallbacks(t *testing.T) {
	t.Setenv("PATH", "")
	brew := ResolveUpgradeHint("/opt/homebrew/Caskroom/updatecheck-test-absent/0.0.0/camp")
	if brew != (UpgradeHint{Kind: HintRun, Text: BrewUpgradeCommand}) {
		t.Errorf("homebrew path hint = %+v", brew)
	}
	see := ResolveUpgradeHint("/updatecheck-test-absent/bin/camp")
	if see != (UpgradeHint{Kind: HintSee, Text: DefaultReleasePage}) {
		t.Errorf("unknown path hint = %+v", see)
	}
}
