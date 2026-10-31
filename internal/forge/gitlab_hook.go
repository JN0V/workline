package forge

// GitLab starts no pipeline on a label: a project webhook on issue events,
// pointed at the project's own trigger URL, does (ADR-0059, decision 8;
// docs/gitlab-trigger.md). `workline setup` creates it and its trigger
// token, `workline doctor` reads them. Both need a Maintainer's token.
// The token never leaves the calls: the hook holds it as a masked URL
// variable, which GitLab never shows again, and no message says it.

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// TriggerDescription names the trigger token workline creates, as the
// project's Settings → CI/CD → Pipeline trigger tokens show it.
const TriggerDescription = "workline: the issue webhook (ADR-0059)"

// IssueHook is a forge whose issue events can start the project's own
// pipeline through a webhook: GitLab's.
type IssueHook interface {
	// IssueHook reads the project's webhook to its own trigger URL with
	// WORKLINE_TASK=issue, and workline's trigger token.
	IssueHook() (HookState, error)
	// SetIssueHook creates what is missing, the trigger token and the
	// hook, or turns the hook's issue events on: what it did, none when
	// all was there.
	SetIssueHook() ([]string, error)
}

var _ IssueHook = &gitlab{}

// HookState is what the doctor reads of the issue webhook.
type HookState struct {
	Found       bool   // a hook to the project's trigger URL with WORKLINE_TASK=issue
	IssueEvents bool   // it fires on issue events
	Alert       string // GitLab's alert_status: executable, temporarily_disabled, disabled
	Ref         string // the branch its URL starts pipelines on
	Default     string // the project's default branch
	Trigger     bool   // a trigger token named TriggerDescription
}

type gitlabHook struct {
	ID          int    `json:"id"`
	URL         string `json:"url"`
	IssueEvents bool   `json:"issues_events"`
	Alert       string `json:"alert_status"`
}

type gitlabTrigger struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	Token       string `json:"token"`
	Owner       *struct {
		Username string `json:"username"`
	} `json:"owner"`
}

// project reads the project's numeric id, path and default branch.
func (g *gitlab) projectRefs() (id, path, branch string, err error) {
	out, err := g.api("projects/:id")
	if err != nil {
		return "", "", "", err
	}
	var p struct {
		ID     int    `json:"id"`
		Path   string `json:"path_with_namespace"`
		Branch string `json:"default_branch"`
	}
	if err := json.Unmarshal(out, &p); err != nil {
		return "", "", "", fmt.Errorf("%w: unexpected answer: %v", ErrUnreachable, err)
	}
	return strconv.Itoa(p.ID), p.Path, p.Branch, nil
}

// issueHookRef is the branch a hook's URL starts pipelines on, when it is
// the project's own trigger URL with WORKLINE_TASK=issue; ok false else.
func issueHookRef(raw, id, path string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	p := u.EscapedPath()
	var rest string
	found := false
	for _, project := range []string{id, url.PathEscape(path)} {
		if project == "" {
			continue
		}
		if _, after, ok := strings.Cut(p, "/projects/"+project+"/"); ok {
			rest, found = after, true
			break
		}
	}
	if !found || rest != "trigger/pipeline" && !strings.HasSuffix(rest, "/trigger/pipeline") {
		return "", false
	}
	if u.Query().Get("variables[WORKLINE_TASK]") != "issue" {
		return "", false
	}
	ref := u.Query().Get("ref") // /trigger/pipeline?ref=, else the path's /ref/<branch>/
	if r, ok := strings.CutPrefix(strings.TrimSuffix(rest, "/trigger/pipeline"), "ref/"); ok {
		ref, _ = url.PathUnescape(r)
	}
	return ref, true
}

// issueHook is the best of the project's hooks to its own trigger URL
// with WORKLINE_TASK=issue: on issue events, then not disabled, then on
// the default branch; found false when none is.
func issueHook(hooks []gitlabHook, id, path, branch string) (h gitlabHook, ref string, found bool) {
	score := func(h gitlabHook, ref string) int {
		n := 0
		if h.IssueEvents {
			n += 4
		}
		if h.Alert != "disabled" && h.Alert != "temporarily_disabled" {
			n += 2
		}
		if ref == branch {
			n++
		}
		return n
	}
	best := -1
	for _, c := range hooks {
		r, ok := issueHookRef(c.URL, id, path)
		if ok && score(c, r) > best {
			h, ref, found, best = c, r, true, score(c, r)
		}
	}
	return h, ref, found
}

func (g *gitlab) hooks() ([]gitlabHook, error) {
	out, err := g.api("--paginate", "projects/:id/hooks?per_page=100")
	if err != nil {
		return nil, err
	}
	return pages[gitlabHook](out)
}

func (g *gitlab) triggers() ([]gitlabTrigger, error) {
	out, err := g.api("--paginate", "projects/:id/triggers?per_page=100")
	if err != nil {
		return nil, err
	}
	return pages[gitlabTrigger](out)
}

// IssueHook reads the hook and the trigger token, never the token's value.
func (g *gitlab) IssueHook() (HookState, error) {
	var s HookState
	id, path, branch, err := g.projectRefs()
	if err != nil {
		return s, err
	}
	s.Default = branch
	hooks, err := g.hooks()
	if err != nil {
		return s, err
	}
	if h, ref, ok := issueHook(hooks, id, path, branch); ok {
		s.Found, s.IssueEvents, s.Alert, s.Ref = true, h.IssueEvents, h.Alert, ref
	}
	triggers, err := g.triggers()
	if err != nil {
		return s, err
	}
	for _, t := range triggers {
		s.Trigger = s.Trigger || t.Description == TriggerDescription
	}
	return s, nil
}

// SetIssueHook creates the trigger token and the hook when no hook is
// there; a hook there with its issue events off gets them on; one GitLab
// disabled, or on another branch, is said, left for a person to mend: its
// token is not read back. A trigger
// token of workline's whose value GitLab shows (the token's user owns it)
// is used again, else a new one is created.
func (g *gitlab) SetIssueHook() ([]string, error) {
	id, path, branch, err := g.projectRefs()
	if err != nil {
		return nil, err
	}
	if branch == "" {
		return nil, fmt.Errorf("%w: the project has no default branch yet: push one first", errRefused)
	}
	hooks, err := g.hooks()
	if err != nil {
		return nil, err
	}
	if h, ref, ok := issueHook(hooks, id, path, branch); ok {
		var did []string
		if !h.IssueEvents {
			if _, err := g.api("-X", "PUT", fmt.Sprintf("projects/:id/hooks/%d", h.ID), "-f", "url="+h.URL, "-f", "issues_events=true"); err != nil {
				return nil, err
			}
			did = append(did, fmt.Sprintf("turned on the issue events of webhook %d", h.ID))
		}
		if h.Alert == "disabled" || h.Alert == "temporarily_disabled" {
			did = append(did, fmt.Sprintf("webhook %d is there, but GitLab disabled it after its calls failed: Settings → Webhooks, read its recent events, fix the cause, then Enable", h.ID))
		}
		if ref != branch {
			did = append(did, fmt.Sprintf("webhook %d is there, but starts pipelines on %q, not on the default branch %s: Settings → Webhooks, its URL's /ref/<branch>/", h.ID, ref, branch))
		}
		return did, nil
	}
	var did []string
	triggers, err := g.triggers()
	if err != nil {
		return nil, err
	}
	token := ""
	for _, t := range triggers {
		// GitLab shows a token in full to its owner only, its first
		// characters to others.
		if t.Description == TriggerDescription && len(t.Token) >= 20 {
			token = t.Token
			break
		}
	}
	if token == "" {
		out, err := g.api("-X", "POST", "projects/:id/triggers", "-f", "description="+TriggerDescription)
		if err != nil {
			return nil, err
		}
		var t gitlabTrigger
		if err := json.Unmarshal(out, &t); err != nil || t.Token == "" {
			return nil, fmt.Errorf("%w: the new trigger token did not read", ErrUnreachable)
		}
		token = t.Token
		did = append(did, fmt.Sprintf("created the trigger token %d, %q", t.ID, TriggerDescription))
	}
	target := fmt.Sprintf("%s/projects/%s/ref/%s/trigger/pipeline?token={token}&variables[WORKLINE_TASK]=issue",
		g.base, id, url.PathEscape(branch))
	out, err := g.api("-X", "POST", "projects/:id/hooks", "-f", "name=workline", "-f", "url="+target,
		"-f", "issues_events=true", "-f", "push_events=false",
		"-f", "url_variables[][key]=token", "-f", "url_variables[][value]="+token)
	if err != nil && strings.Contains(err.Error(), token) { // an answer quoting the form: the token never said
		return did, fmt.Errorf("%w: %s", errRefused, strings.ReplaceAll(err.Error(), token, "[token]"))
	}
	if err != nil {
		return did, err
	}
	var h gitlabHook
	_ = json.Unmarshal(out, &h)
	return append(did, fmt.Sprintf("created webhook %d on issue events: %s", h.ID, target)), nil
}

// Forbidden says whether err is the forge refusing the token's role or
// scope (HTTP 403).
func Forbidden(err error) bool { return errors.Is(err, errForbidden) }
