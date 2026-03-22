package camputil

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// resolvePath resolves symlinks so path comparisons work on macOS
// where /var -> /private/var.
func resolvePath(t *testing.T, p string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatalf("resolve path %q: %v", p, err)
	}
	return resolved
}

// makeCampaign creates a .campaign directory inside dir.
func makeCampaign(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, CampaignDir), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestFindCampaignRoot(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) (startDir string, cleanup func())
		env     string // CAMP_ROOT value, empty means unset
		wantErr bool
		errMsg  string
	}{
		{
			name: "finds root from root itself",
			setup: func(t *testing.T) (string, func()) {
				dir := t.TempDir()
				makeCampaign(t, dir)
				return dir, func() {}
			},
		},
		{
			name: "finds root from nested subdirectory",
			setup: func(t *testing.T) (string, func()) {
				root := t.TempDir()
				makeCampaign(t, root)
				nested := filepath.Join(root, "projects", "myapp", "src")
				if err := os.MkdirAll(nested, 0o755); err != nil {
					t.Fatal(err)
				}
				return nested, func() {}
			},
		},
		{
			name: "CAMP_ROOT env var takes priority",
			setup: func(t *testing.T) (string, func()) {
				// Create two campaign directories — env should win.
				envRoot := t.TempDir()
				makeCampaign(t, envRoot)
				cwdRoot := t.TempDir()
				makeCampaign(t, cwdRoot)
				return cwdRoot, func() {}
			},
			env: "", // set dynamically in test body
		},
		{
			name: "CAMP_ROOT pointing to invalid dir returns error",
			setup: func(t *testing.T) (string, func()) {
				return t.TempDir(), func() {}
			},
			env:     "/nonexistent/path/that/does/not/exist",
			wantErr: true,
			errMsg:  "CAMP_ROOT",
		},
		{
			name: "CAMP_ROOT dir exists but no .campaign returns error",
			setup: func(t *testing.T) (string, func()) {
				dir := t.TempDir()
				return dir, func() {}
			},
			env:     "", // set dynamically
			wantErr: true,
			errMsg:  "does not contain",
		},
		{
			name: "context cancellation stops walk",
			setup: func(t *testing.T) (string, func()) {
				return t.TempDir(), func() {}
			},
			wantErr: true,
			errMsg:  "context canceled",
		},
		{
			name: "returns error when no .campaign found",
			setup: func(t *testing.T) (string, func()) {
				return t.TempDir(), func() {}
			},
			wantErr: true,
			errMsg:  "not inside a campaign",
		},
		{
			name: "symlink resolution works",
			setup: func(t *testing.T) (string, func()) {
				root := t.TempDir()
				makeCampaign(t, root)
				// Create a symlink to the root.
				linkDir := t.TempDir()
				link := filepath.Join(linkDir, "campaign-link")
				if err := os.Symlink(root, link); err != nil {
					t.Skipf("symlinks not supported: %v", err)
				}
				return link, func() {}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startDir, cleanup := tt.setup(t)
			defer cleanup()

			// Handle env var.
			switch tt.name {
			case "CAMP_ROOT env var takes priority":
				// Create a separate env root.
				envRoot := t.TempDir()
				makeCampaign(t, envRoot)
				t.Setenv(EnvCampaignRoot, envRoot)

				ctx := context.Background()
				got, err := FindCampaignRoot(ctx, startDir)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				// Should resolve to the env root, not the startDir root.
				want := resolvePath(t, envRoot)
				got = resolvePath(t, got)
				if got != want {
					t.Errorf("got %q, want %q (CAMP_ROOT should take priority)", got, want)
				}
				return

			case "CAMP_ROOT pointing to invalid dir returns error":
				t.Setenv(EnvCampaignRoot, tt.env)

			case "CAMP_ROOT dir exists but no .campaign returns error":
				noCampaignDir := t.TempDir()
				t.Setenv(EnvCampaignRoot, noCampaignDir)

			case "context cancellation stops walk":
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				_, err := FindCampaignRoot(ctx, startDir)
				if err == nil {
					t.Fatal("expected error for cancelled context")
				}
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err, tt.errMsg)
				}
				return

			default:
				t.Setenv(EnvCampaignRoot, "")
			}

			ctx := context.Background()
			got, err := FindCampaignRoot(ctx, startDir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err, tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Normalize both paths for comparison (macOS symlinks).
			want := resolvePath(t, startDir)
			got = resolvePath(t, got)

			// For nested tests, the result should be an ancestor of startDir.
			if tt.name == "finds root from nested subdirectory" {
				// The root is the temp dir, not the nested subdir.
				// Walk startDir up to find the campaign root for comparison.
				root := want
				for {
					if _, err := os.Stat(filepath.Join(root, CampaignDir)); err == nil {
						want = root
						break
					}
					parent := filepath.Dir(root)
					if parent == root {
						t.Fatal("could not find campaign root in test setup")
					}
					root = parent
				}
			}

			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
