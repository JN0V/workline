package tester

// A test file the base holds is shared (#1259): where a test lives is the
// tester's judgement, so it may add its tests to a file already there, as
// a person keeping one test file per class would. What converges stays
// hard: its patch only adds lines, never changes nor removes one of the
// base's; and what it froze is its own lines, kept whole, whatever is
// later added beside them.

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/intent"
)

// lines is a file's content as its lines, its last newline not a line.
func lines(content string) []string {
	if content == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(content, "\n"), "\n")
}

// keptIn says whether each of base's lines is in content, in order, so
// that content only adds lines to base; when not, the first base line
// that content changes or removes, and its number.
func keptIn(base, content []string) (int, string, bool) {
	i := 0
	for _, l := range content {
		if i < len(base) && l == base[i] {
			i++
		}
	}
	if i == len(base) {
		return 0, "", true
	}
	return i + 1, base[i], false
}

// baseLineChanged is the first line of file, as the base holds it, that
// the patch p changes or removes; ok when p only adds lines, or when the
// base lacks file. A whole file is read against the base's; a diff by its
// removed lines, a changed line being one removed and one added.
func baseLineChanged(repo, base, file string, p intent.Intention) (string, bool) {
	old, err := git(repo, "show", base+":"+file)
	if err != nil {
		return "", true // a new file: nothing of the base's to keep
	}
	if m, ok := p.Value.(map[string]any); ok {
		content, _ := m["content"].(string)
		n, l, kept := keptIn(lines(old), lines(content))
		if kept {
			return "", true
		}
		return fmt.Sprintf("line %d, `%s`", n, l), false
	}
	diff, _ := p.Value.(string)
	ls := strings.Split(intent.NormalizeDiff(diff), "\n")
	in := false // in file's part of the diff, as it names it before or after
	for i, l := range ls {
		if old, ok := strings.CutPrefix(l, "--- "); ok && i+1 < len(ls) && strings.HasPrefix(ls[i+1], "+++ ") {
			name := func(s string) string { s, _, _ = strings.Cut(strings.TrimSpace(s), "\t"); return s }
			in = strings.TrimPrefix(name(old), "a/") == file || strings.TrimPrefix(name(ls[i+1][4:]), "b/") == file
			continue
		}
		if strings.HasPrefix(l, "+++ ") && i > 0 && strings.HasPrefix(ls[i-1], "--- ") {
			continue
		}
		if r, ok := strings.CutPrefix(l, "-"); ok && in {
			return "`" + r + "`", false
		}
	}
	return "", true
}

// frozenRun is a run of lines the tester added to a test file: start is
// the first one's number in the file it recorded.
type frozenRun struct {
	start int
	lines []string
}

// name is a run as a person finds it: its first line that is not blank.
func (r frozenRun) name() string {
	for i, l := range r.lines {
		if strings.TrimSpace(l) != "" {
			return fmt.Sprintf("line %d, `%s`", r.start+i, strings.TrimSpace(l))
		}
	}
	return fmt.Sprintf("line %d", r.start)
}

// frozenRuns are the runs of lines the tester added to the base's file,
// as recorded: the base's lines matched in order, first fit, the rest
// its own. A file the base lacks is one run, whole.
func frozenRuns(base, recorded []string) []frozenRun {
	var runs []frozenRun
	i := 0
	for n, l := range recorded {
		if i < len(base) && l == base[i] {
			i++
			continue
		}
		if k := len(runs) - 1; k >= 0 && runs[k].start+len(runs[k].lines) == n+1 {
			runs[k].lines = append(runs[k].lines, l)
			continue
		}
		runs = append(runs, frozenRun{start: n + 1, lines: []string{l}})
	}
	return runs
}

// missingRun is the first of runs that head does not hold whole, each
// after the one before; ok when head holds each.
func missingRun(runs []frozenRun, head []string) (frozenRun, bool) {
	at := 0
	for _, r := range runs {
		found := -1
		for i := at; i+len(r.lines) <= len(head); i++ {
			if slices.Equal(head[i:i+len(r.lines)], r.lines) {
				found = i
				break
			}
		}
		if found < 0 {
			return r, false
		}
		at = found + len(r.lines)
	}
	return frozenRun{}, true
}

// keptBeside reads file at the head against the tester's record: ok when
// the head holds each run of lines the tester wrote, whole and in order,
// whatever was added beside them; else the run it changed. read is false
// when the recorded file cannot be read: nothing is then taken as kept.
func keptBeside(repo string, r record, file string) (missing frozenRun, kept, read bool) {
	recorded, err := git(repo, "cat-file", "blob", r.Files[file])
	if err != nil {
		return frozenRun{}, false, false
	}
	head, err := git(repo, "show", "HEAD:"+file)
	if err != nil {
		return frozenRun{}, false, false
	}
	var base []string
	if old, err := git(repo, "show", r.Base+":"+file); err == nil {
		base = lines(old)
	}
	missing, kept = missingRun(frozenRuns(base, lines(recorded)), lines(head))
	return missing, kept, true
}
