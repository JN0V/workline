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
// sends nothing; a branch, new or not, sends the commits no ref of that
// remote holds.
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
		// What is new is what no ref of the remote holds, nor the tip the push
		// replaces: a branch rebased onto main sends its own commits, not
		// main's, which the remote has already.
		args := []string{"rev-list", "--reverse", local, "--not", "--remotes=" + remote}
		if strings.Trim(remoteSha, "0") != "" && known(repo, remoteSha) {
			args = append(args, remoteSha)
		}
		listed, err := git(repo, args...)
		if err != nil {
			return nil, err
		}
		if listed == "" {
			continue // nothing the remote does not already have
		}
		// The new commits stand on parents the remote has: one, and the range
		// reads base..local; several — main merged into the branch — and it
		// reads `local ^base…`, each one left out, as git log takes it.
		isNew := map[string]bool{}
		for _, c := range strings.Fields(listed) {
			isNew[c] = true
		}
		var bases []string
		seen := map[string]bool{}
		for _, c := range strings.Fields(listed) {
			parents, _ := git(repo, "rev-list", "--parents", "-n", "1", c)
			for _, p := range strings.Fields(parents)[1:] {
				if !isNew[p] && !seen[p] {
					seen[p] = true
					bases = append(bases, p)
				}
			}
		}
		switch len(bases) {
		case 0:
			out = append(out, local) // the first commit of the history
		case 1:
			out = append(out, bases[0]+".."+local)
		default:
			out = append(out, local+" ^"+strings.Join(bases, " ^"))
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
