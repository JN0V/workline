package reviewer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
)

// Record is what the reviewer remembers between runs (ADR-0013's way): the
// commits a review answered whole, so they are not asked again; the changes
// they made, by git's patch id, so a commit rebased with its change intact
// is not asked again either (#286); and how many reviews ran, which turns
// the lens a push gets; and the findings of the last review that block,
// which stand until a commit answers them (#326); and whether every lens
// read the change since it was last a draft (#359). On a merge request it
// is hidden in the summary comment; on a machine, in the git directory.
type Record struct {
	Reviewed []string `json:"reviewed"`
	Patches  []string `json:"patches,omitempty"`
	Runs     int      `json:"runs"`
	Owed     bool     `json:"owed,omitempty"`     // every lens owed: the run on ready, a rule holding it, asked none (#129)
	ReadAll  bool     `json:"read-all,omitempty"` // every lens asked on the whole change, since the merge request was last a draft (#359)
	Blocks   []Block  `json:"blocks,omitempty"`   // the findings that block, carried to the next run (#326)
}

// Block is a lens's finding that blocked a review: it stands, run after
// run, while its cause is found again on the change and no review of its
// lens has read it again without finding it (#326).
type Block struct {
	Lens  string `json:"lens"`
	At    string `json:"at,omitempty"` // the head of the review that found it
	Where string `json:"where"`
	Title string `json:"title"` // its severity and title, as its row reads beside its comment on the line (#277)
	Cause Quote  `json:"cause"`
}

// SummaryKey keys the reviewer's one comment on a merge request.
const SummaryKey = "summary"

// recordLine reads a record; one an engine wrote before patch ids holds no
// `patches=`, nor `read=all` before #359, and still reads. blocksLine reads
// its blocks, a line of their own an engine before them leaves.
var (
	recordLine = regexp.MustCompile(`<!-- workline:record reviewed=([0-9a-f,]*)(?: patches=([0-9a-f,]*))? runs=(\d+)( owed=all)?( read=all)? -->`)
	blocksLine = regexp.MustCompile(`<!-- workline:blocks (.*?) -->`)
)

// The openings of the record and of its blocks: a text holding one that
// does not read holds a record damaged, never none.
const (
	recordOpening = "<!-- workline:record"
	blocksOpening = "<!-- workline:blocks"
)

// String is the record as the summary comment and the local file hold it;
// with no patch id, as an engine before them wrote it. The blocks follow,
// as JSON on one line: a dash written \u002d, and `<`, `>` escaped by the
// JSON library, so nothing in a finding ends the comment hiding them.
func (r Record) String() string {
	patches := ""
	if len(r.Patches) > 0 {
		patches = " patches=" + strings.Join(r.Patches, ",")
	}
	owed := ""
	if r.Owed {
		owed = " owed=all"
	}
	if r.ReadAll {
		owed += " read=all"
	}
	s := fmt.Sprintf("<!-- workline:record reviewed=%s%s runs=%d%s -->", strings.Join(r.Reviewed, ","), patches, r.Runs, owed)
	if len(r.Blocks) > 0 {
		data, _ := json.Marshal(r.Blocks)
		s += "\n<!-- workline:blocks " + strings.ReplaceAll(string(data), "-", `\u002d`) + " -->"
	}
	return s
}

// parseRecord reads a record from a text holding one; an empty one when
// none. A record or blocks that do not read are an error (#302): never
// read as none.
func parseRecord(text string) (Record, error) {
	m := recordLine.FindStringSubmatch(text)
	if m == nil {
		if strings.Contains(text, recordOpening) || strings.Contains(text, blocksOpening) {
			return Record{}, backlog.UnreadRecord(text, recordOpening)
		}
		return Record{}, nil
	}
	var r Record
	r.Reviewed, r.Patches = ids(m[1]), ids(m[2])
	r.Runs, _ = strconv.Atoi(m[3])
	r.Owed = m[4] != ""
	r.ReadAll = m[5] != ""
	if b := blocksLine.FindStringSubmatch(text); b != nil {
		if err := json.Unmarshal([]byte(b[1]), &r.Blocks); err != nil {
			return Record{}, fmt.Errorf("the findings that blocked do not read: %v", err)
		}
	}
	return r, nil
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
// each, and one more run; the blocks kept, and whether every lens read the
// change.
func (r Record) add(commits, patches []string) Record {
	out := Record{Reviewed: slices.Clone(r.Reviewed), Patches: slices.Clone(r.Patches), Runs: r.Runs + 1, Blocks: slices.Clone(r.Blocks), ReadAll: r.ReadAll}
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

// standing are the blocks of the last review that still stand at the head
// (#326), each at its line there: its lens still blocks (`ai-findings`),
// and its cause is found again on the change, as a lens's is. A cause no
// longer found is a commit's answer. One quoted from an issue stands
// until its lens reads the issue again.
func standing(repo string, s Settings, st state, blocks []Block) []Block {
	var out []Block
	for _, b := range blocks {
		if !s.AIFindings.Blocks(b.Lens) {
			continue
		}
		if !strings.HasPrefix(b.Cause.Path, "#") {
			f, why := found(repo, st, b.Lens, false, map[string]any{"severity": "important", "title": b.Lens, "cause": b.Cause})
			if why != "" || !f.Related {
				continue
			}
			b.Where = f.Where
		}
		out = append(out, b)
	}
	return out
}

// reread leaves out the standing blocks a review read again (#326): a
// whole one, their lens answering, given the cause's file among those the
// new commits change, or its issue. Found again and blocking, it is the
// review's own finding; found again, its judge not saying yes, it still
// stands; not found, or its judge saying no, it is answered.
func reread(repo string, s Settings, st state, c candidates, related []Finding, whole bool) ([]Block, error) {
	if len(st.Standing) == 0 {
		return nil, nil
	}
	var files []string
	if whole {
		var err error
		if files, err = codeFiles(repo, s, st.lensFrom(), st.Head); err != nil {
			return nil, err
		}
	}
	answered := slices.DeleteFunc(slices.Clone(facets(st)), func(l string) bool { return slices.Contains(c.Failed, l) })
	return slices.DeleteFunc(slices.Clone(st.Standing), func(b Block) bool {
		for _, g := range related {
			for _, f := range members(g) {
				if f.Lens == b.Lens && f.Where == b.Where {
					return s.blocking(f)
				}
			}
		}
		if !whole || !slices.Contains(answered, b.Lens) {
			return false
		}
		if strings.HasPrefix(b.Cause.Path, "#") {
			return issueOf(st, b.Cause.Path) != nil
		}
		return slices.Contains(files, b.Cause.Path)
	}), nil
}

// span is what since finds of a range.
type span struct {
	From    string   // the newest commit reviewed, with all it holds: the base when none is, the head when every one is
	Commits []string // the range's, newest first
	New     int      // how many of them are not reviewed: a merge's line may lie between those that are
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
		if !reviewed[c] {
			sp.New++
		}
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
// where it read it, and why it could not, when it could not. The summary
// comment and the machine's file always hold a record: one holding none
// was damaged, said as one that does not read (#302).
func loadRecord(repo string, s Settings) (Record, string, error) {
	if t := mergeRequest(); t != nil && s.ForgeWrites {
		where := fmt.Sprintf("in its summary comment on merge request %d", t.ID)
		f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
		if err != nil {
			return Record{}, where, fmt.Errorf("the forge: %v", err)
		}
		if f == nil {
			return Record{}, where, fmt.Errorf("no forge to read the merge request's comments from")
		}
		b, ok := f.(forge.Backlog)
		if !ok {
			return Record{}, where, fmt.Errorf("this forge cannot list a merge request's comments")
		}
		comments, err := b.Comments(*t)
		if err != nil {
			return Record{}, where, err
		}
		for _, c := range comments {
			if strings.Contains(c, forge.Marker("sticky=reviewer/"+SummaryKey)) {
				r, err := readRecord(c)
				return r, where, err
			}
		}
		return Record{}, where, nil
	}
	file := localRecord(repo)
	where := "in " + file
	if rel, err := filepath.Rel(repo, file); err == nil && !strings.HasPrefix(rel, "..") {
		where = "in " + rel
	}
	data, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return Record{}, where, nil
	}
	if err != nil {
		return Record{}, where, err
	}
	r, err := readRecord(string(data))
	return r, where, err
}

// readRecord reads the record of a text that always holds one.
func readRecord(text string) (Record, error) {
	if !strings.Contains(text, recordOpening) {
		return Record{}, backlog.UnreadRecord(text, recordOpening)
	}
	return parseRecord(text)
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
