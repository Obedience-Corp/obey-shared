package updatecheck

import (
	"os"
	"strings"
)

// SuppressInput is the caller's process context for the ambient path.
type SuppressInput struct {
	// LookupEnv reads the environment; nil means os.Getenv.
	LookupEnv func(string) string

	// StderrIsTTY must be true for any ambient work.
	StderrIsTTY bool

	// JSONMode is true when the command emits machine-readable output.
	JSONMode bool

	// AgentMode is true for agent-oriented invocations (for example --agent).
	AgentMode bool
}

var ciMarkers = []string{"CI", "GITHUB_ACTIONS"}

// agentMarkers matches fest's operatorAgentMarkers. Agents hosted in a real
// PTY pass the stderr TTY check, so the environment is the only signal.
var agentMarkers = []string{
	"OBEY_AGENT",
	"OBEY_SESSION_ID",
	"CLAUDE_CODE",
	"CLAUDECODE",
	"CLAUDE_CODE_SESSION_ID",
	"CODEX_TASK",
}

// AmbientSuppressed reports whether the ambient path must skip all work,
// including cache reads and network refresh.
func AmbientSuppressed(in SuppressInput) bool {
	if in.JSONMode || in.AgentMode || !in.StderrIsTTY {
		return true
	}
	getenv := in.LookupEnv
	if getenv == nil {
		getenv = os.Getenv
	}
	if envSet(getenv(EnvNoUpdateCheck)) {
		return true
	}
	for _, name := range ciMarkers {
		if envSet(getenv(name)) {
			return true
		}
	}
	for _, name := range agentMarkers {
		if envSet(getenv(name)) {
			return true
		}
	}
	return false
}

func envSet(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "0", "false", "no", "off":
		return false
	default:
		return true
	}
}
