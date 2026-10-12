package reviewer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// Record is what the reviewer remembers between runs (ADR-0013's way): the
// commits a review answered whole, so they are not asked again; the changes
// they made, by git's patch id, so a commit rebased with its change intact
// is not asked again either (#286); and how many reviews ran, which turns
// the lens a push gets. On a merge request it is hidden in the summary
// comment; on a machine, in the git directory.
type Record struct {
	Reviewed []string `json:"reviewed"`
	Patches  []string `json:"patches,omitempty"`
	Runs     int      `json:"runs"`
}

// SummaryKey keys the reviewer's one comment on a merge request.
const SummaryKey = "summary"

// recordLine reads a record; one an engine wrote before patch ids holds no
// `patches=`, and still reads.
var recordLine = regexp.MustCompile(`<!-- workline:record reviewed=([0-9a-f,]*)(?: patches=([0-9a-f,]*))? runs=(\d+) -->`)

// String is the record as the summary comment and the local file hold it;
// with no patch id, as an engine before them wrote it.
func (r Record) String() string {
	patches := ""
	if len(r.Patches) > 0 {
		patches = " patches=" + strings.Join(r.Patches, ",")
	}
	return fmt.Sprintf("<!-- workline:record reviewed=%s%s runs=%d -->", strings.Join(r.Reviewed, ","), patches, r.Runs)
}

// parseRecord reads a record from a text holding one; an empty one when none.
func parseRecord(text string) Record {
	m := recordLine.FindStringSubmatch(text)
	if m == nil {
		return Record{}
	}
	var r Record
	r.Reviewed, r.Patches = ids(m[1]), ids(m[2])
	r.Runs, _ = strconv.Atoi(m[3])
	return r
}

// ids are the ids of a list a comma separates, none empty.
func ids(list string) []string {
	var out []string
	for _, c := range strings.Split(list, ",") {
		if c != "" {
			out = append(out, c)
		}
	}
	return out
}

// add records commits as reviewed, and the patch ids of their changes, once
// each, and one more run.
func (r Record) add(commits, patches []string) Record {
	out := Record{Reviewed: slices.Clone(r.Reviewed), Patches: slices.Clone(r.Patches), Runs: r.Runs + 1}
	for _, c := range commits {
		if !slices.Contains(out.Reviewed, c) {
			out.Reviewed = append(out.Reviewed, c)
		}
	}
	for _, p := range patches {
		if !slices.Contains(out.Patches, p) {
			out.Patches = append(out.Patches, p)
		}
	}
	return out
}

// span is what since finds of a range.
type span struct {
	From    string   // the newest commit reviewed, with all it holds: the base when none is, the head when every one is
	Commits []string // the range's, newest first
	Patches []string // the patch ids of their changes
	Unread  error    // why git gave no patch ids, when it gave none: commit ids alone tell
}

// since reads the commits base..head and the patch ids of their changes,
// and finds the newest commit reviewed. A commit is reviewed when the
// record holds it, or a commit holding it; or when the record holds its
// change and its parents in the range are reviewed: a rebase gives a
// commit a new id, not a new change (#286). One whose change a conflict
// altered is not, nor are those after it. A merge has no patch id: only
// its id tells.
func (r Record) since(repo, base, head string) (span, error) {
	out, err := git(repo, "rev-list", "--topo-order", "--parents", base+".."+head)
	if err != nil {
		return span{}, err
	}
	var sp span
	parents := map[string][]string{}
	var single []string // the commits with one parent: those a patch id names
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		f := strings.Fields(l)
		if len(f) == 0 {
			continue
		}
		sp.Commits = append(sp.Commits, f[0]) // each before its parents
		parents[f[0]] = f[1:]
		if len(f) == 2 {
			single = append(single, f[0])
		}
	}
	patch, err := patchIDs(repo, single)
	if err != nil {
		sp.Unread, patch = err, nil
	}
	held := map[string]bool{} // recorded, or held by a commit recorded
	for _, c := range sp.Commits {
		if held[c] || slices.Contains(r.Reviewed, c) {
			held[c] = true
			for _, p := range parents[c] {
				held[p] = true
			}
		}
	}
	reviewed := map[string]bool{}
	for _, c := range slices.Backward(sp.Commits) { // parents first
		p, ok := patch[c]
		reviewed[c] = held[c] || ok && slices.Contains(r.Patches, p) && !slices.ContainsFunc(parents[c], func(q string) bool {
			_, inRange := parents[q]
			return inRange && !reviewed[q]
		})
	}
	sp.From = base
	for _, c := range sp.Commits {
		if reviewed[c] {
			sp.From = c
			break
		}
	}
	for _, c := range sp.Commits {
		if p, ok := patch[c]; ok {
			sp.Patches = append(sp.Patches, p)
		}
	}
	return sp, nil
}

// patchIDs are the patch ids of commits, none a merge, by commit: git's id
// of a change, the same whatever commit it sits on (`git patch-id
// --verbatim`: the stable algorithm, whitespace kept, since an indent can
// be code), read from `git diff-tree`, plumbing no diff setting of the
// machine alters. A commit changing nothing has none.
func patchIDs(repo string, commits []string) (map[string]string, error) {
	ids := map[string]string{}
	if len(commits) == 0 {
		return ids, nil
	}
	diff, err := gitIn(repo, strings.Join(commits, "\n")+"\n", "diff-tree", "--stdin", "-p", "--no-color")
	if err != nil {
		return nil, err
	}
	out, err := gitIn(repo, diff, "patch-id", "--verbatim")
	if err != nil {
		return nil, err
	}
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if id, commit, ok := strings.Cut(l, " "); ok {
			ids[commit] = id
		}
	}
	return ids, nil
}

// localRecord is where a machine keeps the record: in the git directory,
// never tracked.
func localRecord(repo string) string {
	dir, err := git(repo, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return filepath.Join(repo, ".git", "workline", "reviewer-record")
	}
	return filepath.Join(strings.TrimSpace(dir), "workline", "reviewer-record")
}

// loadRecord reads the record: from the merge request's summary comment when
// the run is on one and may write there, else from the machine's. It says
// where it read it, and why it could not, when it could not.
func loadRecord(repo string, s Settings) (Record, string, error) {
	if t := mergeRequest(); t != nil && s.ForgeWrites {
		f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
		if err != nil {
			return Record{}, "forge", fmt.Errorf("the forge: %v", err)
		}
		if f == nil {
			return Record{}, "forge", fmt.Errorf("no forge to read the merge request's comments from")
		}
		b, ok := f.(forge.Backlog)
		if !ok {
			return Record{}, "forge", fmt.Errorf("this forge cannot list a merge request's comments")
		}
		comments, err := b.Comments(*t)
		if err != nil {
			return Record{}, "forge", err
		}
		for _, c := range comments {
			if strings.Contains(c, forge.Marker("sticky=reviewer/"+SummaryKey)) {
				return parseRecord(c), "forge", nil
			}
		}
		return Record{}, "forge", nil
	}
	data, err := os.ReadFile(localRecord(repo))
	if os.IsNotExist(err) {
		return Record{}, "local", nil
	}
	return parseRecord(string(data)), "local", err
}

// saveLocal keeps the record on this machine.
func saveLocal(repo string, r Record) error {
	file := localRecord(repo)
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return os.WriteFile(file, []byte(r.String()+"\n"), 0o644)
}

// mergeRequest is the merge request the run is on, when a forge and a
// target are given and the target is one.
func mergeRequest() *forge.Target {
	if os.Getenv("WORKLINE_FORGE") == "" {
		return nil
	}
	kind, id, ok := strings.Cut(os.Getenv("WORKLINE_TARGET"), ":")
	n, err := strconv.Atoi(id)
	if !ok || err != nil || kind != "merge-request" {
		return nil
	}
	return &forge.Target{Kind: kind, ID: n}
}
