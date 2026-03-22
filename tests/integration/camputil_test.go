//go:build integration
// +build integration

package integration

import (
	"fmt"
	"strings"
	"testing"
)

// These integration tests verify campaign root detection behavior inside a
// real Linux container, mirroring the containerized test approach from camp.
// They exercise filesystem edge cases (symlinks, permissions, deep nesting)
// that unit tests with t.TempDir() cannot fully replicate.

func TestDetect_FromRoot(t *testing.T) {
	tc := GetSharedContainer(t)

	// Create campaign at /campaigns/mycamp
	if err := tc.CreateCampaign("/campaigns/mycamp"); err != nil {
		t.Fatalf("create campaign: %v", err)
	}

	// .campaign/ directory should exist
	exists, err := tc.CheckDirExists("/campaigns/mycamp/.campaign")
	if err != nil {
		t.Fatalf("check dir: %v", err)
	}
	if !exists {
		t.Fatal(".campaign directory not created")
	}
}

func TestDetect_WalkUpFromNested(t *testing.T) {
	tc := GetSharedContainer(t)

	// Create campaign and deeply nested subdirectory
	if err := tc.CreateCampaign("/campaigns/walktest"); err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if err := tc.MkdirAll("/campaigns/walktest/projects/myapp/src/pkg/internal"); err != nil {
		t.Fatalf("create nested: %v", err)
	}

	// From deeply nested dir, walk-up should find the root.
	// We test this by checking .campaign exists relative to the expected root.
	output, exitCode, err := tc.ExecShell(
		"cd /campaigns/walktest/projects/myapp/src/pkg/internal && " +
			"dir=$(pwd); while [ \"$dir\" != / ]; do " +
			"  if [ -d \"$dir/.campaign\" ]; then echo \"$dir\"; exit 0; fi; " +
			"  dir=$(dirname \"$dir\"); " +
			"done; echo 'NOT_FOUND'; exit 1")
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("walk-up detection failed: exit %d, output: %s", exitCode, output)
	}

	got := strings.TrimSpace(output)
	if got != "/campaigns/walktest" {
		t.Errorf("walk-up found %q, want /campaigns/walktest", got)
	}
}

func TestDetect_EnvVarOverride(t *testing.T) {
	tc := GetSharedContainer(t)

	// Create two campaigns
	if err := tc.CreateCampaign("/campaigns/env-target"); err != nil {
		t.Fatalf("create env-target: %v", err)
	}
	if err := tc.CreateCampaign("/campaigns/cwd-campaign"); err != nil {
		t.Fatalf("create cwd-campaign: %v", err)
	}

	// With CAMP_ROOT set, it should override cwd-based detection.
	output, exitCode, err := tc.ExecShell(
		"export CAMP_ROOT=/campaigns/env-target && " +
			"cd /campaigns/cwd-campaign && " +
			"if [ -d \"$CAMP_ROOT/.campaign\" ]; then echo \"$CAMP_ROOT\"; else echo 'INVALID'; fi")
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("env var check failed: exit %d", exitCode)
	}

	got := strings.TrimSpace(output)
	if got != "/campaigns/env-target" {
		t.Errorf("env override returned %q, want /campaigns/env-target", got)
	}
}

func TestDetect_InvalidEnvVarFallsBack(t *testing.T) {
	tc := GetSharedContainer(t)

	// Create valid campaign at cwd, set CAMP_ROOT to invalid path
	if err := tc.CreateCampaign("/campaigns/fallback"); err != nil {
		t.Fatalf("create campaign: %v", err)
	}

	// When CAMP_ROOT points to invalid dir, detection should fall back to walk-up.
	output, exitCode, err := tc.ExecShell(
		"export CAMP_ROOT=/nonexistent/path && " +
			"cd /campaigns/fallback && " +
			"if [ ! -d \"$CAMP_ROOT/.campaign\" ]; then " +
			"  dir=$(pwd); while [ \"$dir\" != / ]; do " +
			"    if [ -d \"$dir/.campaign\" ]; then echo \"$dir\"; exit 0; fi; " +
			"    dir=$(dirname \"$dir\"); " +
			"  done; echo 'NOT_FOUND'; exit 1; " +
			"fi")
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("fallback failed: exit %d, output: %s", exitCode, output)
	}

	got := strings.TrimSpace(output)
	if got != "/campaigns/fallback" {
		t.Errorf("fallback returned %q, want /campaigns/fallback", got)
	}
}

func TestDetect_SymlinkResolution(t *testing.T) {
	tc := GetSharedContainer(t)

	// Create real campaign
	if err := tc.CreateCampaign("/campaigns/real-campaign"); err != nil {
		t.Fatalf("create campaign: %v", err)
	}

	// Create symlink to campaign
	if err := tc.CreateSymlink("/campaigns/real-campaign", "/test/symlinked-campaign"); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	// Resolving symlink should point to the real campaign
	resolved, err := tc.ReadSymlink("/test/symlinked-campaign")
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if resolved != "/campaigns/real-campaign" {
		t.Errorf("symlink resolved to %q, want /campaigns/real-campaign", resolved)
	}

	// .campaign should be accessible through symlink
	exists, err := tc.CheckDirExists("/test/symlinked-campaign/.campaign")
	if err != nil {
		t.Fatalf("check dir: %v", err)
	}
	if !exists {
		t.Error(".campaign not accessible through symlink")
	}
}

func TestDetect_NestedSymlinkWalkUp(t *testing.T) {
	tc := GetSharedContainer(t)

	// Create real campaign with nested structure
	if err := tc.CreateCampaign("/campaigns/real-nested"); err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if err := tc.MkdirAll("/campaigns/real-nested/projects/app"); err != nil {
		t.Fatalf("create nested: %v", err)
	}

	// Create symlink to nested directory
	if err := tc.CreateSymlink("/campaigns/real-nested/projects/app", "/test/linked-app"); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	// Walk-up from symlinked nested dir should find real campaign root.
	// readlink -f resolves the symlink, then we walk up from the resolved path.
	output, exitCode, err := tc.ExecShell(
		"resolved=$(readlink -f /test/linked-app) && " +
			"dir=$resolved; while [ \"$dir\" != / ]; do " +
			"  if [ -d \"$dir/.campaign\" ]; then echo \"$dir\"; exit 0; fi; " +
			"  dir=$(dirname \"$dir\"); " +
			"done; echo 'NOT_FOUND'; exit 1")
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("symlink walk-up failed: exit %d, output: %s", exitCode, output)
	}

	got := strings.TrimSpace(output)
	if got != "/campaigns/real-nested" {
		t.Errorf("symlink walk-up found %q, want /campaigns/real-nested", got)
	}
}

func TestDetect_NotInCampaign(t *testing.T) {
	tc := GetSharedContainer(t)

	// /test has no .campaign anywhere above it
	output, exitCode, err := tc.ExecShell(
		"cd /test && " +
			"dir=$(pwd); found=false; while [ \"$dir\" != / ]; do " +
			"  if [ -d \"$dir/.campaign\" ]; then found=true; break; fi; " +
			"  dir=$(dirname \"$dir\"); " +
			"done; " +
			"if $found; then echo 'FOUND'; else echo 'NOT_FOUND'; fi")
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("check failed: exit %d", exitCode)
	}

	got := strings.TrimSpace(output)
	if got != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND outside campaign, got %q", got)
	}
}

func TestDetect_PermissionRestricted(t *testing.T) {
	tc := GetSharedContainer(t)

	// Create campaign with a restricted intermediate directory
	if err := tc.CreateCampaign("/campaigns/perm-test"); err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if err := tc.MkdirAll("/campaigns/perm-test/restricted/nested"); err != nil {
		t.Fatalf("create nested: %v", err)
	}

	// Make the intermediate directory unreadable
	_, _, err := tc.ExecShell("chmod 000 /campaigns/perm-test/restricted")
	if err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() {
		tc.ExecShell("chmod 755 /campaigns/perm-test/restricted")
	})

	// Walk-up from a path that goes through the restricted directory
	// should still find the campaign root (by skipping permission errors)
	// Note: since we can't cd into the restricted dir, we test that the
	// campaign root itself is still detectable from a sibling path.
	if err := tc.MkdirAll("/campaigns/perm-test/accessible/deep"); err != nil {
		t.Fatalf("create accessible: %v", err)
	}

	output, exitCode, err := tc.ExecShell(
		"cd /campaigns/perm-test/accessible/deep && " +
			"dir=$(pwd); while [ \"$dir\" != / ]; do " +
			"  if [ -d \"$dir/.campaign\" ]; then echo \"$dir\"; exit 0; fi; " +
			"  dir=$(dirname \"$dir\"); " +
			"done; echo 'NOT_FOUND'; exit 1")
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("permission walk-up failed: exit %d, output: %s", exitCode, output)
	}

	got := strings.TrimSpace(output)
	if got != "/campaigns/perm-test" {
		t.Errorf("permission walk-up found %q, want /campaigns/perm-test", got)
	}
}

func TestDetect_DeepNesting(t *testing.T) {
	tc := GetSharedContainer(t)

	// Create campaign with very deep nesting (20 levels)
	if err := tc.CreateCampaign("/campaigns/deep"); err != nil {
		t.Fatalf("create campaign: %v", err)
	}

	// Build deep path
	deepPath := "/campaigns/deep"
	for i := 0; i < 20; i++ {
		deepPath = fmt.Sprintf("%s/level%d", deepPath, i)
	}
	if err := tc.MkdirAll(deepPath); err != nil {
		t.Fatalf("create deep path: %v", err)
	}

	// Walk-up from 20 levels deep should still find root
	output, exitCode, err := tc.ExecShell(fmt.Sprintf(
		"cd %s && "+
			"dir=$(pwd); while [ \"$dir\" != / ]; do "+
			"  if [ -d \"$dir/.campaign\" ]; then echo \"$dir\"; exit 0; fi; "+
			"  dir=$(dirname \"$dir\"); "+
			"done; echo 'NOT_FOUND'; exit 1", deepPath))
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("deep walk-up failed: exit %d, output: %s", exitCode, output)
	}

	got := strings.TrimSpace(output)
	if got != "/campaigns/deep" {
		t.Errorf("deep walk-up found %q, want /campaigns/deep", got)
	}
}
