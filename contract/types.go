package contract

// Contract represents the complete watchers.yaml file. It contains a version
// number for forward compatibility and a list of entries describing watched
// files and directories.
type Contract struct {
	Version int     `yaml:"version"`
	Entries []Entry `yaml:"entries"`
}

// Entry represents a single watched file or directory in the contract.
// Each entry declares a file or directory that exists in the campaign
// workspace, its format, how it should be watched, and which tool owns it.
type Entry struct {
	// ID is a unique, dot-separated identifier for this entry (e.g., "festival.lifecycle").
	ID string `yaml:"id"`

	// Path is the file or directory path relative to the campaign root.
	Path string `yaml:"path"`

	// Type categorizes the entry (e.g., "festival.lifecycle", "campaign.metadata").
	// Use the Type* constants defined in entry_types.go.
	Type string `yaml:"type"`

	// Format describes the data format of the watched file.
	Format Format `yaml:"format"`

	// Watch defines how the daemon should watch this file or directory.
	Watch WatchMode `yaml:"watch"`

	// Owner identifies which tool wrote this entry ("camp" or "fest").
	// Used for owner-scoped merging: WriteEntries only replaces entries
	// matching the specified owner.
	Owner string `yaml:"owner"`

	// Status is an optional current status string, used for status_dir entries.
	Status string `yaml:"status,omitempty"`

	// Template indicates whether this entry uses per-festival path templates.
	// When true, the Path field contains template variables like {festival_dir}
	// that the daemon expands for each active festival.
	Template bool `yaml:"template,omitempty"`

	// FallbackPaths lists alternative paths to try if the primary Path doesn't
	// exist. The daemon tries each in order until it finds one that exists.
	FallbackPaths []string `yaml:"fallback_paths,omitempty"`

	// StatusField names the frontmatter field to extract status from, used
	// when Format is FormatMarkdownFrontmatter.
	StatusField string `yaml:"status_field,omitempty"`
}

// WatchMode defines how a file or directory should be watched by the daemon.
type WatchMode string

const (
	// WatchFile watches the parent directory and filters events by filename.
	// On change, the daemon re-reads the entire file. Use for config files,
	// YAML state, and any file that is rewritten atomically.
	WatchFile WatchMode = "file"

	// WatchDirectory watches a directory for child create, delete, and rename
	// events. The daemon tracks which children exist. Use for directories where
	// the presence or absence of subdirectories is meaningful (e.g., festival
	// status directories like planned/, active/).
	WatchDirectory WatchMode = "directory"

	// WatchAppend watches a file and reads only new bytes from a tracked offset.
	// Use for append-only log files like JSONL progress streams where the daemon
	// should not re-read the entire file on each change.
	WatchAppend WatchMode = "append"
)

// Format defines the data format of a watched file, so the daemon knows how
// to parse it.
type Format string

const (
	// FormatYAML indicates the file contains YAML data.
	FormatYAML Format = "yaml"

	// FormatJSON indicates the file contains JSON data.
	FormatJSON Format = "json"

	// FormatJSONL indicates the file contains newline-delimited JSON (one JSON
	// object per line). Used for append-only log streams.
	FormatJSONL Format = "jsonl"

	// FormatDirectory indicates the entry is a directory, not a file. The
	// "data" is the list of child entries (subdirectories or files).
	FormatDirectory Format = "directory"

	// FormatMarkdownFrontmatter indicates the file is a Markdown document with
	// YAML frontmatter. The daemon extracts structured data from the frontmatter
	// block (delimited by --- lines) rather than parsing the full document.
	FormatMarkdownFrontmatter Format = "markdown_frontmatter"
)
