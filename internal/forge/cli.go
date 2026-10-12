package forge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// run calls a forge CLI (gh) in the repository, which tells it which
// project to talk to. A CLI that is missing or cannot authenticate makes the
// forge unreachable, not the role wrong.
func run(repo, bin string, args ...string) ([]byte, error) {
	return runInput(repo, nil, bin, args...)
}

// runInput is run, given what the CLI reads on its standard input: a
// request's JSON body (`gh api --input -`). A write the forge refused for
// who the token is (HTTP 403) is errForbidden; one it answered and did not
// take (HTTP 422), errRefused, with what it said.
func runInput(repo string, input []byte, bin string, args ...string) ([]byte, error) {
	if _, err := exec.LookPath(bin); err != nil {
		return nil, fmt.Errorf("%w: %s is not installed", ErrUnreachable, bin)
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = repo
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errOut.String())
		switch {
		case strings.Contains(msg, "404") || strings.Contains(strings.ToLower(msg), "not found"):
			return nil, fmt.Errorf("%w: %s", errNotFound, msg)
		case strings.Contains(msg, "(HTTP 403)"):
			return nil, fmt.Errorf("%w: %s %s: %s", errForbidden, bin, args[0], msg)
		case strings.Contains(msg, "(HTTP 422)"):
			if said := strings.TrimSpace(out.String()); said != "" {
				msg += ": " + said
			}
			return nil, fmt.Errorf("%w: %s %s: %s", errRefused, bin, args[0], msg)
		}
		return nil, fmt.Errorf("%w: %s %s: %s", ErrUnreachable, bin, args[0], msg)
	}
	return out.Bytes(), nil
}

var errNotFound = fmt.Errorf("not found")

// errForbidden marks a write the forge refused for who the token is: a
// note another user wrote, a role too low.
var errForbidden = fmt.Errorf("%w: forbidden", ErrUnreachable)

func decode(data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%w: unexpected answer: %v", ErrUnreachable, err)
	}
	return nil
}
