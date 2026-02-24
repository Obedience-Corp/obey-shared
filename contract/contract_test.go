package contract

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRead_MissingFile(t *testing.T) {
	c, err := Read("/nonexistent/path/watchers.yaml")
	if err != nil {
		t.Fatalf("Read missing file: unexpected error: %v", err)
	}
	if c.Version != ContractVersion {
		t.Errorf("Version = %d, want %d", c.Version, ContractVersion)
	}
	if len(c.Entries) != 0 {
		t.Errorf("Entries length = %d, want 0", len(c.Entries))
	}
}

func TestRead_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "watchers.yaml")

	content := `version: 1
entries:
  - id: "test.entry"
    path: "test/path"
    type: "campaign.metadata"
    format: yaml
    watch: file
    owner: camp
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	c, err := Read(path)
	if err != nil {
		t.Fatalf("Read valid file: unexpected error: %v", err)
	}
	if c.Version != 1 {
		t.Errorf("Version = %d, want 1", c.Version)
	}
	if len(c.Entries) != 1 {
		t.Fatalf("Entries length = %d, want 1", len(c.Entries))
	}
	if c.Entries[0].ID != "test.entry" {
		t.Errorf("Entry ID = %q, want %q", c.Entries[0].ID, "test.entry")
	}
}

func TestRead_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "watchers.yaml")

	if err := os.WriteFile(path, []byte(":::not yaml:::"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	_, err := Read(path)
	if err == nil {
		t.Fatal("Read invalid YAML: expected error, got nil")
	}
}

func TestRead_InvalidContract(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "watchers.yaml")

	// Version 0 is invalid.
	content := `version: 0
entries: []
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	_, err := Read(path)
	if err == nil {
		t.Fatal("Read invalid contract: expected error, got nil")
	}
}

func TestContractPath(t *testing.T) {
	got := ContractPath("/home/user/campaign")
	want := "/home/user/campaign/.campaign/watchers.yaml"
	if got != want {
		t.Errorf("ContractPath = %q, want %q", got, want)
	}
}

func TestWriteEntries_FirstWrite(t *testing.T) {
	dir := t.TempDir()
	campaignDir := filepath.Join(dir, ".campaign")
	path := filepath.Join(campaignDir, "watchers.yaml")

	entries := []Entry{
		{
			ID:     "test.entry",
			Path:   "test/path",
			Type:   TypeCampaignMetadata,
			Format: FormatYAML,
			Watch:  WatchFile,
			Owner:  OwnerCamp,
		},
	}

	if err := WriteEntries(path, OwnerCamp, entries); err != nil {
		t.Fatalf("WriteEntries: unexpected error: %v", err)
	}

	// Read it back and verify.
	c, err := Read(path)
	if err != nil {
		t.Fatalf("Read after write: %v", err)
	}
	if len(c.Entries) != 1 {
		t.Fatalf("Entries length = %d, want 1", len(c.Entries))
	}
	if c.Entries[0].ID != "test.entry" {
		t.Errorf("Entry ID = %q, want %q", c.Entries[0].ID, "test.entry")
	}
}

func TestWriteEntries_OwnerScopedMerge(t *testing.T) {
	dir := t.TempDir()
	campaignDir := filepath.Join(dir, ".campaign")
	path := filepath.Join(campaignDir, "watchers.yaml")

	// Camp writes its entry.
	campEntries := []Entry{
		{
			ID:     "camp.entry",
			Path:   "camp/path",
			Type:   TypeCampaignMetadata,
			Format: FormatYAML,
			Watch:  WatchFile,
			Owner:  OwnerCamp,
		},
	}
	if err := WriteEntries(path, OwnerCamp, campEntries); err != nil {
		t.Fatalf("WriteEntries camp: %v", err)
	}

	// Fest writes its entry.
	festEntries := []Entry{
		{
			ID:     "fest.entry",
			Path:   "fest/path",
			Type:   TypeFestivalLifecycle,
			Format: FormatYAML,
			Watch:  WatchFile,
			Owner:  OwnerFest,
		},
	}
	if err := WriteEntries(path, OwnerFest, festEntries); err != nil {
		t.Fatalf("WriteEntries fest: %v", err)
	}

	// Both entries should exist.
	c, err := Read(path)
	if err != nil {
		t.Fatalf("Read after both writes: %v", err)
	}
	if len(c.Entries) != 2 {
		t.Fatalf("Entries length = %d, want 2", len(c.Entries))
	}

	// Now camp overwrites its entry. Fest's should remain.
	newCampEntries := []Entry{
		{
			ID:     "camp.new_entry",
			Path:   "camp/new_path",
			Type:   TypeCampaignRegistry,
			Format: FormatYAML,
			Watch:  WatchFile,
			Owner:  OwnerCamp,
		},
	}
	if err := WriteEntries(path, OwnerCamp, newCampEntries); err != nil {
		t.Fatalf("WriteEntries camp overwrite: %v", err)
	}

	c, err = Read(path)
	if err != nil {
		t.Fatalf("Read after overwrite: %v", err)
	}
	if len(c.Entries) != 2 {
		t.Fatalf("Entries length = %d, want 2", len(c.Entries))
	}

	// Find entries by owner.
	var campFound, festFound bool
	for _, e := range c.Entries {
		if e.Owner == OwnerCamp {
			campFound = true
			if e.ID != "camp.new_entry" {
				t.Errorf("Camp entry ID = %q, want %q", e.ID, "camp.new_entry")
			}
		}
		if e.Owner == OwnerFest {
			festFound = true
			if e.ID != "fest.entry" {
				t.Errorf("Fest entry ID = %q, want %q", e.ID, "fest.entry")
			}
		}
	}
	if !campFound {
		t.Error("Camp entry not found after overwrite")
	}
	if !festFound {
		t.Error("Fest entry not found after overwrite")
	}
}

func TestValidate_ValidContract(t *testing.T) {
	c := &Contract{
		Version: 1,
		Entries: []Entry{
			{
				ID:     "test.entry",
				Path:   "test/path",
				Type:   TypeCampaignMetadata,
				Format: FormatYAML,
				Watch:  WatchFile,
				Owner:  OwnerCamp,
			},
		},
	}
	if err := Validate(c); err != nil {
		t.Errorf("Validate valid contract: unexpected error: %v", err)
	}
}

func TestValidate_EmptyContract(t *testing.T) {
	c := &Contract{
		Version: 1,
		Entries: []Entry{},
	}
	if err := Validate(c); err != nil {
		t.Errorf("Validate empty contract: unexpected error: %v", err)
	}
}

func TestValidate_Errors(t *testing.T) {
	tests := []struct {
		name     string
		contract *Contract
		wantErr  string
	}{
		{
			name:     "version zero",
			contract: &Contract{Version: 0},
			wantErr:  "contract version must be >= 1",
		},
		{
			name:     "version too high",
			contract: &Contract{Version: ContractVersion + 1},
			wantErr:  "is newer than supported version",
		},
		{
			name: "empty ID",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{{Path: "p", Type: "t", Format: FormatYAML, Watch: WatchFile, Owner: "o"}},
			},
			wantErr: "ID is required",
		},
		{
			name: "duplicate ID",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{
					{ID: "dup", Path: "p1", Type: "t", Format: FormatYAML, Watch: WatchFile, Owner: "o"},
					{ID: "dup", Path: "p2", Type: "t", Format: FormatYAML, Watch: WatchFile, Owner: "o"},
				},
			},
			wantErr: "duplicate ID",
		},
		{
			name: "empty path",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{{ID: "id", Type: "t", Format: FormatYAML, Watch: WatchFile, Owner: "o"}},
			},
			wantErr: "Path is required",
		},
		{
			name: "empty type",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{{ID: "id", Path: "p", Format: FormatYAML, Watch: WatchFile, Owner: "o"}},
			},
			wantErr: "Type is required",
		},
		{
			name: "empty owner",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{{ID: "id", Path: "p", Type: "t", Format: FormatYAML, Watch: WatchFile}},
			},
			wantErr: "Owner is required",
		},
		{
			name: "empty watch",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{{ID: "id", Path: "p", Type: "t", Format: FormatYAML, Owner: "o"}},
			},
			wantErr: "Watch is required",
		},
		{
			name: "unknown watch mode",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{{ID: "id", Path: "p", Type: "t", Format: FormatYAML, Watch: "bad", Owner: "o"}},
			},
			wantErr: "unknown Watch mode",
		},
		{
			name: "empty format",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{{ID: "id", Path: "p", Type: "t", Watch: WatchFile, Owner: "o"}},
			},
			wantErr: "Format is required",
		},
		{
			name: "unknown format",
			contract: &Contract{
				Version: 1,
				Entries: []Entry{{ID: "id", Path: "p", Type: "t", Format: "bad", Watch: WatchFile, Owner: "o"}},
			},
			wantErr: "unknown Format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.contract)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if got := err.Error(); !contains(got, tt.wantErr) {
				t.Errorf("error = %q, want substring %q", got, tt.wantErr)
			}
		})
	}
}

func TestRead_PermissionError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "watchers.yaml")

	if err := os.WriteFile(path, []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	// Make file unreadable.
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(path, 0o644) })

	_, err := Read(path)
	if err == nil {
		t.Fatal("Read unreadable file: expected error, got nil")
	}
}

func TestWriteEntries_InvalidEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".campaign", "watchers.yaml")

	// Write entries with missing required field (no ID).
	entries := []Entry{
		{
			Path:   "test/path",
			Type:   TypeCampaignMetadata,
			Format: FormatYAML,
			Watch:  WatchFile,
			Owner:  OwnerCamp,
		},
	}

	err := WriteEntries(path, OwnerCamp, entries)
	if err == nil {
		t.Fatal("WriteEntries with invalid entries: expected error, got nil")
	}
}

func TestWriteEntries_ReadError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "watchers.yaml")

	// Write an invalid YAML file so Read returns a parse error.
	if err := os.WriteFile(path, []byte(":::bad:::"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	entries := []Entry{
		{
			ID:     "test.entry",
			Path:   "test/path",
			Type:   TypeCampaignMetadata,
			Format: FormatYAML,
			Watch:  WatchFile,
			Owner:  OwnerCamp,
		},
	}

	err := WriteEntries(path, OwnerCamp, entries)
	if err == nil {
		t.Fatal("WriteEntries with unreadable existing file: expected error, got nil")
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
