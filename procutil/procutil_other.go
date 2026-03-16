//go:build !unix

package procutil

import (
	"context"
	"errors"
	"os/exec"
)

// ErrInterrupted indicates the process was terminated by an OS signal.
var ErrInterrupted = errors.New("process interrupted by signal")

// ErrUnsupported indicates process group isolation is not available on this platform.
var ErrUnsupported = errors.New("procutil: process group isolation is not supported on this platform")

// SetProcessGroup is a no-op on non-Unix platforms.
func SetProcessGroup(_ *exec.Cmd) {}

// RunWithCleanup is not supported on non-Unix platforms.
func RunWithCleanup(_ context.Context, _ *exec.Cmd) error {
	return ErrUnsupported
}
