package festivalbundle

import (
	"context"
)

// Pack packs sourceDir into a compressed .festival ZIP at outputPath.
//
// It snapshots the work unit under payload/, rewrites out-of-root file links
// into payload/.artifacts/, writes info.json, and sets bundle.id to the payload
// content hash. Network URLs are not rewritten or fetched.
//
// Returns the Info document written into the archive.
func Pack(ctx context.Context, sourceDir, outputPath string, opts PackOptions) (*Info, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Implemented in subsequent tasks (hash, rewrite, zip).
	_ = sourceDir
	_ = outputPath
	_ = opts
	return nil, ErrNotImplemented
}

// Unbundle extracts a .festival ZIP into destDir.
//
// Materializes payload/ (including .artifacts/) into destDir. Does not execute
// festivals or rituals. Optionally verifies bundle.id and writes a received
// history record under destDir/.bundles/received/.
//
// Returns the Info document from the archive.
func Unbundle(ctx context.Context, festivalPath, destDir string, opts UnbundleOptions) (*Info, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_ = festivalPath
	_ = destDir
	_ = opts
	return nil, ErrNotImplemented
}

// Verify recomputes the payload content hash of a .festival and compares it to
// info.json bundle.id. Returns nil when they match.
func Verify(ctx context.Context, festivalPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_ = festivalPath
	return ErrNotImplemented
}

// PayloadContentID is implemented in hash.go (SPEC §7).
