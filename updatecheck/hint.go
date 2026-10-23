package updatecheck

import (
	"debug/buildinfo"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	festivalBinary    = "festival"
	suiteModulePrefix = "github.com/Obedience-Corp/"
)

// homebrewStores are the Cellar and Caskroom dirs of the default Homebrew
// prefixes, matching the hub's brewOwns. A bare prefix is not enough: npm's
// global root under /opt/homebrew/lib is not a cask install.
var homebrewStores = []string{
	"/opt/homebrew/caskroom/",
	"/opt/homebrew/cellar/",
	"/usr/local/caskroom/",
	"/usr/local/cellar/",
	"/home/linuxbrew/.linuxbrew/caskroom/",
	"/home/linuxbrew/.linuxbrew/cellar/",
}

// ResolveUpgradeHint picks the upgrade action for the binary at
// executablePath ("" means os.Executable):
//
//  1. the suite's festival hub beside the symlink-resolved executable, or on
//     PATH: run "festival update", which knows the install origin. A binary
//     counts only if its Go build info names an Obedience-Corp module, since
//     Debian and Ubuntu ship an unrelated speech synthesizer as
//     /usr/bin/festival. npm installs keep the binary as festival-bin beside a
//     node shim named festival.
//  2. a resolved path in a Homebrew Cellar or Caskroom: run
//     "brew upgrade --cask festival"
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
	if exe != "" && suiteFestivalAt(filepath.Join(filepath.Dir(exe), festivalBinary)) {
		return true
	}
	p, err := exec.LookPath(festivalBinary)
	if err != nil {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		p = resolved
	}
	return isSuiteFestival(p) || suiteFestivalAt(p)
}

// suiteFestivalAt checks the hub's file names for base: festival.exe on
// Windows, else festival and npm's festival-bin.
func suiteFestivalAt(base string) bool {
	if runtime.GOOS == "windows" {
		return isSuiteFestival(base + ".exe")
	}
	return isSuiteFestival(base) || isSuiteFestival(base+"-bin")
}

func isSuiteFestival(p string) bool {
	if !isExecutableFile(p) {
		return false
	}
	bi, err := buildinfo.ReadFile(p)
	if err != nil {
		return false
	}
	return strings.HasPrefix(bi.Main.Path, suiteModulePrefix) || strings.HasPrefix(bi.Path, suiteModulePrefix)
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
	for _, store := range homebrewStores {
		if strings.HasPrefix(p, store) {
			return true
		}
	}
	return false
}
