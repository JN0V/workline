package doctor

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/steer"
)

// boundsDoc is where a person reads how to guard the bounds' job.
const boundsDoc = "docs/gitlab/bounds.md"

// boundsGuard says, on GitLab, what keeps a developer's merge request
// from editing out workline:bounds, the job that holds it to its issue's
// bounds (ADR-0058): GitLab runs a merge request's pipeline from its own
// branch's .gitlab-ci.yml. A code owner's approval of the CI files, on
// the default branch, on Premium and Ultimate; a pipeline execution
// policy on Ultimate, which the doctor cannot read; on Free, the review
// alone. Only when the developer is routed.
func boundsGuard(r *Report, o Options, projectForge string, line *routing.Config) {
	spec := o.Forge
	if spec == "" {
		spec = projectForge
	}
	if spec == "" && onGitLab(o.Repo) {
		spec = "gitlab"
	}
	if spec != "gitlab" || !routesRole(line, "developer") {
		return
	}
	f, err := forge.Open(spec, o.Repo)
	if err != nil {
		return // said by findingsOnMergeRequests
	}
	pg, ok := f.(forge.PipelineGuard)
	if !ok {
		return
	}
	g, err := pg.PipelineGuard()
	if err != nil {
		r.add(Check{Area: "repository", Rule: "bounds-guard-unread", Level: Warn,
			Message: "could not read what keeps a developer's merge request from editing out workline:bounds: " + forge.Said(err),
			Fix:     boundsDoc})
		return
	}
	ci := ciFiles(o.Repo)
	bare, unread := ownerless(o.Repo, ci)
	switch {
	case g.Protected && g.CodeOwnerApproval && len(bare) == 0 && unread == "":
		r.add(Check{Area: "repository", Rule: "bounds-guard-codeowners", Where: strings.Join(ci, ", "), Level: OK,
			Message: "a change to the CI files waits for a code owner's approval on " + g.Branch + ": a merge request cannot edit out workline:bounds unseen"})
	case g.Plan == "ultimate":
		r.add(Check{Area: "repository", Rule: "bounds-guard-policy", Level: Note,
			Message: "GitLab Ultimate: include workline-bounds.gitlab-ci.yml from a pipeline execution policy, which a merge request cannot edit; the doctor does not read policies. Or require a code owner's approval of the CI files",
			Fix:     boundsDoc})
	case g.Plan == "premium":
		var missing []string
		if !g.Protected {
			missing = append(missing, g.Branch+" is not protected")
		} else if !g.CodeOwnerApproval {
			missing = append(missing, g.Branch+"'s protection does not require a code owner's approval")
		}
		switch {
		case unread != "":
			missing = append(missing, "whether CODEOWNERS owns the CI files cannot be told: "+unread)
		case len(bare) > 0:
			missing = append(missing, "CODEOWNERS names no owner whose approval is required for "+strings.Join(bare, ", "))
		}
		r.add(Check{Area: "repository", Rule: "bounds-guard-codeowners-missing", Where: strings.Join(bare, ", "), Level: Warn,
			Message: "GitLab Premium: a developer's merge request runs its own branch's pipeline and can edit out workline:bounds: " + strings.Join(missing, "; "),
			Fix:     "own the CI files in CODEOWNERS, then Settings → Repository → Protected branches: " + g.Branch + ", Code owner approval (" + boundsDoc + ")"})
	case g.Plan == "free":
		r.add(Check{Area: "repository", Rule: "bounds-guard-free", Level: Warn,
			Message: "GitLab Free: a developer's merge request runs its own branch's pipeline and can edit out workline:bounds; nothing but the review keeps it: read each change to the CI files. Premium's code owners, or Ultimate's policies, guard it",
			Fix:     boundsDoc})
	default:
		r.add(Check{Area: "repository", Rule: "bounds-guard-tier-unread", Level: Warn,
			Message: "GitLab's tier could not be read (" + g.PlanUnread + "), nor a code owner's approval of the CI files found: on Ultimate, include workline-bounds.gitlab-ci.yml from a pipeline execution policy; on Premium, require a code owner's approval; on Free, read each change to the CI files",
			Fix:     boundsDoc})
	}
}

// noBypassDoc is where a person reads why nobody bypasses the default
// branch, and the emergency.
const noBypassDoc = "docs/merge-policy.md#nobody-bypasses-main"

// bypassWords says what a token may do past a ruleset, by GitHub's
// current_user_can_bypass.
var bypassWords = map[string]string{
	"pull_requests_only": "on a pull request (`gh pr merge --admin`)",
	"always":             "on a pull request and a push",
	"exempt":             "its rules never applied to it",
}

// bypassGuard says whether this token's user may merge into the default
// branch past its rulesets: nobody does, the person included, so every
// merge goes through its checks and workline/accepted (ADR-0068 §6).
// One API read a ruleset, no agent; read where the forge keeps rulesets
// (GitHub), its rulesets only.
func bypassGuard(r *Report, o Options, projectForge string) {
	spec := o.Forge
	if spec == "" {
		spec = projectForge
	}
	if spec == "" && onGitHub(o.Repo) {
		spec = "github"
	}
	f, err := forge.Open(spec, o.Repo)
	if err != nil {
		return // said by findingsOnMergeRequests
	}
	b, ok := f.(forge.Bypasser)
	if !ok {
		return
	}
	branch, sets, err := b.Bypass("")
	if err != nil {
		where := branch
		if where == "" {
			where = "the default branch"
		}
		r.add(Check{Area: "repository", Rule: "bypass-unread", Where: branch, Level: Note,
			Message: "could not read who may bypass " + where + "'s rulesets: " + forge.Said(err), Fix: noBypassDoc})
		return
	}
	var held []string
	for _, s := range sets {
		if s.CanBypass == "never" {
			held = append(held, fmt.Sprintf("%q", s.Name))
			continue
		}
		how := bypassWords[s.CanBypass]
		if how == "" {
			how = "GitHub does not say it cannot (" + cmp.Or(s.CanBypass, "unsaid") + ")"
		}
		r.add(Check{Area: "repository", Rule: "bypass-allowed", Where: branch, Level: Warn,
			Message: fmt.Sprintf("this token can bypass %s's ruleset %q %s: a merge can skip its checks, workline/accepted included", branch, s.Name, how),
			Fix: fmt.Sprintf("empty its bypass list: %s, or `gh api repos/{owner}/{repo}/rulesets/%d` then PUT it back with \"bypass_actors\": []; the emergency: %s",
				cmp.Or(s.URL, "Settings → Rules → Rulesets"), s.ID, noBypassDoc)})
	}
	if len(held) > 0 && len(held) == len(sets) {
		r.add(Check{Area: "repository", Rule: "no-bypass", Where: branch, Level: OK,
			Message: fmt.Sprintf("nobody bypasses %s's ruleset %s with this token: every merge goes through its checks", branch, strings.Join(held, ", "))})
	}
}

// routesRole says whether the line runs the role on an event.
func routesRole(line *routing.Config, name string) bool {
	if line == nil {
		return false
	}
	for _, steps := range line.Events {
		if slices.ContainsFunc(steps, func(s string) bool { return s == name || strings.HasPrefix(s, name+" ") }) {
			return true
		}
	}
	return false
}

var localInclude = regexp.MustCompile(`local:\s*['"]?/?([^'"}\s,\]]+)`)

// ciFiles are the files that define the pipeline: .gitlab-ci.yml and the
// local files it includes, by name.
func ciFiles(repo string) []string {
	files := []string{".gitlab-ci.yml"}
	data, err := os.ReadFile(filepath.Join(repo, ".gitlab-ci.yml"))
	if err != nil {
		return files
	}
	for _, m := range localInclude.FindAllStringSubmatch(string(data), -1) {
		if !slices.Contains(files, m[1]) {
			files = append(files, m[1])
		}
	}
	return files
}

// ownerless are the files no CODEOWNERS rule gives an owner whose
// approval is required, GitLab's three places read, the first found. In
// each section the last rule that matches decides; an optional section
// (`^[…]`) requires no approval, a section's default owners stand for a
// rule naming none (steer.Codeowners). An exclusion (`!pattern`) the
// doctor does not read: said in unread, never taken for owned.
func ownerless(repo string, files []string) (bare []string, unread string) {
	var co *steer.Codeowners
	for _, p := range []string{"CODEOWNERS", "docs/CODEOWNERS", ".gitlab/CODEOWNERS"} {
		if data, err := os.ReadFile(filepath.Join(repo, p)); err == nil {
			co = steer.ParseCodeowners(p, data)
			break // GitLab reads the first file it finds
		}
	}
	if co == nil {
		return files, ""
	}
	if co.Unread != "" {
		return files, co.Path + " " + co.Unread + " (the doctor does not read it)"
	}
	for _, file := range files {
		if _, required := co.Owns(file); !required {
			bare = append(bare, file)
		}
	}
	return bare, ""
}

// guardedDoc is where a person reads what never merges by the policy
// alone, and why CODEOWNERS is its list.
const guardedDoc = "docs/merge-policy-normal.md#what-never-merges-alone"

// guardedList says which list the engine reads as what never merges by
// the policy alone, nor is the developer's to change (ADR-0068 §4): the
// project's CODEOWNERS, the first found where the forges read it, else
// the files that steer agents. Where the policy merges, or the
// developer runs.
func guardedList(r *Report, o Options, cfg *role.ProjectConfig, line *routing.Config) {
	byPolicy := cfg.Merge != nil && (cfg.Merge.Autonomy == "normal" || cfg.Merge.Autonomy == "enterprising")
	if !byPolicy && !routesRole(line, "developer") {
		return
	}
	for _, p := range steer.CodeownersPlaces {
		data, err := os.ReadFile(filepath.Join(o.Repo, p))
		if err != nil {
			continue
		}
		r.add(Check{Area: "repository", Rule: "guarded-list", Where: p, Level: OK,
			Message: "what never merges by the policy alone, nor is the developer's to change: the paths " + p + " lists, and it; a person's `workline:accepted` merges them"})
		if co := steer.ParseCodeowners(p, data); co.Unread != "" {
			r.add(Check{Area: "repository", Rule: "guarded-list-read-past", Where: p, Level: Note,
				Message: p + " " + co.Unread + ": the engine holds every path its other rules list all the same, the exclusion unread", Fix: guardedDoc})
		}
		return
	}
	r.add(Check{Area: "repository", Rule: "guarded-list-none", Level: Note,
		Message: "no CODEOWNERS: what never merges by the policy alone is the files that steer agents (AGENTS.md, CLAUDE.md…) and the pages they link; CI and workline's config are held apart",
		Fix:     "list the files that define oversight and authority in CODEOWNERS (" + guardedDoc + ")"})
}
