//go:build container_fs

package updatecheck

import (
	"os"
	"path/filepath"
	"testing"
)

var (
	festivalHint = UpgradeHint{Kind: HintRun, Text: FestivalUpdateCommand}
	seeHint      = UpgradeHint{Kind: HintSee, Text: DefaultReleasePage}
)

// suiteFestival copies the running test binary, whose build info names the
// obey-shared module, to path.
func suiteFestival(t *testing.T, path string) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	writeRaw(t, path, string(data))
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func touch(t *testing.T, path string, mode os.FileMode) string {
	t.Helper()
	writeRaw(t, path, "#!/bin/sh\n")
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	return path
}

// camp is a symlink in bin/ to a versioned install dir that also holds
// festival, the cask and npm layout.
func linkedInstall(t *testing.T) (root, linked string) {
	t.Helper()
	root = t.TempDir()
	target := touch(t, filepath.Join(root, "cask", "0.3.24", "camp"), 0o755)
	linked = filepath.Join(root, "bin", "camp")
	if err := os.MkdirAll(filepath.Dir(linked), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, linked); err != nil {
		t.Fatal(err)
	}
	return root, linked
}

func TestHintPrefersFestivalBesideResolvedExecutable(t *testing.T) {
	t.Setenv("PATH", "")
	root, linked := linkedInstall(t)
	suiteFestival(t, filepath.Join(root, "cask", "0.3.24", "festival"))

	if got := ResolveUpgradeHint(linked); got != festivalHint {
		t.Errorf("hint = %+v, want %+v", got, festivalHint)
	}
}

// Only the resolved executable's directory counts as "beside".
func TestHintIgnoresFestivalBesideTheSymlink(t *testing.T) {
	t.Setenv("PATH", "")
	root, linked := linkedInstall(t)
	suiteFestival(t, filepath.Join(root, "bin", "festival"))

	if got := ResolveUpgradeHint(linked); got != seeHint {
		t.Errorf("hint = %+v, want %+v", got, seeHint)
	}
}

func TestHintFindsFestivalOnPath(t *testing.T) {
	pathDir := t.TempDir()
	suiteFestival(t, filepath.Join(pathDir, "festival"))
	t.Setenv("PATH", pathDir)
	_, linked := linkedInstall(t)

	if got := ResolveUpgradeHint(linked); got != festivalHint {
		t.Errorf("hint = %+v, want %+v", got, festivalHint)
	}
}

// An executable named festival that is not the suite hub (Debian's speech
// synthesizer) must never earn "run: festival update", beside or on PATH.
func TestHintRejectsForeignFestival(t *testing.T) {
	root, linked := linkedInstall(t)
	touch(t, filepath.Join(root, "cask", "0.3.24", "festival"), 0o755)
	pathDir := t.TempDir()
	touch(t, filepath.Join(pathDir, "festival"), 0o755)
	t.Setenv("PATH", pathDir)

	if got := ResolveUpgradeHint(linked); got != seeHint {
		t.Errorf("hint = %+v, want %+v", got, seeHint)
	}
}

func TestHintSkipsNonExecutableFestival(t *testing.T) {
	t.Setenv("PATH", "")
	root, linked := linkedInstall(t)
	dir := filepath.Join(root, "cask", "0.3.24")

	touch(t, filepath.Join(dir, "festival"), 0o644)
	if got := ResolveUpgradeHint(linked); got != seeHint {
		t.Errorf("non-executable file: hint = %+v", got)
	}

	if err := os.Remove(filepath.Join(dir, "festival")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "festival"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := ResolveUpgradeHint(linked); got != seeHint {
		t.Errorf("directory: hint = %+v", got)
	}
}

func TestHintDefaultsToRunningExecutable(t *testing.T) {
	t.Setenv("PATH", "")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(exe), "festival")); err == nil {
		t.Skip("festival sits beside the test binary")
	}
	if got := ResolveUpgradeHint(""); got != seeHint {
		t.Errorf("hint = %+v, want %+v", got, seeHint)
	}
}

// npm global install: the package's bin/ holds node shims named camp, fest,
// festival and the Go binaries as camp-bin, fest-bin, festival-bin; the
// prefix's bin/ symlinks to the shims.
func npmInstall(t *testing.T) (prefixBin, pkgBin string) {
	t.Helper()
	root := t.TempDir()
	pkgBin = filepath.Join(root, "lib", "node_modules", "@obedience-corp", "festival", "bin")
	prefixBin = filepath.Join(root, "bin")
	for _, name := range []string{"camp", "fest", "festival"} {
		touch(t, filepath.Join(pkgBin, name), 0o755)
		if err := os.MkdirAll(prefixBin, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(pkgBin, name), filepath.Join(prefixBin, name)); err != nil {
			t.Fatal(err)
		}
	}
	touch(t, filepath.Join(pkgBin, "camp-bin"), 0o755)
	suiteFestival(t, filepath.Join(pkgBin, "festival-bin"))
	return prefixBin, pkgBin
}

func TestHintFindsNpmFestivalBinBesideExecutable(t *testing.T) {
	t.Setenv("PATH", "")
	_, pkgBin := npmInstall(t)

	if got := ResolveUpgradeHint(filepath.Join(pkgBin, "camp-bin")); got != festivalHint {
		t.Errorf("hint = %+v, want %+v", got, festivalHint)
	}
}

func TestHintFindsNpmFestivalShimOnPath(t *testing.T) {
	prefixBin, _ := npmInstall(t)
	t.Setenv("PATH", prefixBin)

	if got := ResolveUpgradeHint(absentExe); got != festivalHint {
		t.Errorf("hint = %+v, want %+v", got, festivalHint)
	}
}
