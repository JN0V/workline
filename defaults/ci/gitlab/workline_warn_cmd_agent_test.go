// Package gitlab tests the GitLab CI template by running the script of its
// jobs against a stub of the engine.
package gitlab

import (
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"
)

// warnScript builds the shell script of workline:warn from the template,
// its YAML items read as the runner would run them; the bot's token guard,
// the ready lookup and the gates download (the forge's API) are left out.
// The agent the project sets is `agent`, put in the route command.
func warnScript(t *testing.T, agent string) string {
    t.Helper()
    raw, err := os.ReadFile("workline.gitlab-ci.yml")
    if err != nil {
        t.Fatal(err)
    }
    lines := strings.Split(string(raw), "\n")
    var claude []string
    start, inScript := -1, false
    for i, l := range lines {
        if strings.HasPrefix(l, ".workline:claude: &claude") {
            for _, c := range lines[i+1:] {
                if !strings.HasPrefix(c, "  - ") {
                    break
                }
                claude = append(claude, strings.TrimPrefix(c, "  - "))
            }
        }
        if l == "workline:warn:" {
            start = i
        }
    }
    if start < 0 {
        t.Fatal("no job workline:warn in the template")
    }
    var items [][]string
    for _, l := range lines[start+1:] {
        if l != "" && !strings.HasPrefix(l, " ") {
            break
        }
        if l == "  script:" {
            inScript = true
            continue
        }
        if !inScript {
            continue
        }
        if strings.HasPrefix(l, "    - ") {
            items = append(items, []string{strings.TrimPrefix(l, "    - ")})
        } else if strings.HasPrefix(l, "      ") && len(items) > 0 {
            items[len(items)-1] = append(items[len(items)-1], strings.TrimPrefix(l, "      "))
        } else if strings.HasPrefix(l, "  ") && !strings.HasPrefix(l, "   ") && strings.TrimSpace(l) != "" {
            break
        }
    }
    var out []string
    for _, it := range items {
        head, rest := it[0], it[1:]
        switch {
        case head == "*bot-token" || head == "*ready":
        case head == "*claude":
            out = append(out, claude...)
        case head == "|":
            body := strings.Join(rest, "\n")
            if strings.Contains(body, "api=") {
                out = append(out, "gates=")
            } else {
                out = append(out, body)
            }
        case head == ">":
            cmd := strings.Join(rest, " ")
            out = append(out, strings.ReplaceAll(cmd, `--ai "$ai"`, `--ai "`+agent+`"`))
        case strings.HasPrefix(head, "'"):
            s := strings.TrimSuffix(strings.TrimPrefix(head, "'"), "'")
            out = append(out, strings.ReplaceAll(s, "''", "'"))
        default:
            out = append(out, head)
        }
    }
    return strings.Join(out, "\n")
}

// Another agent
//
// Given a project whose agent is a `cmd:` and no Claude token,
// When a merge request's gate finds one new finding,
// Then the inspector's comment triages it.
func TestAnotherAgent(t *testing.T) {
    dir := t.TempDir()
    args := filepath.Join(dir, "args")
    stub := "#!/bin/sh\n[ \"$1\" = version ] && exit 0\necho \"$@\" > " + args + "\n"
    if err := os.WriteFile(filepath.Join(dir, "workline"), []byte(stub), 0o755); err != nil {
        t.Fatal(err)
    }
    script := warnScript(t, "cmd:my-agent")
    cmd := exec.Command("sh", "-c", script)
    cmd.Dir = dir
    cmd.Env = []string{
        "PATH=" + dir + ":" + os.Getenv("PATH"),
        "WORKLINE_GITLAB_TOKEN=bot",
        "CI_MERGE_REQUEST_IID=7",
        "CI_MERGE_REQUEST_SOURCE_BRANCH_NAME=feat",
        "CI_MERGE_REQUEST_DIFF_BASE_SHA=aaa",
        "CI_COMMIT_SHA=bbb",
    }
    out, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("workline:warn failed: %v\n%s", err, out)
    }
    got, rerr := os.ReadFile(args)
    if rerr != nil {
        t.Fatalf("workline route was not run with a cmd: agent and no Claude token:\n%s", out)
    }
    for _, want := range []string{"route merge-request", "--warn-only", "--ai cmd:my-agent"} {
        if !strings.Contains(string(got), want) {
            t.Errorf("route arguments %q lack %q", got, want)
        }
    }
}
