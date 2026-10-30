package doctor

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
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
	if spec != "gitlab" || !routesDeveloper(line) {
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
	bare := ownerless(o.Repo, ci)
	switch {
	case g.Protected && g.CodeOwnerApproval && len(bare) == 0:
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
		if len(bare) > 0 {
			missing = append(missing, "CODEOWNERS names no owner for "+strings.Join(bare, ", "))
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

// routesDeveloper says whether the line runs the developer on an event.
func routesDeveloper(line *routing.Config) bool {
	if line == nil {
		return false
	}
	for _, steps := range line.Events {
		if slices.ContainsFunc(steps, func(s string) bool { return s == "developer" || strings.HasPrefix(s, "developer ") }) {
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

// ownerless are the files no CODEOWNERS rule gives an owner, GitLab's
// three places read, the last rule that matches a file deciding.
func ownerless(repo string, files []string) []string {
	var rules [][2]string // pattern, whether owned
	for _, p := range []string{"CODEOWNERS", "docs/CODEOWNERS", ".gitlab/CODEOWNERS"} {
		data, err := os.ReadFile(filepath.Join(repo, p))
		if err != nil {
			continue
		}
		for l := range strings.Lines(string(data)) {
			l = strings.TrimSpace(l)
			if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "[") || strings.HasPrefix(l, "^[") {
				continue
			}
			f := strings.Fields(l)
			owned := "no"
			if len(f) > 1 {
				owned = "yes"
			}
			rules = append(rules, [2]string{f[0], owned})
		}
		break // GitLab reads the first file it finds
	}
	var bare []string
	for _, file := range files {
		owned := false
		for _, rl := range rules {
			if codeownersMatch(rl[0], file) {
				owned = rl[1] == "yes"
			}
		}
		if !owned {
			bare = append(bare, file)
		}
	}
	return bare
}

// codeownersMatch says whether a CODEOWNERS pattern takes the file: a
// leading `/` anchors it at the root, a trailing `/` takes a folder's
// files, a name without `/` matches at any depth, `*` and `**` glob.
func codeownersMatch(pattern, file string) bool {
	if pattern == "*" || pattern == "/**" || pattern == "**" {
		return true
	}
	anchored := strings.HasPrefix(pattern, "/")
	p := strings.TrimPrefix(pattern, "/")
	if strings.HasSuffix(p, "/") {
		p += "**"
	}
	if !anchored && !strings.Contains(strings.TrimSuffix(p, "/**"), "/") {
		p = "**/" + p
	}
	return globMatch(p, file)
}

// globMatch matches a path to a pattern whose `**` spans folders.
func globMatch(pattern, name string) bool {
	if !strings.Contains(pattern, "**") {
		ok, _ := path.Match(pattern, name)
		return ok
	}
	before, after, _ := strings.Cut(pattern, "**")
	if !strings.HasPrefix(name, before) {
		return false
	}
	rest := strings.TrimPrefix(name, before)
	after = strings.TrimPrefix(after, "/")
	if after == "" {
		return true
	}
	for i := 0; i <= len(rest); i++ {
		if i == 0 || rest[i-1] == '/' {
			if globMatch(after, rest[i:]) {
				return true
			}
		}
	}
	return false
}
