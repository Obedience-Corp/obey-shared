package contract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resolvePath resolves symlinks in a path. Required on macOS where
// /var -> /private/var causes path comparison failures with t.TempDir().
func resolvePath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("resolve path %s: %v", path, err)
	}
	return resolved
}

// validEntry returns a minimal valid Entry for use in tests. Callers can
// override fields as needed.
func validEntry(id, owner string) Entry {
	return Entry{
		ID:     id,
		Path:   "some/path/" + id,
		Type:   TypeFestivalLifecycle,
		Format: FormatYAML,
		Watch:  WatchFile,
		Owner:  owner,
	}
}

func TestRead(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, path string)
		wantErr   bool
		wantVer   int
		wantCount int
	}{
		{
			name:      "missing file returns empty contract",
			setup:     func(t *testing.T, path string) {},
			wantErr:   false,
			wantVer:   ContractVersion,
			wantCount: 0,
		},
		{
			name: "valid YAML returns correct contract",
			setup: func(t *testing.T, path string) {
				data := []byte(`version: 1
entries:
  - id: test.entry
    path: some/path
    type: festival.lifecycle
    format: yaml
    watch: file
    owner: fest
`)
				if err := os.WriteFile(path, data, 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr:   false,
			wantVer:   1,
			wantCount: 1,
		},
		{
			name: "invalid YAML returns error",
			setup: func(t *testing.T, path string) {
				if err := os.WriteFile(path, []byte(`{{{not yaml at all`), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: true,
		},
		{
			name: "unknown version returns validation error",
			setup: func(t *testing.T, path string) {
				data := []byte(`version: 999
entries: []
`)
				if err := os.WriteFile(path, data, 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: true,
		},
		{
			name: "permission error returns error",
			setup: func(t *testing.T, path string) {
				if err := os.WriteFile(path, []byte("version: 1\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(path, 0o000); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { os.Chmod(path, 0o644) })
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := resolvePath(t, t.TempDir())
			path := filepath.Join(dir, "watchers.yaml")
			tt.setup(t, path)

			got, err := Read(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Version != tt.wantVer {
				t.Errorf("version = %d, want %d", got.Version, tt.wantVer)
			}
			if len(got.Entries) != tt.wantCount {
				t.Errorf("entries count = %d, want %d", len(got.Entries), tt.wantCount)
			}
		})
	}
}

func TestWriteEntries(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, path string)
		owner   string
		entries []Entry
		wantErr bool
		check   func(t *testing.T, path string)
	}{
		{
			name:  "first write to non-existent file creates contract",
			setup: func(t *testing.T, path string) {},
			owner: OwnerFest,
			entries: []Entry{
				validEntry("fest.entry1", OwnerFest),
			},
			check: func(t *testing.T, path string) {
				c, err := Read(path)
				if err != nil {
					t.Fatal(err)
				}
				if len(c.Entries) != 1 {
					t.Fatalf("entries = %d, want 1", len(c.Entries))
				}
				if c.Entries[0].ID != "fest.entry1" {
					t.Errorf("entry ID = %q, want %q", c.Entries[0].ID, "fest.entry1")
				}
			},
		},
		{
			name: "write preserves other owner entries",
			setup: func(t *testing.T, path string) {
				err := WriteEntries(path, OwnerCamp, []Entry{
					validEntry("camp.entry1", OwnerCamp),
				})
				if err != nil {
					t.Fatal(err)
				}
			},
			owner: OwnerFest,
			entries: []Entry{
				validEntry("fest.entry1", OwnerFest),
			},
			check: func(t *testing.T, path string) {
				c, err := Read(path)
				if err != nil {
					t.Fatal(err)
				}
				if len(c.Entries) != 2 {
					t.Fatalf("entries = %d, want 2", len(c.Entries))
				}
				foundCamp := false
				foundFest := false
				for _, e := range c.Entries {
					if e.ID == "camp.entry1" {
						foundCamp = true
					}
					if e.ID == "fest.entry1" {
						foundFest = true
					}
				}
				if !foundCamp {
					t.Error("camp entry was removed")
				}
				if !foundFest {
					t.Error("fest entry was not added")
				}
			},
		},
		{
			name: "write replaces same owner entries",
			setup: func(t *testing.T, path string) {
				err := WriteEntries(path, OwnerFest, []Entry{
					validEntry("fest.old", OwnerFest),
				})
				if err != nil {
					t.Fatal(err)
				}
			},
			owner: OwnerFest,
			entries: []Entry{
				validEntry("fest.new", OwnerFest),
			},
			check: func(t *testing.T, path string) {
				c, err := Read(path)
				if err != nil {
					t.Fatal(err)
				}
				if len(c.Entries) != 1 {
					t.Fatalf("entries = %d, want 1", len(c.Entries))
				}
				if c.Entries[0].ID != "fest.new" {
					t.Errorf("entry ID = %q, want %q", c.Entries[0].ID, "fest.new")
				}
			},
		},
		{
			name:  "write to non-existent directory creates directory",
			setup: func(t *testing.T, path string) {},
			owner: OwnerFest,
			entries: []Entry{
				validEntry("fest.entry1", OwnerFest),
			},
			check: func(t *testing.T, path string) {
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Error("file was not created")
				}
			},
		},
		{
			name: "write with invalid entries returns error",
			setup: func(t *testing.T, path string) {},
			owner: OwnerCamp,
			entries: []Entry{
				{Path: "test/path", Type: TypeCampaignMetadata, Format: FormatYAML, Watch: WatchFile, Owner: OwnerCamp},
			},
			wantErr: true,
		},
		{
			name: "write with corrupt existing file returns error",
			setup: func(t *testing.T, path string) {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(":::bad:::"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			owner: OwnerCamp,
			entries: []Entry{
				validEntry("camp.entry1", OwnerCamp),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := resolvePath(t, t.TempDir())
			path := filepath.Join(dir, ".campaign", "watchers.yaml")

			if tt.setup != nil {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				tt.setup(t, path)
			}

			err := WriteEntries(path, tt.owner, tt.entries)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, path)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		contract *Contract
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid contract passes",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{
					validEntry("test.one", OwnerFest),
					validEntry("test.two", OwnerCamp),
				},
			},
			wantErr: false,
		},
		{
			name: "empty entries is valid",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{},
			},
			wantErr: false,
		},
		{
			name: "missing version fails",
			contract: &Contract{
				Version: 0,
				Entries: []Entry{},
			},
			wantErr: true,
			errMsg:  "version must be >= 1",
		},
		{
			name: "unknown version fails",
			contract: &Contract{
				Version: ContractVersion + 1,
				Entries: []Entry{},
			},
			wantErr: true,
			errMsg:  "newer than supported",
		},
		{
			name: "duplicate entry IDs fail",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{
					validEntry("same.id", OwnerFest),
					validEntry("same.id", OwnerCamp),
				},
			},
			wantErr: true,
			errMsg:  "duplicate ID",
		},
		{
			name: "entry missing ID fails",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{
					{Path: "some/path", Type: TypeFestivalLifecycle, Format: FormatYAML, Watch: WatchFile, Owner: OwnerFest},
				},
			},
			wantErr: true,
			errMsg:  "ID is required",
		},
		{
			name: "entry missing Path fails",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{
					{ID: "test.entry", Type: TypeFestivalLifecycle, Format: FormatYAML, Watch: WatchFile, Owner: OwnerFest},
				},
			},
			wantErr: true,
			errMsg:  "Path is required",
		},
		{
			name: "entry missing Type fails",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{
					{ID: "test.entry", Path: "some/path", Format: FormatYAML, Watch: WatchFile, Owner: OwnerFest},
				},
			},
			wantErr: true,
			errMsg:  "Type is required",
		},
		{
			name: "entry missing Owner fails",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{
					{ID: "test.entry", Path: "some/path", Type: TypeFestivalLifecycle, Format: FormatYAML, Watch: WatchFile},
				},
			},
			wantErr: true,
			errMsg:  "Owner is required",
		},
		{
			name: "entry with unknown WatchMode fails",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{
					{ID: "test.entry", Path: "some/path", Type: TypeFestivalLifecycle, Format: FormatYAML, Watch: WatchMode("invalid"), Owner: OwnerFest},
				},
			},
			wantErr: true,
			errMsg:  "unknown Watch mode",
		},
		{
			name: "entry with unknown Format fails",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{
					{ID: "test.entry", Path: "some/path", Type: TypeFestivalLifecycle, Format: Format("invalid"), Watch: WatchFile, Owner: OwnerFest},
				},
			},
			wantErr: true,
			errMsg:  "unknown Format",
		},
		{
			name: "entry missing Watch fails",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{
					{ID: "test.entry", Path: "some/path", Type: TypeFestivalLifecycle, Format: FormatYAML, Owner: OwnerFest},
				},
			},
			wantErr: true,
			errMsg:  "Watch is required",
		},
		{
			name: "entry missing Format fails",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{
					{ID: "test.entry", Path: "some/path", Type: TypeFestivalLifecycle, Watch: WatchFile, Owner: OwnerFest},
				},
			},
			wantErr: true,
			errMsg:  "Format is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.contract)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errMsg != "" {
					if got := err.Error(); !strings.Contains(got, tt.errMsg) {
						t.Errorf("error = %q, want substring %q", got, tt.errMsg)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestContractPath(t *testing.T) {
	got := ContractPath("/home/user/my-campaign")
	want := "/home/user/my-campaign/.campaign/watchers.yaml"
	if got != want {
		t.Errorf("ContractPath = %q, want %q", got, want)
	}
}
