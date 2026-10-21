package documentalist

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
	"go.yaml.in/yaml/v3"
)

// Records kept off a person's branch (ADR-0013 and ADR-0010, amended
// 2026-10-17): a patch changing only the lines recording when a doc was
// read gets no commit on a person's merge request; its record waits in the
// documentalist's comment, with the ids of the doc and its sources, read by
// the next push, and written on the default branch by gardening once
// merged, naming the commit the merge request became.

// keptRecord is a doc's record, as the comment keeps it.
type keptRecord struct {
	Doc string            `json:"doc" yaml:"doc"`
	Key string            `json:"key" yaml:"key"` // the header line it sets: judged, or judged-in-parts
	At  string            `json:"at" yaml:"at"`   // the commit judged at, whole
	IDs map[string]string `json:"ids" yaml:"ids"` // the doc's and each source's git object id there; "" when absent
}

// keptInput is what pre tells post of the records (in/kept.yaml): whether
// a record may be kept in the comment rather than committed, and those the
// comment keeps already.
type keptInput struct {
	Keep    bool         `yaml:"keep"`
	Records []keptRecord `yaml:"records,omitempty"`
	// Newer: Records are those an earlier round of this run proposed, its
	// comment not written yet (#572); OnForge, those the forge's keeps,
	// which the comment proposed is told from.
	Newer   bool         `yaml:"newer,omitempty"`
	OnForge []keptRecord `yaml:"on-forge,omitempty"`
}

// written are the records the forge's comment keeps now.
func (in keptInput) written() []keptRecord {
	if in.Newer {
		return in.OnForge
	}
	return in.Records
}

// keptKey keys the documentalist's comment that keeps the records.
const keptKey = "judged"

var (
	// keptLine is a record as the comment hides it.
	keptLine = regexp.MustCompile(`<!-- workline:doc-judged (.*?) -->`)
	// recordLine is a header line recording when a doc was read, and by
	// whom: never what it says, nor `checked`, which vouches (ADR-0014).
	recordLine = regexp.MustCompile(`^(judged|judged-in-parts|verified):`)
)

// recordsTask is the gardening task writing the records the merge
// requests merged kept: its branch, workline/documentalist/records.
const recordsTask = "records"

// onPersonsMergeRequest says whether the run judges a merge request a
// forge names on the merge-request event: a person's, since a release
// tool's runs as the release (ADR-0017).
func onPersonsMergeRequest() bool {
	_, ok := targetMergeRequest()
	return ok && os.Getenv("WORKLINE_EVENT") == "merge-request"
}

// targetMergeRequest is the merge request the run is on, when a forge and
// a target name one.
func targetMergeRequest() (forge.Target, bool) {
	if os.Getenv("WORKLINE_FORGE") == "" {
		return forge.Target{}, false
	}
	kind, id, ok := strings.Cut(os.Getenv("WORKLINE_TARGET"), ":")
	n, err := strconv.Atoi(id)
	if !ok || err != nil || kind != "merge-request" {
		return forge.Target{}, false
	}
	return forge.Target{Kind: kind, ID: n}, true
}

// withoutRecords is a doc without the lines of its header recording when
// it was read: two docs the same but for them say the same.
func withoutRecords(content string) string {
	_, n := header(content)
	if n == 0 {
		return content
	}
	lines := strings.Split(content, "\n")
	out := []string{lines[0]}
	for i := 1; i < n-1; i++ {
		if !recordLine.MatchString(lines[i]) {
			out = append(out, lines[i])
		}
	}
	return strings.Join(append(out, lines[n-1:]...), "\n")
}

// headerValue is what a doc's header says in key, as written; "" when it
// says nothing of it.
func headerValue(content, key string) string {
	_, n := header(content)
	lines := strings.Split(content, "\n")
	for i := 1; i < n-1; i++ {
		if v, ok := strings.CutPrefix(lines[i], key+":"); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// withHeaderValue is a doc whose header sets key to value: the line
// replaced, or added last; false for a doc with no header.
func withHeaderValue(content, key, value string) (string, bool) {
	_, n := header(content)
	if n == 0 {
		return content, false
	}
	lines := strings.Split(content, "\n")
	for i := 1; i < n-1; i++ {
		if strings.HasPrefix(lines[i], key+":") {
			lines[i] = key + ": " + value
			return strings.Join(lines, "\n"), true
		}
	}
	lines = slices.Insert(lines, n-1, key+": "+value)
	return strings.Join(lines, "\n"), true
}

// docPaths are the paths whose state a doc's judgement stands on: the doc
// and each of its sources, the section cut ("" the whole tree); false when
// a source is in another repository, whose state this one does not hold.
func docPaths(d *Doc) ([]string, bool) {
	paths := []string{d.Path}
	for _, src := range d.Sources {
		name, p, _ := splitSource(src)
		if name != "" {
			return nil, false
		}
		p = strings.Trim(strings.TrimPrefix(p, "./"), "/")
		if p == "." {
			p = ""
		}
		if !slices.Contains(paths, p) {
			paths = append(paths, p)
		}
	}
	return paths, true
}

// objectIDs are the git object ids of paths at commit: a file's blob, a
// folder's tree; "" for a path the commit does not hold.
func objectIDs(repo, commit string, paths []string) (map[string]string, error) {
	var in strings.Builder
	for _, p := range paths {
		fmt.Fprintf(&in, "%s:%s\n", commit, p)
	}
	out, err := gitIn(repo, in.String(), "cat-file", "--batch-check=%(objectname)")
	if err != nil {
		return nil, fmt.Errorf("the ids at %s: %v", abbrev(commit), err)
	}
	answers := strings.Split(out, "\n")
	if len(answers) != len(paths) {
		return nil, fmt.Errorf("the ids at %s: %d answers for %d paths", abbrev(commit), len(answers), len(paths))
	}
	ids := map[string]string{}
	for i, p := range paths {
		if a := answers[i]; !strings.HasSuffix(a, " missing") && !strings.HasSuffix(a, " ambiguous") {
			ids[p] = a
		} else {
			ids[p] = ""
		}
	}
	return ids, nil
}

// asJudgedAt says whether the doc and the sources a record names are, at
// commit, as they were judged; else the first path that is not.
func asJudgedAt(repo, commit string, r keptRecord) (bool, string, error) {
	paths := slices.Sorted(maps.Keys(r.IDs))
	now, err := objectIDs(repo, commit, paths)
	if err != nil {
		return false, "", err
	}
	if _, ok := r.IDs[r.Doc]; !ok {
		return false, r.Doc, nil
	}
	if r.IDs[r.Doc] != now[r.Doc] { // the doc first: it names its sources
		return false, r.Doc, nil
	}
	for _, p := range paths {
		if now[p] != r.IDs[p] {
			return false, p, nil
		}
	}
	return true, "", nil
}

// hiddenJSON is a value as JSON on one line, safe in a comment hidden in
// Markdown: a dash written -, `<` and `>` escaped by the JSON library,
// so nothing in it ends the comment hiding it (as the reviewer's record).
func hiddenJSON(v any) string {
	data, _ := json.Marshal(v)
	return strings.ReplaceAll(string(data), "-", `-`)
}

// parseKept reads the records a comment keeps, by doc; a line that does
// not read is no record, and its doc is judged again.
func parseKept(comment string) map[string]keptRecord {
	out := map[string]keptRecord{}
	for _, m := range keptLine.FindAllStringSubmatch(comment, -1) {
		var r keptRecord
		if json.Unmarshal([]byte(m[1]), &r) != nil || r.Doc == "" || r.At == "" || len(r.IDs) == 0 ||
			r.Key != "judged" && r.Key != "judged-in-parts" {
			continue
		}
		out[r.Doc] = r
	}
	return out
}

// keptOn reads the records the documentalist's comment on a merge request
// keeps: the last comment carrying its marker that workline's own account
// wrote, the one Sticky edits (ADR-0051); never another's, which anyone
// could have posted on a public merge request. A record that does not
// stand for the doc as it is holds nothing.
func keptOn(f forge.Forge, t forge.Target, u *othersKept) (map[string]keptRecord, error) {
	body, err := stickyOn(f, t, keptKey, u)
	if err != nil {
		return nil, err
	}
	return parseKept(body), nil
}

// stickyOn is the last of the documentalist's comments keyed key on a merge
// request that workline's own account wrote; "" when none. Those carrying
// its marker that another wrote are never read: u gathers whose.
func stickyOn(f forge.Forge, t forge.Target, key string, u *othersKept) (string, error) {
	kept, err := forge.ReadKept(f, t)
	if err != nil {
		return "", err
	}
	marker := forge.Marker("sticky=documentalist/" + key)
	if body, ok := kept.Last(marker); ok {
		return body, nil
	}
	if err := kept.Unread(marker); err != nil {
		u.unread = append(u.unread, fmt.Sprintf("on merge request %d (%v)", t.ID, err))
	}
	return "", nil
}

// othersKept is what a run's reads of the documentalist's comments left
// unread: on each merge request, the comments carrying its marker that
// workline's own account did not write (ADR-0051). Said once (findings).
type othersKept struct {
	unread []string
}

// findings says which comments were not read as the documentalist's, and
// what that changes: nothing when none was left.
func (u *othersKept) findings() []verdict.Finding {
	if len(u.unread) == 0 {
		return nil
	}
	return []verdict.Finding{{Rule: "record-unread", Level: "warn",
		Message: "the documentalist's comment " + strings.Join(u.unread, "; ") + " could not be read: the docs it would keep are judged again"}}
}

// keptBody is the comment keeping the records: what a person reads, then
// each record hidden.
func keptBody(records []keptRecord) string {
	if len(records) == 0 {
		return "**Docs judged, nothing pushed**: none kept now."
	}
	var b strings.Builder
	b.WriteString("**Docs judged, nothing pushed** — the documentalist found nothing to fix in these docs, " +
		"and could not vouch for every sentence, so `checked` stays. Their record is kept here, not committed to the branch: " +
		"the next push does not judge them again unless they or their sources change, and once this is merged, gardening writes " +
		"it on the default branch, if they are still as judged.\n\n")
	for _, r := range records {
		fmt.Fprintf(&b, "- `%s`, judged at %s\n", r.Doc, abbrev(r.At))
	}
	b.WriteString("\n")
	for _, r := range records {
		fmt.Fprintf(&b, "<!-- workline:doc-judged %s -->\n", hiddenJSON(r))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// readKept reads, on a person's merge request, the records the
// documentalist's comment keeps, and holds the docs they still stand for:
// the doc and its sources as they were judged. It tells post, in
// in/kept.yaml, whether a record may be kept there rather than committed:
// not on a forge that does not say which commit a merge request became
// (local, cmd:), which gardening could not read back, nor when the comment
// could not be read (said): the record is committed, as before.
func readKept(runDir, repo string, suspects map[string]*suspectDoc) ([]verdict.Finding, error) {
	if !onPersonsMergeRequest() {
		return nil, nil
	}
	in := keptInput{}
	var findings []verdict.Finding
	var u othersKept
	t, _ := targetMergeRequest()
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err == nil && f != nil {
		if _, lists := f.(forge.Merged); lists {
			records, err := keptOn(f, t, &u)
			if err != nil {
				findings = append(findings, verdict.Finding{Rule: "record-unread", Level: "warn",
					Message: fmt.Sprintf("the documentalist's comment on merge request %d could not be read (%v): a doc found with nothing to fix gets its record committed, as before, and those it kept are judged again; on GitLab, give the judging job GITLAB_TOKEN (read_api)", t.ID, err)})
			} else {
				in.Keep = true
				for _, p := range slices.Sorted(maps.Keys(records)) {
					in.Records = append(in.Records, records[p])
				}
				if err := holdKept(suspects, records, repo); err != nil {
					return nil, err
				}
			}
		}
	}
	findings = append(findings, u.findings()...)
	return findings, writeYAML(filepath.Join(runDir, "in", "kept.yaml"), in)
}

// holdKept keeps from the agent the suspect docs a record kept on the
// merge request still stands for.
func holdKept(suspects map[string]*suspectDoc, records map[string]keptRecord, repo string) error {
	for _, p := range sortedDocs(suspects) {
		sd := suspects[p]
		r, ok := records[p]
		if !ok || sd.note != "" {
			continue
		}
		same, _, err := asJudgedAt(repo, "HEAD", r)
		if err != nil {
			return err
		}
		if same {
			sd.note = fmt.Sprintf("(judged at %s on this merge request, nothing to fix, not vouched for; the doc and its sources unchanged since: not put before an agent again. Its record is kept in the documentalist's comment, and written on the default branch once merged; a person reads it against its sources, then moves `checked`)", abbrev(r.At))
		}
	}
	return nil
}

// readKeptInput reads what pre told post of the records; none kept when
// the file is missing.
func readKeptInput(runDir string) (keptInput, error) {
	var in keptInput
	data, err := os.ReadFile(filepath.Join(runDir, "in", "kept.yaml"))
	if os.IsNotExist(err) {
		return in, nil
	}
	if err != nil {
		return in, err
	}
	return in, yaml.Unmarshal(data, &in)
}

// recordOnly says whether a patch took a doc from was to now changing only
// the lines recording when it was read.
func recordOnly(was, now string) bool {
	return was != now && withoutRecords(was) == withoutRecords(now)
}

// keepRecords takes out of what is committed, on a person's merge request,
// each doc whose patch only records when it was judged: its record goes to
// the documentalist's comment, beside those kept before, and the comment
// is proposed when what it keeps changed. A doc fixed, or vouched for, is
// committed as before, and a record kept for it before is dropped. It
// returns the proposals left, and the docs kept.
func keepRecords(repo string, in keptInput, intents, fallback []intent.Intention, judged map[string]map[string]string, inParts map[string]string, fix *judgedFixes) ([]intent.Intention, map[string]bool, error) {
	if fix == nil {
		return intents, nil, nil
	}
	kept := map[string]keptRecord{}
	var committed []string
	dropped := false
	for _, p := range slices.Sorted(maps.Keys(fix.after)) {
		was, now := fix.before[p], fix.after[p]
		if was == now {
			continue
		}
		if !recordOnly(was, now) {
			committed = append(committed, p)
			continue
		}
		key, at := "judged", ""
		if want := judged[p]; want != nil {
			at = want[""]
		}
		if v := headerValue(now, "judged-in-parts"); v != "" && v != headerValue(was, "judged-in-parts") {
			key, at = "judged-in-parts", inParts[p]
		} else if v := headerValue(now, "judged"); v == "" || v == headerValue(was, "judged") {
			dropped = true // nothing recorded: `verified` alone, which `checked` goes with
			continue
		}
		d, err := ParseDoc(p, []byte(was))
		if err != nil || d == nil || at == "" {
			committed = append(committed, p)
			continue
		}
		paths, local := docPaths(d)
		if !local { // its state is partly another repository's: committed, as before
			committed = append(committed, p)
			continue
		}
		ids, err := objectIDs(repo, at, paths)
		if err != nil {
			return nil, nil, err
		}
		kept[p] = keptRecord{Doc: p, Key: key, At: at, IDs: ids}
	}
	if len(kept) == 0 && !dropped {
		return intents, nil, nil
	}
	var out []intent.Intention
	for _, i := range intents {
		if i.Kind != "patch" || isFallback(i, fallback) {
			out = append(out, i)
		}
	}
	var diff strings.Builder
	for _, p := range committed {
		d, err := unifiedDiff(p, fix.before[p], fix.after[p], 3)
		if err != nil {
			return nil, nil, err
		}
		diff.WriteString(d)
	}
	if diff.Len() > 0 {
		out = append(out, intent.Intention{Kind: "patch", Value: diff.String()})
	}
	merged := map[string]keptRecord{}
	for _, r := range in.Records {
		if !slices.Contains(committed, r.Doc) {
			merged[r.Doc] = r
		}
	}
	maps.Copy(merged, kept)
	var records []keptRecord
	for _, p := range slices.Sorted(maps.Keys(merged)) {
		records = append(records, merged[p])
	}
	if hiddenJSON(records) != hiddenJSON(in.Records) {
		value := map[string]any{"sticky": keptKey, "body": keptBody(records)}
		if len(records) == 0 {
			value["update-only"] = true
		}
		out = append(out, intent.Intention{Kind: "comment", Value: value})
	}
	docs := map[string]bool{}
	for p := range kept {
		docs[p] = true
	}
	return out, docs, nil
}

// personsTitle is the message of the commit a person's merge request gets:
// naming the doc it fixes, or, its record committed for want of a comment
// to keep it, the doc it records.
func personsTitle(fixed, recorded []string) string {
	docs, title, more, short := fixed, "docs: bring %s in line with the code", "docs: bring %s and %d more in line with the code", "docs: bring the docs in line with the code"
	if len(fixed) == 0 {
		docs, title, more, short = recorded, "docs: record when %s was judged", "docs: record when %s and %d more were judged", "docs: record when docs were judged"
	}
	if len(docs) == 0 {
		return ""
	}
	t := fmt.Sprintf(title, docs[0])
	if len(docs) > 1 {
		t = fmt.Sprintf(more, docs[0], len(docs)-1)
	}
	if len([]rune(t)) > 72 {
		return short
	}
	return t
}

// patchedDocs are the docs the agent's patches change, apart: those whose
// text or `checked` they change, and those whose record alone.
func patchedDocs(intents, fallback []intent.Intention, fix *judgedFixes) (fixed, recorded []string) {
	for _, i := range intents {
		if i.Kind != "patch" || isFallback(i, fallback) {
			continue
		}
		for _, p := range intent.PatchFiles(i.Value) {
			switch {
			case slices.Contains(fixed, p) || slices.Contains(recorded, p):
			case fix != nil && recordOnly(fix.before[p], fix.after[p]):
				recorded = append(recorded, p)
			default:
				fixed = append(fixed, p)
			}
		}
	}
	sort.Strings(fixed)
	sort.Strings(recorded)
	return fixed, recorded
}

// fromMerged is a record a merged merge request kept, standing at the
// commit it became: what `judged` names on the default branch.
type fromMerged struct {
	keptRecord `yaml:",inline"`
	MR         int    `yaml:"merge-request"`
	As         string `yaml:"as"` // the commit the merge request became, whole
}

// recordsMerged reads, when gardening, the records merged merge requests
// kept for the suspect docs: for each, the last first-parent commit
// changing it or a source, the merge request the forge says became it or
// the closest after, and its record. A doc as judged in that commit and
// now is returned, for `judged` to name the commit; one that is not is
// judged again, said. A forge naming no such commit reads none: its merge
// requests committed their records.
func recordsMerged(repo string, suspects map[string]*suspectDoc) ([]fromMerged, []verdict.Finding, error) {
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil || f == nil {
		return nil, nil, nil
	}
	lister, ok := f.(forge.Merged)
	if !ok {
		return nil, nil, nil
	}
	type cand struct {
		sd   *suspectDoc
		last string
	}
	var cands []cand
	oldest, oldestAt := "", int64(0)
	for _, p := range sortedDocs(suspects) {
		sd := suspects[p]
		// A doc read in parts tonight: the record of a change read alone
		// would hold it from them (#462).
		if sd.note != "" || sd.rest {
			continue
		}
		paths, local := docPaths(sd.doc)
		if !local {
			continue
		}
		specs := slices.Clone(paths)
		for i, p := range specs {
			if p == "" { // the whole tree, as a pathspec says it
				specs[i] = "."
			}
		}
		out, err := git(repo, append([]string{"log", "--first-parent", "-1", "--format=%H %ct", "HEAD", "--"}, specs...)...)
		if err != nil {
			return nil, nil, err
		}
		c, at, ok := strings.Cut(out, " ")
		if !ok {
			continue
		}
		t, _ := strconv.ParseInt(at, 10, 64)
		if oldest == "" || t < oldestAt {
			oldest, oldestAt = c, t
		}
		cands = append(cands, cand{sd, c})
	}
	if len(cands) == 0 {
		return nil, nil, nil
	}
	unread := func(what string, err error) verdict.Finding {
		return verdict.Finding{Rule: "record-unread", Level: "warn",
			Message: fmt.Sprintf("%s could not be read (%v): the docs judged on merged merge requests are judged again", what, err)}
	}
	var u othersKept
	// A merge request is merged after its commits are made: an hour's
	// leeway for clocks apart.
	all, err := lister.MergedSince(time.Unix(oldestAt, 0).Add(-time.Hour))
	if err != nil {
		return nil, []verdict.Finding{unread("the merge requests merged", err)}, nil
	}
	became := map[string]forge.MergeRequest{}
	for _, m := range all {
		if m.MergedAs == "" {
			continue
		}
		if full, err := git(repo, "rev-parse", "--verify", "-q", m.MergedAs+"^{commit}"); err == nil {
			became[full] = m
		}
	}
	out, err := git(repo, "rev-list", "--first-parent", oldest+"..HEAD")
	if err != nil {
		return nil, nil, err
	}
	var chain []string // newest first, down to the oldest change
	if out != "" {
		chain = strings.Split(out, "\n")
	}
	chain = append(chain, oldest)
	at := map[string]int{}
	for i, c := range chain {
		at[c] = i
	}
	comments := map[int]map[string]keptRecord{}
	var found []fromMerged
	var findings []verdict.Finding
	for _, c := range cands {
		var mr forge.MergeRequest
		as := ""
		for i := at[c.last]; i >= 0; i-- {
			if m, ok := became[chain[i]]; ok {
				mr, as = m, chain[i]
				break
			}
		}
		if as == "" {
			continue
		}
		records, read := comments[mr.ID]
		if !read {
			records, err = keptOn(f, forge.Target{Kind: "merge-request", ID: mr.ID}, &u)
			if err != nil {
				findings = append(findings, unread(fmt.Sprintf("the documentalist's comment on merge request %d", mr.ID), err))
				records = map[string]keptRecord{}
			}
			comments[mr.ID] = records
		}
		r, ok := records[c.sd.doc.Path]
		if !ok {
			continue
		}
		same, path, err := asJudgedAt(repo, as, r)
		why := "is not as it was then in %s, the commit it became"
		if err == nil && same {
			same, path, err = asJudgedAt(repo, "HEAD", r)
			why = "changed since %s, the commit it became"
		}
		if err != nil {
			return nil, nil, err
		}
		if !same {
			c.sd.said = fmt.Sprintf("(merge request #%d kept its record, judged at %s with nothing to fix, but %s "+why+": judged again)",
				mr.ID, abbrev(r.At), orDefault(path, "the whole tree"), abbrev(as))
			continue
		}
		found = append(found, fromMerged{keptRecord: r, MR: mr.ID, As: as})
	}
	return found, append(findings, u.findings()...), nil
}

// recordPatches are the patches writing, on the default branch, each
// record a merged merge request kept: its line set to the commit it
// became. A doc whose derived block is regenerated in the same round
// gets both in one patch, its derived one left out (derived).
func recordPatches(docs map[string]string, records []fromMerged, derived map[string]string) (patches []intent.Intention, replaced []string, err error) {
	for _, r := range records {
		was := docs[r.Doc]
		now := was
		if d, ok := derived[r.Doc]; ok {
			now = d
			replaced = append(replaced, r.Doc)
		}
		now, ok := withHeaderValue(now, r.Key, r.As[:7])
		if !ok {
			continue
		}
		diff, err := zeroContextDiff(r.Doc, was, now)
		if err != nil {
			return nil, nil, err
		}
		patches = append(patches, intent.Intention{Kind: "patch", Value: diff})
	}
	return patches, replaced, nil
}

// recordsRequest describes the merge request writing the records merged
// merge requests kept (in/records.yaml): it only records when docs were
// read, the merge policy's `records` kind (ADR-0042).
func recordsRequest(runDir string) map[string]string {
	var records []fromMerged
	if data, err := os.ReadFile(filepath.Join(runDir, "in", "records.yaml")); err == nil {
		yaml.Unmarshal(data, &records)
	}
	var body strings.Builder
	body.WriteString("Proposed by workline's documentalist, when gardening (task: records).\n\n" +
		"Each doc below was judged on its merge request, with nothing to fix and not vouched for; its record was kept in the documentalist's comment there, not committed. " +
		"The doc and its sources are as judged, at the commit the merge request became and now: the line written names that commit, so the doc is not judged again before one of them changes. " +
		"It stays suspect, for a person, who reads it against its sources, then moves `checked`.\n\nDocs:\n\n")
	for _, r := range records {
		fmt.Fprintf(&body, "- `%s` — judged on #%d at %s; `%s: %s`\n", r.Doc, r.MR, abbrev(r.At), r.Key, r.As[:7])
	}
	body.WriteString("\nRunning the same task again updates this merge request. Once its base moved, gardening rebuilds it on the new tip, or closes it, saying why, when nothing is left to merge; a commit added to its branch by hand is never overwritten, and the branch is then left to that person.\n")
	return map[string]string{"key": recordsTask, "title": "docs: record when docs were judged on their merge requests", "body": body.String()}
}
