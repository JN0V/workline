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

    unsets := false
    for _, line := range strings.Split(effective, "\n") {
        if strings.Contains(line, "unset") && strings.Contains(line, "WORKLINE_GITLAB_TOKEN") && strings.Contains(line, "GITLAB_TOKEN") {
            unsets = true
        }
    }
    otherEnv := false
    for _, line := range strings.Split(judge, "\n") {
        l := strings.TrimSpace(line)
        if strings.HasPrefix(l, "environment:") && !strings.Contains(l, "*environment") {
            otherEnv = true
        }
    }
    if !unsets && !otherEnv {
        t.Errorf("the judge still receives the bot's token: it neither leaves the environment workline nor unsets WORKLINE_GITLAB_TOKEN and GITLAB_TOKEN")
    }
}
