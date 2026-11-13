package developer

// A request that conflicts with its base (ADR-0067, #1353): once another
// request merged, the developer's no longer merges. When the merge policy
// would merge it but git's merge of the base meets a conflict, `follow`
// says it on the request, the conflict's record in it; the next review
// round merges the base in, its agent settling the conflict markers only,
// and the engine commits the merge, two parents: the branch's head and the
// base, nothing rewritten, pushed with no force. One round that leaves a
// marker, or turns the tests red, stops: its issue blocked for a person.

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// RefusedKey is the sticky `follow` says a refused merge in, on the
// request (#1354): a conflict's carries the record a merge round reads.
const RefusedKey = "line/merge-refused"

var conflictRecord = regexp.MustCompile(`<!-- workline:conflict head=([0-9a-f]{40,64}) base=([0-9a-f]{40,64}) -->`)

// ConflictMark is the record of a conflict of head with base, the tip of
// its base branch, the next merge round reads.
func ConflictMark(head, base string) string {
	return fmt.Sprintf("<!-- workline:conflict head=%s base=%s -->", head, base)
}

// ConflictFailed says a merge round failed at head, as the developer's
// record on mr keeps it: no second one there, a person's.
func ConflictFailed(f forge.Forge, mr forge.MergeRequest, head string) (bool, error) {
	kept, err := forge.ReadKept(f, forge.Target{Kind: "merge-request", ID: mr.ID})
	if err != nil {
		return false, err
	}
	body, ok := kept.Last(forge.Marker("sticky=developer/" + RecordKey))
	a, read := ReadAct(body)
	return ok && read && a.Conflict == head, nil
}

// conflictAt reads the conflict `follow` said on the request: the head and
// the base it was read at; ok false when none.
func conflictAt(kept forge.Kept) (head, base string, ok bool) {
	body, read := kept.Last(forge.Marker("sticky=" + RefusedKey))
	m := conflictRecord.FindStringSubmatch(body)
	if !read || m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

// conflictDue says a merge round is due: `follow` said a conflict at head,
// the branch's head now, no round failed at it, and git still meets it in
// repo (a base not fetched here counts it).
func conflictDue(kept forge.Kept, a Act, repo, head string) (base string, due bool) {
	h, b, ok := conflictAt(kept)
	if !ok || head == "" || h != head || a.Conflict == head {
		return "", false
	}
	_, files, err := MergeTree(repo, head, b)
	switch {
	case err == nil && len(files) == 0:
		return "", false // settled since, or merged clean now
	case err != nil:
		// A base not fetched here: the round fetches it. Any other
		// failure would fail each round alike: not due.
		if _, missing := gitOut(repo, "cat-file", "-e", b+"^{commit}"); missing == nil {
			return "", false
		}
	}
	return b, true
}

// branchHead is a request's branch's head in repo: this clone's own
// branch on a forge that keeps them here (the local one), else as
// fetched from origin; "" neither.
func branchHead(f forge.Forge, repo, branch string) string {
	refs := []string{"refs/remotes/origin/" + branch, "refs/heads/" + branch}
	if forge.KeepsBranches(f) {
		refs = refs[1:]
	}
	for _, ref := range refs {
		if h, err := gitOut(repo, "rev-parse", "-q", "--verify", ref+"^{commit}"); err == nil && h != "" {
			return h
		}
	}
	return ""
}

// MergeTree merges base into head as git does, in memory (`git
// merge-tree`, 2.38): the tree, its conflicts' markers left in it, and
// the files in conflict; none when it merges clean.
func MergeTree(repo, head, base string) (tree string, files []string, err error) {
	cmd := exec.Command("git", "-C", repo, "merge-tree", "--write-tree", "-z", "--name-only", "--no-messages", head, base)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	var ee *exec.ExitError
	if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
		return "", nil, fmt.Errorf("git merge-tree (2.38 or later): %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	parts := strings.Split(string(out), "\x00")
	tree = strings.TrimSpace(parts[0])
	for _, p := range parts[1:] {
		if p != "" && !slices.Contains(files, p) {
			files = append(files, p)
		}
	}
	if err != nil && len(files) == 0 {
		return "", nil, errors.New("git merge-tree met a conflict it named no file of")
	}
	return tree, files, nil
}

// markerLine is a line git's merge opens or closes a conflict with: a
// bare ======= alone is no conflict (a Markdown heading's underline).
var markerLine = regexp.MustCompile(`(?m)^(<<<<<<<|>>>>>>>)( |$)`)

// markersLeft are the files of files whose content in tree still holds a
// conflict's marker; one that cannot be read counts.
func markersLeft(repo, tree string, files []string) []string {
	var out []string
	for _, f := range files {
		data, err := exec.Command("git", "-C", repo, "show", tree+":"+f).Output()
		if err != nil {
			if _, gone := gitOut(repo, "rev-parse", "-q", "--verify", tree+":"+f); gone != nil {
				continue // removed: settled as a deletion
			}
			out = append(out, f)
			continue
		}
		if markerLine.Match(data) {
			out = append(out, f)
		}
	}
	return out
}

// settled lays patches on start's tree, in an index of its own, and
// writes the tree they leave.
func settled(repo, start string, patches []intent.Intention) (string, error) {
	tmp, err := os.MkdirTemp("", "workline-merge-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	env := append(os.Environ(), "GIT_INDEX_FILE="+filepath.Join(tmp, "index"))
	run := func(stdin string, args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		var stderr bytes.Buffer
		cmd.Env, cmd.Stdin, cmd.Stderr = env, strings.NewReader(stdin), &stderr
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(stderr.String()))
		}
		return strings.TrimSpace(string(out)), nil
	}
	if _, err := run("", "read-tree", start+"^{tree}"); err != nil {
		return "", err
	}
	for _, p := range patches {
		if m, ok := p.Value.(map[string]any); ok {
			file, _ := m["file"].(string)
			content, _ := m["content"].(string)
			blob, err := run(content, "hash-object", "-w", "--stdin")
			if err != nil {
				return "", err
			}
			if _, err := run("", "update-index", "--add", "--cacheinfo", "100644,"+blob+","+file); err != nil {
				return "", err
			}
			continue
		}
		diff, _ := p.Value.(string)
		if _, err := run(intent.NormalizeDiff(diff), "apply", "--cached", "--recount", "--unidiff-zero", "-"); err != nil {
			return "", err
		}
	}
	return run("", "write-tree")
}

// mergeCommit is the merge of base into head with tree, made apart for
// the proof: the engine makes the one pushed, with its message. It needs
// no identity of the checkout's: a review job's runner has none (#1439).
func mergeCommit(repo, tree, head, base, msg string) (string, error) {
	cmd := exec.Command("git", "-C", repo, "commit-tree", tree, "-p", head, "-p", base, "-m", msg)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=workline", "GIT_AUTHOR_EMAIL=workline@localhost",
		"GIT_COMMITTER_NAME=workline", "GIT_COMMITTER_EMAIL=workline@localhost")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git commit-tree: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

// conflictHolds is who asks a merge round: the base, by its conflict.
const conflictHolds = "the base"

// conflictPre starts the merge round on the request: its agent's worktree
// starts from the branch's head with base merged in, the markers in it.
func conflictPre(runDir, repo string, s Settings, f forge.Forge, b forge.Backlog, mr forge.MergeRequest, a Act, head, base string) int {
	where := fmt.Sprintf("merge request %d", mr.ID)
	if _, err := gitOut(repo, "cat-file", "-e", base+"^{commit}"); err != nil {
		_ = exec.Command("git", "-C", repo, "fetch", "-q", "origin", mr.Base).Run()
	}
	tree, files, err := MergeTree(repo, head, base)
	if err != nil {
		return fail(fmt.Errorf("main merged into its head, in memory: %w", err))
	}
	if len(files) == 0 {
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "its head merges clean with " + mr.Base + " now: no round",
			Findings: []verdict.Finding{{Rule: "conflict-gone", Level: "info", Where: where, Message: "the conflict said on it is no more: the merge policy merges it as it is"}}})
	}
	if ai := os.Getenv("WORKLINE_AI"); ai == "" || ai == "none" {
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no agent: the conflict waits for a run with one, no round counted",
			Findings: []verdict.Finding{{Rule: "no-agent", Level: "warn", Where: where, Message: "settling a conflict is judgement on code: with no agent, nothing is merged (principle 5); in conflict: " + strings.Join(files, ", ")}}})
	}
	r := round{MR: mr.ID, Branch: mr.Branch, Record: a, At: head, Merge: base, Conflicts: files,
		Asked: []Asked{{Key: "conflict|" + base, Where: mr.Base, Holds: conflictHolds, Text: "conflicts with " + mr.Base + " in " + strings.Join(files, ", ")}}}
	if Backstop != nil {
		said, err := Backstop(f, repo, a.Issue)
		if err != nil {
			return external(err)
		}
		if said != "" {
			r.Backstop = said
			if err := writeIn(runDir, roundFile, r); err != nil {
				return fail(err)
			}
			return writeIntents(runDir, "fallback.yaml", stopped(r, repo, a.Issue))
		}
	}
	is, bounds, err := roundBounds(f, b, repo, a, a.Issue)
	if err != nil {
		return external(err)
	}
	if r.Base, err = mergeCommit(repo, tree, head, base, "workline: "+mr.Base+" merged in, its conflicts' markers left"); err != nil {
		return fail(err)
	}
	if err := writeIn(runDir, roundFile, r); err != nil {
		return fail(err)
	}
	ts, tsErr := testerSettings(repo)
	return ask(runDir, repo, r.Base, bounds, conflictTask(r, mr.Base, *is, bounds, s, ts.Command, tsErr))
}

// conflictTask is the merge round's question: the files in conflict, the
// markers to settle and nothing else, then the issue an issue run reads.
func conflictTask(r round, baseName string, is forge.Issue, b Bounds, s Settings, test string, testErr error) string {
	var t strings.Builder
	fmt.Fprintf(&t, "# Merge %s into your pull request %d, for issue #%d: %s\n\n", baseName, r.MR, is.ID, is.Title)
	fmt.Fprintf(&t, "Another change merged into %s since your pull request was written, and the two conflict. "+
		"Your worktree holds your branch with %[1]s merged in; git left its conflict markers in these files:\n\n", baseName)
	for _, f := range r.Conflicts {
		fmt.Fprintf(&t, "- `%s`\n", f)
	}
	fmt.Fprintf(&t, "\nIn each, the side after `<<<<<<<` is your branch's (%s), the side before `>>>>>>>` is %s's (%s).\n\n", shortSHA(r.At), baseName, shortSHA(r.Merge))
	t.WriteString("## What to do\n\n" +
		"- Settle each conflict: keep both intents, yours and " + baseName + "'s, as one code that does both. Read both changes before you choose.\n" +
		"- Remove every marker. Change nothing else: only these files, and in them only what the conflicts need. An edit to another file is refused.\n" +
		"- Never a frozen test: a conflict in one is the tester's; object to it, as below, and leave its markers.\n" +
		"- No commit, no git command that moves the branch: the engine makes the merge commit, two parents, your branch and " + baseName + ".\n" +
		"- Run the tests: the engine proves the merge again, the whole test command, and pushes nothing red.\n" +
		"- In your answer, a `note` a file in conflict, one line: how you settled it. A note is an item of your answer, never a file you write: "+
		"the engine puts it in the round's comment on the pull request. Write nothing outside your worktree.\n\n" +
		"One round settles it, or the issue goes to a person.\n\n")
	t.WriteString(question(is, b, test, testErr, s))
	return t.String()
}

// conflictRule refuses a merge round's edit beyond its files in conflict.
const conflictRule = "beyond-the-conflict"

// mergePost holds the merge round's answer: its agent's edits within the
// files in conflict, no marker left, the tests not red on the merge; then
// asks the engine for the merge commit. Else nothing is pushed, and its
// issue is blocked for a person: no second round at this head.
func mergePost(runDir, repo string, s Settings, r round) int {
	where := fmt.Sprintf("merge request %d", r.MR)
	var b Bounds
	if err := readIn(runDir, boundsFile, &b); err != nil {
		return fail(err)
	}
	proposed, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
	if err != nil {
		return fail(err)
	}
	var notes []string
	var rest []intent.Intention
	for _, in := range proposed {
		if in.Kind != "note" {
			rest = append(rest, in)
		} else if text := noteText(in.Value); text != "" {
			notes = append(notes, text)
		}
	}
	// Its edits from the merge with its markers: its own, none of base's.
	kept, refused, said, unread, _, err := held(runDir, repo, r.Base, b, nil, rest)
	if err != nil {
		return fail(err)
	}
	var patches, others []intent.Intention
	for _, in := range kept {
		if in.Kind != "patch" {
			others = append(others, in) // an objection on the tester's Draft
			continue
		}
		files, _ := touched(in.Value)
		if out := slices.DeleteFunc(slices.Clone(files), func(f string) bool { return slices.Contains(r.Conflicts, f) }); len(out) > 0 {
			for _, f := range out {
				refused = append(refused, Refusal{File: f, Rule: conflictRule})
			}
			continue
		}
		patches = append(patches, in)
	}
	a := r.Record
	n := a.Round + 1
	this := Round{N: n, Asked: texts(r.Asked)}
	v := verdict.Verdict{Status: verdict.Pass, Summary: fmt.Sprintf("round %d on %s: %s merged in", n, where, r.Asked[0].Where),
		Findings: append([]verdict.Finding{rightsFinding(where)}, refusalFindings(where, refused, said)...)}
	if len(unread) > 0 {
		v.Findings = append(v.Findings, unreadFinding(where, unread, true))
	}
	failed := ""
	tree, err := settled(repo, r.Base, patches)
	switch {
	case err != nil:
		failed = "its edits do not apply on the merge: " + err.Error()
	default:
		if left := markersLeft(repo, tree, r.Conflicts); len(left) > 0 {
			failed = "conflict markers are left in " + strings.Join(left, ", ")
		}
	}
	var p Proof
	u := used(runDir)
	if failed == "" {
		// The proof on the merge itself, both parents' code: the whole
		// test command, the frozen tests of both requests in it.
		at, err := mergeCommit(repo, tree, r.At, r.Merge, "workline: the merge proven")
		if err != nil {
			return fail(err)
		}
		var caps []Cap
		var found []verdict.Finding
		p, u, caps, found = proven(runDir, repo, at, where, nil, union(a.Acceptance, union(a.Code, a.Tests)), again(a.Proof), s)
		for _, f := range found {
			f.Message = strings.ReplaceAll(f.Message, "; its pull request opens as a Draft", "")
			v.Findings = append(v.Findings, f)
		}
		if p.Result == Red {
			failed = "the merge turns the tests red: " + strings.Join(p.Red, ", ")
		} else {
			a.Proof, a.Caps = &p, caps
		}
	}
	this.Turns, this.Tokens, this.Cached, this.Tools = u.Turns, u.Tokens, u.Cached, u.Tools
	a.Refused = append(a.Refused, refused...)
	var out []intent.Intention
	if failed == "" {
		this.Result, this.Merge = p.Result, "settled"
		msg := fmt.Sprintf("Merge branch '%s' into %s\n\nConflicts settled: %s", r.Asked[0].Where, r.Branch, strings.Join(r.Conflicts, ", "))
		if err := writeMergeRequest(runDir, map[string]any{"push": msg, "merge": r.Merge, "tree": tree}); err != nil {
			return fail(err)
		}
		v.Summary += ", its conflicts settled, the tests " + p.Result
	} else {
		this.Result, this.Merge = "no-code", "failed"
		a.Conflict = r.At // no second round at this head
		v.Summary += "; not settled: " + failed
		v.Findings = append(v.Findings, verdict.Finding{Rule: "conflict-not-settled", Level: "warn", Where: where,
			Message: fmt.Sprintf("%s: nothing pushed; #%d blocked for a person", failed, a.Issue)})
		why := fmt.Sprintf("its round on pull request %d could not merge %s in: %s", r.MR, r.Asked[0].Where, failed)
		next := fmt.Sprintf("a person merges %s into `%s`, settles the conflict and pushes, never a rebase; then moves the issue back by its label", r.Asked[0].Where, r.Branch)
		out = append(out, holdFor(repo, a, r, why, next, notes, others)...)
		if objectedOn(others) {
			a.Widen = &Widen{At: r.At, Scope: scopeKey(b.Scope), Objected: true, Draft: b.Draft, Kept: r.Kept}
		}
	}
	a.Round = n
	a.Rounds = append(a.Rounds, this)
	out = append([]intent.Intention{
		{Kind: "comment", Value: map[string]any{"sticky": "round", "body": mergeBody(this, r, notes, failed)}},
		{Kind: "comment", Value: map[string]any{"sticky": RecordKey, "body": recordBody(a)}},
	}, append(out, others...)...)
	if err := writeIntents(runDir, "", out); err != 0 {
		return err
	}
	return writeVerdict(runDir, v)
}

// mergeBody is the merge round's comment on its pull request.
func mergeBody(x Round, r round, notes []string, failed string) string {
	var b strings.Builder
	base := r.Asked[0].Where
	if failed == "" {
		fmt.Fprintf(&b, "**The developer**, round %d: %s merged in, its conflicts settled (`%s`), the tests %s. A merge commit, two parents, nothing rewritten: the checks and the reviewer read it next.\n",
			x.N, base, strings.Join(r.Conflicts, "`, `"), x.Result)
	} else {
		fmt.Fprintf(&b, "**The developer**, round %[1]d: %[2]s not merged in, %[3]s. Nothing pushed; its issue #%[4]d is blocked for a person, who merges %[2]s into `%[5]s` and settles the conflict.\n",
			x.N, base, failed, r.Record.Issue, r.Branch)
	}
	if len(notes) > 0 {
		b.WriteString("\nWhat it did:\n\n")
		for _, n := range notes {
			fmt.Fprintf(&b, "- %s\n", strings.ReplaceAll(strings.TrimSpace(n), "\n", " "))
		}
	}
	return b.String()
}
