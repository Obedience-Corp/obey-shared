package festivalbundle

import "errors"

var (
	// ErrNotImplemented is returned by APIs not yet filled in this scaffold.
	// Removed as implementations land in subsequent tasks.
	ErrNotImplemented = errors.New("festivalbundle: not implemented")

	// ErrInvalidInfo indicates info.json is missing required fields or is malformed.
	ErrInvalidInfo = errors.New("festivalbundle: invalid info.json")

	// ErrHashMismatch indicates computed payload hash does not match bundle.id.
	ErrHashMismatch = errors.New("festivalbundle: bundle.id content hash mismatch")

	// ErrSymlinkRejected indicates a symlink was encountered (v1 forbids them).
	ErrSymlinkRejected = errors.New("festivalbundle: symlinks not allowed in v1")

	// ErrEmptyPayload indicates payload/ has no regular files.
	ErrEmptyPayload = errors.New("festivalbundle: payload has no files")

	// ErrUnsafeZipPath indicates a zip member path fails zip-slip checks.
	ErrUnsafeZipPath = errors.New("festivalbundle: unsafe zip member path")

	// ErrDestNotEmpty indicates unbundle destination exists and is non-empty without Force.
	ErrDestNotEmpty = errors.New("festivalbundle: destination exists and is not empty")
)
