package gitlab

import (
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"
)

// reviewHeadScript builds the shell script of workline:review:head from the
// template, its YAML items read as the runner would run them.
func reviewHeadScript(t *testing.T) string {
    t.Helper()
    raw, err := os.ReadFile("workline.gitlab-ci.yml")
    if err != nil {
        t.Fatal(err)
    }
    lines := strings.Split(string(raw), "\n")
    start := -1
    for i, l := range lines {
        if l == "workline:review:head:" {
            start = i
        }
    }
    if start < 0 {
        t.Fatal("no job workline:review:head in the template")
    }
    var items [][]string
    inScript := false
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
        case head == "|" || head == ">":
            out = append(out, strings.Join(rest, "\n"))
        case strings.HasPrefix(head, "'"):
            s := strings.TrimSuffix(strings.TrimPrefix(head, "'"), "'")
            out = append(out, strings.ReplaceAll(s, "''", "'"))
        default:
            out = append(out, head)
        }
    }
    return strings.Join(out, "\n")
}

// runReviewHead runs the script of workline:review:head in a folder of its
// own, with a git whose remote has the branch (or not) and a forge whose API
// says the merge request is in `state`. It returns the output and the error.
func runReviewHead(t *testing.T, state string, branchKept bool) (string, error) {
    t.Helper()
    dir := t.TempDir()
    bin := filepath.Join(dir, "bin")
    if err := os.MkdirAll(bin, 0o755); err != nil {
        t.Fatal(err)
    }
    gitStub := "#!/bin/sh\n" +
        "if [ \"$1\" = ls-remote ] && [ -n \"$STUB_BRANCH_SHA\" ]; then printf '%s\\trefs/heads/x\\n' \"$STUB_BRANCH_SHA\"; fi\n" +
        "exit 0\n"
    curlStub := "#!/bin/sh\nprintf '{\"state\":\"%s\"}\\n' \"$STUB_MR_STATE\"\n"
    for name, body := range map[string]string{"git": gitStub, "curl": curlStub} {
        if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0o755); err != nil {
            t.Fatal(err)
        }
    }
    sha := ""
    if branchKept {
        sha = strings.Repeat("a", 40)
    }
    cmd := exec.Command("sh", "-c", reviewHeadScript(t))
    cmd.Dir = dir
    cmd.Env = []string{
        "PATH=" + bin + ":" + os.Getenv("PATH"),
        "WORKLINE_GITLAB_TOKEN=bot",
        "GITLAB_TOKEN=bot",
        "CI_JOB_TOKEN=job",
        "CI_API_V4_URL=https://gitlab.example/api/v4",
        "CI_PROJECT_ID=42",
        "MR_IID=7",
        "MR_SOURCE_BRANCH=workline/developer/fix",
        "STUB_MR_STATE=" + state,
        "STUB_BRANCH_SHA=" + sha,
    }
    out, err := cmd.CombinedOutput()
    return string(out), err
}

// Merged meanwhile
//
// Given a review round started on an open merge request,
// When it is merged before the round runs (its branch deleted by the merge),
// Then the pipeline says so and is green.
func TestMergedMeanwhile(t *testing.T) {
    out, err := runReviewHead(t, "merged", false)
    if err != nil {
        t.Fatalf("workline:review:head failed on a merged merge request: %v\n%s", err, out)
    }
    if !strings.Contains(strings.ToLower(out), "merged") {
        t.Errorf("the job does not say the merge request is merged:\n%s", out)
    }
}

// Closed meanwhile
//
// Given the same,
// When it is closed instead, branch kept or not,
// Then the pipeline says so and is green.
func TestClosedMeanwhile(t *testing.T) {
    for name, kept := range map[string]bool{"branch gone": false, "branch kept": true} {
        t.Run(name, func(t *testing.T) {
            out, err := runReviewHead(t, "closed", kept)
            if err != nil {
                t.Fatalf("workline:review:head failed on a closed merge request: %v\n%s", err, out)
            }
            if !strings.Contains(strings.ToLower(out), "closed") {
                t.Errorf("the job does not say the merge request is closed:\n%s", out)
            }
        })
    }
}
