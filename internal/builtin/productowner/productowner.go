// Package productowner holds the deterministic steps of the product owner
// role (roles/product-owner): pre lists the open issues with what the engine
// knows of each; the acts the agent proposes are checked when the engine
// applies them (internal/backlog). The role's scripts call them through
// `workline builtin product-owner pre|post`.
package productowner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

const (
	exitNothing   = 10
	exitExternal  = 3
	bodyMax       = 3000 // characters of an issue's body given to the agent
	commentsMax   = 5    // its last comments given
	filesPerIssue = 3    // files an issue names, given whole
	codeLines     = 400  // lines of a file given
)

// Pre lists the open issues in in/task.md. An issue without a state comment
// gets one, through a fallback comment, and is judged from the next run; one
// whose state comment does not read is left out, and said.
func Pre(runDir, repo string) int {
	role := os.Getenv("WORKLINE_ROLE")
	if role == "" {
		role = "product-owner"
	}
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil {
		return fail(err)
	}
	b, ok := f.(forge.Backlog)
	if f == nil || !ok {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no backlog to keep",
			Findings: []verdict.Finding{{Rule: "no-forge", Message: "the product owner keeps a forge's issues; " + forge.Missing}}})
	}
	open, err := b.Issues()
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return exitExternal
	}
	if err != nil {
		return fail(err)
	}
	head, err := exec.Command("git", "-C", repo, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return fail(fmt.Errorf("the commit the run is on: %v", err))
	}
	commit := strings.TrimSpace(string(head))
	var task strings.Builder
	var fallback []intent.Intention
	var findings []verdict.Finding
	judged := 0
	tracked := trackedFiles(repo)
	var code []string // the files the issues name, given once each
	for _, is := range open {
		if is.Title == backlog.ReportTitle(role) {
			continue
		}
		comments, err := b.Comments(forge.Target{Kind: "issue", ID: is.ID})
		if errors.Is(err, forge.ErrUnreachable) {
			fmt.Fprintln(os.Stderr, err)
			return exitExternal
		}
		if err != nil {
			return fail(err)
		}
		st, found, err := backlog.ReadState(comments, role)
		switch {
		case !found:
			fallback = append(fallback, intent.Intention{Kind: "comment", Value: map[string]any{
				"issue": is.ID, "sticky": "state", "body": backlog.FormatState(backlog.State{Confirmed: commit})}})
			continue
		case err != nil:
			findings = append(findings, verdict.Finding{Rule: "state-broken", Where: fmt.Sprintf("#%d", is.ID),
				Message: "its state comment does not read (" + err.Error() + "): the issue is not judged, nothing is written on it"})
			continue
		}
		judged++
		files := named(is, st, tracked)
		writeIssue(&task, is, st, comments, files)
		for _, f := range files {
			if !slices.Contains(code, f) {
				code = append(code, f)
			}
		}
	}
	if len(fallback) > 0 {
		if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), fallback); err != nil {
			return fail(err)
		}
	}
	if judged == 0 {
		if len(fallback) > 0 { // no task.md: the agent is not asked, the state comments are written
			return writeFindings(runDir, findings)
		}
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no issue to judge", Findings: findings})
	}
	intro := fmt.Sprintf("The run is on commit %s. %d open issues to read against the code.\n\n", commit, judged)
	writeCode(&task, repo, code)
	if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(intro+task.String()), 0o644); err != nil {
		return fail(err)
	}
	return writeFindings(runDir, findings)
}

// writeIssue gives one issue to the agent: what the engine knows of it, its
// body, its last comments, the engine's own left out.
func writeIssue(b *strings.Builder, is forge.Issue, st *backlog.State, comments []string, files []string) {
	fmt.Fprintf(b, "## #%d %s\n\n", is.ID, is.Title)
	if len(is.Labels) > 0 {
		fmt.Fprintf(b, "Labels: %s\n", strings.Join(is.Labels, ", "))
	}
	sources := "none named yet"
	if len(st.Sources) > 0 {
		sources = strings.Join(st.Sources, ", ")
	}
	fmt.Fprintf(b, "Sources: %s. Confirmed at: %s.\n", sources, st.Confirmed)
	if len(files) > 0 {
		fmt.Fprintf(b, "Code it names, given below: %s.\n", strings.Join(files, ", "))
	}
	b.WriteString("\n")
	fmt.Fprintf(b, "%s\n\n", clip(is.Body, bodyMax))
	var kept []string
	for _, c := range comments {
		if !strings.Contains(c, "<!-- workline:") {
			kept = append(kept, c)
		}
	}
	if len(kept) > commentsMax {
		fmt.Fprintf(b, "(%d earlier comments left out)\n\n", len(kept)-commentsMax)
		kept = kept[len(kept)-commentsMax:]
	}
	for _, c := range kept {
		fmt.Fprintf(b, "Comment:\n> %s\n\n", strings.ReplaceAll(clip(c, bodyMax), "\n", "\n> "))
	}
}

// trackedFiles lists the files of the commit the run is on.
func trackedFiles(repo string) map[string]bool {
	out, _ := exec.Command("git", "-C", repo, "ls-files").Output()
	files := map[string]bool{}
	for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		files[f] = true
	}
	return files
}

var pathLike = regexp.MustCompile(`[\w.-]+(?:/[\w.-]+)+|[\w-]+\.[A-Za-z]{1,5}\b`)

// named lists the files an issue is about: its sources, then the paths its
// title and body name that the commit holds, at most filesPerIssue.
func named(is forge.Issue, st *backlog.State, tracked map[string]bool) []string {
	var out []string
	add := func(p string) {
		p, _, _ = strings.Cut(p, "#")
		p = strings.Trim(p, "./`'\"")
		if tracked[p] && !slices.Contains(out, p) && len(out) < filesPerIssue {
			out = append(out, p)
		}
	}
	for _, s := range st.Sources {
		add(s)
	}
	for _, m := range pathLike.FindAllString(is.Title+"\n"+is.Body, -1) {
		add(m)
	}
	return out
}

// writeCode gives each file named, whole up to codeLines, with its last
// commits: what changed it, so an issue the code solved can be told.
func writeCode(b *strings.Builder, repo string, files []string) {
	if len(files) == 0 {
		return
	}
	b.WriteString("# Code the issues name, as it is at this commit\n\n")
	for _, f := range files {
		data, err := exec.Command("git", "-C", repo, "show", "HEAD:"+f).Output()
		if err != nil {
			continue
		}
		lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
		more := ""
		if len(lines) > codeLines {
			more = fmt.Sprintf("\n(%d more lines left out)", len(lines)-codeLines)
			lines = lines[:codeLines]
		}
		for i := range lines {
			lines[i] = fmt.Sprintf("%4d  %s", i+1, lines[i])
		}
		log, _ := exec.Command("git", "-C", repo, "log", "-5", "--format=%h %as %s", "--", f).Output()
		fmt.Fprintf(b, "## %s\n\nLast commits:\n%s\n```\n%s\n```%s\n\n", f, strings.TrimSpace(string(log)), strings.Join(lines, "\n"), more)
	}
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + " […]"
}

// Post passes: each act is checked when the engine applies it, against the
// code and the forge as they are then.
func Post(runDir, repo string) int {
	v, err := verdict.Read(filepath.Join(runDir, "out", "verdict.yaml"))
	if err != nil {
		v = &verdict.Verdict{}
	}
	v.Status, v.Summary = verdict.Pass, "the acts are checked as they are applied"
	return write(runDir, *v)
}

func writeFindings(runDir string, findings []verdict.Finding) int {
	if len(findings) == 0 {
		return 0
	}
	return write(runDir, verdict.Verdict{Status: verdict.Pass, Findings: findings})
}

func final(runDir string, v verdict.Verdict) int {
	if code := write(runDir, v); code != 0 {
		return code
	}
	return exitNothing
}

func write(runDir string, v verdict.Verdict) int {
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
		return fail(err)
	}
	return 0
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "product-owner:", err)
	return 1
}
