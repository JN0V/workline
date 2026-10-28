package tester

// The frozen guard (ADR-0055, guards): the tester keeps, in workline's own
// comment on the Draft it opened, each test file it wrote and the blob
// git names it by. On the merge request's every push, a test file whose
// blob at the head is not the recorded one — changed, deleted — is the
// blocking finding tester-test-changed. Only the tester's own runs write
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
	// Scenarios: the issue's, as the run that wrote the tests read them,
	// for the auditor (ADR-0046); none in a record written before.
	Scenarios []Scenario `json:"scenarios,omitempty"`
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
	where := fmt.Sprintf("merge request %d", id)
	kept, err := forge.ReadKept(f, forge.Target{Kind: "merge-request", ID: id})
	if err != nil {
		return record{}, nil, err
	}
	marker := forge.Marker("sticky=tester/" + recordKey)
	body, ok := kept.Last(marker)
	if !ok {
		msg := "the tester's branch holds no record of its tests in workline's own comment: which tests are frozen cannot be told, nothing is read as unchanged"
		if err := kept.Unread(marker); err != nil {
			msg += "; " + err.Error()
		}
		return record{}, &verdict.Finding{Rule: "tester-record-missing", Where: where, Message: msg}, nil
	}
	m := recordData.FindStringSubmatch(body)
	var r record
	if m == nil || json.Unmarshal([]byte(m[1]), &r) != nil || len(r.Files) == 0 {
		return record{}, &verdict.Finding{Rule: "tester-record-missing", Where: where,
			Message: "the tester's comment holds no record it can read: which tests are frozen cannot be told"}, nil
	}
	return r, nil, nil
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

// guardPre is the frozen guard, the tester on a merge request, then, on
// the one that closes the issue, the green proof: no agent.
func guardPre(runDir, repo string, s Settings) int {
	kind, num, _ := strings.Cut(os.Getenv("WORKLINE_TARGET"), ":")
	id, _ := strconv.Atoi(num)
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
	if issueOf(mr.Branch) == 0 {
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "not a tester's branch: no test of its to guard"})
	}
	r, missing, err := readRecord(f, id)
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return exitExternal
	} else if err != nil {
		return fail(err)
	}
	if missing != nil {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the tester's tests cannot be told", Findings: []verdict.Finding{*missing}})
	}
	v := guard(repo, r)
	switch {
	case v.Status != verdict.Pass:
		v.Summary += "; the green proof waits for the tests as the tester wrote them"
	case mr.Draft: // its tests are meant red until whoever builds marks it ready
		v.Summary += "; the green proof waits for the pull request marked ready"
	default:
		g := green(repo, r.Base, s)
		g.Summary = v.Summary + "; " + g.Summary
		v = g
	}
	return final(runDir, v)
}

// guard compares each recorded test file with the head's.
func guard(repo string, r record) verdict.Verdict {
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
		}
		by := "a commit not the tester's"
		if c, err := git(repo, "log", "-1", "--format=%h %s", r.Base+"..HEAD", "--", file); err == nil && strings.TrimSpace(c) != "" {
			by = "commit " + strings.TrimSpace(c)
		}
		findings = append(findings, verdict.Finding{Rule: "tester-test-changed", Where: file,
			Message: fmt.Sprintf("%s %s this test of the tester's: only the tester changes its tests. Put it back as it was, "+
				"and object on its line, the comment opened by `workline:objection`: the tester answers, kept or rewritten (ADR-0055)", by, what)})
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
