package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/role"
)

// A role's call started from a Claude Code session ran as its child: the
// editor's entrypoint made Claude Code add to the prompt and have a small
// model read the whole question again, twice the tokens in
// (docs/research/claude-session-env.md). claude drops the session's
// variables and keeps the login a CI job needs and the person's settings;
// a cmd: agent, the person's own command, gets their environment as it is.
func TestClaudeCallLeavesTheSessionBehind(t *testing.T) {
	session := map[string]string{
		"CLAUDECODE": "1", "CLAUDE_CODE_ENTRYPOINT": "claude-vscode", "CLAUDE_CODE_SESSION_ID": "s-1",
		"CLAUDE_CODE_CHILD_SESSION": "1", "CLAUDE_CODE_MESSAGING_SOCKET": "/tmp/s.sock",
		"CLAUDE_CODE_MESSAGING_TOKEN": "parent", "CLAUDE_PID": "42", "CLAUDE_EFFORT": "xhigh",
		"MCP_CONNECTION_NONBLOCKING": "true", "AI_AGENT": "claude-code",
	}
	kept := map[string]string{
		"CLAUDE_CODE_OAUTH_TOKEN": "login-kept", "ANTHROPIC_API_KEY": "key-kept",
		"ANTHROPIC_BASE_URL": "http://proxy.example", "CLAUDE_CODE_USE_BEDROCK": "1",
		"CLAUDE_CODE_MAX_OUTPUT_TOKENS": "4000",
	}
	for k, v := range session {
		t.Setenv(k, v)
	}
	for k, v := range kept {
		t.Setenv(k, v)
	}
	bin, dump := t.TempDir(), filepath.Join(t.TempDir(), "env")
	fake := "#!/bin/sh\nenv > \"$WORKLINE_ENV_DUMP\"\ncat > /dev/null\necho '{\"result\":\"[]\",\"modelUsage\":{}}'\n"
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("WORKLINE_ENV_DUMP", dump)
	request := func() Request {
		dir, run := t.TempDir(), t.TempDir()
		os.WriteFile(filepath.Join(dir, "instruction.md"), []byte("Judge."), 0o644)
		os.MkdirAll(filepath.Join(run, "in"), 0o755)
		os.MkdirAll(filepath.Join(run, "out"), 0o755)
		os.WriteFile(filepath.Join(run, "in", "task.md"), []byte("The change."), 0o644)
		return Request{RunDir: run, Repo: t.TempDir(), Role: &role.Role{Name: "committer", Dir: dir, Intentions: []string{"note"}}}
	}
	childEnv := func(a Agent) map[string]string {
		os.Remove(dump)
		if _, err := a.Propose(request()); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(dump)
		if err != nil {
			t.Fatal(err)
		}
		env := map[string]string{}
		for _, l := range strings.Split(string(data), "\n") {
			if k, v, ok := strings.Cut(l, "="); ok {
				env[k] = v
			}
		}
		return env
	}

	env := childEnv(claude{})
	for k := range session {
		if _, ok := env[k]; ok {
			t.Errorf("claude: the session's %s reached the call", k)
		}
	}
	for k, v := range kept {
		if env[k] != v {
			t.Errorf("claude: %s = %q, want %q kept", k, env[k], v)
		}
	}

	env = childEnv(command{script: `env > "$WORKLINE_ENV_DUMP"; cat > /dev/null; echo '[]'`})
	for k, v := range session {
		if env[k] != v {
			t.Errorf("cmd: the person's %s = %q, want %q as they have it", k, env[k], v)
		}
	}
}
