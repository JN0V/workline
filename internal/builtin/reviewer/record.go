package reviewer

import (
	"encoding/json"
	"fmt"
	"math"
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
// which stand until a commit answers them (#326); whether every lens
// read the change since it was last a draft (#359); and the lenses that
// did not answer on the commits it does not hold, which no run records
// before they read them (#588); and those that left new commits past
// commits-per-push, which read them on the other run's turn (#591). On a
// merge request it is hidden in the summary comment; on a machine, in
// the git directory.
type Record struct {
	Reviewed []string `json:"reviewed"`
	Patches  []string `json:"patches,omitempty"`
	Runs     int      `json:"runs"`
	Owed     bool     `json:"owed,omitempty"`     // every lens owed: the run on ready, a rule holding it, asked none (#129)
	ReadAll  bool     `json:"read-all,omitempty"` // every lens answered on the whole change, since the merge request was last a draft (#359)
	Retried  bool     `json:"retried,omitempty"`  // the run on ready failed once: every lens owed again, once (#359)
	Failed   []string `json:"failed,omitempty"`   // the lenses that did not answer on the commits after the record: asked first, those commits theirs (#588)
	LeftBy   []string `json:"left-by,omitempty"`  // the lenses that left new commits past commits-per-push: asked on the other run's turn, never recorded unread (#591)
	Blocks   []Block  `json:"blocks,omitempty"`   // the findings that block, carried to the next run (#326)
	// Judged are the findings the judge answered, every run's, kept on a
	// merge request for the weekly sample (ADR-0046); Gone, how many of
	// them the summary comment no longer holds. Never on a machine.
	Judged []Answered `json:"judged,omitempty"`
	Gone   int        `json:"judged-gone,omitempty"`
	// comment is the side's own comment the record was read from on a
	// merge request, whole: a draft's run keeps it as it is, its record
	// alone written anew (ADR-0050). "" elsewhere, or with none yet.
	comment string
}

// Answered is a finding the judge answered: kept (yes) or dropped (no), at
// the head the review read, with the models of the lens that found it and
// of its judge. A line each in the summary comment, for the merge
// request's life: the weekly sample draws the reviewer's acts from them
// (ADR-0046). What it was judged on keys it too, so the same finding on
// the same patch is not judged again, its verdict kept (ADR-0050, step 6):
// the patch id of the merge request's change to the cause's file, and the
// finding's own key; with the independence of a yes, the reason of a no.
// A line written before them keys nothing: its finding is judged again.
type Answered struct {
	Lens    string `json:"lens"`
	Where   string `json:"where"`
	At      string `json:"at"`
	Patch   string `json:"patch,omitempty"`
	Finding string `json:"finding,omitempty"`
	Level   string `json:"level,omitempty"`
	Why     string `json:"why,omitempty"`
	Yes     bool   `json:"yes"`
	Finder  string `json:"finder"`
	Judge   string `json:"judge"`
	Title   string `json:"title"`
}

// SummaryMax is the most characters the summary comment holds: GitHub
// takes 65,536, the engine adds to the review region when it applies. The
// lines of the findings judged give way first, the oldest first.
const SummaryMax = 60000

// Block is a lens's finding that blocked a review: it stands, run after
// run, while its cause is found again on the change and no review of its
// lens has read it again without finding it (#326).
type Block struct {
	Lens  string `json:"lens"`
	At    string `json:"at,omitempty"` // the head of the review that found it
	Where string `json:"where"`
	Title string `json:"title"` // its severity and title, as its row reads beside its comment on the line (#277)
	Cause Quote  `json:"cause"`
	Side  string `json:"side,omitempty"` // where its cause was found, onHead or onRemoved: looked for there alone (#570); none in a record before it
}

// SummaryKey keys the reviewer's one comment on a merge request; WarnKey,
// the run apart's, the lenses that warn, when the review is split
// (ADR-0050): each run its own comment and record, written by its own
// applying job, which no other edits at the same time.
const (
	SummaryKey = "summary"
	WarnKey    = "warn"
)

// splitMark is hidden in the summary comment of a review split (ADR-0050):
// its record is the verdict's run's alone. One without it, written by a
// run not split, holds every lens's turn: the run apart's first record.
const splitMark = "<!-- workline:warn-apart -->"

// keyOf is the comment a run of that side keeps its summary and record in.
func keyOf(side string) string {
	if side == warnOnly {
		return WarnKey
	}
	return SummaryKey
}

// recordLine reads a record; one an engine wrote before patch ids holds no
// `patches=`, nor `read=all` and `retried=once` before #359, nor `failed=`
// before #588, nor `left-by=` before #591, and still reads. blocksLine
// reads its blocks, a line of their own an engine before them leaves.
var (
	recordLine = regexp.MustCompile(`<!-- workline:record reviewed=([0-9a-f,]*)(?: patches=([0-9a-f,]*))? runs=(\d+)( owed=all)?( read=all)?( retried=once)?(?: failed=([\w.,-]+))?(?: left-by=([\w.,-]+))? -->`)
	blocksLine = regexp.MustCompile(`<!-- workline:blocks (.*?) -->`)
	// judgedHead opens the lines of the findings judged, saying how many
	// the comment no longer holds; an engine before them wrote none.
	judgedHead = regexp.MustCompile(`<!-- workline:judged-lines gone=(\d+) -->`)
	judgedLine = regexp.MustCompile(`<!-- workline:judged (.*?) -->`)
)

// The openings of the record and of its blocks: a text holding one that
// does not read holds a record damaged, never none.
const (
	recordOpening = "<!-- workline:record"
	blocksOpening = "<!-- workline:blocks"
)

// String is the record as the summary comment and the local file hold it;
// with no patch id, as an engine before them wrote it. The blocks follow,
// on one line (hidden); the findings judged, apart (judgedLines).
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
	if r.Retried {
		owed += " retried=once"
	}
	if len(r.Failed) > 0 {
		owed += " failed=" + strings.Join(r.Failed, ",")
	}
	if len(r.LeftBy) > 0 {
		owed += " left-by=" + strings.Join(r.LeftBy, ",")
	}
	s := fmt.Sprintf("<!-- workline:record reviewed=%s%s runs=%d%s -->", strings.Join(r.Reviewed, ","), patches, r.Runs, owed)
	if len(r.Blocks) > 0 {
		s += "\n<!-- workline:blocks " + hidden(r.Blocks) + " -->"
	}
	return s
}

// hidden is a value as JSON on one line, safe in a comment hidden in
// Markdown: a dash written \u002d, and `<`, `>` escaped by the JSON
// library, so nothing in a finding ends the comment hiding it.
func hidden(v any) string {
	data, _ := json.Marshal(v)
	return strings.ReplaceAll(string(data), "-", `\u002d`)
}

// judgedLines are the lines of the findings judged as the summary comment
// holds them, within room characters: the newest kept, the oldest gone
// first, and how many went counted in the line opening them (ADR-0046).
func (r Record) judgedLines(room int) string {
	lines := make([]string, len(r.Judged))
	size := 0
	for i, j := range r.Judged {
		lines[i] = "\n<!-- workline:judged " + hidden(j) + " -->"
		size += len(lines[i])
	}
	head := func(gone int) string { return fmt.Sprintf("\n<!-- workline:judged-lines gone=%d -->", gone) }
	drop := 0
	for drop < len(lines) && len(head(r.Gone+drop))+size > room {
		size -= len(lines[drop])
		drop++
	}
	return head(r.Gone+drop) + strings.Join(lines[drop:], "")
}

// ReadJudged reads the lines of the findings judged in a summary comment:
// each line; how many the comment no longer holds — past its size, or a
// line that does not read; and whether it holds them at all: one an
// engine before them wrote does not (ADR-0046).
func ReadJudged(comment string) (judged []Answered, gone int, held bool) {
	if m := judgedHead.FindStringSubmatch(comment); m != nil {
		gone, _ = strconv.Atoi(m[1])
		held = true
	}
	for _, m := range judgedLine.FindAllStringSubmatch(comment, -1) {
		var j Answered
		if err := json.Unmarshal([]byte(m[1]), &j); err != nil || j.Lens == "" || j.Where == "" {
			gone++
			continue
		}
		judged = append(judged, j)
	}
	return judged, gone, held
}

// parseRecord reads a record from a text holding one; an empty one when
// none. A record or blocks that do not read are an error (#302): never
// read as none. The findings judged read apart, and are kept still.
func parseRecord(text string) (Record, error) {
	var r Record
	r.Judged, r.Gone, _ = ReadJudged(text)
	m := recordLine.FindStringSubmatch(text)
	if m == nil {
		if strings.Contains(text, recordOpening) || strings.Contains(text, blocksOpening) {
			return Record{Judged: r.Judged, Gone: r.Gone}, backlog.UnreadRecord(text, recordOpening)
		}
		return r, nil
	}
	r.Reviewed, r.Patches = ids(m[1]), ids(m[2])
	r.Runs, _ = strconv.Atoi(m[3])
	r.Owed = m[4] != ""
	r.ReadAll = m[5] != ""
	r.Retried = m[6] != ""
	r.Failed = ids(m[7])
	r.LeftBy = ids(m[8])
	if b := blocksLine.FindStringSubmatch(text); b != nil {
		if err := json.Unmarshal([]byte(b[1]), &r.Blocks); err != nil {
			return Record{Judged: r.Judged, Gone: r.Gone}, fmt.Errorf("the findings that blocked do not read: %v", err)
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
// each, and one more run; the blocks kept, the findings judged, and
// whether every lens read the change. No lens owes them any more, nor
// left any: a caller records only what a lens of its run read, or what
// none had to read (#588), and names who left the rest (recorded, #591).
func (r Record) add(commits, patches []string) Record {
	out := Record{Reviewed: slices.Clone(r.Reviewed), Patches: slices.Clone(r.Patches), Runs: r.Runs + 1, Blocks: slices.Clone(r.Blocks), ReadAll: r.ReadAll, Retried: r.Retried,
		Judged: slices.Clone(r.Judged), Gone: r.Gone}
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

// owing are the lenses that owe the commits after the record once a run
// asked some of them and the failed did not answer (#588): those owing
// before that it did not ask, or that did not answer again, then the new.
func (r Record) owing(asked, failed []string) []string {
	var out []string
	for _, l := range r.Failed {
		if !slices.Contains(asked, l) || slices.Contains(failed, l) {
			out = append(out, l)
		}
	}
	return union(out, failed)
}

// standing are the blocks of the last review that still stand at the head
// (#326), each at its line there: its lens still blocks (`ai-findings`),
// and its cause is found again on the change, as a lens's is, on the side
// it was found on (#570): one found on the head's lines only there, never
// on a line the change removes that holds its quote too; one found on a
// removed line while the change still removes it. A cause no longer found
// is a commit's answer; one whose file moved is found on its new path, and
// kept there (#358). One quoted from an issue stands until its lens reads
// the issue again. A block recorded before its side was is looked for on
// both, as it was then.
func standing(repo string, s Settings, st state, blocks []Block) []Block {
	var out []Block
	for _, b := range blocks {
		if !s.AIFindings.Blocks(b.Lens) {
			continue
		}
		if !strings.HasPrefix(b.Cause.Path, "#") {
			f, why := found(repo, st, b.Lens, false, map[string]any{"severity": "important", "title": b.Lens, "cause": b.Cause}, b.Side)
			if why != "" || !f.Related {
				continue
			}
			b.Where, b.Cause.Path = f.Where, f.Cause.Path
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
	var files []File
	if whole {
		var err error
		if files, err = codeFiles(repo, s, st.lensFrom(), st.lensTo()); err != nil {
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
		return slices.Contains(paths(files), b.Cause.Path)
	}), nil
}

// span is what since finds of a range.
type span struct {
	From    string   // the newest commit reviewed, with all it holds: the base when none is, the head when every one is
	Commits []string // the range's, newest first
	New     int      // how many of them are not reviewed: a merge's line may lie between those that are
	Fresh   []string // those not reviewed, parents first: what a cap on the commits a push reads counts (ADR-0050)
	Patches []string // the patch ids of their changes
	Unread  error    // why git gave no patch ids, when it gave none: commit ids alone tell
	patchOf map[string]string
	merges  map[string]bool
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
	sp := span{merges: map[string]bool{}}
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
		sp.merges[f[0]] = len(f) > 2
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
			sp.Fresh = append(sp.Fresh, c)
		}
	}
	sp.patchOf = patch
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
// machine alters. Without renames, unlike what a lens reads (#358): a
// file moved whole would show no line, and its commit no patch id; and
// the records written before keep their ids. A commit changing nothing
// has none.
func patchIDs(repo string, commits []string) (map[string]string, error) {
	ids := map[string]string{}
	if len(commits) == 0 {
		return ids, nil
	}
	diff, err := gitIn(repo, strings.Join(commits, "\n")+"\n", "diff-tree", "--stdin", "-p", "--no-color", "--no-renames")
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

// Read is the record a run read, as the summary comment held it: the
// applying job merges the comment it finds with it (MergeSummary, #375).
func (r Record) Read() string {
	return r.String() + r.judgedLines(math.MaxInt)
}

// MergeSummary is body, the summary comment a run writes, its record
// merged with the one the comment holds now, found, when a run applied
// since this one read read (#375): every run's commits, patch ids and
// findings judged kept, each run counted once; what blocks and what is
// owed, this run's. A record that does not read, found or read, leaves
// body as it is (roles/reviewer/docs/run.md, step 3).
func MergeSummary(body, read, found string) string {
	at := strings.Index(body, recordOpening)
	ours, err := parseRecord(body)
	if at < 0 || err != nil {
		return body
	}
	now, err := readRecord(found)
	if err != nil {
		return body
	}
	was, err := readRecord(read)
	if err != nil {
		return body
	}
	if now.String() == was.String() && slices.Equal(now.Judged, was.Judged) {
		return body // none applied since this run read it
	}
	merged := ours
	merged.Reviewed = union(now.Reviewed, ours.Reviewed)
	merged.Patches = union(now.Patches, ours.Patches)
	merged.Runs = now.Runs + max(0, ours.Runs-was.Runs)
	merged.Judged, merged.Gone = slices.Clone(now.Judged), now.Gone
	for _, j := range ours.Judged {
		if !slices.Contains(was.Judged, j) && !slices.Contains(merged.Judged, j) {
			merged.Judged = append(merged.Judged, j)
		}
	}
	out := body[:at] + merged.String()
	if _, _, held := ReadJudged(body); held {
		out += merged.judgedLines(SummaryMax - len(out))
	}
	return out
}

// union is a's ids, then b's not in a.
func union(a, b []string) []string {
	out := slices.Clone(a)
	for _, x := range b {
		if !slices.Contains(out, x) {
			out = append(out, x)
		}
	}
	return out
}

// localRecord is where a machine keeps the record: in the git directory,
// never tracked; the run apart's beside it (ADR-0050).
func localRecord(repo, side string) string {
	name := "reviewer-record"
	if side == warnOnly {
		name += "-" + WarnKey
	}
	dir, err := git(repo, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return filepath.Join(repo, ".git", "workline", name)
	}
	return filepath.Join(strings.TrimSpace(dir), "workline", name)
}

// keptReview is a side's comment, text, as it stands, its record and the
// blocks beside it replaced by rec's: what the last review said is kept,
// only the record changes (a draft again, ADR-0050). "" when text holds no
// record that reads.
func keptReview(text, side string, rec Record) string {
	loc := recordLine.FindStringIndex(text)
	if loc == nil {
		return ""
	}
	end := loc[1]
	if m := blocksLine.FindStringIndex(text[end:]); m != nil && strings.TrimSpace(text[end:end+m[0]]) == "" {
		end += m[1]
	}
	body := text[:loc[0]] + rec.String() + text[end:]
	return strings.TrimSpace(strings.Replace(body, forge.Marker("sticky=reviewer/"+keyOf(side)), "", 1))
}

// loadRecord reads the record: from the merge request's summary comment when
// the run is on one and may write there, else from the machine's. It says
// where it read it, and why it could not, when it could not. The summary
// comment and the machine's file always hold a record: one holding none
// was damaged, said as one that does not read (#302). On a merge request,
// only a comment workline's own account wrote is read, the last, the one
// Sticky edits (ADR-0051): one another wrote carrying its marker is never
// taken for the record, and the range is read again, said.
func loadRecord(repo string, s Settings, side string) (Record, string, error) {
	if t := mergeRequest(); t != nil && s.ForgeWrites {
		where := fmt.Sprintf("in its summary comment on merge request %d", t.ID)
		if side == warnOnly {
			where = fmt.Sprintf("in the comment of the lenses that warn on merge request %d", t.ID)
		}
		f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
		if err != nil {
			return Record{}, where, fmt.Errorf("the forge: %v", err)
		}
		if f == nil {
			return Record{}, where, fmt.Errorf("no forge to read the merge request's comments from")
		}
		kept, err := forge.ReadKept(f, *t)
		if err != nil {
			return Record{}, where, err
		}
		marker := forge.Marker("sticky=reviewer/" + keyOf(side))
		if c, ok := kept.Last(marker); ok {
			r, err := readRecord(c)
			r.comment = c
			return r, where, err
		}
		// The run apart's first (ADR-0050): a summary comment a run not
		// split wrote holds every lens's commits and turn, and the lenses
		// owing them or that left some, which it takes up; its blocks and
		// judged lines stay there. One a split review wrote is the
		// verdict's run's alone: it starts from the base.
		if c, ok := kept.Last(forge.Marker("sticky=reviewer/" + SummaryKey)); side == warnOnly && ok && !strings.Contains(c, splitMark) {
			r, err := readRecord(c)
			if err != nil {
				return Record{}, where, nil
			}
			return Record{Reviewed: r.Reviewed, Patches: r.Patches, Runs: r.Runs, Owed: r.Owed, ReadAll: r.ReadAll, Retried: r.Retried, Failed: r.Failed, LeftBy: r.LeftBy}, where, nil
		}
		return Record{}, where, kept.Unread(marker)
	}
	file := localRecord(repo, side)
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
		r, _ := parseRecord(text) // the findings judged, read apart
		return r, backlog.UnreadRecord(text, recordOpening)
	}
	return parseRecord(text)
}

// saveLocal keeps the record on this machine: never the findings judged,
// which only a merge request's summary keeps (String leaves them out).
func saveLocal(repo, side string, r Record) error {
	file := localRecord(repo, side)
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
