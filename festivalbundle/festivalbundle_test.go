package festivalbundle_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Obedience-Corp/obey-shared/festivalbundle"
)

func TestPackageConstants(t *testing.T) {
	if festivalbundle.FormatVersion != 1 {
		t.Fatalf("FormatVersion = %d, want 1", festivalbundle.FormatVersion)
	}
	if festivalbundle.HashAlgSHA256 != "sha256" {
		t.Fatalf("HashAlgSHA256 = %q", festivalbundle.HashAlgSHA256)
	}
}

func TestInfoJSONRoundTrip(t *testing.T) {
	info := festivalbundle.Info{
		FormatVersion: festivalbundle.FormatVersion,
		Kind:          festivalbundle.KindNote,
		Bundle: festivalbundle.BundleMeta{
			ID:        "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			HashAlg:   festivalbundle.HashAlgSHA256,
			Name:      "Example",
			CreatedAt: time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			PackedAt:  time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			Creator:   "scaffold-test",
		},
		From: &festivalbundle.FromMeta{
			CampaignID:   "074751cf-8b79-49c6-8f78-ea7dce784bd8",
			CampaignName: "obedience-growth-rd",
		},
		Subject: &festivalbundle.SubjectMeta{
			Ref:   "WI-93268b",
			Type:  "design",
			Title: "Portable .festival text-bundle",
		},
	}

	raw, err := json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	var got festivalbundle.Info
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Kind != festivalbundle.KindNote || got.Bundle.Name != "Example" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if got.From == nil || got.From.CampaignID == "" {
		t.Fatal("expected From to round-trip")
	}
}

func TestAPIStubsRespectContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := festivalbundle.Pack(ctx, ".", "out.festival", festivalbundle.PackOptions{Kind: festivalbundle.KindNote}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Pack: want context.Canceled, got %v", err)
	}
	if _, err := festivalbundle.Unbundle(ctx, "x.festival", "dest", festivalbundle.UnbundleOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Unbundle: want context.Canceled, got %v", err)
	}
	if err := festivalbundle.Verify(ctx, "x.festival"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Verify: want context.Canceled, got %v", err)
	}
	if _, err := festivalbundle.PayloadContentID(ctx, "."); !errors.Is(err, context.Canceled) {
		t.Fatalf("PayloadContentID: want context.Canceled, got %v", err)
	}
}

func TestAPIStubsNotImplemented(t *testing.T) {
	ctx := context.Background()
	if _, err := festivalbundle.Pack(ctx, ".", "out.festival", festivalbundle.PackOptions{Kind: festivalbundle.KindNote}); !errors.Is(err, festivalbundle.ErrNotImplemented) {
		t.Fatalf("Pack: want ErrNotImplemented, got %v", err)
	}
	if _, err := festivalbundle.Unbundle(ctx, "x.festival", "dest", festivalbundle.UnbundleOptions{}); !errors.Is(err, festivalbundle.ErrNotImplemented) {
		t.Fatalf("Unbundle: want ErrNotImplemented, got %v", err)
	}
	if err := festivalbundle.Verify(ctx, "x.festival"); !errors.Is(err, festivalbundle.ErrNotImplemented) {
		t.Fatalf("Verify: want ErrNotImplemented, got %v", err)
	}
	if _, err := festivalbundle.PayloadContentID(ctx, "."); !errors.Is(err, festivalbundle.ErrNotImplemented) {
		t.Fatalf("PayloadContentID: want ErrNotImplemented, got %v", err)
	}
}
