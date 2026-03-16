//go:build unix

package procutil

import (
	"context"
	"errors"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestSetProcessGroup(t *testing.T) {
	t.Run("nil SysProcAttr", func(t *testing.T) {
		cmd := exec.Command("true")
		SetProcessGroup(cmd)
		if cmd.SysProcAttr == nil {
			t.Fatal("SysProcAttr should not be nil")
		}
		if !cmd.SysProcAttr.Setpgid {
			t.Fatal("Setpgid should be true")
		}
	})

	t.Run("existing SysProcAttr preserved", func(t *testing.T) {
		cmd := exec.Command("true")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Noctty: true,
		}
		SetProcessGroup(cmd)
		if !cmd.SysProcAttr.Setpgid {
			t.Fatal("Setpgid should be true")
		}
		if !cmd.SysProcAttr.Noctty {
			t.Fatal("existing Noctty should be preserved")
		}
	})
}

func TestRunWithCleanup_HappyPath(t *testing.T) {
	ctx := context.Background()
	cmd := exec.Command("echo", "hello")
	err := RunWithCleanup(ctx, cmd)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestRunWithCleanup_ProcessExitError(t *testing.T) {
	ctx := context.Background()
	cmd := exec.Command("false")
	err := RunWithCleanup(ctx, cmd)
	if err == nil {
		t.Fatal("expected error from failing process")
	}
}

func TestRunWithCleanup_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Start a long-running process
	cmd := exec.Command("sleep", "60")
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunWithCleanup(ctx, cmd)
	}()

	// Give the process time to start
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for cleanup")
	}
}

func TestRunWithCleanup_StartFailure(t *testing.T) {
	ctx := context.Background()
	cmd := exec.Command("/nonexistent/binary/path")
	err := RunWithCleanup(ctx, cmd)
	if err == nil {
		t.Fatal("expected error for nonexistent binary")
	}
}

func TestErrInterrupted_IsSentinel(t *testing.T) {
	wrapped := errors.Join(ErrInterrupted, errors.New("SIGTERM"))
	if !errors.Is(wrapped, ErrInterrupted) {
		t.Fatal("should be detectable via errors.Is")
	}
}
