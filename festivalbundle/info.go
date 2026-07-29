package festivalbundle

// FormatVersion is the current wire format version (info.json format_version).
const FormatVersion = 1

// HashAlgSHA256 is the v1 content-hash algorithm id.
const HashAlgSHA256 = "sha256"

// Known kinds (v1 registry). Unknown kinds remain openable as opaque trees.
const (
	KindFestival = "festival"
	KindRitual   = "ritual"
	KindExplore  = "explore"
	KindDesign   = "design"
	KindIntent   = "intent"
	KindWorkitem = "workitem"
	KindNote     = "note"
	KindPackage  = "package"
)

// Info is the root info.json document.
type Info struct {
	FormatVersion int            `json:"format_version"`
	Kind          string         `json:"kind"`
	Bundle        BundleMeta     `json:"bundle"`
	From          *FromMeta      `json:"from,omitempty"`
	Subject       *SubjectMeta   `json:"subject,omitempty"`
	App           map[string]any `json:"app,omitempty"`
}

// BundleMeta is the share-artifact identity and timestamps.
type BundleMeta struct {
	ID        string   `json:"id"`
	HashAlg   string   `json:"hash_alg"`
	Name      string   `json:"name"`
	CreatedAt string   `json:"created_at"`
	PackedAt  string   `json:"packed_at"`
	Summary   string   `json:"summary,omitempty"`
	Creator   string   `json:"creator,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}

// FromMeta is the origin campaign (optional when packing outside a campaign).
type FromMeta struct {
	CampaignID   string `json:"campaign_id"`
	CampaignName string `json:"campaign_name,omitempty"`
	RelativePath string `json:"relative_path,omitempty"`
}

// SubjectMeta identifies the live festival/workitem inside the campaign.
type SubjectMeta struct {
	ID        string `json:"id,omitempty"`
	UUID      string `json:"uuid,omitempty"`
	Ref       string `json:"ref,omitempty"`
	Type      string `json:"type,omitempty"`
	Title     string `json:"title,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}
