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

func (c *command) Release(tag, notes string) error {
	return c.call("release", map[string]any{"tag": tag, "notes": notes}, nil)
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

func (c *command) MergeRequestBranch(id int) (string, bool, error) {
	var a struct {
		Branch string `json:"branch"`
		Here   bool   `json:"here"`
	}
	err := c.call("merge-request-branch", map[string]any{"id": id}, &a)
	return a.Branch, a.Here, err
}
