// Package intent reads the proposals an agent writes and checks them against
// the closed catalogue (docs/spec/role-contract.md, "Intentions").
package intent

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Catalogue is every kind of intention the engine knows how to apply.
var Catalogue = map[string]bool{
	"commit-message": true,
	"patch":          true,
	"comment":        true,
	"label":          true,
	"issue":          true,
	"handoff":        true,
	"note":           true,
	"close":          true, // an issue closed as a duplicate or obsolete, its evidence quoted (docs/spec/backlog-acts.md)
	"claim":          true, // a part's answer (in/parts), read by pre; or why a patch takes words out, read by post; never applied
}

// Intention is one proposal: its kind and its value.
type Intention struct {
	Kind  string
	Value any
}

// Read loads out/intentions.yaml: a list of one-key maps, e.g.
// `- commit-message: "fix: ..."`. A missing file means no intentions.
func Read(path string) ([]Intention, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var raw []map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("intentions: %w", err)
	}
	var out []Intention
	for i, m := range raw {
		if len(m) != 1 {
			return nil, fmt.Errorf("intentions: entry %d must have exactly one kind", i+1)
		}
		for k, v := range m {
			out = append(out, Intention{Kind: k, Value: v})
		}
	}
	return out, nil
}

// Kinds lists the kinds of a set of intentions, in order.
func Kinds(in []Intention) []string {
	var k []string
	for _, i := range in {
		k = append(k, i.Kind)
	}
	return k
}

// Merge returns the fallback proposals with the agent's in place of those of
// the same kind (docs/spec/role-contract.md, "One run"). A patch replaces only
// the fallback patches of the files it touches: the agent fixing one doc does
// not drop what the role regenerated in another. A fallback patch given as a
// diff replaces nothing and is never replaced: it holds only the lines it
// changes, and is applied after the agent's, beside them.
func Merge(fallback, agent []Intention) []Intention {
	proposed := map[string]bool{}
	patched := map[string]bool{}
	for _, a := range agent {
		proposed[a.Kind] = true
		if a.Kind == "patch" {
			for _, f := range PatchFiles(a.Value) {
				patched[f] = true
			}
		}
	}
	var out, beside []Intention
	for _, f := range fallback {
		_, isDiff := f.Value.(string)
		switch {
		case f.Kind == "patch" && isDiff:
			beside = append(beside, f)
		case f.Kind == "patch":
			kept := true
			for _, file := range PatchFiles(f.Value) {
				kept = kept && !patched[file]
			}
			if kept {
				out = append(out, f)
			}
		case !proposed[f.Kind] || sticky(f):
			out = append(out, f) // a role's sticky proposal is its own record, not an alternative to the agent's
		}
	}
	return append(append(out, agent...), beside...)
}

// PatchFiles lists the files a patch names: its `file`, or the `+++` lines of
// its diff. It reads names only; whether the diff applies is checked later.
func PatchFiles(v any) []string {
	if m, ok := v.(map[string]any); ok {
		if f, ok := m["file"].(string); ok {
			return []string{path.Clean(f)}
		}
		return nil
	}
	diff, _ := v.(string)
	var files []string
	old := ""
	for _, l := range strings.Split(diff, "\n") {
		if p, ok := strings.CutPrefix(l, "--- "); ok {
			old, _, _ = strings.Cut(strings.TrimSpace(p), "\t")
		}
		if p, ok := strings.CutPrefix(l, "+++ "); ok {
			p, _, _ = strings.Cut(strings.TrimSpace(p), "\t")
			if p == "/dev/null" { // a deletion: the file is the one it deletes
				p = strings.TrimPrefix(old, "a/")
			}
			files = append(files, path.Clean(strings.TrimPrefix(p, "b/")))
		}
	}
	return files
}

// Write saves proposals in the same shape Read expects. No proposals, no file.
func Write(path string, in []Intention) error {
	if len(in) == 0 {
		return nil
	}
	list := make([]map[string]any, len(in))
	for i, x := range in {
		list[i] = map[string]any{x.Kind: x.Value}
	}
	data, err := yaml.Marshal(list)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// applyOrder is the order intentions are applied in, whatever order they were
// proposed in: files first, then what depends on them, then what only informs.
var applyOrder = map[string]int{
	"commit-message": 0, "patch": 1, "label": 2, "comment": 3, "close": 4, "issue": 5, "handoff": 6, "note": 7,
}

// SortForApply puts intentions in apply order, keeping the proposed order within a kind.
func SortForApply(in []Intention) {
	sort.SliceStable(in, func(i, j int) bool { return applyOrder[in[i].Kind] < applyOrder[in[j].Kind] })
}

// NormalizeDiff adds the `diff --git` line each file of a diff needs, when an
// agent left it out. git apply --recount relies on it to tell where a file
// ends: without it, a new file's `--- /dev/null` is read as a line removed
// from the file before. A file created or deleted also gets its mode line.
func NormalizeDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	var out []string
	for i, l := range lines {
		if strings.HasPrefix(l, "--- ") && i+1 < len(lines) && strings.HasPrefix(lines[i+1], "+++ ") &&
			(i == 0 || !strings.HasPrefix(lines[i-1], "diff --git ") && !strings.HasPrefix(lines[i-1], "new file mode") &&
				!strings.HasPrefix(lines[i-1], "deleted file mode") && !strings.HasPrefix(lines[i-1], "index ")) {
			from := name(strings.TrimPrefix(l, "--- "), "a/")
			to := name(strings.TrimPrefix(lines[i+1], "+++ "), "b/")
			switch {
			case from == "/dev/null":
				out = append(out, "diff --git a/"+to+" b/"+to, "new file mode 100644")
			case to == "/dev/null":
				out = append(out, "diff --git a/"+from+" b/"+from, "deleted file mode 100644")
			default:
				out = append(out, "diff --git a/"+from+" b/"+to)
			}
		}
		out = append(out, l)
	}
	return mergeOverlapping(strings.Join(out, "\n"))
}

var hunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// mergeOverlapping joins two hunks of a file that overlap on lines both give
// as the same context: agents write a change twice over that way, and git
// apply refuses it. Hunks overlapping otherwise are left for git to refuse.
func mergeOverlapping(diff string) string {
	type hunk struct {
		header             string // kept as written unless the hunk is merged
		merged             bool
		oldStart, newStart int
		lines              []string
	}
	oldLines := func(h hunk) []string {
		var o []string
		for _, l := range h.lines {
			if l == "" || l[0] == ' ' || l[0] == '-' {
				o = append(o, l)
			}
		}
		return o
	}
	emit := func(out []string, h hunk) []string {
		if !h.merged {
			return append(append(out, h.header), h.lines...)
		}
		oldN, newN := 0, 0
		for _, l := range h.lines {
			switch {
			case l == "" || l[0] == ' ':
				oldN, newN = oldN+1, newN+1
			case l[0] == '-':
				oldN++
			case l[0] == '+':
				newN++
			}
		}
		return append(append(out, fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.oldStart, oldN, h.newStart, newN)), h.lines...)
	}
	lines := strings.Split(diff, "\n")
	var out []string
	var cur *hunk
	flush := func() {
		if cur != nil {
			out = emit(out, *cur)
			cur = nil
		}
	}
	for i := 0; i < len(lines); i++ {
		m := hunkHeader.FindStringSubmatch(lines[i])
		if m == nil {
			if cur != nil && (lines[i] == "" || strings.ContainsRune(" -+\\", rune(lines[i][0]))) &&
				!(strings.HasPrefix(lines[i], "--- ") && i+1 < len(lines) && strings.HasPrefix(lines[i+1], "+++ ")) {
				cur.lines = append(cur.lines, lines[i])
				continue
			}
			flush()
			out = append(out, lines[i])
			continue
		}
		next := hunk{header: lines[i]}
		fmt.Sscan(m[1], &next.oldStart)
		fmt.Sscan(m[2], &next.newStart)
		if cur != nil {
			have := oldLines(*cur)
			overlap := cur.oldStart + len(have) - next.oldStart
			// The next hunk's own lines, up to the next header.
			j := i + 1
			for ; j < len(lines) && !hunkHeader.MatchString(lines[j]) && !strings.HasPrefix(lines[j], "diff --git ") &&
				!(strings.HasPrefix(lines[j], "--- ") && j+1 < len(lines) && strings.HasPrefix(lines[j+1], "+++ ")); j++ {
				next.lines = append(next.lines, lines[j])
			}
			if overlap > 0 && overlap <= len(next.lines) && sameContext(have[len(have)-overlap:], next.lines[:overlap]) {
				cur.lines = append(cur.lines, next.lines[overlap:]...)
				cur.merged = true
				i = j - 1
				continue
			}
			flush()
			cur = &next
			i = j - 1
			continue
		}
		cur = &next
	}
	flush()
	return strings.Join(out, "\n")
}

// sameContext says whether both runs are the same lines, all context.
func sameContext(a, b []string) bool {
	for k := range a {
		ca, cb := a[k], b[k]
		if ca == "" {
			ca = " "
		}
		if cb == "" {
			cb = " "
		}
		if ca[0] != ' ' || cb[0] != ' ' || ca != cb {
			return false
		}
	}
	return true
}

func name(header, prefix string) string {
	header, _, _ = strings.Cut(strings.TrimSpace(header), "\t")
	return strings.TrimPrefix(header, prefix)
}

// sticky says whether a proposal keeps one comment or issue in place.
func sticky(in Intention) bool {
	m, ok := in.Value.(map[string]any)
	if !ok {
		return false
	}
	_, key := m["sticky"]
	return key
}
