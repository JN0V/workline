package tester

// The frozen guard (ADR-0055, guards): the tester keeps, in workline's own
// comment on the Draft it opened, each test file it wrote and the blob
// git names it by. On the merge request's every push, a test file whose
// blob at the head is not the recorded one is read against it: the lines
// the tester wrote kept whole, others added beside them, it passes
// (#1259); one of them changed, or the file deleted, is the blocking
// finding tester-test-changed. Only the tester's own runs write
// the record: no label lifts it, no commit's trailer, which anyone can
// write, is trusted.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// branchPrefix is the tester's work branches', one an issue
// (workline/tester/issue-<n>).
const branchPrefix = "workline/tester/issue-"

// recordKey is the record's sticky key: the comment's marker is
// sticky=tester/tests, as the engine writes it.
const recordKey = "tests"

// record is what the tester keeps on its Draft: the issue, the base its
// tests were proven red on, and each test file's blob.
type record struct {
	Issue int               `json:"issue"`
	Base  string            `json:"base"`
	Files map[string]string `json:"files"`
	// Tests: the ids of the tests the red proof found new; the green proof
	// requires each on the code (ADR-0065, change 2): a report's id,
	// file:name for a test whose build broke on the base, or the file
	// alone when its ecosystem names no test (#1293). None in a record
	// written before.
	Tests []string `json:"tests,omitempty"`
	// Errored: those of Tests the red run wrote with no class, failed —
	// maybe the runner's record of a file it could not collect, not a
	// test case —, or red by an error alone, alone in its suite, a name
	// no test file holds (a suite's record, #1325): gone on the code, the
	// judge reads it, never a block.
	Errored []string `json:"errored,omitempty"`
	// Scenarios: the issue's, as the run that wrote the tests read them,
	// for the auditor (ADR-0046); none in a record written before.
	Scenarios []Scenario `json:"scenarios,omitempty"`
	// Covers: each scenario's tests, as the agent declared them and the
	// red proof found them (ADR-0065, change 4): the developer's
	// objection names a scenario by them. None in a record written before,
	// nor for a scenario no test can check.
	Covers map[string][]string `json:"covers,omitempty"`
	// RedUnproven: why the judge did not settle the tests' red proof, a
	// build broken on the base (ADR-0065); the developer's pull request
	// says it too. None when proven.
	RedUnproven string `json:"red-unproven,omitempty"`
}

var recordData = regexp.MustCompile(`<!-- workline:tester-record (\{.*\}) -->`)

// body is the comment a person reads: the tests, frozen, and how to
// object; the record hidden at its end.
func (r record) body() string {
	var b strings.Builder
	fmt.Fprintf(&b, "**The acceptance tests of #%d, frozen**: the tester wrote them before the code, red on the base. "+
		"Only the tester's own commit changes them: a test that does not fit, object on its line, the comment opened by `workline:objection` "+
		"(the test, its scenario, the kind, why). Any other change to them blocks, `tester-test-changed`.\n\n| Test file | Blob |\n|---|---|\n", r.Issue)
	for _, f := range r.files() {
		fmt.Fprintf(&b, "| `%s` | `%s` |\n", f, short(r.Files[f]))
	}
	if len(r.Tests) > 0 {
		fmt.Fprintf(&b, "\nThe green proof requires each of these tests on the code, three runs out of three: `%s`.\n", strings.Join(r.Tests, "`, `"))
		var alone []string
		for _, t := range r.Tests {
			if fileAlone(r, t) {
				alone = append(alone, t)
			}
		}
		if len(alone) > 0 {
			fmt.Fprintf(&b, "By its file alone, its build broken on the base and no test named in it: %s, met by a test of that file.\n", codes(alone))
		}
	}
	if r.RedUnproven != "" {
		fmt.Fprintf(&b, "\n**Red unproven**: %s\n", r.RedUnproven)
	}
	data, _ := json.Marshal(r)
	fmt.Fprintf(&b, "\n<!-- workline:tester-record %s -->", data)
	return b.String()
}

func (r record) files() []string {
	var out []string
	for f := range r.Files {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// recordIntention is the comment keeping r on the merge request a run
// targets.
func (r record) intention() map[string]any {
	return map[string]any{"comment": map[string]any{"sticky": recordKey, "body": r.body()}}
}

// readRecord reads the record from workline's own comment on the merge
// request: never from another's, nor one another edited last.
func readRecord(f forge.Forge, id int) (record, *verdict.Finding, error) {
	r, missing, _, err := readRecordKept(f, id)
	return r, missing, err
}

// readRecordKept is readRecord, with the merge request's comments as
// read: who workline's own account is, learnt from them too — on GitHub,
// the App's login from a comment that names the App, where a reply in a
// thread names none.
func readRecordKept(f forge.Forge, id int) (record, *verdict.Finding, forge.Kept, error) {
	where := fmt.Sprintf("merge request %d", id)
	kept, err := forge.ReadKept(f, forge.Target{Kind: "merge-request", ID: id})
	if err != nil {
		return record{}, nil, kept, err
	}
	marker := forge.Marker("sticky=tester/" + recordKey)
	body, ok := kept.Last(marker)
	if !ok {
		msg := "the tester's branch holds no record of its tests in workline's own comment: which tests are frozen cannot be told, nothing is read as unchanged"
		if err := kept.Unread(marker); err != nil {
			msg += "; " + err.Error()
		}
		return record{}, &verdict.Finding{Rule: "tester-record-missing", Where: where, Message: msg}, kept, nil
	}
	m := recordData.FindStringSubmatch(body)
	var r record
	if m == nil || json.Unmarshal([]byte(m[1]), &r) != nil || len(r.Files) == 0 {
		return record{}, &verdict.Finding{Rule: "tester-record-missing", Where: where,
			Message: "the tester's comment holds no record it can read: which tests are frozen cannot be told"}, kept, nil
	}
	return r, nil, kept, nil
}

// issueOf is the issue a tester's branch is for, 0 when the branch is not
// one.
func issueOf(branch string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(branch, branchPrefix))
	if !strings.HasPrefix(branch, branchPrefix) || err != nil || n <= 0 {
		return 0
	}
	return n
}

// closingDrafts are the tester's open Drafts of the issues mr closes,
// none when it closes none or no issue has one: the developer's pull
// request, stacked on that Draft, merges its tests, so they are guarded
// and proven green there, each record read from its Draft (#1281). Any
// branch closing an issue is read so, not only the developer's: the pull
// request that closes an issue carries its frozen tests. ok is false
// when the forge lists no merge request: none could be looked for.
func closingDrafts(f forge.Forge, mr forge.MergeRequest) (drafts []int, ok bool, err error) {
	closes := forge.Closes(mr.Title, mr.Body)
	if len(closes) == 0 {
		return nil, true, nil
	}
	open, err := f.ListMergeRequests()
	if errors.Is(err, forge.ErrUnreachable) {
		return nil, false, err
	} else if err != nil {
		return nil, false, nil
	}
	for _, n := range closes {
		for _, o := range open {
			if o.ID != mr.ID && issueOf(o.Branch) == n && !slices.Contains(drafts, o.ID) {
				drafts = append(drafts, o.ID)
			}
		}
	}
	return drafts, true, nil
}

// merged is the records of several Drafts as one, for one green proof:
// their files, tests and scenarios together, the first's base.
func merged(rs []record) record {
	if len(rs) == 1 {
		return rs[0]
	}
	m := rs[0]
	m.Files = map[string]string{}
	m.Tests, m.Errored, m.Scenarios = nil, nil, nil
	for _, r := range rs {
		for k, v := range r.Files {
			m.Files[k] = v
		}
		m.Tests = append(m.Tests, r.Tests...)
		m.Errored = append(m.Errored, r.Errored...)
		m.Scenarios = append(m.Scenarios, r.Scenarios...)
	}
	return m
}

// guardPre is the frozen guard, the tester on a merge request — its own
// Draft, or one closing the Draft's issue —, then, on one ready, the
// green proof: no agent but the judge of a frozen test not found under
// its name (renamed.go).
func guardPre(runDir, repo string, s Settings) int {
	kind, num, _ := strings.Cut(os.Getenv("WORKLINE_TARGET"), ":")
	id, _ := strconv.Atoi(num)
	if os.Getenv("WORKLINE_JUDGED") == "answered" {
		if v, matched, ok, err := renameAnswered(runDir); err != nil {
			return fail(err)
		} else if ok {
			return greenEnd(runDir, id, v, matched)
		}
	}
	if kind != "merge-request" || id <= 0 {
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no merge request: no test of the tester's to guard"})
	}
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil {
		return fail(err)
	}
	if f == nil {
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no forge: the tester's tests are not guarded",
			Findings: []verdict.Finding{{Rule: "no-forge", Level: "warn", Where: fmt.Sprintf("merge request %d", id),
				Message: "the frozen guard reads the merge request's branch and the tester's record on a forge; " + forge.Missing}}})
	}
	mr, err := f.MergeRequest(id)
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return exitExternal
	} else if err != nil {
		return fail(err)
	}
	from := []int{id}
	if issueOf(mr.Branch) == 0 {
		drafts, ok, err := closingDrafts(f, mr)
		if errors.Is(err, forge.ErrUnreachable) {
			fmt.Fprintln(os.Stderr, err)
			return exitExternal
		}
		if !ok {
			return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "the tester's Drafts not read: no test of its guarded",
				Findings: []verdict.Finding{{Rule: "drafts-unread", Level: "warn", Where: fmt.Sprintf("merge request %d", id),
					Message: "it closes an issue, and the forge lists no merge request (`merge-requests`): the tester's Draft of that issue, and the tests it froze, cannot be found"}}})
		}
		if len(drafts) == 0 {
			return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no tester's Draft for an issue it closes: no test of its to guard"})
		}
		from = drafts
	}
	var rs []record
	var v verdict.Verdict // every Draft's guard
	for _, d := range from {
		r, missing, err := readRecord(f, d)
		if errors.Is(err, forge.ErrUnreachable) {
			fmt.Fprintln(os.Stderr, err)
			return exitExternal
		} else if err != nil {
			return fail(err)
		}
		if missing != nil {
			return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the tester's tests cannot be told", Findings: []verdict.Finding{*missing}})
		}
		draft := 0
		if d != id {
			draft = d
		}
		g := guard(repo, r, draft)
		if len(rs) == 0 || g.Status != verdict.Pass {
			v.Status, v.Summary = g.Status, g.Summary
		}
		v.Findings = append(v.Findings, g.Findings...)
		rs = append(rs, r)
	}
	r := merged(rs)
	if len(rs) > 1 { // the summary counts every Draft's
		v.Summary = fmt.Sprintf("the tester's %d test files of %d Drafts as it wrote them", len(r.Files), len(rs))
		if v.Status != verdict.Pass {
			v.Summary = fmt.Sprintf("%d of the tester's tests changed by another", len(v.Findings))
		}
	}
	switch {
	case v.Status != verdict.Pass:
		v.Summary += "; the green proof waits for the tests as the tester wrote them"
	case mr.Draft: // its tests are meant red until whoever builds marks it ready
		v.Summary += "; the green proof waits for the pull request marked ready"
	default:
		g, doubt := green(repo, r, s)
		g.Summary = v.Summary + "; " + g.Summary
		v = g
		if doubt != nil {
			doubt.Verdict = g
			if k, err := forge.ReadKept(f, forge.Target{Kind: "merge-request", ID: id}); err == nil {
				if body, ok := k.Last(forge.Marker("sticky=tester/green")); ok {
					doubt.remembered(matchedBefore(body))
				}
			}
			if len(doubt.Missing) == 0 {
				return greenEnd(runDir, id, doubt.Verdict, doubt.Matched)
			}
			if ai := os.Getenv("WORKLINE_AI"); ai == "" || ai == "none" {
				return greenEnd(runDir, id, doubt.unproven("no agent judges it"), doubt.Matched)
			}
			if err := doubt.ask(runDir); err != nil {
				return fail(err)
			}
			return 0 // the engine asks the judge, and runs pre again with its answer
		}
	}
	return final(runDir, v)
}

// guard compares each recorded test file with the head's; draft is the
// tester's Draft the record was read from, 0 when it is the merge request
// guarded: an objection is answered there only.
func guard(repo string, r record, draft int) verdict.Verdict {
	where := "on its line"
	if draft > 0 {
		where = fmt.Sprintf("on its line in the tester's Draft, merge request %d (not here: the tester answers on its own branch)", draft)
	}
	var findings []verdict.Finding
	for _, file := range r.files() {
		blob, err := git(repo, "rev-parse", "--verify", "-q", "HEAD:"+file)
		blob = strings.TrimSpace(blob)
		if err == nil && blob == r.Files[file] {
			continue
		}
		what := "deletes"
		if err == nil {
			what = "changes"
			// Its lines kept whole, others added beside them: a file
			// the tests share, each test as it was (#1259).
			missing, kept, read := keptBeside(repo, r, file)
			if kept {
				continue
			}
			if read {
				what = "changes the tester's lines from " + missing.name() + ", in"
			}
		}
		by := "a commit not the tester's"
		if c, err := git(repo, "log", "-1", "--format=%h %s", r.Base+"..HEAD", "--", file); err == nil && strings.TrimSpace(c) != "" {
			by = "commit " + strings.TrimSpace(c)
		}
		findings = append(findings, verdict.Finding{Rule: "tester-test-changed", Where: file,
			Message: fmt.Sprintf("%s %s this test of the tester's: only the tester changes its tests. Put it back as it was, "+
				"and object %s, the comment opened by `workline:objection`: the tester answers, kept or rewritten (ADR-0055)", by, what, where)})
	}
	if len(findings) > 0 {
		return verdict.Verdict{Status: verdict.Block, Summary: fmt.Sprintf("%d of the tester's tests changed by another", len(findings)), Findings: findings}
	}
	n := len(r.Files)
	return verdict.Verdict{Status: verdict.Pass, Summary: fmt.Sprintf("the tester's %d test file%s as it wrote %s", n, plural(n), map[bool]string{true: "it", false: "them"}[n == 1])}
}

// blobsOf names each file by its blob at rev.
func blobsOf(dir, rev string, files []string) (map[string]string, error) {
	out := map[string]string{}
	for _, f := range slices.Compact(slices.Sorted(slices.Values(files))) {
		b, err := git(dir, "rev-parse", rev+":"+f)
		if err != nil {
			return nil, err
		}
		out[f] = strings.TrimSpace(b)
	}
	return out, nil
}
