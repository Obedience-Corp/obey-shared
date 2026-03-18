//go:build unix

package procutil

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
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

func TestRunWithCleanup_StartFailurePreservesCause(t *testing.T) {
	ctx := context.Background()
	cmd := exec.Command("/nonexistent/binary/path")
	err := RunWithCleanup(ctx, cmd)
	if err == nil {
		t.Fatal("expected error for nonexistent binary")
	}

	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("expected wrapped *os.PathError, got: %v", err)
	}
}

func TestErrInterrupted_IsSentinel(t *testing.T) {
	wrapped := errors.Join(ErrInterrupted, errors.New("SIGTERM"))
	if !errors.Is(wrapped, ErrInterrupted) {
		t.Fatal("should be detectable via errors.Is")
	}
}

func TestHelperProcessIgnoresSIGTERM(t *testing.T) {
	if os.Getenv("PROCUTIL_HELPER_IGNORE_TERM") != "1" {
		return
	}

	signal.Ignore(syscall.SIGTERM)
	_, _ = os.Stdout.WriteString("ready\n")
	for {
		time.Sleep(250 * time.Millisecond)
	}
}

func TestKillProcessGroup_EscalatesToSIGKILL(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcessIgnoresSIGTERM")
	cmd.Env = append(os.Environ(), "PROCUTIL_HELPER_IGNORE_TERM=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe failed: %v", err)
	}
	SetProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	ready, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatalf("failed waiting for helper readiness: %v", err)
	}
	if ready != "ready\n" {
		t.Fatalf("unexpected helper readiness output: %q", ready)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	cancelKill := killProcessGroup(cmd.Process.Pid)
	defer cancelKill()

	select {
	case err := <-done:
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("expected exit error after kill, got: %v", err)
		}

		status, ok := exitErr.Sys().(syscall.WaitStatus)
		if !ok {
			t.Fatalf("expected syscall.WaitStatus, got: %T", exitErr.Sys())
		}
		if !status.Signaled() {
			t.Fatalf("expected signaled exit, got: %v", status)
		}
		if status.Signal() != syscall.SIGKILL {
			t.Fatalf("expected SIGKILL escalation, got: %v", status.Signal())
		}
	case <-time.After(killGracePeriod + 5*time.Second):
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		<-done
		t.Fatal("timed out waiting for SIGKILL escalation")
	}
}
