package documentalist

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

// identifier is a code span that looks like a name from the code: `Revoke`,
// `auth.RefreshToken`, `refresh_token()`. The last part is what is searched.
var identifier = regexp.MustCompile(`^(?:[A-Za-z_][A-Za-z0-9_]*(?:\.|::))*([A-Za-z_][A-Za-z0-9_]{2,})(?:\(\))?$`)

// named lists the identifiers a doc names in code spans, outside code blocks.
func named(content string) []string {
	seen := map[string]bool{}
	for _, l := range scan(content) {
		if l.code {
			continue
		}
		for _, span := range codeSpan.FindAllString(l.text, -1) {
			if m := identifier.FindStringSubmatch(strings.Trim(span, "`")); m != nil {
				seen[m[1]] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// IdentifiersGone reports the identifiers a doc names that were in the code
// when the doc was last edited, and are gone from it now (DOCER). A name that
// was never in the code is not reported: it may be a product term.
func IdentifiersGone(repo string, docs map[string]string) ([]Problem, error) {
	byCommit := map[string][]string{} // last commit that edited the doc -> docs
	for p := range docs {
		c, err := git(repo, "log", "-1", "--format=%H", "--", p)
		if err != nil {
			return nil, err
		}
		if c != "" { // never committed: nothing to compare with
			byCommit[c] = append(byCommit[c], p)
		}
	}
	var out []Problem
	for c, paths := range byCommit {
		sort.Strings(paths)
		want := map[string]bool{}
		for _, p := range paths {
			for _, id := range named(docs[p]) {
				want[id] = true
			}
		}
		then, err := inCode(repo, c, want)
		if err != nil {
			return nil, err
		}
		now, err := inCode(repo, "HEAD", then)
		if err != nil {
			return nil, err
		}
		for _, p := range paths {
			for _, id := range named(docs[p]) {
				if then[id] && !now[id] {
					out = append(out, Problem{Rule: "identifier-gone", Where: p, Key: "identifier-gone " + p + " " + id,
						Message: fmt.Sprintf("`%s` was in the code when this doc was last edited (%.7s) and is gone now", id, c)})
				}
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Where < out[j].Where })
	return out, nil
}

// inCode returns which of the names appear as whole words in the files of rev
// that are not Markdown.
func inCode(repo, rev string, names map[string]bool) (map[string]bool, error) {
	found := map[string]bool{}
	if len(names) == 0 {
		return found, nil
	}
	var list strings.Builder
	for n := range names {
		list.WriteString(n + "\n")
	}
	cmd := exec.Command("git", "-C", repo, "grep", "-I", "-h", "-o", "-w", "-F", "-f", "-", rev, "--", ".", ":(exclude,glob)**/*.md")
	cmd.Stdin = strings.NewReader(list.String())
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 || errOut.Len() > 0 {
			return nil, fmt.Errorf("git grep at %.7s: %s", rev, strings.TrimSpace(errOut.String()))
		}
		return found, nil // exit 1, nothing on stderr: no match
	}
	for _, n := range strings.Split(out.String(), "\n") {
		if names[n] {
			found[n] = true
		}
	}
	return found, nil
}
