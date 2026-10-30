package forge

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Guard is what keeps a GitLab merge request from editing out the jobs
// that check it (ADR-0058): GitLab runs its pipeline from its source
// branch's .gitlab-ci.yml. On Ultimate, a pipeline execution policy runs
// them from elsewhere; on Premium, a code owner's approval guards the CI
// files; on Free, nothing but the review.
type Guard struct {
	// Plan is the namespace's tier: free, premium or ultimate; "" when
	// GitLab does not say it (a self-managed instance's, or a token that
	// may not read the namespace), PlanUnread saying why.
	Plan       string
	PlanUnread string
	// Branch is the default branch; Protected, whether a rule protects
	// it; CodeOwnerApproval, whether that rule asks a code owner's
	// approval of a change to the files they own.
	Branch            string
	Protected         bool
	CodeOwnerApproval bool
}

// PipelineGuard is a forge whose merge requests run a pipeline their own
// branch defines: what guards it is read from the project.
type PipelineGuard interface {
	PipelineGuard() (Guard, error)
}

var _ PipelineGuard = &gitlab{}

// PipelineGuard reads the project's default branch and namespace, the
// namespace's plan, and the default branch's protection. Only the
// project's read fails it; a plan unread is said in the Guard.
func (g *gitlab) PipelineGuard() (Guard, error) {
	out, err := g.api("projects/:id")
	if err != nil {
		return Guard{}, err
	}
	var p struct {
		Default   string `json:"default_branch"`
		Namespace struct {
			ID int `json:"id"`
		} `json:"namespace"`
	}
	if err := decode(out, &p); err != nil {
		return Guard{}, err
	}
	gd := Guard{Branch: p.Default}
	if out, err := g.api(fmt.Sprintf("namespaces/%d", p.Namespace.ID)); err != nil {
		gd.PlanUnread = Said(err)
	} else {
		var n struct {
			Plan string `json:"plan"`
		}
		if json.Unmarshal(out, &n) != nil || tier(n.Plan) == "" {
			gd.PlanUnread = fmt.Sprintf("the namespace's plan is %q, no tier GitLab.com names", n.Plan)
		}
		gd.Plan = tier(n.Plan)
	}
	if gd.Branch == "" {
		return gd, nil
	}
	out, err = g.api("projects/:id/protected_branches/" + url.PathEscape(gd.Branch))
	switch {
	case errors.Is(err, errNotFound): // not protected
		return gd, nil
	case err != nil:
		return Guard{}, err
	}
	var b struct {
		CodeOwner bool `json:"code_owner_approval_required"`
	}
	if err := decode(out, &b); err != nil {
		return Guard{}, err
	}
	gd.Protected, gd.CodeOwnerApproval = true, b.CodeOwner
	return gd, nil
}

// tier reads a GitLab.com plan as its tier: the old names (bronze,
// silver, gold), a trial's and the open source program's among them.
// A self-managed instance says "default", no tier.
func tier(plan string) string {
	switch p := strings.ToLower(plan); {
	case strings.Contains(p, "ultimate"), strings.Contains(p, "gold"), p == "opensource":
		return "ultimate"
	case strings.Contains(p, "premium"), strings.Contains(p, "silver"), p == "bronze":
		return "premium"
	case p == "free":
		return "free"
	}
	return ""
}
