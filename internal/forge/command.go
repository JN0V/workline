package forge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// command is a forge plugged by a command, the way a `cmd:` agent is: for a
// forge workline does not speak natively (Gitea, Forgejo, Bitbucket…). Each
// operation is one run of the command through sh -c, in the repository: one
// JSON request on its input, one JSON answer on its output
// (docs/spec/forge-command.md). An exit other than 0 is a forge unreachable;
// an answer {"error": "…"} a refusal. The command keeps each write
// idempotent, as the interface says.
type command struct {
	repo, script string
}

// commandTimeout bounds one operation: a forge that does not answer in time
// is unreachable, never waited for forever.
const commandTimeout = 2 * time.Minute

// call runs one operation with its arguments and decodes the answer into v.
func (c *command) call(op string, args map[string]any, v any) error {
	req := map[string]any{"operation": op}
	for k, x := range args {
		req[k] = x
	}
	var in bytes.Buffer
	enc := json.NewEncoder(&in)
	enc.SetEscapeHTML(false) // a marker stays <!-- … -->, as written
	if err := enc.Encode(req); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", c.script)
	cmd.Dir = c.repo
	cmd.Env = append(os.Environ(), "WORKLINE_FORGE_OPERATION="+op)
	cmd.Stdin = &in
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("%w: the forge command did not answer %s within %s", ErrUnreachable, op, commandTimeout)
		}
		return fmt.Errorf("%w: the forge command failed on %s: %s", ErrUnreachable, op, lastLine(errOut.String()+out.String()))
	}
	answer := bytes.TrimSpace(out.Bytes())
	if len(answer) == 0 {
		answer = []byte("{}")
	}
	var refused struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(answer, &refused); err != nil {
		return fmt.Errorf("%w: the forge command answered %s with no JSON object: %v", ErrUnreachable, op, err)
	}
	if refused.Error != "" {
		return fmt.Errorf("the forge refused %s: %s", op, refused.Error)
	}
	if v == nil {
		return nil
	}
	return decode(answer, v)
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

type idAnswer struct {
	ID int `json:"id"`
}

func (c *command) Issue(id int) (*Issue, error) {
	var is Issue
	if err := c.call("issue", map[string]any{"id": id}, &is); err != nil {
		return nil, err
	}
	if is.Labels == nil {
		is.Labels = []string{}
	}
	return &is, nil
}

func (c *command) Issues() ([]Issue, error) {
	var a struct {
		Issues []Issue `json:"issues"`
	}
	if err := c.call("issues", nil, &a); err != nil {
		return nil, err
	}
	var out []Issue // only the open ones, by number, whatever the command sent
	for _, is := range a.Issues {
		if !is.Closed {
			if is.Labels == nil {
				is.Labels = []string{}
			}
			out = append(out, is)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// AllIssues asks its own operation, all-issues: a command that does not
// know it refuses it, and the opening fails loud, rather than a closed
// subject being taken for one never seen.
func (c *command) AllIssues() ([]Issue, error) {
	var a struct {
		Issues []Issue `json:"issues"`
	}
	if err := c.call("all-issues", nil, &a); err != nil {
		return nil, err
	}
	for i := range a.Issues {
		if a.Issues[i].Labels == nil {
			a.Issues[i].Labels = []string{}
		}
	}
	sort.Slice(a.Issues, func(i, j int) bool { return a.Issues[i].ID < a.Issues[j].ID })
	return a.Issues, nil
}

func (c *command) Comments(t Target) ([]string, error) {
	var a struct {
		Comments []string `json:"comments"`
	}
	err := c.call("comments", map[string]any{"target": t}, &a)
	return a.Comments, err
}

func (c *command) Close(id, dup int) error {
	args := map[string]any{"id": id}
	if dup > 0 {
		args["duplicate-of"] = dup
	}
	return c.call("close", args, nil)
}

func (c *command) Comment(t Target, body, marker string) error {
	return c.call("comment", map[string]any{"target": t, "body": body, "marker": marker}, nil)
}

func (c *command) Sticky(t Target, body, marker string, create bool) error {
	return c.call("sticky", map[string]any{"target": t, "body": body, "marker": marker, "create": create}, nil)
}

func (c *command) Label(t Target, add, remove []string) error {
	if add == nil {
		add = []string{}
	}
	if remove == nil {
		remove = []string{}
	}
	return c.call("label", map[string]any{"target": t, "add": add, "remove": remove}, nil)
}

func (c *command) OpenIssue(title, body, marker string) (int, error) {
	var a idAnswer
	err := c.call("open-issue", map[string]any{"title": title, "body": body, "marker": marker}, &a)
	return a.ID, err
}

func (c *command) KeepIssue(title, body string, create bool) (int, error) {
	var a idAnswer
	err := c.call("keep-issue", map[string]any{"title": title, "body": body, "create": create}, &a)
	return a.ID, err
}

func (c *command) OpenMergeRequest(branch, base, title, body string) (int, error) {
	var a idAnswer
	err := c.call("open-merge-request", map[string]any{"branch": branch, "base": base, "title": title, "body": body}, &a)
	return a.ID, err
}

func (c *command) OpenMergeRequests(prefix string) ([]string, error) {
	var a struct {
		Branches []string `json:"branches"`
	}
	if err := c.call("open-merge-requests", map[string]any{"prefix": prefix}, &a); err != nil {
		return nil, err
	}
	var out []string // only the prefix's, sorted, whatever the command sent
	for _, b := range a.Branches {
		if strings.HasPrefix(b, prefix) {
			out = append(out, b)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (c *command) MergeRequest(id int) (MergeRequest, error) {
	var a struct {
		Branch string `json:"branch"`
		Base   string `json:"base"`
		Here   bool   `json:"here"`
	}
	err := c.call("merge-request-branch", map[string]any{"id": id}, &a)
	return MergeRequest{Branch: a.Branch, Base: a.Base, Here: a.Here}, err
}

func (c *command) Milestones() ([]string, error) {
	var a struct {
		Milestones []string `json:"milestones"`
	}
	err := c.call("milestones", nil, &a)
	return a.Milestones, err
}

func (c *command) EnsureLabel(name, color, description string) error {
	return c.call("ensure-label", map[string]any{"name": name, "color": color, "description": description}, nil)
}

func (c *command) SetBody(id int, body string) error {
	return c.call("set-body", map[string]any{"id": id, "body": body}, nil)
}

func (c *command) SetMilestone(id int, title string) error {
	return c.call("set-milestone", map[string]any{"id": id, "milestone": title}, nil)
}
