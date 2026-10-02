package documentalist

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// Adopting a repository (`workline init`): each doc declaring no sources is
// put before the agent, which proposes the code it describes. The doc's
// `checked` is the commit that last changed it, never HEAD: the doc was
// written against the code as it was then, and a source changed since makes
// it suspect, to be judged, rather than vouched for unread.

// treeMaxLines caps the listing of the repository given with the task.
const treeMaxLines = 400

// docShownChars caps what one doc shows of itself: its first lines tell what
// it describes, and a changelog of years would not fit the role's budget.
const docShownChars = 12000

// noSources reports each doc under the docs globs whose header does not say
// what code it describes, with the header that would track it.
func noSources(repo string, t Tree, globs []string) ([]verdict.Finding, map[string]string, error) {
	var paths []string
	for p, content := range t.Docs {
		if declared, _ := declaresSources(content); matchAny(globs, p) && !declared {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	var findings []verdict.Finding
	last := map[string]string{}
	for _, p := range paths {
		c, err := git(repo, "log", "-1", "--format=%H", "--", p)
		if err != nil {
			return nil, nil, err
		}
		if c == "" {
			continue // not committed yet: nothing to date it by
		}
		last[p] = c
		findings = append(findings, verdict.Finding{Rule: "no-sources", Where: p,
			Message: fmt.Sprintf("declares no sources, so it is never found suspect: name the code it describes in its header, `sources: [...]` with `checked: %s`, the commit that last changed it; or `sources: []` if it describes no code", c[:7])})
	}
	return findings, last, nil
}

// headerForm says where a doc's header goes: inside the one it has, else a
// comment, which no preview, forge or PDF shows, where a frontmatter would
// show on a forge and in a PDF (ADR-0007).
func headerForm(content string) string {
	if _, n := header(content); n > 0 {
		return "It has a header already (its first lines): add `sources` and `checked` inside it, keeping what is there."
	}
	return "It has no header. Add one at line 1, as a comment, which no rendering shows:\n\n```\n<!-- workline\nsources: [...]\nchecked: %s\n-->\n```"
}

// sourcesTask asks the agent to name the code each doc describes.
func sourcesTask(repo string, t Tree, last map[string]string, s Settings) (string, map[string]map[string]string, []string) {
	var b strings.Builder
	b.WriteString(`Kind: sources

These docs declare no sources, so nothing tells when the code they describe
changes. For each, name that code: the files or folders of the repository
listed below whose change could make the doc wrong, and nothing it only
mentions in passing. A doc that follows another doc names it, with a heading's
anchor when it follows one section (docs/tech/auth.md#token-refresh). Answer
with a patch adding the header shown, with context lines, citing the doc's
lines by the numbers shown; the body stays as it is. A doc describing no code
(a decision record, a backlog, a changelog) gets ` + "`sources: []`" + ` and no
` + "`checked`" + `. If you cannot tell, no patch for that doc: say why in a note.

`)
	b.WriteString("The repository's files:\n\n```\n" + listing(t) + "```\n\n")
	judged := map[string]map[string]string{}
	var left []string
	paths := make([]string, 0, len(last))
	for p := range last {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		if s.AIMaxCalls > 0 && len(judged) >= s.AIMaxCalls {
			left = append(left, p)
			continue
		}
		content := t.Docs[p]
		var entry strings.Builder
		form := headerForm(content)
		if strings.Contains(form, "%s") {
			form = fmt.Sprintf(form, last[p][:7])
		}
		fmt.Fprintf(&entry, "## %s\n\nIts `checked` is %s, the commit that last changed it. %s\n\nThe doc, with its line numbers:\n\n```\n", p, last[p][:7], form)
		lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
		shown := 0
		for i, l := range lines {
			if shown += len(l) + 8; shown > docShownChars && i > 0 {
				// What is cut still says what it covers, by its headings.
				fmt.Fprintf(&entry, "     | … %d more lines, not shown; their headings:\n", len(lines)-i)
				for _, rest := range scan(strings.Join(lines[i:], "\n")) {
					if rest.heading > 0 {
						fmt.Fprintf(&entry, "%4d | %s\n", i+rest.n, rest.text)
					}
				}
				break
			}
			fmt.Fprintf(&entry, "%4d | %s\n", i+1, l)
		}
		entry.WriteString("```\n\n")
		if len(judged) > 0 && b.Len()+entry.Len() > taskMaxChars {
			left = append(left, p)
			continue
		}
		b.WriteString(entry.String())
		judged[p] = map[string]string{"": last[p]}
	}
	if len(judged) == 0 {
		return "", nil, left
	}
	return b.String(), judged, left
}

// listing lists the repository's files, a folder at a time once they are
// too many to list one by one.
func listing(t Tree) string {
	files := make([]string, 0, len(t.Files))
	for f := range t.Files {
		files = append(files, f)
	}
	sort.Strings(files)
	if len(files) <= treeMaxLines {
		return strings.Join(files, "\n") + "\n"
	}
	count := map[string]int{}
	for _, f := range files {
		count[filepath.Dir(f)]++
	}
	dirs := make([]string, 0, len(count))
	for d := range count {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	var b strings.Builder
	for i, d := range dirs {
		if i == treeMaxLines {
			fmt.Fprintf(&b, "… and %d more folders\n", len(dirs)-i)
			break
		}
		fmt.Fprintf(&b, "%s/ (%d files)\n", d, count[d])
	}
	return b.String()
}

// judgeSources checks each patch of a sources task: on the docs given only,
// adding to the header alone, naming sources that exist, with the `checked`
// given. It returns the refusals, and the docs the patches track.
func judgeSources(repo string, s Settings, judged map[string]map[string]string, intents, fallback []intent.Intention) ([]verdict.Finding, map[string]bool, error) {
	var refused []verdict.Finding
	refuse := func(rule, where, msg string) {
		refused = append(refused, verdict.Finding{Rule: rule, Where: where, Message: msg})
	}
	patched := map[string]bool{}
	var tree Tree
	for _, in := range intents {
		if in.Kind != "patch" || isFallback(in, fallback) {
			continue
		}
		if tree.Docs == nil {
			var err error
			if tree, err = loadTree(repo, s.Docs); err != nil {
				return nil, nil, err
			}
		}
		diff, ok := in.Value.(string)
		if !ok {
			refuse("patch-not-diff", "patch", "send a unified diff, not a whole file")
			continue
		}
		if _, err := gitIn(repo, intent.NormalizeDiff(diff), "apply", "--recount", "--unidiff-zero", "--check", "-"); err != nil {
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
				refuse("patch-off-task", f.path, "only the docs put before you may be patched")
				continue
			}
			old := tree.Docs[f.path]
			f = placed(old, f)
			if why := misquoted(old, f); why != "" {
				refuse("misquoted", f.path, why)
				continue
			}
			now := applyHunks(old, f)
			if byGit, err := gitApplied(f.path, old, diff, false); err != nil || byGit != now {
				refuse("patch-ambiguous", f.path, "git would apply this diff differently from how it reads; send a plain unified diff")
				continue
			}
			// A blank line between the header and the doc is how many write it.
			if line, was, is := firstChange(body(old), strings.TrimPrefix(body(now), "\n")); line > 0 {
				refuse("body-changed", f.path, fmt.Sprintf("the patch changes the doc itself, from its body's line %d: %q becomes %q; add the header only", line, was, is))
				continue
			}
			declared, sources := declaresSources(now)
			if !declared {
				refuse("no-sources-declared", f.path, "the doc has no `sources` in its header after the patch")
				continue
			}
			var missing []string
			for _, src := range sources {
				name, p, anchor := splitSource(src)
				switch {
				case name != "":
					missing = append(missing, src+" (another repository: name only this one's files)")
				case !tree.exists(strings.TrimSuffix(p, "/")):
					missing = append(missing, src)
				case anchor != "":
					if _, ok := Section(tree.Docs[p], anchor); !ok {
						missing = append(missing, src+" (no such heading)")
					}
				}
			}
			if len(missing) > 0 {
				refuse("source-missing", f.path, "these sources are not in the repository: "+strings.Join(missing, ", "))
				continue
			}
			if len(sources) > 0 && !checkedMatches(now, want) {
				refuse("checked-wrong", f.path, fmt.Sprintf("`checked` must be %s, the commit that last changed the doc", wanted(want)))
				continue
			}
			patched[f.path] = true
		}
	}
	return refused, patched, nil
}

// adoption is the part of Pre run on `init`: the docs declaring no sources
// are reported, and put before the agent.
func adoption(runDir, repo string, t Tree, s Settings) ([]verdict.Finding, string, map[string]map[string]string, error) {
	findings, last, err := noSources(repo, t, s.Docs)
	if err != nil {
		return nil, "", nil, err
	}
	task, judged, left := sourcesTask(repo, t, last, s)
	if task != "" {
		if err := os.WriteFile(filepath.Join(runDir, "in", "task-kind"), []byte("sources\n"), 0o644); err != nil {
			return nil, "", nil, err
		}
	}
	if len(left) > 0 && task != "" { // the engine runs another round, once this one is applied
		var b strings.Builder
		for _, p := range left {
			fmt.Fprintf(&b, "no-sources %s\n", p)
		}
		if err := os.WriteFile(filepath.Join(runDir, "in", "more"), []byte(b.String()), 0o644); err != nil {
			return nil, "", nil, err
		}
	}
	return findings, task, judged, nil
}

// firstChange returns the first line where two texts differ, numbered from 1,
// and that line in each; 0 when they are the same.
func firstChange(a, b string) (int, string, string) {
	if a == b {
		return 0, "", ""
	}
	al, bl := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := 0; ; i++ {
		var x, y string
		if i < len(al) {
			x = al[i]
		}
		if i < len(bl) {
			y = bl[i]
		}
		if x != y || i >= len(al) || i >= len(bl) {
			return i + 1, x, y
		}
	}
}
