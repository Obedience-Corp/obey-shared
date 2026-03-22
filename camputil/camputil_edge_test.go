package camputil

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestFindCampaignRoot_Symlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test not supported on Windows")
	}

	// Create temp directory structure
	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	// Create real campaign
	realCampaign := filepath.Join(tmpDir, "real-campaign")
	if err := os.MkdirAll(filepath.Join(realCampaign, CampaignDir), 0755); err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	// Create symlink to campaign
	symlinkPath := filepath.Join(tmpDir, "symlinked-campaign")
	if err := os.Symlink(realCampaign, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	ctx := context.Background()

	// Detection from symlink should return the resolved real path
	got, err := FindCampaignRoot(ctx, symlinkPath)
	if err != nil {
		t.Fatalf("FindCampaignRoot() from symlink error = %v", err)
	}

	if got != realCampaign {
		t.Errorf("FindCampaignRoot() from symlink = %v, want %v", got, realCampaign)
	}
}

func TestFindCampaignRoot_NestedSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test not supported on Windows")
	}

	// Create temp directory structure
	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	// Create real campaign with nested structure
	realCampaign := filepath.Join(tmpDir, "real-campaign")
	nestedDir := filepath.Join(realCampaign, "projects", "foo")
	if err := os.MkdirAll(filepath.Join(realCampaign, CampaignDir), 0755); err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	// Create symlink to nested directory
	symlinkPath := filepath.Join(tmpDir, "symlinked-nested")
	if err := os.Symlink(nestedDir, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	ctx := context.Background()

	// Detection from symlinked nested directory should find the real campaign
	got, err := FindCampaignRoot(ctx, symlinkPath)
	if err != nil {
		t.Fatalf("FindCampaignRoot() from nested symlink error = %v", err)
	}

	if got != realCampaign {
		t.Errorf("FindCampaignRoot() from nested symlink = %v, want %v", got, realCampaign)
	}
}

func TestFindCampaignRoot_EnvVarCanonicalized(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test not supported on Windows")
	}

	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	// Create real campaign
	realCampaign := filepath.Join(tmpDir, "real-campaign")
	if err := os.MkdirAll(filepath.Join(realCampaign, CampaignDir), 0755); err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	// Create symlink to campaign
	symlinkPath := filepath.Join(tmpDir, "linked-campaign")
	if err := os.Symlink(realCampaign, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Set CAMP_ROOT to the symlink — returned path must be the resolved real path
	t.Setenv(EnvCampaignRoot, symlinkPath)

	got, err := FindCampaignRoot(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("FindCampaignRoot() error = %v", err)
	}

	if got != realCampaign {
		t.Errorf("FindCampaignRoot() with symlinked CAMP_ROOT = %v, want %v (resolved path)", got, realCampaign)
	}
}

func TestFindCampaignRoot_NonCanonicalPath(t *testing.T) {
	// Create temp campaign
	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	campaignRoot := filepath.Join(tmpDir, "campaign")
	if err := os.MkdirAll(filepath.Join(campaignRoot, CampaignDir), 0755); err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	ctx := context.Background()

	// Test with path containing ..
	nonCanonicalPath := filepath.Join(campaignRoot, "subdir", "..")
	os.MkdirAll(filepath.Join(campaignRoot, "subdir"), 0755)

	got, err := FindCampaignRoot(ctx, nonCanonicalPath)
	if err != nil {
		t.Fatalf("FindCampaignRoot() with non-canonical path error = %v", err)
	}

	if got != campaignRoot {
		t.Errorf("FindCampaignRoot() with non-canonical path = %v, want %v", got, campaignRoot)
	}
}

func TestFindCampaignRoot_PathWithDot(t *testing.T) {
	// Create temp campaign
	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	campaignRoot := filepath.Join(tmpDir, "campaign")
	if err := os.MkdirAll(filepath.Join(campaignRoot, CampaignDir), 0755); err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	ctx := context.Background()

	// Test with path containing .
	pathWithDot := filepath.Join(campaignRoot, ".", "subdir", ".")
	os.MkdirAll(filepath.Join(campaignRoot, "subdir"), 0755)

	got, err := FindCampaignRoot(ctx, pathWithDot)
	if err != nil {
		t.Fatalf("FindCampaignRoot() with dot path error = %v", err)
	}

	if got != campaignRoot {
		t.Errorf("FindCampaignRoot() with dot path = %v, want %v", got, campaignRoot)
	}
}

func TestFindWithTimeout(t *testing.T) {
	// Create temp campaign
	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	campaignRoot := filepath.Join(tmpDir, "campaign")
	if err := os.MkdirAll(filepath.Join(campaignRoot, CampaignDir), 0755); err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	got, err := FindWithTimeout(campaignRoot)
	if err != nil {
		t.Fatalf("FindWithTimeout() error = %v", err)
	}

	if got != campaignRoot {
		t.Errorf("FindWithTimeout() = %v, want %v", got, campaignRoot)
	}
}

func TestFindFromCwdWithTimeout(t *testing.T) {
	// Create temp campaign
	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	campaignRoot := filepath.Join(tmpDir, "campaign")
	if err := os.MkdirAll(filepath.Join(campaignRoot, CampaignDir), 0755); err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	// Save and restore working directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(campaignRoot); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	got, err := FindFromCwdWithTimeout()
	if err != nil {
		t.Fatalf("FindFromCwdWithTimeout() error = %v", err)
	}

	if got != campaignRoot {
		t.Errorf("FindFromCwdWithTimeout() = %v, want %v", got, campaignRoot)
	}
}

func TestFindCampaignRoot_TimeoutExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(1 * time.Millisecond)

	_, err := FindCampaignRoot(ctx, "/some/path")
	if err != context.DeadlineExceeded {
		t.Errorf("FindCampaignRoot() with expired timeout: got %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestFindCampaignRoot_InaccessibleStartDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission test not reliable on Windows")
	}

	if os.Geteuid() == 0 {
		t.Skip("test requires non-root user")
	}

	// Create temp directory structure
	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	// Create campaign with a nested path that becomes inaccessible before detection.
	campaignRoot := filepath.Join(tmpDir, "campaign")
	restrictedDir := filepath.Join(campaignRoot, "restricted")
	nestedDir := filepath.Join(restrictedDir, "nested")

	if err := os.MkdirAll(filepath.Join(campaignRoot, CampaignDir), 0755); err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	if err := os.Chmod(restrictedDir, 0o000); err != nil {
		t.Fatalf("failed to chmod restricted dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(restrictedDir, 0o755)
	})

	// FindCampaignRoot resolves startDir before walking up, so an unreadable start
	// path must surface a permission error instead of falling back past it.
	_, err := FindCampaignRoot(context.Background(), nestedDir)
	if !os.IsPermission(err) {
		t.Fatalf("FindCampaignRoot() error = %v, want permission denied", err)
	}
}

func BenchmarkFindWithTimeout(b *testing.B) {
	tmpDir := b.TempDir()
	campaignRoot := filepath.Join(tmpDir, "campaign")
	campaignDir := filepath.Join(campaignRoot, CampaignDir)
	deepDir := filepath.Join(campaignRoot, "a", "b", "c", "d", "e")

	os.MkdirAll(campaignDir, 0755)
	os.MkdirAll(deepDir, 0755)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FindWithTimeout(deepDir)
	}
}
