//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// demuxDockerOutput strips Docker exec multiplexed stream headers from output.
// Docker exec output is multiplexed with 8-byte headers:
// - byte 0: stream type (1=stdout, 2=stderr)
// - bytes 1-3: padding (zeros)
// - bytes 4-7: big-endian uint32 payload size
func demuxDockerOutput(data []byte) []byte {
	var result bytes.Buffer
	offset := 0
	for offset < len(data) {
		if offset+8 > len(data) {
			result.Write(data[offset:])
			break
		}
		payloadSize := binary.BigEndian.Uint32(data[offset+4 : offset+8])
		payloadStart := offset + 8
		payloadEnd := payloadStart + int(payloadSize)
		if payloadEnd > len(data) {
			payloadEnd = len(data)
		}
		result.Write(data[payloadStart:payloadEnd])
		offset = payloadEnd
	}
	return result.Bytes()
}

// TestContainer wraps container operations for testing.
type TestContainer struct {
	container testcontainers.Container
	ctx       context.Context
	t         *testing.T
}

// NewSharedContainer creates a container for reuse across multiple tests.
func NewSharedContainer() (*TestContainer, error) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:      "alpine:latest",
		Cmd:        []string{"sleep", "3600"},
		WaitingFor: wait.ForExec([]string{"true"}).WithStartupTimeout(30 * time.Second),
		AutoRemove: true,
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Create initial working directories
	exitCode, _, err := container.Exec(ctx, []string{"mkdir", "-p", "/test", "/campaigns"})
	if err != nil || exitCode != 0 {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to create initial directories: %w", err)
	}

	return &TestContainer{
		container: container,
		ctx:       ctx,
		t:         nil,
	}, nil
}

// Reset clears container state between tests.
func (tc *TestContainer) Reset() error {
	exitCode, _, err := tc.container.Exec(tc.ctx, []string{
		"sh", "-c",
		"rm -rf /test /campaigns 2>/dev/null; " +
			"mkdir -p /test /campaigns; sync",
	})
	if err != nil {
		return fmt.Errorf("failed to reset container: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("reset command failed with exit code %d", exitCode)
	}
	return nil
}

// Cleanup terminates the container.
func (tc *TestContainer) Cleanup() {
	if tc.container != nil {
		tc.container.Terminate(tc.ctx)
	}
}

// ExecCommand executes an arbitrary command in the container.
func (tc *TestContainer) ExecCommand(args ...string) (string, int, error) {
	exitCode, reader, err := tc.container.Exec(tc.ctx, args)
	if err != nil {
		return "", -1, fmt.Errorf("failed to execute command: %w", err)
	}

	rawOutput, err := io.ReadAll(reader)
	if err != nil {
		return "", exitCode, fmt.Errorf("failed to read output: %w", err)
	}

	output := demuxDockerOutput(rawOutput)
	return string(output), exitCode, nil
}

// ExecShell executes a shell command string in the container.
func (tc *TestContainer) ExecShell(cmd string) (string, int, error) {
	return tc.ExecCommand("sh", "-c", cmd)
}

// MkdirAll creates a directory tree in the container.
func (tc *TestContainer) MkdirAll(path string) error {
	exitCode, _, err := tc.container.Exec(tc.ctx, []string{"mkdir", "-p", path})
	if err != nil || exitCode != 0 {
		return fmt.Errorf("failed to mkdir %s: %w", path, err)
	}
	return nil
}

// CheckDirExists checks if a directory exists in the container.
func (tc *TestContainer) CheckDirExists(path string) (bool, error) {
	exitCode, _, err := tc.container.Exec(tc.ctx, []string{"test", "-d", path})
	if err != nil {
		return false, fmt.Errorf("failed to check directory: %w", err)
	}
	return exitCode == 0, nil
}

// CreateCampaign creates a .campaign directory at the given path.
func (tc *TestContainer) CreateCampaign(path string) error {
	campaignDir := filepath.Join(path, ".campaign")
	return tc.MkdirAll(campaignDir)
}

// CreateSymlink creates a symlink in the container.
func (tc *TestContainer) CreateSymlink(target, link string) error {
	exitCode, _, err := tc.container.Exec(tc.ctx, []string{"ln", "-s", target, link})
	if err != nil || exitCode != 0 {
		return fmt.Errorf("failed to create symlink %s -> %s: %w", link, target, err)
	}
	return nil
}

// ReadSymlink reads the target of a symlink.
func (tc *TestContainer) ReadSymlink(path string) (string, error) {
	output, exitCode, err := tc.ExecCommand("readlink", "-f", path)
	if err != nil || exitCode != 0 {
		return "", fmt.Errorf("failed to readlink %s: %w", path, err)
	}
	return strings.TrimSpace(output), nil
}
