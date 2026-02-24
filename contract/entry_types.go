package contract

// ContractFileName is the name of the contract file within the .campaign directory.
const ContractFileName = "watchers.yaml"

// ContractVersion is the current version of the contract schema. Bump this
// when making breaking changes to the contract format.
const ContractVersion = 1

// Owner constants identify which tool owns a contract entry.
// WriteEntries uses the owner to scope its merge: it only replaces entries
// matching the specified owner, leaving other owners' entries untouched.
const (
	// OwnerCamp identifies entries written by the camp CLI.
	OwnerCamp = "camp"

	// OwnerFest identifies entries written by the fest CLI.
	OwnerFest = "fest"
)

// Campaign entry types describe campaign-wide state files.
const (
	// TypeCampaignMetadata is the campaign.yaml configuration file.
	TypeCampaignMetadata = "campaign.metadata"

	// TypeCampaignRegistry is the registry of projects in the campaign.
	TypeCampaignRegistry = "campaign.registry"
)

// Festival entry types describe festival state files and directories.
const (
	// TypeFestivalNavigation tracks the currently active festival for
	// navigation commands like fgo.
	TypeFestivalNavigation = "festival.navigation"

	// TypeFestivalLifecycle is the lifecycle state file that tracks
	// a festival's current status (planned, active, completed, etc.).
	TypeFestivalLifecycle = "festival.lifecycle"

	// TypeFestivalTypes defines the available festival types and their
	// phase templates.
	TypeFestivalTypes = "festival.types"

	// TypeFestivalRegistry is the index of all known festivals.
	TypeFestivalRegistry = "festival.registry"

	// TypeFestivalIntegrity tracks validation state across festivals.
	TypeFestivalIntegrity = "festival.integrity"

	// TypeFestivalStatusDir represents a directory whose children indicate
	// festivals in a particular status (e.g., planned/, active/, completed/).
	TypeFestivalStatusDir = "festival.status_dir"

	// TypeFestivalMetadata is the per-festival metadata file (FESTIVAL_OVERVIEW.md
	// frontmatter or similar).
	TypeFestivalMetadata = "festival.metadata"

	// TypeFestivalProgress is the append-only JSONL progress log for a festival.
	TypeFestivalProgress = "festival.progress"

	// TypeFestivalTask represents a task document within a festival's phase
	// and sequence hierarchy.
	TypeFestivalTask = "festival.task"
)

// Project entry types.
const (
	// TypeProjectRegistry is the registry of projects tracked by camp.
	TypeProjectRegistry = "project.registry"
)

// UI entry types.
const (
	// TypeUIPins tracks user-pinned items in the TUI.
	TypeUIPins = "ui.pins"
)

// Settings entry types.
const (
	// TypeSettingsNavigation stores navigation preferences and history.
	TypeSettingsNavigation = "settings.navigation"

	// TypeSettingsPermissions defines per-tool permission boundaries.
	TypeSettingsPermissions = "settings.permissions"
)

// Metrics entry types.
const (
	// TypeMetricsSnapshots is the metrics snapshot file for dashboard display.
	TypeMetricsSnapshots = "metrics.snapshots"
)

// Intent entry types.
const (
	// TypeIntentStatusDir represents the directory containing intent documents
	// organized by status.
	TypeIntentStatusDir = "intent.status_dir"
)

// Workflow entry types.
const (
	// TypeWorkflowConfig is the workflow configuration file.
	TypeWorkflowConfig = "workflow.config"
)
