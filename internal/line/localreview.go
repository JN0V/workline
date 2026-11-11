package line

// The local forge's merge requests (#1324): no job runs on a push nor on
// a comment there, so the reconciler does what a forge's would. On each
// open, ready developer's request whose head has no verdict kept, it
// runs the merge-request line and keeps the verdict, for `workline
// follow`; on each a review round is due on, it runs the round. Both
// work in a worktree of the head, detached, removed after: the person's
// checkout is never read nor written (#1328).

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/tester"
	"github.com/JN0V/workline/internal/verdict"
)

// HeadVerdict is the line's verdict on head, as the merge policy reads it
// (ADR-0042).
func (r *Result) HeadVerdict(head string) engine.HeadVerdict {
	v := engine.HeadVerdict{Head: head, Status: r.Status, Findings: r.Findings}
	for _, s := range r.Steps {
		st := engine.StepStatus{Name: s.Name, Status: s.Status}
		if s.Status != verdict.Pass { // what blocked, said on the request (#1327)
			switch {
			case s.Result != nil:
				st.Findings = engine.Blocking(s.Result.Findings)
			case s.Gate != nil:
				st.Findings = engine.Blocking(s.Gate.Findings)
			}
		}
		v.Steps = append(v.Steps, st)
	}
	return v
}

// runsHere says the reconciler runs the merge requests' events itself:
// a forge kept in this clone, on a person's machine, not listing, its
// acts applied (a run that applies nothing starts none).
func runsHere(f forge.Forge, o engine.Options) bool {
	return forge.KeepsBranches(f) && forge.InCI() == "" && o.Inputs[ReconcileInput] != "list" && !o.NoApply && !o.DryRun
}

// machineError says the line's findings hold a machine error: an agent
// out of reach, its answer unread, the engine failing. Its red is no
// verdict on the head: the next reconcile runs it again.
func machineError(findings []verdict.Finding) string {
	for _, f := range findings {
		switch f.Rule {
		case "agent-unavailable", "engine-error", "agent-invalid-output", "config-invalid":
			return f.Rule
		}
	}
	return ""
}

// CheckedOut says where branch is checked out, in repo or one of its
// worktrees; "" nowhere.
func CheckedOut(repo, branch string) string {
	out, err := exec.Command("git", "-C", repo, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return ""
	}
	tree := ""
	for _, l := range strings.Split(string(out), "\n") {
		if t, ok := strings.CutPrefix(l, "worktree "); ok {
			tree = t
		}
		if l == "branch refs/heads/"+branch {
			return tree
		}
	}
	return ""
}

// localLines runs the merge-request line on each open, ready developer's
// request of a forge kept in this clone whose head has no verdict kept,
// up to reconcile-max (started), and keeps its verdict there. A draft
// waits to be marked ready: the merge policy merges none. It returns how
// many it ran.
func localLines(res *Result, cfg *routing.Config, f forge.Forge, o engine.Options, started *int) int {
	m, ok := f.(forge.Merger)
	if !ok || len(cfg.Events["merge-request"]) == 0 || !runsHere(f, o) {
		return 0
	}
	mrs, err := f.ListMergeRequests()
	if err != nil {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "lines-unread", Level: "warn", Where: "merge requests",
			Message: "the developer's merge requests not read, so none is reviewed: " + forge.Said(err)})
		return 0
	}
	ran := 0
	for _, mr := range mrs {
		if !strings.HasPrefix(mr.Branch, developer.Branch) || mr.Draft || mr.Base == "" {
			continue
		}
		where := fmt.Sprintf("!%d", mr.ID)
		head := TipOf(o.Repo, mr.Branch)
		if head == "" {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "line-not-run", Level: "warn", Where: where,
				Message: fmt.Sprintf("its branch %s is not in this clone: its line is not run", mr.Branch)})
			continue
		}
		if class, _, err := m.Verdict(head); err != nil || class != "" {
			continue // judged already, or said by follow
		}
		if *started >= cfg.ReconcileMax {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "reconcile-left", Level: "info", Where: where,
				Message: fmt.Sprintf("its head %s to review, past reconcile-max (%d) this run: the next run reviews it", short(head), cfg.ReconcileMax)})
			continue
		}
		*started++
		ran++
		inTree(res, o.Repo, head, where, func(tree string) {
			lo := inTreeOptions(o, tree)
			lo.Target, lo.Branch = &forge.Target{Kind: "merge-request", ID: mr.ID}, mr.Branch
			lo.Inputs = map[string]string{"range": mr.Base + ".." + head}
			r := RunWith("merge-request", lo, &Gates{})
			res.Steps = append(res.Steps, r.Steps...)
			res.Findings = append(res.Findings, r.Findings...)
			res.AgentCalls += r.AgentCalls
			res.Calls = append(res.Calls, r.Calls...)
			if why := machineError(r.Findings); r.Status == verdict.BlockedExternal || why != "" { // no verdict: run again
				if why == "" {
					why = r.Status
				}
				res.Findings = append(res.Findings, verdict.Finding{Rule: "line-ran", Level: "warn", Where: where,
					Message: fmt.Sprintf("its merge-request line failed on %s (%s): %s; no verdict kept, it runs again at the next reconcile", short(head), why, r.Summary)})
				return
			}
			v := r.HeadVerdict(head)
			kept, err := engine.KeepVerdict(m, v)
			if err != nil {
				res.Findings = append(res.Findings, verdict.Finding{Rule: "verdict-not-kept", Level: "warn", Where: where,
					Message: fmt.Sprintf("the line's verdict is not kept in the clone (%v): it runs again at the next reconcile", err)})
				return
			}
			if err := engine.SayVerdict(f, mr.ID, v); err != nil { // kept all the same (#1327)
				kept.Message += "; not said on the merge request: " + err.Error()
			}
			res.Findings = append(res.Findings, verdict.Finding{Rule: "line-ran", Level: "info", Where: where,
				Message: fmt.Sprintf("its merge-request line ran on %s: %s; %s", short(head), r.Summary, kept.Message)})
		})
	}
	return ran
}

// localRound runs the review round due on mr, the developer's, in a
// worktree of its head: its commit lands on its branch, never in the
// person's checkout.
func localRound(res *Result, cfg *routing.Config, o engine.Options, mr forge.MergeRequest) {
	where := fmt.Sprintf("!%d", mr.ID)
	head := TipOf(o.Repo, mr.Branch)
	if head == "" {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "round-not-run", Level: "warn", Where: where,
			Message: fmt.Sprintf("its branch %s is not in this clone: its review round is not run", mr.Branch)})
		return
	}
	// Its commit moves the branch: git moves none checked out elsewhere,
	// and the round's work would be lost with its worktree.
	if at := CheckedOut(o.Repo, mr.Branch); at != "" {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "round-not-run", Level: "warn", Where: where,
			Message: fmt.Sprintf("its branch %s is checked out in %s: its review round is not run; check another branch out there, or run `workline develop --merge-request %d` in it", mr.Branch, at, mr.ID)})
		return
	}
	inTree(res, o.Repo, head, where, func(tree string) {
		ro := inTreeOptions(o, tree)
		ro.Role, ro.Event, ro.Command = developerRole, "review", "workline develop"
		ro.Target = &forge.Target{Kind: "merge-request", ID: mr.ID}
		ran := len(res.Steps)
		_, status := runRole(res, cfg, ro, 0)
		res.Status = worst(res.Status, status)
		if status == verdict.Pass {
			res.Findings = append(res.Findings, CountRuns(nil, 0, runDirs(res.Steps[ran:]))...) // a round's tokens, on its issue (ADR-0064)
		}
		res.Findings = append(res.Findings, verdict.Finding{Rule: "round-ran", Level: "info", Where: where,
			Message: fmt.Sprintf("its review round ran on %s: %s", short(head), status)})
	})
}

// localAnswers runs the tester's answer on each of its open Drafts an
// objection waits on (#1265): the review's test findings the developer
// sent there, or the developer's own objection. A forge's job runs it on
// the comment (ADR-0055); in this clone none does, so the reconciler
// does, in a worktree of the Draft's head, counted in reconcile-max.
func localAnswers(res *Result, cfg *routing.Config, f forge.Forge, o engine.Options, started *int) int {
	th, ok := f.(forge.Threaded)
	if !ok || !cfg.Runs("tester") || !runsHere(f, o) {
		return 0
	}
	mrs, err := f.ListMergeRequests()
	if err != nil {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "answers-unread", Level: "warn", Where: "merge requests",
			Message: "the tester's Drafts not read, so no objection on them is answered: " + forge.Said(err)})
		return 0
	}
	answered := 0
	for _, mr := range mrs {
		if !strings.HasPrefix(mr.Branch, tester.TestsBranch) {
			continue
		}
		where := fmt.Sprintf("!%d", mr.ID)
		threads, err := th.Threads(mr.ID)
		if err != nil {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "answers-unread", Level: "warn", Where: where,
				Message: "its conversations on lines not read, so no objection there is answered: " + forge.Said(err)})
			continue
		}
		if !tester.ObjectionsWait(threads) {
			continue
		}
		if *started >= cfg.ReconcileMax {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "reconcile-left", Level: "info", Where: where,
				Message: fmt.Sprintf("an objection waits on the tester's answer, past reconcile-max (%d) this run: the next run answers it", cfg.ReconcileMax)})
			continue
		}
		head := TipOf(o.Repo, mr.Branch)
		if head == "" {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "answer-not-run", Level: "warn", Where: where,
				Message: fmt.Sprintf("its branch %s is not in this clone: the objection on it is not answered", mr.Branch)})
			continue
		}
		if at := CheckedOut(o.Repo, mr.Branch); at != "" {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "answer-not-run", Level: "warn", Where: where,
				Message: fmt.Sprintf("its branch %s is checked out in %s: the objection on it is not answered; check another branch out there, or run `workline test answer --merge-request %d` in it", mr.Branch, at, mr.ID)})
			continue
		}
		*started++
		answered++
		inTree(res, o.Repo, head, where, func(tree string) {
			ao := inTreeOptions(o, tree)
			ao.Role, ao.Event, ao.Command = "tester", "objection", "workline test answer"
			ao.Target, ao.Branch = &forge.Target{Kind: "merge-request", ID: mr.ID}, mr.Branch
			ran := len(res.Steps)
			_, status := runRole(res, cfg, ao, 0)
			res.Status = worst(res.Status, status)
			if status == verdict.Pass {
				res.Findings = append(res.Findings, CountRuns(nil, 0, runDirs(res.Steps[ran:]))...) // an answer's tokens, on its issue (ADR-0064)
			}
			res.Findings = append(res.Findings, verdict.Finding{Rule: "answer-ran", Level: "info", Where: where,
				Message: fmt.Sprintf("the tester's answer to an objection on it ran on %s: %s", short(head), status)})
		})
	}
	return answered
}

// TipOf is branch's commit in repo, "" when it is not there.
func TipOf(repo, branch string) string {
	out, err := exec.Command("git", "-C", repo, "rev-parse", "-q", "--verify", "refs/heads/"+branch+"^{commit}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// inTreeOptions is o for a run in tree: its roles' folder, given relative to
// where the command runs, made absolute.
func inTreeOptions(o engine.Options, tree string) engine.Options {
	a := engine.Options{Repo: tree, RolesDir: o.RolesDir, AI: o.AI, DefaultAI: o.DefaultAI, Forge: o.Forge}
	if a.RolesDir != "" {
		if abs, err := filepath.Abs(a.RolesDir); err == nil {
			a.RolesDir = abs
		}
	}
	return a
}

// inTree runs do in a worktree of repo at head, detached, removed after;
// one that could not be made is said, do not run.
func inTree(res *Result, repo, head, where string, do func(tree string)) {
	dir, done, err := HeadTree(repo, head)
	if err != nil {
		res.Status = worst(res.Status, verdict.Block)
		res.Findings = append(res.Findings, verdict.Finding{Rule: "tree-not-made", Where: where, Message: err.Error()})
		return
	}
	defer done()
	do(dir)
}

// HeadTree checks head out in a worktree of repo apart, detached, repo's
// .workline/config.yaml laid in it; done removes it. A round or a line
// run there leaves the person's checkout as it was (#1328).
func HeadTree(repo, head string) (tree string, done func(), err error) {
	dir, err := os.MkdirTemp("", "workline-review-")
	if err == nil {
		_ = os.Remove(dir) // git makes it
		err = tester.Worktree(repo, head, dir)
	}
	if err != nil {
		return "", nil, fmt.Errorf("its head %s could not be checked out apart: %v", short(head), err)
	}
	done = func() { tester.RemoveWorktree(repo, dir) }
	if err := sameConfig(repo, dir); err != nil {
		done()
		return "", nil, fmt.Errorf("the project's config could not be laid in its head %s's tree: %v", short(head), err)
	}
	return dir, done, nil
}

// sameConfig lays repo's .workline/config.yaml, as the reconciler read
// it, in tree: its runs take the line the person runs, even a config not
// committed yet, as a checkout of the head would keep it.
func sameConfig(repo, tree string) error {
	data, err := os.ReadFile(filepath.Join(repo, ".workline", "config.yaml"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(tree, ".workline"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(tree, ".workline", "config.yaml"), data, 0o644)
}
