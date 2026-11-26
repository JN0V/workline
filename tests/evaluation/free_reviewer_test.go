package evaluation

// A free reviewer on the reviewer's replay set: one agent call on the
// checkout, read tools and read-only git, the maintainer's question, no
// lens, no judge, no quote check. Its answer is scored as the reviewer's
// (scoreReview), each defect it would block on an Important, so the two
// compare on the same points (docs/spec/conformance/replay.md).
//
//	WORKLINE_FREE_REVIEWER=claude:sonnet@medium WORKLINE_EVAL_RESULTS=rows.tsv \
//	  go test -count=1 -p 1 -parallel 3 -timeout 120m -run 'TestFreeReviewer/' ./tests/evaluation/
//
// WORKLINE_FREE_TRANSCRIPTS names a folder that keeps each call's
// messages and answer.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"
)

// freeTools: read and search, and git to read the history and the
// change; dontAsk denies any other command.
var freeTools = []string{"Bash(git log:*)", "Bash(git show:*)", "Bash(git diff:*)"}

func TestFreeReviewer(t *testing.T) {
	onDemand(t)
	spec := os.Getenv("WORKLINE_FREE_REVIEWER")
	if spec == "" {
		t.Skip("set WORKLINE_FREE_REVIEWER=claude:<model>@<effort> to run it: it calls a real agent and costs tokens")
	}
	model, effort, _ := strings.Cut(strings.TrimPrefix(spec, "claude:"), "@")
	keep := os.Getenv("WORKLINE_FREE_TRANSCRIPTS")
	files, _ := filepath.Glob("replay/reviewer/*.yaml")
	sort.Strings(files)
	version := worklineVersion()
	for _, f := range files {
		c, err := readCase(f)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(strings.TrimPrefix(c.Case, "reviewer/"), func(t *testing.T) {
			t.Parallel()
			start := time.Now()
			_, repo, env, err := build(t, &c)
			if err != nil {
				t.Fatal(err)
			}
			// What came after the change stays out of reach: the mirror's
			// branches and the tags past its head.
			if err := sh(repo, env, "git remote remove origin && git tag --no-merged change | xargs -r git tag -d >/dev/null"); err != nil {
				t.Fatal(err)
			}
			changed, err := changedFiles(repo)
			if err != nil {
				t.Fatal(err)
			}
			stream, cl, err := askFree(repo, model, effort, freePrompt(&c))
			name := strings.TrimPrefix(c.Case, "reviewer/") + "-" + start.UTC().Format("150405.000")
			if keep != "" {
				if werr := errors.Join(os.MkdirAll(keep, 0o755),
					os.WriteFile(filepath.Join(keep, name+".jsonl"), stream, 0o644),
					os.WriteFile(filepath.Join(keep, name+".answer.md"), []byte(cl.answer), 0o644)); werr != nil {
					t.Fatal(werr)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			findings := freeFindings(cl.answer, changed)
			rc := *c.Review
			rc.Replay, rc.Clean = true, strings.Contains(c.Case, "/clean-")
			passed, failed, measure, notes := scoreReview(&rc, &reviewRun{Findings: findings})
			score := fmt.Sprintf("%d/%d", passed, passed+len(failed))
			t.Logf("%s — %s; failed: %s", c.Case, score, strings.Join(failed, "; "))
			for _, n := range notes {
				t.Logf("  finding: %s", n)
			}
			measure = fmt.Sprintf("free: blocks=%d fresh=%d cache-read=%d cache-written=%d out=%d turns=%d tools=%d; %s",
				len(findings), cl.in-cl.cached+cl.out, cl.cached, cl.written, cl.out, cl.turns, cl.tools, measure)
			line := strings.Join([]string{time.Now().UTC().Format(time.RFC3339), version, "free:" + spec, cl.model, effort,
				c.Case, score, "1", fmt.Sprint(cl.in), fmt.Sprint(cl.out), fmt.Sprintf("%.4f", cl.cost),
				fmt.Sprintf("%.0f", time.Since(start).Seconds()), strings.Join(failed, "; "), "", measure}, "\t")
			if err := appendResult(line); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// freePrompt is the maintainer's question: the issues, the pull request,
// where the change is; nothing the line prepares.
func freePrompt(c *caseFile) string {
	var b strings.Builder
	var forge struct {
		Issues []struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
			Body  string `json:"body"`
		} `json:"issues"`
		MRs []struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
			Body  string `json:"body"`
		} `json:"merge-requests"`
	}
	data, _ := json.Marshal(c.Given.Forge)
	_ = json.Unmarshal(data, &forge) // a case with no forge: the question alone
	pr := "this pull request"
	if len(forge.MRs) > 0 {
		pr = fmt.Sprintf("pull request #%d", forge.MRs[0].ID)
	}
	fmt.Fprintf(&b, "You are the maintainer of %s, reviewing %s before you merge it.\n\n", c.Given.Clone.Repo, pr)
	b.WriteString("The repository is checked out in the current directory: the branch `change` (HEAD) is the pull request's head, `main` its base. `git diff main...HEAD` and `git log main..HEAD` show the change. Read whatever you need in the repository: the change, the code it calls, its tests, the CI.\n\n")
	for _, i := range forge.Issues {
		fmt.Fprintf(&b, "## Issue #%d: %s\n\n%s\n\n", i.ID, i.Title, strings.TrimSpace(i.Body))
	}
	for _, m := range forge.MRs {
		fmt.Fprintf(&b, "## Pull request #%d: %s\n\n%s\n\n", m.ID, m.Title, strings.TrimSpace(m.Body))
	}
	b.WriteString(`## Your answer

List only the defects you would block the merge on. Each on a line of its own, in this form:

BLOCK <path>:<line> — <why>

<path> from the repository's root, <line> the line at HEAD where the cause is. When the defect is that the change does not deliver what an issue asks, cite the issue instead: BLOCK #<n> — <why>.

If you would merge it as it is, end with the line: no blocking defect.
`)
	return b.String()
}

// freeCall is what one call used and answered.
type freeCall struct {
	answer, model                   string
	in, cached, written, out, turns int
	tools                           int
	cost                            float64
}

// askFree runs one claude call in repo, read tools and read-only git,
// and returns its messages as they came.
func askFree(repo, model, effort, prompt string) ([]byte, freeCall, error) {
	var cl freeCall
	args := []string{"-p", "--output-format", "stream-json", "--verbose", "--tools", "Read,Glob,Grep,Bash",
		"--allowedTools", strings.Join(freeTools, ","), "--permission-mode", "dontAsk", "--strict-mcp-config",
		"--setting-sources", "", "--no-session-persistence"}
	if model != "" {
		args = append(args, "--model", model)
	}
	if effort != "" {
		args = append(args, "--effort", effort)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir, cmd.Stdin = repo, strings.NewReader(prompt)
	cmd.Env = slices.DeleteFunc(os.Environ(), func(kv string) bool {
		name, _, _ := strings.Cut(kv, "=")
		return name == "CLAUDECODE" || name == "AI_AGENT" || name == "CLAUDE_PID" || name == "CLAUDE_EFFORT" ||
			strings.HasPrefix(name, "CLAUDE_CODE_") || name == "GH_TOKEN" || name == "GITHUB_TOKEN"
	})
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	runErr := cmd.Run()
	var res *claudeStreamResult
	sc := bufio.NewScanner(bytes.NewReader(out.Bytes()))
	sc.Buffer(make([]byte, 0, 1<<20), 64<<20)
	for sc.Scan() {
		var m struct {
			Type    string `json:"type"`
			Message struct {
				Content []struct {
					Type string `json:"type"`
				} `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(sc.Bytes(), &m) != nil {
			continue
		}
		for _, p := range m.Message.Content {
			if m.Type == "assistant" && p.Type == "tool_use" {
				cl.tools++
			}
		}
		if m.Type == "result" {
			res = &claudeStreamResult{}
			if err := json.Unmarshal(sc.Bytes(), res); err != nil {
				return out.Bytes(), cl, fmt.Errorf("claude's result unread: %v", err)
			}
		}
	}
	if res == nil {
		return out.Bytes(), cl, fmt.Errorf("claude gave no result (%v): %s", runErr, lastLines(errOut.String()))
	}
	cl.answer, cl.turns, cl.cost = res.Result, res.NumTurns, res.CostUSD
	most := -1
	for m, u := range res.ModelUsage {
		if u.OutputTokens > most {
			cl.model, most = m, u.OutputTokens
		}
		cl.in += u.InputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens
		cl.cached += u.CacheReadInputTokens
		cl.written += u.CacheCreationInputTokens
		cl.out += u.OutputTokens
	}
	if res.IsError {
		return out.Bytes(), cl, fmt.Errorf("claude failed: %s", res.Result)
	}
	return out.Bytes(), cl, nil
}

type claudeStreamResult struct {
	Result     string  `json:"result"`
	IsError    bool    `json:"is_error"`
	NumTurns   int     `json:"num_turns"`
	CostUSD    float64 `json:"total_cost_usd"`
	ModelUsage map[string]struct {
		InputTokens              int `json:"inputTokens"`
		OutputTokens             int `json:"outputTokens"`
		CacheReadInputTokens     int `json:"cacheReadInputTokens"`
		CacheCreationInputTokens int `json:"cacheCreationInputTokens"`
	} `json:"modelUsage"`
}

func lastLines(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.Join(lines[max(0, len(lines)-5):], "\n")
}

// blockRe reads a BLOCK line: an issue, or a path and a line (a range
// taken at its first line), backticks and bold marks allowed.
var blockRe = regexp.MustCompile("(?m)^[\\s>*-]*\\**BLOCK\\**:?\\s+`?(#\\d+|[^\\s:`]+:L?(\\d+))(?:-L?\\d+)?`?\\s*(.*)$")

// freeFindings are the defects an answer blocks on, each an Important:
// on the change when its file is one the change touches (or an issue),
// outside it otherwise, as the reviewer's own findings are placed.
func freeFindings(answer string, changed map[string]bool) []lensFinding {
	var out []lensFinding
	for _, m := range blockRe.FindAllStringSubmatch(answer, -1) {
		where := m[1]
		if m[2] != "" {
			file, _, _ := strings.Cut(where, ":")
			file = strings.TrimPrefix(file, "./")
			where = file + ":" + m[2]
		}
		file, _, _ := strings.Cut(where, ":")
		related := strings.HasPrefix(where, "#") || changed[file]
		title := strings.TrimLeft(strings.TrimSpace(m[3]), "—–-: ")
		out = append(out, lensFinding{Lens: "free", Where: where, Severity: "important", Title: title, Related: related})
	}
	return out
}

// changedFiles are the files the change touches, main...change.
func changedFiles(repo string) (map[string]bool, error) {
	out, err := exec.Command("git", "-C", repo, "diff", "--name-only", "main...change").Output()
	if err != nil {
		return nil, err
	}
	files := map[string]bool{}
	for _, f := range strings.Fields(string(out)) {
		files[f] = true
	}
	return files, nil
}

// TestFreeFindingsRead: the BLOCK lines read as Importants, placed on the
// change or outside it, whatever marks the agent wraps them in.
func TestFreeFindingsRead(t *testing.T) {
	answer := "Review.\n\nBLOCK `src/a.h:209` — the poll drains 8\n- **BLOCK** src/b.h:L12-L20 — untested\nBLOCK #139 — link-down scenario not met\nBLOCK lib/c.go:5: off the change\nno blocking defect elsewhere\n"
	got := freeFindings(answer, map[string]bool{"src/a.h": true, "src/b.h": true})
	want := []string{"src/a.h:209 true", "src/b.h:12 true", "#139 true", "lib/c.go:5 false"}
	var said []string
	for _, f := range got {
		said = append(said, fmt.Sprintf("%s %v", f.Where, f.Related))
	}
	if !slices.Equal(said, want) {
		t.Errorf("read %q, want %q", said, want)
	}
}
