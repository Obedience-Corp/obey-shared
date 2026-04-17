package camputil

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLookupRegisteredCampaignRoot_InvalidRegistryJSONIgnored(t *testing.T) {
	registryPath := filepath.Join(t.TempDir(), "registry.json")
	if err := os.WriteFile(registryPath, []byte("{invalid json"), 0o644); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	t.Setenv(envCampaignRegistryPath, registryPath)

	root, found, err := lookupRegisteredCampaignRoot(context.Background(), "linked-campaign-id")
	if err != nil {
		t.Fatalf("lookupRegisteredCampaignRoot() error = %v", err)
	}
	if found {
		t.Fatalf("lookupRegisteredCampaignRoot() found = %v, want false", found)
	}
	if root != "" {
		t.Fatalf("lookupRegisteredCampaignRoot() root = %q, want empty", root)
	}
}
