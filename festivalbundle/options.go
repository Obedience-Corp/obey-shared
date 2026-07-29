package festivalbundle

// PackOptions configures Pack.
type PackOptions struct {
	// Kind is required (e.g. KindNote, KindExplore).
	Kind string

	// Name is the human-readable bundle title; defaults to source directory name.
	Name string

	// Creator is recorded in bundle.creator when set.
	Creator string

	// Summary is optional one-line preview text.
	Summary string

	// Tags are optional freeform tags.
	Tags []string

	// From is origin campaign metadata (optional).
	From *FromMeta

	// Subject is festival/workitem identity (optional).
	Subject *SubjectMeta

	// App is cooperative app-specific metadata (optional).
	App map[string]any

	// Strict fails pack when a linked out-of-root file is missing.
	Strict bool

	// WriteSentRecord writes source/.bundles/sent/<id>.json after a successful pack.
	// Default false at the library layer; CLIs may enable it.
	WriteSentRecord bool

	// ExcludePatterns are extra path patterns to skip when copying the source tree.
	// Built-in excludes always include .git, .bundles, and common junk.
	ExcludePatterns []string
}

// UnbundleOptions configures Unbundle.
type UnbundleOptions struct {
	// Force allows writing into a non-empty destination.
	Force bool

	// Verify checks bundle.id against the payload content hash (default true when zero-value is used via helper).
	// Set SkipVerify to disable.
	SkipVerify bool

	// WriteReceivedRecord writes dest/.bundles/received/<id>.json after unbundle.
	// Default false at the library layer; CLIs may enable it.
	WriteReceivedRecord bool
}
