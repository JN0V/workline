package tester

// A bug's parent, its parts all closed (#1212): the tester runs the tests
// it wrote on the parent's Draft before its split — the reporter's
// symptom, red there — over the default branch's current commit, through
// its green proof, as a person in QA reruns the reproduction once the
// fixes merged. What it found goes in its one comment on the parent
// (backlog.Symptom), which the engine reads at the parent's closing:
// green closes it alone past cautious; red, missing or unproven leaves
// the closing a person's, the reason said. No agent.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/clock"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// symptomInput reads the line's input on a bug's parent: its parts
// closed, by number; ok false when the run was not asked for its symptom.
func symptomInput(runDir string) (parts []int, ok bool) {
	data, err := os.ReadFile(filepath.Join(runDir, "in", "input", backlog.SymptomInput))
	if err != nil || strings.TrimSpace(string(data)) == "" {
		return nil, false
	}
	for _, w := range strings.Split(strings.TrimSpace(string(data)), ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(w)); err == nil {
			parts = append(parts, n)
		}
	}
	return parts, true
}

// symptomPre proves the symptom of the parent the run targets and says
// it on the parent, in the tester's comment there.
func symptomPre(runDir, repo string, s Settings, parts []int) int {
	_, num, _ := strings.Cut(os.Getenv("WORKLINE_TARGET"), ":")
	id, _ := strconv.Atoi(num)
	if id <= 0 {
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no issue: no symptom to prove"})
	}
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil {
		return fail(err)
	}
	if f == nil {
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no forge: the symptom of a bug's parent is not proven",
			Findings: []verdict.Finding{{Rule: "no-forge", Level: "warn", Where: fmt.Sprintf("#%d", id), Message: "the tester reads its Draft on a forge; " + forge.Missing}}})
	}
	sym, err := proveSymptom(repo, f, id, parts, s)
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return exitExternal
	}
	if err != nil {
		return fail(err)
	}
	said := []intent.Intention{{Kind: "comment", Value: map[string]any{"issue": id, "sticky": backlog.SymptomKey, "on-block": true, "body": sym.Body()}}}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), said); err != nil {
		return fail(err)
	}
	v := verdict.Verdict{Status: verdict.Pass, Final: true, Summary: "the symptom of #" + strconv.Itoa(id) + ": " + sym.Why}
	if !sym.Green {
		v.Findings = []verdict.Finding{{Rule: "symptom-unproven", Level: "info", Where: fmt.Sprintf("#%d", id),
			Message: sym.Why + ": its closing stays a person's"}}
	}
	data, err := json.Marshal(v)
	if err != nil {
		return fail(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", saidFile), data, 0o644); err != nil {
		return fail(err)
	}
	return 0
}

// proveSymptom runs the tests of id's Draft over its base branch's
// current commit: the symptom record, green or why not. An error is
// the forge's or the machine's, never a test's.
func proveSymptom(repo string, f forge.Forge, id int, parts []int, s Settings) (backlog.Symptom, error) {
	sym := backlog.Symptom{Issue: id, Parts: slices.Sorted(slices.Values(parts)), Ran: clock.Now().UTC().Format(time.RFC3339)}
	mrs, err := f.ListMergeRequests()
	if errors.Is(err, forge.ErrUnreachable) {
		return sym, err
	} else if err != nil {
		sym.Why = "the forge lists no merge request (" + err.Error() + "): its Draft, and the reproduction test it holds, cannot be found"
		return sym, nil
	}
	i := slices.IndexFunc(mrs, func(m forge.MergeRequest) bool { return issueOf(m.Branch) == id })
	if i < 0 {
		sym.Why = fmt.Sprintf("no reproduction test of its own: the tester has no open Draft of #%d, its tests written before its split", id)
		return sym, nil
	}
	mr := mrs[i]
	sym.Draft = mr.ID
	rec, missing, err := readRecord(f, mr.ID)
	if err != nil {
		return sym, err
	}
	if missing != nil {
		sym.Why = fmt.Sprintf("its Draft, merge request %d, keeps no record of its tests the tester can read", mr.ID)
		return sym, nil
	}
	sym.Tests = rec.Tests
	sym.Branch = mr.Base
	if sym.Branch == "" {
		sym.Branch = "main"
	}
	main, err := tip(repo, sym.Branch)
	if err != nil {
		sym.Why = fmt.Sprintf("the commit of %s could not be read (%v)", sym.Branch, err)
		return sym, nil
	}
	sym.Commit = main
	draft, err := tip(repo, mr.Branch)
	if err != nil && mr.Head != "" {
		draft, err = tip(repo, mr.Head) // a fork's, kept under the forge's ref
	}
	if err != nil {
		sym.Why = fmt.Sprintf("its Draft's branch %s could not be read (%v)", mr.Branch, err)
		return sym, nil
	}
	v, why := symptomGreen(repo, rec, main, draft, s)
	if why != "" {
		sym.Why = why
		return sym, nil
	}
	sym.Green = v.Status == verdict.Pass
	if sym.Green {
		sym.Why = sym.Said()
		return sym, nil
	}
	sym.Why = fmt.Sprintf("its reproduction tests, from merge request %d, are not green on %s at %s: %s", mr.ID, sym.Branch, short(main), v.Summary)
	if len(v.Findings) > 0 {
		sym.Why += " (" + v.Findings[0].Rule + ": " + v.Findings[0].Message + ")"
	}
	return sym, nil
}

// tip is ref's commit as origin holds it now, fetched; the local branch
// only when there is no origin to ask: a fetch that fails never falls
// back on a clone's copy, maybe behind.
func tip(repo, ref string) (string, error) {
	if !strings.HasPrefix(ref, "refs/") {
		ref = "refs/heads/" + ref
	}
	if _, err := git(repo, "remote", "get-url", "origin"); err == nil {
		if _, err := git(repo, "fetch", "-q", "--no-tags", "origin", ref); err != nil {
			return "", fmt.Errorf("origin could not be read (%v)", err)
		}
		out, err := git(repo, "rev-parse", "--verify", "-q", "FETCH_HEAD^{commit}")
		return strings.TrimSpace(out), err
	}
	out, err := git(repo, "rev-parse", "--verify", "-q", ref+"^{commit}")
	if err != nil {
		return "", errors.New("neither origin nor this clone holds it")
	}
	return strings.TrimSpace(out), nil
}

// symptomGreen lays the Draft's test files over main and runs the green
// proof there; why says what kept it from running. A file main lacks is
// the Draft's; one main holds as the Draft does is left; one both
// changed since the record's base is merged three ways (#1259: the
// tester adds to a file others add to).
func symptomGreen(repo string, rec record, main, draft string, s Settings) (verdict.Verdict, string) {
	tree, err := os.MkdirTemp("", "workline-symptom-")
	if err != nil {
		return verdict.Verdict{}, err.Error()
	}
	if err := Worktree(repo, main, tree); err != nil {
		return verdict.Verdict{}, err.Error()
	}
	defer RemoveWorktree(repo, tree)
	laid := false
	for _, file := range rec.files() {
		ours, err := blob(repo, draft, file)
		if err != nil {
			return verdict.Verdict{}, fmt.Sprintf("its test file %s is not on its Draft's branch", file)
		}
		theirs, _ := blob(repo, main, file)
		if theirs == ours {
			continue // on main as the Draft holds it
		}
		text, err := git(repo, "cat-file", "blob", ours)
		if err != nil {
			return verdict.Verdict{}, err.Error()
		}
		if theirs != "" {
			if text, err = merge3(repo, rec.Base, file, theirs, ours); err != nil {
				return verdict.Verdict{}, fmt.Sprintf("its test file %s does not merge over %s: %v", file, short(main), err)
			}
		}
		path := filepath.Join(tree, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return verdict.Verdict{}, err.Error()
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			return verdict.Verdict{}, err.Error()
		}
		if _, err := git(tree, "add", "--", file); err != nil {
			return verdict.Verdict{}, err.Error()
		}
		laid = true
	}
	if laid {
		if _, err := git(tree, "-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false", "-c", "user.name=workline", "-c", "user.email=workline@invalid",
			"commit", "-q", "--no-verify", "-m", "test: the reproduction tests over the default branch"); err != nil {
			return verdict.Verdict{}, err.Error()
		}
	}
	v, doubt := green(tree, rec, s)
	if doubt != nil {
		v = doubt.unproven("no judge is asked at a parent's closing")
		v.Status = verdict.Human
	}
	return v, ""
}

// blob is file's blob at rev, "" with an error when rev has none.
func blob(repo, rev, file string) (string, error) {
	out, err := git(repo, "rev-parse", "--verify", "-q", rev+":"+file)
	return strings.TrimSpace(out), err
}

// merge3 merges the Draft's file (ours) into main's (theirs), from the
// file at base, empty when base had none: an error on a conflict.
func merge3(repo, base, file, theirs, ours string) (string, error) {
	dir, err := os.MkdirTemp("", "workline-merge-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	write := func(name, b string) (string, error) {
		text := ""
		if b != "" {
			out, err := git(repo, "cat-file", "blob", b)
			if err != nil {
				return "", err
			}
			text = out
		}
		p := filepath.Join(dir, name)
		return p, os.WriteFile(p, []byte(text), 0o644)
	}
	from, _ := blob(repo, base, file)
	paths := map[string]string{}
	for name, b := range map[string]string{"main": theirs, "base": from, "draft": ours} {
		if paths[name], err = write(name, b); err != nil {
			return "", err
		}
	}
	out, err := exec.Command("git", "merge-file", "-p", paths["main"], paths["base"], paths["draft"]).Output()
	if err != nil {
		return "", errors.New("its lines and main's conflict")
	}
	return string(out), nil
}
