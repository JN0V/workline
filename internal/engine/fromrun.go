package engine

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/release"
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

// plainModel is an agent and model as a commit's trailer names them.
var plainModel = regexp.MustCompile(`^[A-Za-z0-9._:@/+-]+$`)

// ResumeFrom applies a run a merge request's own CI run judged, listed as
// listed: found by its name in this job's runs folder (RunsDir), never at
// a path the run wrote. Only its proposals are read, and only what the
// merge request's line may do is applied: by a role of this engine's that
// runs on a merge request, proposals of its own kinds, to that merge
// request — its branch, its comments —, never another merge request
// opened but the release's own fix (ADR-0017). Nothing is applied once
// the merge request's branch moved on from the head judged.
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
		for _, in := range intents {
			if why := outOfLine(r, in); why != "" {
				refuse(why)
				return nil
			}
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
		local := forge.KeepsBranches(f)
		tip, err := fetchTip(from.Repo, mr.Branch, local)
		if err != nil {
			return err
		}
		if from.Head == "" || tip != from.Head {
			res.Status = verdict.Pass
			res.Findings = append(res.Findings, verdict.Finding{Rule: "moved-on", Level: "warn", Where: mr.Branch,
				Message: fmt.Sprintf("merge request #%d's branch is at %s, not at %s, the head its run judged: nothing applied; the run of its new head applies its own", from.ID, short(tip), short(from.Head))})
			return nil
		}
		st := runState{Role: r.Name, RolesDir: from.RolesDir, Repo: from.Repo, Forge: from.Forge,
			Target: &forge.Target{Kind: "merge-request", ID: from.ID}, Digest: was.Digest,
			PushToMergeRequest: true}
		for _, m := range was.Models {
			if plainModel.MatchString(m) {
				st.Models = append(st.Models, m)
			}
		}
		// A release tool's pull request, as this job's settings say it, not
		// as the run did: its fix goes to a merge request of its own.
		if r.Accepts("release") {
			rs, err := release.Load(from.Repo)
			switch {
			case err != nil:
				res.Findings = append(res.Findings, verdict.Finding{Rule: "release-unknown", Level: "warn", Message: err.Error()})
			case rs.IsBranch(mr.Branch):
				st.Release = mr.Branch
				if st.releaseHeld, err = releaseNotOwn(f, mr.Branch); err != nil {
					return err
				}
			}
		}
		back, err := onHead(from.Repo, from.Head)
		if err != nil {
			return err
		}
		defer back()
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

// releaseNotOwn says why a release tool's branch is not workline's App's
// alone, "" when it is: whoever else pushed to it could change its run, and
// that run's fix would become a merge request the App opens, workline's own
// work at the merge policy's `normal`. With the job's own token, the fix's
// merge request is never workline's own: nothing to check.
func releaseNotOwn(f forge.Forge, branch string) (string, error) {
	app := strings.TrimSpace(os.Getenv("WORKLINE_APP"))
	if app == "" {
		return "", nil
	}
	who, err := forge.BranchPushers(f, branch)
	if err != nil {
		return "", err
	}
	if who == nil {
		return fmt.Sprintf("who pushed to %s since it was created, the forge does not say; only workline's App may have, for its run's fix to be a merge request of the App's", branch), nil
	}
	for _, p := range who {
		if p != app+"[bot]" {
			name := "a hand the forge does not name"
			if p != "" {
				name = "@" + p
			}
			return fmt.Sprintf("%s pushed to %s, not workline's App alone: its run's fix is not opened as the App's merge request", name, branch), nil
		}
	}
	return "", nil
}

// outOfLine says why a proposal is not one a merge request's line may
// apply: a kind its role never proposes, an act on the backlog, a comment
// on another issue; "" when it is.
func outOfLine(r *role.Role, in intent.Intention) string {
	switch {
	case !slices.Contains(r.Intentions, in.Kind):
		return fmt.Sprintf("it proposes %s, which the %s never does", in.Kind, r.Name)
	case slices.Contains(backlog.Kinds, in.Kind):
		return fmt.Sprintf("it proposes %s, an act on the backlog, which no merge request's run does", in.Kind)
	}
	if m, ok := in.Value.(map[string]any); ok && in.Kind == "comment" && m["issue"] != nil {
		return "it proposes a comment on another issue than its merge request"
	}
	return ""
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
