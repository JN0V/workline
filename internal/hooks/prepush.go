package hooks

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

const zero = "0000000000000000000000000000000000000000"

// PushRanges turns what git gives a pre-push hook on its standard input —
// "<local ref> <local sha> <remote ref> <remote sha>" per ref — into the
// ranges of commits the push sends, as `git log` reads them. A deleted ref
// sends nothing; a new branch sends the commits no ref of that remote holds.
func PushRanges(repo, remote string, stdin io.Reader) ([]string, error) {
	var out []string
	s := bufio.NewScanner(stdin)
	for s.Scan() {
		f := strings.Fields(s.Text())
		if len(f) != 4 {
			continue
		}
		local, remoteSha := f[1], f[3]
		if strings.Trim(local, "0") == "" {
			continue // deleting a ref sends no commit
		}
		if strings.Trim(remoteSha, "0") != "" && known(repo, remoteSha) {
			out = append(out, remoteSha+".."+local)
			continue
		}
		// A new branch, or a remote tip this clone has not fetched: what is new
		// is what no ref of the remote holds.
		first, err := git(repo, "rev-list", "--reverse", local, "--not", "--remotes="+remote)
		if err != nil {
			return nil, err
		}
		if first == "" {
			continue // nothing the remote does not already have
		}
		first = strings.SplitN(first, "\n", 2)[0]
		if parent, err := git(repo, "rev-parse", "-q", "--verify", first+"^"); err == nil && parent != "" {
			out = append(out, parent+".."+local)
		} else {
			out = append(out, local) // the first commit of the history
		}
	}
	return out, s.Err()
}

func known(repo, sha string) bool {
	_, err := git(repo, "cat-file", "-e", sha+"^{commit}")
	return err == nil
}

func git(repo string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return strings.TrimSpace(string(out)), nil
}
