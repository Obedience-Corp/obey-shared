package festivalbundle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HistoryRecord is a .bundles/sent or .bundles/received entry.
type HistoryRecord struct {
	RecordVersion int            `json:"record_version"`
	Direction     string         `json:"direction"`
	RecordedAt    string         `json:"recorded_at"`
	Bundle        map[string]any `json:"bundle"`
	From          *FromMeta      `json:"from,omitempty"`
	Subject       *SubjectMeta   `json:"subject,omitempty"`
	Output        map[string]any `json:"output,omitempty"`
	Source        map[string]any `json:"source,omitempty"`
	InfoSnapshot  *Info          `json:"info_snapshot,omitempty"`
}

// WriteSentRecord writes source/.bundles/sent/<safe-id>.json.
func WriteSentRecord(sourceDir string, info *Info, outputPath string) error {
	dir := filepath.Join(sourceDir, ".bundles", "sent")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	rec := HistoryRecord{
		RecordVersion: 1,
		Direction:     "sent",
		RecordedAt:    time.Now().UTC().Truncate(time.Second).Format(time.RFC3339),
		Bundle: map[string]any{
			"id":        info.Bundle.ID,
			"name":      info.Bundle.Name,
			"kind":      info.Kind,
			"packed_at": info.Bundle.PackedAt,
			"hash_alg":  info.Bundle.HashAlg,
		},
		Output:       map[string]any{"path": outputPath},
		InfoSnapshot: info,
	}
	return writeHistoryJSON(filepath.Join(dir, safeBundleFileName(info.Bundle.ID)+".json"), rec)
}

// WriteReceivedRecord writes dest/.bundles/received/<safe-id>-<stamp>.json.
func WriteReceivedRecord(destDir string, info *Info, festivalPath string) error {
	dir := filepath.Join(destDir, ".bundles", "received")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	rec := HistoryRecord{
		RecordVersion: 1,
		Direction:     "received",
		RecordedAt:    time.Now().UTC().Truncate(time.Second).Format(time.RFC3339),
		Bundle: map[string]any{
			"id":        info.Bundle.ID,
			"name":      info.Bundle.Name,
			"kind":      info.Kind,
			"packed_at": info.Bundle.PackedAt,
			"hash_alg":  info.Bundle.HashAlg,
		},
		From:    info.From,
		Subject: info.Subject,
		Source: map[string]any{
			"filename": filepath.Base(festivalPath),
			"path":     festivalPath,
		},
		InfoSnapshot: info,
	}
	name := safeBundleFileName(info.Bundle.ID) + "-" + stamp + ".json"
	return writeHistoryJSON(filepath.Join(dir, name), rec)
}

func safeBundleFileName(id string) string {
	return strings.ReplaceAll(id, ":", "-")
}

func writeHistoryJSON(path string, rec HistoryRecord) error {
	raw, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}
