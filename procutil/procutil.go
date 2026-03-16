// Package procutil provides process group isolation for subprocess management.
// It ensures child processes (especially interactive editors) are cleaned up
// when the parent process exits, preventing orphaned processes that can
// accumulate memory indefinitely.
package procutil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

// SetProcessGroup configures cmd to run in its own process group.
// This ensures the child process can be killed as a group and does not
// become an orphan if the parent process dies.
func SetProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// RunWithCleanup starts cmd and waits for it to finish, cleaning up the
// process group if the context is cancelled or a termination signal is
// received. Use this for CLI (non-TUI) editor launches where the caller
// manages the process lifecycle directly.
//
// For BubbleTea TUI contexts, use SetProcessGroup on the cmd before
// passing it to tea.ExecProcess — BubbleTea manages the lifecycle.
func RunWithCleanup(ctx context.Context, cmd *exec.Cmd) error {
	SetProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting process: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigCh)

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		return err
	case sig := <-sigCh:
		killProcessGroup(cmd.Process.Pid)
		<-done
		return fmt.Errorf("interrupted by signal: %v", sig)
	case <-ctx.Done():
		killProcessGroup(cmd.Process.Pid)
		<-done
		return ctx.Err()
	}
}

// killProcessGroup sends SIGTERM to the process group, then SIGKILL
// after a grace period if the group is still alive.
func killProcessGroup(pid int) {
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		// Process already exited.
		return
	}
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	time.AfterFunc(3*time.Second, func() {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	})
}
