package engine

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/verdict"
	"github.com/JN0V/workline/internal/role"
)

// The caps of what the shared pre reads for a role's agent, in bytes: a
// text past its cap is cut, and the cut is said (in/cuts).
const (
	textCap = 20000 // an issue's or merge request's text, one comment
	diffCap = 60000 // a diff
	logCap  = 20000 // the messages of a range

	commentsCap = 60000 // all the comments of a target together
)

// capText cuts text to cap bytes, on a line, and says the cut.
func capText(name, text string, limit int, cuts *[]string) string {
	if len(text) <= limit {
		return text
	}
	cut := text[:limit]
	if i := strings.LastIndex(cut, "\n"); i > 0 {
		cut = cut[:i+1]
	} else {
		for len(cut) > 0 && !utf8.RuneStart(text[len(cut)]) {
			cut = cut[:len(cut)-1]
		}
	}
	*cuts = append(*cuts, fmt.Sprintf("%s cut from %d to %d bytes, the inputs' cap", name, len(text), len(cut)))
	return cut + "\n[cut]\n"
}

// readInputs is what the role's `inputs` give its agent in its task, and the
// cuts it made. An input the forge or git does not give is an error naming it.
func readInputs(r *role.Role, env []string) (string, []string, error) {
	var out strings.Builder
	var cuts []string
	repo := envValue(env, "WORKLINE_REPO")
	rng := envValue(env, "WORKLINE_RANGE")
	diffOf := ""  // the range whose diff is already in the input
	logRng := rng // the range of the commit list: two dots when derived, the merge request's own commits
	if slices.Contains(r.Inputs, "target") {
		text, base, branch, err := readTarget(env, &cuts)
		if err != nil {
			return "", nil, fmt.Errorf("input target not read: %w", err)
		}
		out.WriteString(text)
		if base != "" && rng == "" {
			ref := mergeRequestBranch(repo, branch)
			rng = base + "..." + ref
			logRng = base + ".." + ref
		}
		if base != "" {
			d, err := rangeDiff(repo, rng, &cuts)
			if err != nil {
				return "", nil, fmt.Errorf("input target not read: %w", err)
			}
			out.WriteString("\n## Diff\n\n```diff\n" + d + "\n```\n")
			diffOf = rng
		}
	}
	if slices.Contains(r.Inputs, "commits") {
		if rng == "" {
			return "", nil, errors.New("input commits not read: the run has no range")
		}
		msgs, err := git(repo, nil, "log", "--reverse", "--format=commit %h%n%B", logRng)
		if err != nil {
			return "", nil, fmt.Errorf("input commits not read: %w", err)
		}
		out.WriteString("\n## Commits\n\n" + capText("commit messages", msgs, logCap, &cuts) + "\n")
		if diffOf != rng { // the target's diff, when it is this one, is not given twice
			d, err := rangeDiff(repo, rng, &cuts)
			if err != nil {
				return "", nil, fmt.Errorf("input commits not read: %w", err)
			}
			out.WriteString("\n## Diff\n\n```diff\n" + d + "\n```\n")
		}
	}
	return out.String(), cuts, nil
}

// mergeRequestBranch is the ref of a merge request's branch in the checkout:
// origin/<branch> when the checkout has it (a CI checkout holds no local
// branch), else the local branch.
func mergeRequestBranch(repo, branch string) string {
	for _, ref := range []string{"origin/" + branch, branch} {
		if _, err := git(repo, nil, "rev-parse", "--verify", "--quiet", ref+"^{commit}"); err == nil {
			return ref
		}
	}
	return branch
}

func rangeDiff(repo, rng string, cuts *[]string) (string, error) {
	d, err := gitRaw(repo, "diff", "--no-color", "--no-ext-diff", rng)
	if err != nil {
		return "", err
	}
	return capText("diff", d, diffCap, cuts), nil
}

// readTarget reads the issue or merge request the run is on; base and branch
// are those of a merge request.
func readTarget(env []string, cuts *[]string) (text, base, branch string, err error) {
	spec := envValue(env, "WORKLINE_TARGET")
	if spec == "" {
		return "", "", "", errors.New("the run has no target")
	}
	var t forge.Target
	if _, err := fmt.Sscanf(strings.Replace(spec, ":", " ", 1), "%s %d", &t.Kind, &t.ID); err != nil {
		return "", "", "", fmt.Errorf("target %q: %w", spec, err)
	}
	f, err := forge.Open(envValue(env, "WORKLINE_FORGE"), envValue(env, "WORKLINE_REPO"))
	if err != nil || f == nil {
		return "", "", "", fmt.Errorf("the forge cannot be opened: %v", err)
	}
	var title, body string
	var labels []string
	if t.Kind == "merge-request" {
		mr, err := f.MergeRequest(t.ID)
		if err != nil {
			return "", "", "", err
		}
		title, body, labels, base, branch = mr.Title, mr.Body, mr.Labels, mr.Base, mr.Branch
		if base == "" {
			base = "HEAD"
		}
	} else {
		is, err := f.Issue(t.ID)
		if err != nil {
			return "", "", "", err
		}
		title, body, labels = is.Title, is.Body, is.Labels
	}
	bl, ok := f.(forge.Backlog)
	if !ok {
		return "", "", "", errors.New("the forge does not give comments")
	}
	comments, err := bl.Comments(t)
	if err != nil {
		return "", "", "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\n## %s\n\n# %s\n\n%s\n\nLabels: %s\n", t, title, capText("target text", body, textCap, cuts), strings.Join(labels, ", "))
	// The newest comments are kept, up to the total budget; the older ones
	// left out are said.
	start, used := len(comments), 0
	for start > 0 {
		n := min(len(comments[start-1]), textCap)
		if used+n > commentsCap {
			break
		}
		used += n
		start--
	}
	if start > 0 {
		*cuts = append(*cuts, fmt.Sprintf("target comments: the %d oldest of %d left out, the inputs' cap of %d bytes", start, len(comments), commentsCap))
		fmt.Fprintf(&b, "\n[%d older comments left out]\n", start)
	}
	for _, c := range comments[start:] {
		b.WriteString("\n### Comment\n\n" + capText("target comment", c, textCap, cuts) + "\n")
	}
	return b.String(), base, branch, nil
}

// sayInputCuts records the cuts the inputs made in the run's in/cuts.
func sayInputCuts(runDir string, cuts []string) error {
	for _, c := range cuts {
		if err := verdict.SayCut(runDir, c); err != nil {
			return err
		}
	}
	return nil
}
