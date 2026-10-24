package updatecheck

import "testing"

func envOf(kv ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return func(k string) string { return m[k] }
}

func TestAmbientSuppressedMatrix(t *testing.T) {
	tests := []struct {
		name string
		in   SuppressInput
		want bool
	}{
		{"human tty", SuppressInput{StderrIsTTY: true, LookupEnv: envOf()}, false},
		{"stderr not a tty", SuppressInput{LookupEnv: envOf()}, true},
		{"json", SuppressInput{StderrIsTTY: true, JSONMode: true, LookupEnv: envOf()}, true},
		{"agent flag", SuppressInput{StderrIsTTY: true, AgentMode: true, LookupEnv: envOf()}, true},
		{"opt-out 1", SuppressInput{StderrIsTTY: true, LookupEnv: envOf(EnvNoUpdateCheck, "1")}, true},
		{"opt-out yes", SuppressInput{StderrIsTTY: true, LookupEnv: envOf(EnvNoUpdateCheck, "YES")}, true},
		{"opt-out 0", SuppressInput{StderrIsTTY: true, LookupEnv: envOf(EnvNoUpdateCheck, "0")}, false},
		{"opt-out false", SuppressInput{StderrIsTTY: true, LookupEnv: envOf(EnvNoUpdateCheck, "false")}, false},
		{"CI true", SuppressInput{StderrIsTTY: true, LookupEnv: envOf("CI", "true")}, true},
		{"CI vendor name", SuppressInput{StderrIsTTY: true, LookupEnv: envOf("CI", "woodpecker")}, true},
		{"CI false", SuppressInput{StderrIsTTY: true, LookupEnv: envOf("CI", "false")}, false},
		{"GITHUB_ACTIONS", SuppressInput{StderrIsTTY: true, LookupEnv: envOf("GITHUB_ACTIONS", "true")}, true},
		{"OBEY_AGENT", SuppressInput{StderrIsTTY: true, LookupEnv: envOf("OBEY_AGENT", "1")}, true},
		{"OBEY_SESSION_ID", SuppressInput{StderrIsTTY: true, LookupEnv: envOf("OBEY_SESSION_ID", "s-1")}, true},
		{"CLAUDECODE", SuppressInput{StderrIsTTY: true, LookupEnv: envOf("CLAUDECODE", "1")}, true},
		{"CLAUDE_CODE", SuppressInput{StderrIsTTY: true, LookupEnv: envOf("CLAUDE_CODE", "1")}, true},
		{"CLAUDE_CODE_SESSION_ID", SuppressInput{StderrIsTTY: true, LookupEnv: envOf("CLAUDE_CODE_SESSION_ID", "x")}, true},
		{"CODEX_TASK", SuppressInput{StderrIsTTY: true, LookupEnv: envOf("CODEX_TASK", "t")}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AmbientSuppressed(tt.in); got != tt.want {
				t.Errorf("AmbientSuppressed = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAmbientSuppressedDefaultsToProcessEnv(t *testing.T) {
	for _, name := range append(append([]string{EnvNoUpdateCheck}, ciMarkers...), agentMarkers...) {
		t.Setenv(name, "")
	}
	if AmbientSuppressed(SuppressInput{StderrIsTTY: true}) {
		t.Fatal("suppressed with a clean environment")
	}
	t.Setenv(EnvNoUpdateCheck, "1")
	if !AmbientSuppressed(SuppressInput{StderrIsTTY: true}) {
		t.Fatal("process env opt-out ignored")
	}
}
