package contract

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	dir := resolvePath(t, t.TempDir())
	path := filepath.Join(dir, ".campaign", "watchers.yaml")

	entries := []Entry{
		{
			ID:     "fest.lifecycle",
			Path:   "festivals/.state/lifecycle.yaml",
			Type:   TypeFestivalLifecycle,
			Format: FormatYAML,
			Watch:  WatchFile,
			Owner:  OwnerFest,
		},
		{
			ID:       "fest.progress",
			Path:     "festivals/.state/progress.jsonl",
			Type:     TypeFestivalProgress,
			Format:   FormatJSONL,
			Watch:    WatchAppend,
			Owner:    OwnerFest,
			Template: true,
		},
	}

	if err := WriteEntries(path, OwnerFest, entries); err != nil {
		t.Fatalf("WriteEntries: %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if got.Version != ContractVersion {
		t.Errorf("version = %d, want %d", got.Version, ContractVersion)
	}
	if len(got.Entries) != len(entries) {
		t.Fatalf("entries = %d, want %d", len(got.Entries), len(entries))
	}

	for i, want := range entries {
		g := got.Entries[i]
		if g.ID != want.ID {
			t.Errorf("entry[%d].ID = %q, want %q", i, g.ID, want.ID)
		}
		if g.Path != want.Path {
			t.Errorf("entry[%d].Path = %q, want %q", i, g.Path, want.Path)
		}
		if g.Type != want.Type {
			t.Errorf("entry[%d].Type = %q, want %q", i, g.Type, want.Type)
		}
		if g.Format != want.Format {
			t.Errorf("entry[%d].Format = %q, want %q", i, g.Format, want.Format)
		}
		if g.Watch != want.Watch {
			t.Errorf("entry[%d].Watch = %q, want %q", i, g.Watch, want.Watch)
		}
		if g.Owner != want.Owner {
			t.Errorf("entry[%d].Owner = %q, want %q", i, g.Owner, want.Owner)
		}
		if g.Template != want.Template {
			t.Errorf("entry[%d].Template = %v, want %v", i, g.Template, want.Template)
		}
	}
}

func TestMultiOwner(t *testing.T) {
	dir := resolvePath(t, t.TempDir())
	path := filepath.Join(dir, ".campaign", "watchers.yaml")

	festEntries := []Entry{
		validEntry("fest.lifecycle", OwnerFest),
		validEntry("fest.registry", OwnerFest),
	}
	if err := WriteEntries(path, OwnerFest, festEntries); err != nil {
		t.Fatalf("WriteEntries (fest): %v", err)
	}

	campEntries := []Entry{
		validEntry("camp.metadata", OwnerCamp),
		validEntry("camp.registry", OwnerCamp),
	}
	if err := WriteEntries(path, OwnerCamp, campEntries); err != nil {
		t.Fatalf("WriteEntries (camp): %v", err)
	}

	c, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(c.Entries) != 4 {
		t.Fatalf("entries = %d, want 4", len(c.Entries))
	}

	owners := make(map[string]int)
	for _, e := range c.Entries {
		owners[e.Owner]++
	}
	if owners[OwnerFest] != 2 {
		t.Errorf("fest entries = %d, want 2", owners[OwnerFest])
	}
	if owners[OwnerCamp] != 2 {
		t.Errorf("camp entries = %d, want 2", owners[OwnerCamp])
	}
}

func TestOverwriteSameOwner(t *testing.T) {
	dir := resolvePath(t, t.TempDir())
	path := filepath.Join(dir, ".campaign", "watchers.yaml")

	campEntries := []Entry{
		validEntry("camp.metadata", OwnerCamp),
	}
	if err := WriteEntries(path, OwnerCamp, campEntries); err != nil {
		t.Fatalf("WriteEntries (camp): %v", err)
	}

	festEntriesV1 := []Entry{
		validEntry("fest.old1", OwnerFest),
		validEntry("fest.old2", OwnerFest),
	}
	if err := WriteEntries(path, OwnerFest, festEntriesV1); err != nil {
		t.Fatalf("WriteEntries (fest v1): %v", err)
	}

	festEntriesV2 := []Entry{
		validEntry("fest.new1", OwnerFest),
	}
	if err := WriteEntries(path, OwnerFest, festEntriesV2); err != nil {
		t.Fatalf("WriteEntries (fest v2): %v", err)
	}

	c, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if len(c.Entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(c.Entries))
	}

	ids := make(map[string]bool)
	for _, e := range c.Entries {
		ids[e.ID] = true
	}

	if !ids["camp.metadata"] {
		t.Error("camp.metadata was removed")
	}
	if ids["fest.old1"] {
		t.Error("fest.old1 should have been replaced")
	}
	if ids["fest.old2"] {
		t.Error("fest.old2 should have been replaced")
	}
	if !ids["fest.new1"] {
		t.Error("fest.new1 was not written")
	}
}

func TestAtomicWriteSafety(t *testing.T) {
	dir := resolvePath(t, t.TempDir())
	path := filepath.Join(dir, ".campaign", "watchers.yaml")

	original := []Entry{
		validEntry("fest.original", OwnerFest),
	}
	if err := WriteEntries(path, OwnerFest, original); err != nil {
		t.Fatalf("WriteEntries (original): %v", err)
	}

	campEntries := []Entry{
		validEntry("camp.entry", OwnerCamp),
	}
	if err := WriteEntries(path, OwnerCamp, campEntries); err != nil {
		t.Fatalf("WriteEntries (camp): %v", err)
	}

	// Try to write fest entries with duplicate ID matching camp's entry.
	badEntries := []Entry{
		{
			ID:     "camp.entry",
			Path:   "some/other/path",
			Type:   TypeFestivalLifecycle,
			Format: FormatYAML,
			Watch:  WatchFile,
			Owner:  OwnerFest,
		},
	}
	err := WriteEntries(path, OwnerFest, badEntries)
	if err == nil {
		t.Fatal("expected validation error for duplicate ID, got nil")
	}

	// Read back -- original entries should be untouched.
	c, err := Read(path)
	if err != nil {
		t.Fatalf("Read after failed write: %v", err)
	}
	if len(c.Entries) != 2 {
		t.Fatalf("entries = %d, want 2 (original should be untouched)", len(c.Entries))
	}

	ids := make(map[string]bool)
	for _, e := range c.Entries {
		ids[e.ID] = true
	}
	if !ids["fest.original"] {
		t.Error("fest.original was lost after failed write")
	}
	if !ids["camp.entry"] {
		t.Error("camp.entry was lost after failed write")
	}
}

func TestContractPathHelper(t *testing.T) {
	tests := []struct {
		name string
		root string
		want string
	}{
		{
			name: "absolute path",
			root: "/home/user/my-campaign",
			want: "/home/user/my-campaign/.campaign/watchers.yaml",
		},
		{
			name: "path with trailing slash",
			root: "/home/user/my-campaign/",
			want: "/home/user/my-campaign/.campaign/watchers.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContractPath(tt.root)
			if got != tt.want {
				t.Errorf("ContractPath(%q) = %q, want %q", tt.root, got, tt.want)
			}
		})
	}
}

func TestReadWrittenYAMLFile(t *testing.T) {
	dir := resolvePath(t, t.TempDir())
	path := filepath.Join(dir, ".campaign", "watchers.yaml")

	entries := []Entry{
		{
			ID:            "fest.status_dir",
			Path:          "festivals/active/",
			Type:          TypeFestivalStatusDir,
			Format:        FormatDirectory,
			Watch:         WatchDirectory,
			Owner:         OwnerFest,
			Status:        "active",
			FallbackPaths: []string{"festivals/planning/"},
			StatusField:   "fest_status",
		},
	}

	if err := WriteEntries(path, OwnerFest, entries); err != nil {
		t.Fatalf("WriteEntries: %v", err)
	}

	// Verify the file is valid YAML by reading the raw bytes.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("written file is empty")
	}

	// Read back and verify optional fields survived the round-trip.
	c, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(c.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(c.Entries))
	}

	e := c.Entries[0]
	if e.Status != "active" {
		t.Errorf("Status = %q, want %q", e.Status, "active")
	}
	if e.StatusField != "fest_status" {
		t.Errorf("StatusField = %q, want %q", e.StatusField, "fest_status")
	}
	if len(e.FallbackPaths) != 1 || e.FallbackPaths[0] != "festivals/planning/" {
		t.Errorf("FallbackPaths = %v, want [festivals/planning/]", e.FallbackPaths)
	}
}
