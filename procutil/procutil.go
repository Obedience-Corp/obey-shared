//go:build unix

package procutil

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

// killGracePeriod is the time between SIGTERM and SIGKILL escalation.
const killGracePeriod = 3 * time.Second

// ErrInterrupted indicates the process was terminated by an OS signal.
var ErrInterrupted = errors.New("process interrupted by signal")

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
//
// On signal interruption, the returned error wraps ErrInterrupted so callers
// can check with errors.Is(err, procutil.ErrInterrupted).
func RunWithCleanup(ctx context.Context, cmd *exec.Cmd) error {
	SetProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		return errors.New("starting process: " + err.Error())
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
		cancelKill := killProcessGroup(cmd.Process.Pid)
		waitErr := <-done
		cancelKill()
		return errors.Join(ErrInterrupted, errors.New(sig.String()), waitErr)
	case <-ctx.Done():
		cancelKill := killProcessGroup(cmd.Process.Pid)
		waitErr := <-done
		cancelKill()
		return errors.Join(ctx.Err(), waitErr)
	}
}

// killProcessGroup sends SIGTERM to the process group, then SIGKILL after
// killGracePeriod if the group is still alive. It returns a cancel func that
// stops the escalation timer; callers must invoke it once Wait returns to
// prevent a delayed SIGKILL from firing against a reused process group ID.
func killProcessGroup(pid int) (cancel func()) {
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		// Process already exited; nothing to cancel.
		return func() {}
	}
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
		// Process group gone; no escalation needed.
		return func() {}
	}
	t := time.AfterFunc(killGracePeriod, func() {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	})
	return func() { t.Stop() }
}
