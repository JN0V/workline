package committer

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/JN0V/workline/internal/gitrange"
	"github.com/JN0V/workline/internal/verdict"
)

// allowedIdentities is the allow list: the project's `allowed-identities`
// setting, and the user's own list, one pattern a line (# comments), kept in
// <config folder>/workline/allowed-identities, never in the repository. Both
// empty, identities are not checked.
func allowedIdentities(s Settings) ([]*regexp.Regexp, error) {
	patterns := append([]string{}, s.AllowedIdentities...)
	if dir, err := os.UserConfigDir(); err == nil {
		if f, err := os.Open(filepath.Join(dir, "workline", "allowed-identities")); err == nil {
			defer f.Close()
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				if l := strings.TrimSpace(sc.Text()); l != "" && !strings.HasPrefix(l, "#") {
					patterns = append(patterns, l)
				}
			}
		}
	}
	var out []*regexp.Regexp
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("allowed identity %q: %v", p, err)
		}
		out = append(out, re)
	}
	return out, nil
}

// forgeIdentities checks the project's `forge-identities`: the addresses its
// forge commits with (its App's bot, GitHub's web merges), each written
// whole and matched exactly, case included. They are allowed beside the
// allow list but never turn the check on alone; naming one bot never lets
// every bot in. An entry that reads as a pattern is a mistake, said.
func forgeIdentities(s Settings) error {
	for _, f := range s.ForgeIdentities {
		if !strings.Contains(f, "@") || strings.ContainsAny(f, "^$*?\\|(){} \t") {
			return fmt.Errorf("forge identity %q: an address written whole (12345+app[bot]@users.noreply.github.com), not a pattern", f)
		}
	}
	return nil
}

// Identity checks the author and committer addresses git will record (rng
// empty), or those of every commit of rng. git var gives the effective
// identity, so an address given for one commit (-c user.email, --author) is
// caught too: the one that leaks is usually given on purpose.
func Identity(repo, rng string, s Settings) ([]verdict.Finding, error) {
	if err := forgeIdentities(s); err != nil {
		return nil, err
	}
	allowed, err := allowedIdentities(s)
	if err != nil || len(allowed) == 0 {
		return nil, err
	}
	ok := func(email string) bool {
		for _, f := range s.ForgeIdentities {
			if email == f {
				return true
			}
		}
		for _, re := range allowed {
			if re.MatchString(email) {
				return true
			}
		}
		return false
	}
	var findings []verdict.Finding
	// check refuses an address off the list, and an empty one whatever the
	// list holds: it names no one, so no pattern is meant to let it in.
	check := func(where, email, commit string) {
		if email == "" {
			findings = append(findings, verdict.Finding{Rule: "identity", Where: where,
				Message: fmt.Sprintf("the %s address%s is empty; commit with the identity you meant (git config user.email)", where, commit)})
		} else if !ok(email) {
			findings = append(findings, verdict.Finding{Rule: "identity", Where: where,
				Message: fmt.Sprintf("the %s address <%s>%s is not on the allow list; commit with the identity you meant (git config user.email), or add a pattern to the list (an address the forge commits with: the project's forge-identities)", where, email, commit)})
		}
	}
	if rng == "" {
		for _, who := range []string{"author", "committer"} {
			out, err := exec.Command("git", "-C", repo, "var", "GIT_"+strings.ToUpper(who)+"_IDENT").Output()
			if err != nil {
				return nil, fmt.Errorf("git var: %v", err)
			}
			if _, rest, found := strings.Cut(string(out), "<"); found {
				email, _, _ := strings.Cut(rest, ">")
				check(who, email, "")
			}
		}
		return findings, nil
	}
	// A tab apart, an empty address stays a field of its own.
	out, err := exec.Command("git", append([]string{"-C", repo, "log", "--format=%h%x09%ae%x09%ce"}, gitrange.Args(rng)...)...).Output()
	if err != nil {
		return nil, fmt.Errorf("cannot read the commits of %s: %v", rng, err)
	}
	for _, l := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if l == "" {
			continue
		}
		f := strings.Split(l, "\t")
		if len(f) != 3 {
			return nil, fmt.Errorf("cannot read the author and committer of %q", l)
		}
		check("author", f[1], " of commit "+f[0])
		check("committer", f[2], " of commit "+f[0])
	}
	return findings, nil
}
