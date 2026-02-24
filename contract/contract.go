package contract

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Read loads a contract from the given file path. If the file does not exist,
// Read returns an empty contract with Version set to ContractVersion and an
// empty Entries slice. This supports the first-write scenario where no contract
// file has been created yet.
//
// After loading, Read validates the contract with Validate. If the file exists
// but contains invalid data, Read returns an error.
func Read(path string) (*Contract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Contract{
				Version: ContractVersion,
				Entries: []Entry{},
			}, nil
		}
		return nil, fmt.Errorf("read contract %s: %w", path, err)
	}

	var c Contract
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse contract %s: %w", path, err)
	}

	if err := Validate(&c); err != nil {
		return nil, fmt.Errorf("validate contract %s: %w", path, err)
	}

	return &c, nil
}

// ContractPath returns the standard path to the contract file within the given
// campaign root directory. The contract file lives at .campaign/watchers.yaml.
func ContractPath(campaignRoot string) string {
	return filepath.Join(campaignRoot, ".campaign", ContractFileName)
}

// WriteEntries performs an owner-scoped merge and atomic write of entries to the
// contract file at the given path.
//
// The merge protocol:
//  1. Read the existing contract (missing file is treated as empty contract)
//  2. Remove all existing entries where entry.Owner == owner
//  3. Append the new entries
//  4. Validate the merged contract
//  5. Write to a temp file in the same directory
//  6. Rename the temp file to the target path (atomic on same filesystem)
//
// This ensures that fest's entries are never modified when camp writes, and vice
// versa. Each tool owns its entries and can replace them at any time.
func WriteEntries(path string, owner string, entries []Entry) error {
	c, err := Read(path)
	if err != nil {
		return fmt.Errorf("write entries: %w", err)
	}

	// Remove all entries owned by this owner.
	kept := make([]Entry, 0, len(c.Entries))
	for _, e := range c.Entries {
		if e.Owner != owner {
			kept = append(kept, e)
		}
	}

	// Append new entries.
	kept = append(kept, entries...)
	c.Entries = kept
	c.Version = ContractVersion

	// Validate the merged contract before writing.
	if err := Validate(c); err != nil {
		return fmt.Errorf("write entries: validate merged contract: %w", err)
	}

	// Marshal to YAML.
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("write entries: marshal: %w", err)
	}

	// Ensure the parent directory exists (for first write).
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("write entries: create directory %s: %w", dir, err)
	}

	// Write to a unique temp file in the same directory. Using CreateTemp avoids
	// a write-after-write race when two processes call WriteEntries concurrently.
	f, err := os.CreateTemp(dir, ".watchers.yaml.*.tmp")
	if err != nil {
		return fmt.Errorf("write entries: create temp file: %w", err)
	}
	tmp := f.Name()

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("write entries: write temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("write entries: close temp file: %w", err)
	}

	// Set desired permissions (CreateTemp uses 0600 by default).
	if err := os.Chmod(tmp, 0o644); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("write entries: chmod temp file: %w", err)
	}

	// Atomic rename.
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("write entries: rename temp to target: %w", err)
	}

	return nil
}

// Validate checks that a contract is well-formed. It returns an error
// describing the first problem found, or nil if the contract is valid.
//
// Checks performed:
//   - Version must be between 1 and ContractVersion (inclusive)
//   - No duplicate entry IDs
//   - Every entry has non-empty ID, Path, Type, Format, Watch, and Owner
//   - Watch must be a recognized WatchMode value
//   - Format must be a recognized Format value
func Validate(c *Contract) error {
	if c.Version < 1 {
		return fmt.Errorf("contract version must be >= 1, got %d", c.Version)
	}
	if c.Version > ContractVersion {
		return fmt.Errorf("contract version %d is newer than supported version %d", c.Version, ContractVersion)
	}

	seen := make(map[string]bool, len(c.Entries))
	for i, e := range c.Entries {
		if e.ID == "" {
			return fmt.Errorf("entry %d: ID is required", i)
		}
		if seen[e.ID] {
			return fmt.Errorf("entry %d: duplicate ID %q", i, e.ID)
		}
		seen[e.ID] = true

		if e.Path == "" {
			return fmt.Errorf("entry %q: Path is required", e.ID)
		}
		if e.Type == "" {
			return fmt.Errorf("entry %q: Type is required", e.ID)
		}
		if e.Owner == "" {
			return fmt.Errorf("entry %q: Owner is required", e.ID)
		}

		if err := validateWatchMode(e.Watch); err != nil {
			return fmt.Errorf("entry %q: %w", e.ID, err)
		}
		if err := validateFormat(e.Format); err != nil {
			return fmt.Errorf("entry %q: %w", e.ID, err)
		}
	}

	return nil
}

// validateWatchMode checks that a WatchMode value is one of the recognized
// constants.
func validateWatchMode(w WatchMode) error {
	switch w {
	case WatchFile, WatchDirectory, WatchAppend:
		return nil
	case "":
		return fmt.Errorf("Watch is required")
	default:
		return fmt.Errorf("unknown Watch mode %q (expected file, directory, or append)", w)
	}
}

// validateFormat checks that a Format value is one of the recognized constants.
func validateFormat(f Format) error {
	switch f {
	case FormatYAML, FormatJSON, FormatJSONL, FormatDirectory, FormatMarkdownFrontmatter:
		return nil
	case "":
		return fmt.Errorf("Format is required")
	default:
		return fmt.Errorf("unknown Format %q (expected yaml, json, jsonl, directory, or markdown_frontmatter)", f)
	}
}
