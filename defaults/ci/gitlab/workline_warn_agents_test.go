package gitlab

import (
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"
)

// runWarn runs the script of workline:warn with the agent `agent` and the
// extra environment `env`; it returns the output, the route arguments (empty
// when route was not run) and the error of the script.
func runWarn(t *testing.T, agent string, env ...string) (string, string, error) {
    t.Helper()
    dir := t.TempDir()
    args := filepath.Join(dir, "args")
    stub := "#!/bin/sh\n[ \"$1\" = version ] && exit 0\necho \"$@\" > " + args + "\n"
    if err := os.WriteFile(filepath.Join(dir, "workline"), []byte(stub), 0o755); err != nil {
        t.Fatal(err)
    }
    cmd := exec.Command("sh", "-c", warnScript(t, agent))
    cmd.Dir = dir
    cmd.Env = append([]string{
        "PATH=" + dir + ":" + os.Getenv("PATH"),
        "WORKLINE_GITLAB_TOKEN=bot",
        "CI_MERGE_REQUEST_IID=7",
        "CI_MERGE_REQUEST_SOURCE_BRANCH_NAME=feat",
        "CI_MERGE_REQUEST_DIFF_BASE_SHA=aaa",
        "CI_COMMIT_SHA=bbb",
    }, env...)
    out, err := cmd.CombinedOutput()
    got, _ := os.ReadFile(args)
    return string(out), string(got), err
}

// No agent: no token, no agent: exit 0, no route call, the message printed.
func TestNoAgentNoToken(t *testing.T) {
    out, got, err := runWarn(t, "none")
    if err != nil {
        t.Fatalf("workline:warn failed: %v\n%s", err, out)
    }
    if got != "" {
        t.Errorf("workline route was run with no agent: %q", got)
    }
    if !strings.Contains(out, "no agent, nothing read apart") {
        t.Errorf("no message that no agent is set:\n%s", out)
    }
}

// A Claude token set: the job routes with the Claude agent, as before.
func TestClaudeToken(t *testing.T) {
    out, got, err := runWarn(t, "claude", "CLAUDE_CODE_OAUTH_TOKEN=tok", "draft=true")
    if err != nil {
        t.Fatalf("workline:warn failed: %v\n%s", err, out)
    }
    for _, want := range []string{"route merge-request", "--warn-only", "--ai claude"} {
        if !strings.Contains(got, want) {
            t.Errorf("route arguments %q lack %q", got, want)
        }
    }
}
