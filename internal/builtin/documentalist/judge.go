package documentalist

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// fileDiff is one file of a unified diff.
type fileDiff struct {
	path  string
	hunks []hunk
}

// hunk is one hunk; each line keeps its ' ', '-' or '+' prefix.
type hunk struct {
	oldStart, oldCount int
	lines              []string
}

var hunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// parseDiff reads a unified diff as its lines read: a hunk runs to the next
// header, whatever counts its own header announces — agents often get them
// wrong, and the engine applies with git apply --recount, which reads it the
// same way.
func parseDiff(diff string) ([]fileDiff, error) {
	lines := strings.Split(strings.TrimSuffix(diff, "\n"), "\n")
	var out []fileDiff
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		switch {
		case strings.HasPrefix(l, "+++ "):
			p := strings.TrimSpace(strings.TrimPrefix(l, "+++ "))
			p, _, _ = strings.Cut(p, "\t")
			out = append(out, fileDiff{path: strings.TrimPrefix(p, "b/")})
		case strings.HasPrefix(l, "@@"):
			m := hunkHeader.FindStringSubmatch(l)
			if m == nil || len(out) == 0 {
				return nil, fmt.Errorf("line %d: a hunk header must follow a file header, as `@@ -a,b +c,d @@`", i+1)
			}
			h := hunk{oldStart: atoi(m[1])}
			for ; i+1 < len(lines) && !diffHeader(lines, i+1); i++ {
				hl := lines[i+1]
				if strings.HasPrefix(hl, `\`) { // "\ No newline at end of file"
					continue
				}
				if hl == "" {
					hl = " " // a blank context line whose space was trimmed
				}
				switch hl[0] {
				case ' ', '-':
					h.oldCount++
				case '+':
				default:
					return nil, fmt.Errorf("the hunk at line %d holds a line starting with neither ' ', '-' nor '+': %q", h.oldStart, hl)
				}
				h.lines = append(h.lines, hl)
			}
			if h.oldCount == 0 && h.oldStart > 0 && m[2] != "0" {
				return nil, fmt.Errorf("the hunk at line %d quotes no line of the doc; give context lines", h.oldStart)
			}
			out[len(out)-1].hunks = append(out[len(out)-1].hunks, h)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no file in the diff")
	}
	return out, nil
}

// diffHeader says whether lines[i] starts a hunk or a file.
func diffHeader(lines []string, i int) bool {
	l := lines[i]
	return strings.HasPrefix(l, "@@") || strings.HasPrefix(l, "diff ") || strings.HasPrefix(l, "index ") ||
		strings.HasPrefix(l, "+++ ") || strings.HasPrefix(l, "--- ") && i+1 < len(lines) && strings.HasPrefix(lines[i+1], "+++ ")
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

// misquoted checks that each hunk's context and removed lines are exactly the
// doc's lines at the numbers the hunk cites (Delfini's check on quotes). git
// apply alone would look for them elsewhere in the file and apply anyway.
func misquoted(old string, f fileDiff) string {
	lines := strings.Split(strings.TrimSuffix(old, "\n"), "\n")
	for _, h := range f.hunks {
		n := h.oldStart
		if h.oldCount == 0 {
			continue // a pure insertion quotes nothing
		}
		for _, l := range h.lines {
			if l[0] == '+' {
				continue
			}
			if n < 1 || n > len(lines) {
				return fmt.Sprintf("the hunk at line %d quotes line %d, but the doc has %d lines", h.oldStart, n, len(lines))
			}
			if lines[n-1] != l[1:] {
				return fmt.Sprintf("the hunk at line %d quotes %q as line %d, but line %d reads %q; cite the lines as numbered in the task", h.oldStart, l[1:], n, n, lines[n-1])
			}
			n++
		}
	}
	return ""
}

// applyHunks returns old with the hunks applied. The hunks were checked by
// misquoted first, so they sit where they say.
func applyHunks(old string, f fileDiff) string {
	lines := strings.Split(strings.TrimSuffix(old, "\n"), "\n")
	var out []string
	next := 0 // index of the first old line not copied yet
	for _, h := range f.hunks {
		start := h.oldStart - 1
		if h.oldCount == 0 {
			start = h.oldStart // inserted after that line
		}
		out = append(out, lines[next:start]...)
		next = start
		for _, l := range h.lines {
			switch l[0] {
			case ' ':
				out = append(out, lines[next])
				next++
			case '-':
				next++
			case '+':
				out = append(out, l[1:])
			}
		}
	}
	out = append(out, lines[next:]...)
	return strings.Join(out, "\n") + "\n"
}

// touchesDerived says whether a hunk changes a line between workline:derive
// markers: those lines are regenerated from the code, never written.
func touchesDerived(old string, f fileDiff) bool {
	lines := strings.Split(old, "\n")
	inside := make([]bool, len(lines)+2)
	in := false
	for i, l := range lines {
		if strings.Contains(l, "<!-- workline:derive") {
			in = true
		}
		inside[i+1] = in
		if strings.Contains(l, "<!-- workline:end") {
			in = false
		}
	}
	for _, h := range f.hunks {
		n := h.oldStart
		for _, l := range h.lines {
			switch l[0] {
			case ' ':
				n++
			case '-':
				if n < len(inside) && inside[n] {
					return true
				}
				n++
			case '+':
				if n > 1 && n-1 < len(inside) && inside[n-1] && n < len(inside) && inside[n] {
					return true // inserted between two lines of a derived block
				}
			}
		}
	}
	return false
}

// checkedMatches says whether a doc's `checked`, as patched, names the commits
// it was judged against: each one given at least by its 7-character prefix.
func checkedMatches(content string, want map[string]string) bool {
	d, err := ParseDoc("", []byte(content))
	if err != nil || d == nil {
		return false
	}
	for repo, full := range want {
		got := d.Checked[repo]
		if len(got) < 7 || !strings.HasPrefix(full, got) {
			return false
		}
	}
	return true
}

// judgePatches checks each proposed patch: a diff the engine can apply, on the
// docs put before the agent only, quoting them at the lines it cites, leaving
// derived blocks alone, not growing the doc, recording the commits the doc was
// judged against, and bringing no new budget, link or duplicate problem. It
// returns the refusals, and the docs the patches handle.
func judgePatches(repo string, s Settings, judged map[string]map[string]string, intents []intent.Intention, fallback []intent.Intention) ([]verdict.Finding, map[string]bool, error) {
	var refused []verdict.Finding
	refuse := func(rule, where, msg string) {
		refused = append(refused, verdict.Finding{Rule: rule, Where: where, Message: msg})
	}
	patched := map[string]bool{}
	var tree Tree
	after := map[string]string{}
	for _, in := range intents {
		if in.Kind != "patch" || isFallback(in, fallback) {
			continue // the role's own regenerated blocks are not the agent's to judge
		}
		if tree.Docs == nil {
			var err error
			if tree, err = loadTree(repo, s.Docs); err != nil {
				return nil, nil, err
			}
			for p, c := range tree.Docs {
				after[p] = c
			}
		}
		diff, ok := in.Value.(string)
		if !ok {
			refuse("patch-not-diff", "patch", "send a unified diff, not a whole file, so what it replaces can be checked against the doc")
			continue
		}
		if _, err := gitIn(repo, diff, "apply", "--recount", "--check", "-"); err != nil {
			refuse("patch-does-not-apply", "patch", err.Error())
			continue
		}
		files, err := parseDiff(diff)
		if err != nil {
			refuse("patch-unreadable", "patch", err.Error())
			continue
		}
		for _, f := range files {
			want, ok := judged[f.path]
			if !ok {
				refuse("patch-off-task", f.path, "only the docs put before you may be patched; anything else belongs in an issue or a note")
				continue
			}
			old := after[f.path]
			if why := misquoted(old, f); why != "" {
				refuse("misquoted", f.path, why)
				continue
			}
			if touchesDerived(old, f) {
				refuse("derived-block", f.path, "the patch changes lines between workline:derive markers; they are regenerated from the code, never written")
				continue
			}
			now := applyHunks(old, f)
			if byGit, err := gitApplied(f.path, old, diff); err != nil || byGit != now {
				refuse("patch-ambiguous", f.path, "git would apply this diff differently from how it reads; send a plain unified diff")
				continue
			}
			// The frontmatter records who checked the doc; only the body counts.
			if n, o := len(scan(now)), len(scan(old)); n > o {
				refuse("patch-grows", f.path, fmt.Sprintf("the patch makes the doc %d lines longer; fix what is wrong without adding to it", n-o))
				continue
			}
			if !checkedMatches(now, want) {
				refuse("still-suspect", f.path, fmt.Sprintf("the patch does not set `checked` to %s, the commit given in the task, so the doc would stay suspect", wanted(want)))
				continue
			}
			after[f.path] = now
			patched[f.path] = true
		}
	}
	if len(refused) > 0 || len(patched) == 0 {
		return refused, patched, nil
	}
	before := map[string]Problem{}
	for _, p := range Hygiene(tree, s.Budgets, s.Duplicates) {
		before[p.Key] = p
	}
	for _, p := range Hygiene(Tree{Docs: after, Files: tree.Files}, s.Budgets, s.Duplicates) {
		if old, ok := before[p.Key]; !ok || p.Size > old.Size {
			refuse("patch-introduces", p.Where, p.Rule+": "+p.Message)
		}
	}
	return refused, patched, nil
}

// gitApplied returns what git apply makes of one file of a diff, so the judge
// judges exactly what the engine will apply.
func gitApplied(path, old, diff string) (string, error) {
	dir, err := os.MkdirTemp("", "workline-judge-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(file, []byte(old), 0o644); err != nil {
		return "", err
	}
	if _, err := gitIn(dir, diff, "apply", "--recount", "--include="+path, "-"); err != nil {
		return "", err
	}
	data, err := os.ReadFile(file)
	return string(data), err
}

// wanted names the commits a doc's `checked` must hold, as the task gave them.
func wanted(want map[string]string) string {
	var parts []string
	for repo, full := range want {
		if repo == "" {
			parts = append(parts, full[:7])
		} else {
			parts = append(parts, repo+": "+full[:7])
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

// isFallback says whether a patch is one pre proposed itself, unchanged.
func isFallback(in intent.Intention, fallback []intent.Intention) bool {
	for _, f := range fallback {
		if f.Kind == in.Kind && fmt.Sprint(f.Value) == fmt.Sprint(in.Value) {
			return true
		}
	}
	return false
}
