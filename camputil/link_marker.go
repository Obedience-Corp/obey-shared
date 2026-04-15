package camputil

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	linkMarkerFile          = ".camp"
	envCampaignRegistryPath = "CAMP_REGISTRY_PATH"
	registryFileName        = "registry.json"
	registryOrgName         = "obey"
	registryAppName         = "campaign"
)

type linkMarker struct {
	Version          int    `json:"version"`
	ActiveCampaignID string `json:"active_campaign_id,omitempty"`

	// Legacy fields kept for backward-compatible reads only.
	CampaignID   string `json:"campaign_id,omitempty"`
	CampaignRoot string `json:"campaign_root,omitempty"`
	ProjectName  string `json:"project_name,omitempty"`
}

func readLinkMarkerFile(path string) (*linkMarker, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var marker linkMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		return nil, err
	}

	if marker.Version == 0 {
		marker.Version = 1
	}
	if marker.ActiveCampaignID == "" && marker.CampaignID != "" {
		marker.ActiveCampaignID = marker.CampaignID
	}

	return &marker, nil
}

func (m linkMarker) effectiveCampaignID() string {
	if m.ActiveCampaignID != "" {
		return m.ActiveCampaignID
	}
	return m.CampaignID
}

type registrySnapshot struct {
	Campaigns map[string]registeredCampaign `json:"campaigns"`
}

type registeredCampaign struct {
	Path string `json:"path"`
}

func registryPath() string {
	if override := os.Getenv(envCampaignRegistryPath); override != "" {
		return override
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, registryOrgName, registryAppName, registryFileName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "."+registryOrgName, registryAppName, registryFileName)
}

func lookupRegisteredCampaignRoot(ctx context.Context, campaignID string) (string, bool, error) {
	if ctx.Err() != nil {
		return "", false, ctx.Err()
	}
	if campaignID == "" {
		return "", false, nil
	}

	data, err := os.ReadFile(registryPath())
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}

	var snapshot registrySnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return "", false, err
	}

	entry, ok := snapshot.Campaigns[campaignID]
	if !ok || entry.Path == "" {
		return "", false, nil
	}

	root, err := filepath.Abs(entry.Path)
	if err != nil {
		return "", false, err
	}
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}

	return root, true, nil
}
