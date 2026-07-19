package brand

import (
	"os"
	"strings"
)

// EnvironmentCapabilities applies shared environment policy to caller-owned
// terminal facts. isTTY and colorDepth are parameters so this helper never
// opens, queries, or mutates a terminal on behalf of a consumer.
func EnvironmentCapabilities(isTTY bool, colorDepth ColorDepth) Capabilities {
	term := strings.TrimSpace(os.Getenv("TERM"))
	_, noColor := os.LookupEnv("NO_COLOR")
	return Capabilities{
		IsTTY:                 isTTY,
		ColorDepth:            colorDepth,
		ReducedMotion:         ReducedMotion(),
		NoColor:               noColor,
		ContinuousIntegration: os.Getenv("CI") != "",
		DumbTerminal:          strings.EqualFold(term, "dumb"),
	}
}

// ReducedMotion reports whether shared ambient animation should be disabled.
// OBEY_REDUCED_MOTION is the shared name; FESTIVAL_REDUCED_MOTION remains a
// compatibility alias for existing installer and Festival configurations.
func ReducedMotion() bool {
	for _, name := range []string{"OBEY_REDUCED_MOTION", "FESTIVAL_REDUCED_MOTION"} {
		if isTruthy(os.Getenv(name)) {
			return true
		}
	}
	return false
}

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
