package forge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// run calls a forge CLI (gh or glab) in the repository, which tells it which
// project to talk to. A CLI that is missing or cannot authenticate makes the
// forge unreachable, not the role wrong.
func run(repo, bin string, args ...string) ([]byte, error) {
	if _, err := exec.LookPath(bin); err != nil {
		return nil, fmt.Errorf("%w: %s is not installed", ErrUnreachable, bin)
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = repo
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errOut.String())
		if strings.Contains(msg, "404") || strings.Contains(strings.ToLower(msg), "not found") {
			return nil, fmt.Errorf("%w: %s", errNotFound, msg)
		}
		return nil, fmt.Errorf("%w: %s %s: %s", ErrUnreachable, bin, args[0], msg)
	}
	return out.Bytes(), nil
}

var errNotFound = fmt.Errorf("not found")

func decode(data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%w: unexpected answer: %v", ErrUnreachable, err)
	}
	return nil
}
