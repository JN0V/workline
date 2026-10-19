package documentalist

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
	"go.yaml.in/yaml/v3"
)

// Records of a release (ADR-0034, amended 2026-10-19): what a release fix
// records, `checked` or `judged`, names the release's head, which the tool
// drops when it rebuilds its branch once the fix is merged. The
// documentalist's comment on the release merge request keeps that head and
// the ids of the doc's sources there: a record naming it, its sources the
// same, stands at the new head.

// releaseKeptKey keys the documentalist's comment on a release merge
// request keeping the heads it recorded docs at.
const releaseKeptKey = "release-judged"

// releaseKeptLine is a release record as the comment hides it.
var releaseKeptLine = regexp.MustCompile(`<!-- workline:doc-judged-at-release (.*?) -->`)

// releaseRecorded are the records the release merge request's comment
// keeps that stand for a doc as its header is now, by doc: read once by
// Pre, before any doc is found suspect.
var releaseRecorded map[string]keptRecord

// onReleaseMergeRequest says whether the run judges a release tool's merge
// request, which the engine runs as the release (ADR-0017).
func onReleaseMergeRequest() bool {
	_, ok := targetMergeRequest()
	return ok && os.Getenv("WORKLINE_EVENT") == "release" && os.Getenv("WORKLINE_RELEASE_BRANCH") != ""
}

// parseReleaseKept reads the release records a comment keeps, by doc; a
// line that does not read is no record.
func parseReleaseKept(comment string) map[string]keptRecord {
	out := map[string]keptRecord{}
	for _, m := range releaseKeptLine.FindAllStringSubmatch(comment, -1) {
		var r keptRecord
		if json.Unmarshal([]byte(m[1]), &r) != nil || r.Doc == "" || len(r.At) < 7 || len(r.IDs) == 0 ||
			r.Key != "checked" && r.Key != "judged" {
			continue
		}
		out[r.Doc] = r
	}
	return out
}

// sourcePaths are the paths a doc's record stands on, the doc left out:
// it changes once its fix is merged. False when a source is in another
// repository, or the doc names none.
func sourcePaths(d *Doc) ([]string, bool) {
	paths, local := docPaths(d)
	if !local || len(paths) < 2 {
		return nil, false
	}
	return paths[1:], true
}

// readReleaseKept reads, on a release merge request, the records the
// documentalist's comment keeps, and keeps those that still describe the
// doc: the same sources it names now. It tells post, in
// in/release-kept.yaml, whether it may write the comment: not when it
// could not be read (said), which would lose what it keeps.
func readReleaseKept(runDir, repo string, docs []*Doc) ([]verdict.Finding, error) {
	releaseRecorded = nil
	if !onReleaseMergeRequest() {
		return nil, nil
	}
	in := keptInput{}
	var findings []verdict.Finding
	t, _ := targetMergeRequest()
	unread := func(err error) verdict.Finding {
		return verdict.Finding{Rule: "record-unread", Level: "warn",
			Message: fmt.Sprintf("the documentalist's comment on release merge request %d could not be read (%v): a doc whose fix was merged is judged again on the head the release tool rebuilt; on GitLab, give the judging job GITLAB_TOKEN (read_api)", t.ID, err)}
	}
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil {
		findings = append(findings, unread(err))
	}
	if _, lists := f.(forge.Backlog); err == nil && lists { // a forge that lists no comment keeps none, as before
		body, err := stickyOn(f, t, releaseKeptKey)
		if err != nil {
			findings = append(findings, unread(err))
		} else {
			in.Keep = true
			records := parseReleaseKept(body)
			releaseRecorded = map[string]keptRecord{}
			for _, d := range docs {
				r, ok := records[d.Path]
				paths, local := sourcePaths(d)
				if ok && local && slices.Equal(slices.Sorted(slices.Values(paths)), slices.Sorted(maps.Keys(r.IDs))) {
					releaseRecorded[d.Path] = r
				}
			}
			for _, p := range slices.Sorted(maps.Keys(records)) {
				in.Records = append(in.Records, records[p])
			}
		}
	}
	return findings, writeYAML(filepath.Join(runDir, "in", "release-kept.yaml"), in)
}

// releaseHead is the commit at rev a record of the release stands for: the
// doc's `checked` or `judged` names a head of the release merge request the
// documentalist recorded it at, a head the release tool rebuilt since, and
// its sources at rev are as they were there. ok is false otherwise.
func releaseHead(repo, rev, doc, value string) (string, bool) {
	r, ok := releaseRecorded[doc]
	if !ok || len(value) < 7 || !strings.HasPrefix(r.At, value) {
		return "", false
	}
	paths := slices.Sorted(maps.Keys(r.IDs))
	now, err := objectIDs(repo, rev, paths)
	if err != nil {
		return "", false
	}
	for _, p := range paths {
		if now[p] != r.IDs[p] {
			return "", false
		}
	}
	full, err := git(repo, "rev-parse", "--verify", "-q", rev+"^{commit}")
	return full, err == nil && full != ""
}

// readReleaseKeptInput reads what pre told post of the release records;
// none, and nothing to write, when the file is missing.
func readReleaseKeptInput(runDir string) (keptInput, error) {
	var in keptInput
	data, err := os.ReadFile(filepath.Join(runDir, "in", "release-kept.yaml"))
	if os.IsNotExist(err) {
		return in, nil
	}
	if err != nil {
		return in, err
	}
	return in, yaml.Unmarshal(data, &in)
}

// keepReleaseRecords adds to what a release run proposes the comment
// keeping, beside those kept before, the head each patched doc records
// (`checked` or `judged` set to it) and the ids of its sources there. The
// patches are left as they are: they still go to the fix's merge request,
// which the release waits for (ADR-0017). The comment is proposed only
// when what it keeps changed.
func keepReleaseRecords(repo string, in keptInput, intents []intent.Intention, judged map[string]map[string]string, fix *judgedFixes) ([]intent.Intention, error) {
	if fix == nil {
		return intents, nil
	}
	merged := map[string]keptRecord{}
	for _, r := range in.Records {
		merged[r.Doc] = r
	}
	for _, p := range slices.Sorted(maps.Keys(fix.after)) {
		was, now := fix.before[p], fix.after[p]
		head := judged[p][""]
		if was == now || head == "" {
			continue
		}
		key := ""
		for _, k := range []string{"checked", "judged"} {
			if v := headerValue(now, k); len(v) >= 7 && v != headerValue(was, k) && strings.HasPrefix(head, v) {
				key = k
				break
			}
		}
		d, err := ParseDoc(p, []byte(now))
		if key == "" || err != nil || d == nil {
			continue
		}
		paths, local := sourcePaths(d)
		if !local {
			continue
		}
		ids, err := objectIDs(repo, head, paths)
		if err != nil {
			return nil, err
		}
		merged[p] = keptRecord{Doc: p, Key: key, At: head, IDs: ids}
	}
	var records []keptRecord
	for _, p := range slices.Sorted(maps.Keys(merged)) {
		records = append(records, merged[p])
	}
	if hiddenJSON(records) == hiddenJSON(in.Records) {
		return intents, nil
	}
	return append(intents, intent.Intention{Kind: "comment", Value: map[string]any{"sticky": releaseKeptKey, "body": releaseKeptBody(records)}}), nil
}

// releaseKeptBody is the comment keeping the release records: what a
// person reads, then each record hidden.
func releaseKeptBody(records []keptRecord) string {
	var b strings.Builder
	b.WriteString("**Docs recorded on this release** — the documentalist's record of the heads of this pull request it judged these docs at. " +
		"Their fix goes to a merge request of its own; once that is merged, the release tool rebuilds this branch, and a doc whose record " +
		"names a head below, its sources as they were there, is not judged again.\n\n")
	for _, r := range records {
		fmt.Fprintf(&b, "- `%s`, `%s` at %s\n", r.Doc, r.Key, abbrev(r.At))
	}
	b.WriteString("\n")
	for _, r := range records {
		fmt.Fprintf(&b, "<!-- workline:doc-judged-at-release %s -->\n", hiddenJSON(r))
	}
	return strings.TrimSuffix(b.String(), "\n")
}
