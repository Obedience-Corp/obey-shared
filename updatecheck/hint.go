package updatecheck

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const festivalBinary = "festival"

var homebrewPrefixes = []string{
	"/opt/homebrew/",
	"/usr/local/cellar/",
	"/usr/local/caskroom/",
	"/usr/local/homebrew/",
	"/home/linuxbrew/.linuxbrew/",
}

// ResolveUpgradeHint picks the upgrade action for the binary at
// executablePath ("" means os.Executable):
//
//  1. a festival binary beside the symlink-resolved executable, or on PATH:
//     run "festival update", which knows the install origin
//  2. an obvious Homebrew prefix: run "brew upgrade --cask festival"
//  3. otherwise: see the latest release page
func ResolveUpgradeHint(executablePath string) UpgradeHint {
	exe := resolveExecutable(executablePath)
	if festivalAvailable(exe) {
		return UpgradeHint{Kind: HintRun, Text: FestivalUpdateCommand}
	}
	if isHomebrewPath(exe) {
		return UpgradeHint{Kind: HintRun, Text: BrewUpgradeCommand}
	}
	return UpgradeHint{Kind: HintSee, Text: DefaultReleasePage}
}

func resolveExecutable(p string) string {
	if p == "" {
		exe, err := os.Executable()
		if err != nil {
			return ""
		}
		p = exe
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return p
}

func festivalAvailable(exe string) bool {
	if exe != "" {
		name := festivalBinary
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		if isExecutableFile(filepath.Join(filepath.Dir(exe), name)) {
			return true
		}
	}
	_, err := exec.LookPath(festivalBinary)
	return err == nil
}

func isExecutableFile(p string) bool {
	fi, err := os.Stat(p)
	if err != nil || !fi.Mode().IsRegular() {
		return false
	}
	return runtime.GOOS == "windows" || fi.Mode().Perm()&0o111 != 0
}

func isHomebrewPath(p string) bool {
	if p == "" {
		return false
	}
	p = strings.ToLower(filepath.ToSlash(p))
	for _, prefix := range homebrewPrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}
