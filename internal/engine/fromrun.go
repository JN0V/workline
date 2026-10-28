package engine

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/verdict"
	"go.yaml.in/yaml/v3"
)

// A merge request's own CI run can be changed by its code: on GitHub, a
// pull request from a branch of the repository runs its own copy of the
// workflow. The job applying what that run proposed holds the write token,
// so it reads the run's folders as data (#331): the agents' proposals, and
// nothing else. The repository, the roles, the forge, the merge request,
// and what applying may do are the job's own.

// FromMergeRequest is what a job applying a merge request's own run takes
// from itself, never from the run.
type FromMergeRequest struct {
	Repo     string // its checkout of the base: the project's settings are read there
	RolesDir string // its engine's roles
	Forge    string // its forge
	ID       int    // the merge request whose run it applies
	Head     string // the commit that run judged, as the forge's event names it
}

// runName is how a run's folder is named (newRunDir).
var runName = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}\.[0-9]{9}-[a-z0-9-]+$`)

// RunName says whether name is how a run's folder is named: an
// argument naming anything else is no run to apply.
func RunName(name string) bool { return runName.MatchString(name) }

// plainModel is an agent and model as a commit's trailer names them.
var plainModel = regexp.MustCompile(`^[A-Za-z0-9._:@/+-]+$`)

// ResumeFrom applies a run a merge request's own CI run judged, listed as
// listed: found by its name in this job's runs folder (RunsDir), never at
// a path the run wrote. Only its proposals are read, and only what the
// merge request's line may do is applied: by a role of this engine's that
// runs on a merge request, proposals of its own kinds, to that merge
// request — its branch, its comments —, never another merge request
// opened but the release's own fix (ADR-0017). Once the merge request's
// branch moved on from the head judged, only what its role names to say
// what that head held is applied (`on-moved-on`: the reviewer's review,
// #375), nothing else.
func ResumeFrom(listed string, from FromMergeRequest) *Result {
	res := &Result{Applied: []string{}, Refused: []string{}, RunDir: listed}
	refuse := func(why string) {
		res.Status = verdict.Block
		res.Findings = append(res.Findings, verdict.Finding{Rule: "run-refused", Where: filepath.Base(listed),
			Message: why + ": nothing of this run is applied"})
	}
	err := func() error {
		name := filepath.Base(listed)
		if !runName.MatchString(name) {
			refuse(fmt.Sprintf("%q names no run's folder", listed))
			return nil
		}
		runDir := filepath.Join(RunsDir(from.Repo), name)
		res.RunDir = runDir
		if link := linkIn(runDir); link != "" {
			refuse("its folder holds a link, " + link)
			return nil
		}
		data, err := os.ReadFile(filepath.Join(runDir, "out", "run.yaml"))
		if err != nil {
			return fmt.Errorf("not a run that reached apply: %w", err)
		}
		var was runState
		if err := yaml.Unmarshal(data, &was); err != nil {
			return err
		}
		now, err := dirDigest(filepath.Join(runDir, "in"))
		if err != nil {
			return err
		}
		if now != was.Digest {
			res.Status = verdict.Block
			res.Findings = append(res.Findings, verdict.Finding{Rule: "input-changed", Message: "the prepared input changed since the run; nothing was applied"})
			return nil
		}
		r, err := role.Load(from.RolesDir, was.Role)
		if err != nil {
			refuse(fmt.Sprintf("it names the role %q, none of this engine's", was.Role))
			return nil
		}
		if !r.Accepts("merge-request") {
			refuse(fmt.Sprintf("the %s does not run on a merge request", r.Name))
			return nil
		}
		intents, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
		if err != nil {
			return err
		}
		cfg, err := role.LoadProjectConfig(from.Repo)
		if err != nil {
			return err
		}
		f, err := forge.Open(from.Forge, from.Repo)
		if err != nil {
			return err
		}
		if f == nil {
			return errors.New("applying a merge request's run needs its forge: pass --forge")
		}
		mr, err := f.MergeRequest(from.ID)
		if err != nil {
			return err
		}
		if !mr.Here {
			refuse(fmt.Sprintf("merge request #%d comes from a fork", from.ID))
			return nil
		}
		for _, in := range intents {
			if why := outOfLine(r, in, issueFor(r.Name, mr.Branch)); why != "" {
				refuse(why)
				return nil
			}
		}
		local := forge.KeepsBranches(f)
		tip, err := fetchTip(from.Repo, mr.Branch, local)
		if err != nil {
			return err
		}
		// Moved on, only what says what the head judged held is applied
		// (`on-moved-on`: the reviewer's review, #375); never a patch nor a
		// label, which the run of the new head applies.
		movedOn := from.Head == "" || tip != from.Head
		if movedOn {
			kept := slices.DeleteFunc(slices.Clone(intents), func(i intent.Intention) bool { return !slices.Contains(r.OnMovedOn, i.Kind) })
			res.Status = verdict.Pass
			if from.Head == "" || len(kept) == 0 {
				res.Findings = append(res.Findings, verdict.Finding{Rule: "moved-on", Level: "warn", Where: mr.Branch,
					Message: fmt.Sprintf("merge request #%d's branch is at %s, not at %s, the head its run judged: nothing applied; the run of its new head applies its own", from.ID, short(tip), short(from.Head))})
				return nil
			}
			res.Findings = append(res.Findings, verdict.Finding{Rule: "moved-on", Level: "warn", Where: mr.Branch,
				Message: fmt.Sprintf("merge request #%d's branch is at %s, not at %s, the head its run judged: only the review of that head applied (%s), at that head; the run of its new head applies its own", from.ID, short(tip), short(from.Head), strings.Join(kindsOf(kept), ", "))})
			intents = kept
		}
		st := runState{Role: r.Name, RolesDir: from.RolesDir, Repo: from.Repo, Forge: from.Forge,
			Target: &forge.Target{Kind: "merge-request", ID: from.ID}, Digest: was.Digest}
		for _, m := range was.Models {
			if plainModel.MatchString(m) {
				st.Models = append(st.Models, m)
			}
		}
		if !movedOn { // what applies once moved on reads no file of the head
			back, err := onHead(from.Repo, from.Head)
			if err != nil {
				return err
			}
			defer back()
		}
		res.Status = verdict.Pass
		if err := applyAll(r, r.MergedSettings(cfg), st, runDir, intents, res); err != nil {
			return err
		}
		for _, h := range res.Handoffs {
			res.Findings = append(res.Findings, deferred(r.Name, h))
		}
		return nil
	}()
	switch {
	case errors.Is(err, forge.ErrUnreachable):
		res.Status = verdict.BlockedExternal
		res.Findings = append(res.Findings, verdict.Finding{Rule: "forge-unreachable", Message: err.Error()})
	case err != nil:
		res.Status = verdict.Block
		res.Findings = append(res.Findings, verdict.Finding{Rule: "engine-error", Message: err.Error()})
	}
	return res
}

// kindsOf are the kinds of intentions, each once, in their order.
func kindsOf(intents []intent.Intention) []string {
	var out []string
	for _, i := range intents {
		if !slices.Contains(out, i.Kind) {
			out = append(out, i.Kind)
		}
	}
	return out
}

// outOfLine says why a proposal is not one a merge request's line may
// apply: a kind its role never proposes, an act on the backlog, a comment
// or a label on an issue but own, the one the role's branch was opened
// for (0: none); "" when it is.
func outOfLine(r *role.Role, in intent.Intention, own int) string {
	switch {
	case !slices.Contains(r.Intentions, in.Kind):
		return fmt.Sprintf("it proposes %s, which the %s never does", in.Kind, r.Name)
	case slices.Contains(backlog.Kinds, in.Kind):
		return fmt.Sprintf("it proposes %s, an act on the backlog, which no merge request's run does", in.Kind)
	}
	m, ok := in.Value.(map[string]any)
	if !ok || (in.Kind != "comment" && in.Kind != "label") || m["issue"] == nil {
		return ""
	}
	if n, ok := m["issue"].(int); ok && own != 0 && n == own {
		return ""
	}
	return fmt.Sprintf("it proposes a %s on another issue than its merge request's", in.Kind)
}

// issueFor is the issue a role's branch was opened for, by the engine's
// name for it (workline/<role>/issue-<n>); 0 when the branch is not one.
func issueFor(role, branch string) int {
	prefix := branchPrefix(role) + "issue-"
	n, err := strconv.Atoi(strings.TrimPrefix(branch, prefix))
	if err != nil || n <= 0 || !strings.HasPrefix(branch, prefix) {
		return 0
	}
	return n
}

// linkIn names the first link in dir, "" when it holds none.
func linkIn(dir string) string {
	found := ""
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.Type()&fs.ModeSymlink != 0 {
			found, _ = filepath.Rel(dir, p)
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// throughLink names the first part of file, under repo, that is a link —
// the file itself or a folder above it —; "" when none is.
func throughLink(repo, file string) string {
	at := repo
	for _, part := range strings.Split(filepath.ToSlash(filepath.Clean(file)), "/") {
		at = filepath.Join(at, part)
		info, err := os.Lstat(at)
		if err != nil {
			return ""
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			rel, _ := filepath.Rel(repo, at)
			return filepath.ToSlash(rel)
		}
	}
	return ""
}

// fetchTip is the commit branch is at, fetched from origin, or, on a forge
// kept in the clone, its local branch.
func fetchTip(repo, branch string, local bool) (string, error) {
	if local {
		return git(repo, nil, "rev-parse", "--verify", "refs/heads/"+branch+"^{commit}")
	}
	if _, err := git(repo, nil, "fetch", "-q", "origin", "refs/heads/"+branch); err != nil {
		return "", fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
	}
	return git(repo, nil, "rev-parse", "FETCH_HEAD^{commit}")
}

// onHead checks head out in repo, detached, for the patches to apply on
// the files they were judged on; back puts repo where it was. A working
// tree holding changes is refused: they would be carried along.
func onHead(repo, head string) (back func(), err error) {
	if changed, err := git(repo, nil, "status", "--porcelain", "--untracked-files=no"); err != nil {
		return nil, err
	} else if changed != "" {
		return nil, errors.New("the working tree holds changes: applying a merge request's run checks out its head")
	}
	orig, err := git(repo, nil, "symbolic-ref", "-q", "--short", "HEAD")
	if err != nil {
		if orig, err = git(repo, nil, "rev-parse", "HEAD"); err != nil {
			return nil, err
		}
	}
	if _, err := git(repo, nil, "checkout", "-q", "--detach", head); err != nil {
		return nil, err
	}
	return func() { git(repo, nil, "checkout", "-q", "-f", orig) }, nil
}
