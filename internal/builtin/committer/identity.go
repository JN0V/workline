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

// Identity checks the author and committer addresses git will record (rng
// empty), or those of every commit of rng. git var gives the effective
// identity, so an address given for one commit (-c user.email, --author) is
// caught too: the one that leaks is usually given on purpose.
func Identity(repo, rng string, s Settings) ([]verdict.Finding, error) {
	allowed, err := allowedIdentities(s)
	if err != nil || len(allowed) == 0 {
		return nil, err
	}
	ok := func(email string) bool {
		for _, re := range allowed {
			if re.MatchString(email) {
				return true
			}
		}
		return false
	}
	refuse := func(where, email, commit string) verdict.Finding {
		return verdict.Finding{Rule: "identity", Where: where,
			Message: fmt.Sprintf("the %s address <%s>%s is not on the allow list; commit with the identity you meant (git config user.email), or add a pattern to the list", where, email, commit)}
	}
	var findings []verdict.Finding
	if rng == "" {
		for _, who := range []string{"author", "committer"} {
			out, err := exec.Command("git", "-C", repo, "var", "GIT_"+strings.ToUpper(who)+"_IDENT").Output()
			if err != nil {
				return nil, fmt.Errorf("git var: %v", err)
			}
			if _, rest, found := strings.Cut(string(out), "<"); found {
				if email, _, _ := strings.Cut(rest, ">"); !ok(email) {
					findings = append(findings, refuse(who, email, ""))
				}
			}
		}
		return findings, nil
	}
	out, err := exec.Command("git", append([]string{"-C", repo, "log", "--format=%h %ae %ce"}, gitrange.Args(rng)...)...).Output()
	if err != nil {
		return nil, fmt.Errorf("cannot read the commits of %s: %v", rng, err)
	}
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Fields(l)
		if len(f) != 3 {
			continue
		}
		if !ok(f[1]) {
			findings = append(findings, refuse("author", f[1], " of commit "+f[0]))
		}
		if !ok(f[2]) {
			findings = append(findings, refuse("committer", f[2], " of commit "+f[0]))
		}
	}
	return findings, nil
}
