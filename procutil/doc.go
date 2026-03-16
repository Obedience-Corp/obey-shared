// Package procutil provides process group isolation for subprocess management.
// It ensures child processes (especially interactive editors) are cleaned up
// when the parent process exits, preventing orphaned processes that can
// accumulate memory indefinitely.
//
// Two entry points are provided:
//
//   - [RunWithCleanup]: for CLI contexts where the caller manages the process
//     lifecycle directly. It intercepts SIGINT/SIGTERM/SIGHUP for the duration
//     of the call, so callers with their own signal handling (e.g. BubbleTea apps)
//     should use [SetProcessGroup] instead.
//
//   - [SetProcessGroup]: for TUI contexts (BubbleTea) where the framework
//     manages the child lifecycle via tea.ExecProcess. Call this on the
//     *exec.Cmd before handing it to BubbleTea.
//
// # Signal Handling
//
// RunWithCleanup registers a process-wide signal handler via signal.Notify for
// SIGINT, SIGTERM, and SIGHUP. This temporarily intercepts those signals from
// any other handlers in the process for the duration of the call. If you need
// concurrent signal handling (e.g. graceful shutdown alongside editor subprocess
// management), use SetProcessGroup with your own lifecycle control instead.
package procutil
