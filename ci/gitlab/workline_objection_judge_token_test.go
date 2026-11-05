package gitlab

import (
    "os"
    "strings"
    "testing"
)

// jobBlock returns the lines of the top-level key `name` of the CI file,
// up to the next line that starts in column 0.
func jobBlock(t *testing.T, name string) []string {
    t.Helper()
    data, err := os.ReadFile("workline.gitlab-ci.yml")
    if err != nil {
        t.Fatal(err)
    }
    var block []string
    in := false
    for _, line := range strings.Split(string(data), "\n") {
        if strings.HasPrefix(line, name+":") {
            in = true
            block = append(block, line)
            continue
        }
        if in {
            if line != "" && line[0] != ' ' && line[0] != '\t' {
                break
            }
            block = append(block, line)
        }
    }
    if len(block) == 0 {
        t.Fatalf("job %s not found", name)
    }
    return block
}

// unsetNames returns the variable names a job text unsets, each as a
// separate word, counting only the unsets that run before the job's script
// commands: those in before_script, or at the head of script.
func unsetNames(text string) map[string]bool {
    names := map[string]bool{}
    section := ""
    started := false
    for _, line := range strings.Split(text, "\n") {
        l := strings.TrimSpace(line)
        switch {
        case strings.HasPrefix(l, "before_script:"):
            section = "before"
            continue
        case strings.HasPrefix(l, "script:"):
            section = "script"
            continue
        case strings.HasPrefix(l, "after_script:"):
            section = ""
            continue
        case l != "" && !strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "#"):
            if strings.HasSuffix(l, ":") || strings.Contains(l, ": ") {
                section = ""
            }
        }
        if section == "" {
            continue
        }
        words := strings.Fields(strings.NewReplacer(";", " ", "&&", " ", "\"", " ", "'", " ").Replace(l))
        isUnset := false
        for i, w := range words {
            if w == "unset" {
                isUnset = true
                for _, n := range words[i+1:] {
                    if section == "before" || !started {
                        names[n] = true
                    }
                }
                break
            }
        }
        if section == "script" && !isUnset && strings.HasPrefix(l, "-") {
            started = true
        }
    }
    return names
}

// The judge holds no write token
// Given a GitLab project with the template's objection jobs,
// When an objection wakes `workline:objection:judge`,
// Then the job's environment holds no token that can push or comment.
func TestTheJudgeHoldsNoWriteToken(t *testing.T) {
    judge := strings.Join(jobBlock(t, "workline:objection:judge"), "\n")
    parent := strings.Join(jobBlock(t, ".workline:objection"), "\n")

    effective := judge
    if !strings.Contains(judge, "before_script:") {
        effective += "\n" + parent
    }
    if strings.Contains(effective, "export GITLAB_TOKEN=$WORKLINE_GITLAB_TOKEN") {
        t.Errorf("the judge exports the bot's token as GITLAB_TOKEN")
    }
    if strings.Contains(effective, "*bot-token") {
        t.Errorf("the judge requires the bot's token (WORKLINE_GITLAB_TOKEN)")
    }

    names := unsetNames(effective)
    unsets := names["WORKLINE_GITLAB_TOKEN"] && names["GITLAB_TOKEN"]
    otherEnv := false
    for _, line := range strings.Split(judge, "\n") {
        l := strings.TrimSpace(line)
        if strings.HasPrefix(l, "environment:") && !strings.Contains(l, "*environment") {
            otherEnv = true
        }
    }
    ok := unsets || otherEnv
    if !ok {
        t.Errorf("the judge still receives the bot's token: it neither leaves the environment workline nor unsets WORKLINE_GITLAB_TOKEN and GITLAB_TOKEN, each as its own word, before its script")
    }
}
