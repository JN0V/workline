package hooks

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/review"
)

// Ref is one ref a push updates, as git gives it to a pre-push hook.
type Ref struct {
	Local, Remote string // the ref names
	Range         string // the commits it sends, as `git log` reads them; empty when it sends none
	Delete        bool   // the push deletes the remote ref
}

// PushedRefs reads what git gives a pre-push hook on its standard input.
func PushedRefs(repo, remote string, stdin io.Reader) ([]Ref, error) {
	var out []Ref
	s := bufio.NewScanner(stdin)
	for s.Scan() {
		f := strings.Fields(s.Text())
		if len(f) != 4 {
			continue
		}
		r := Ref{Local: f[0], Remote: f[2]}
		if strings.Trim(f[1], "0") == "" {
			r.Delete = true
			out = append(out, r)
			continue
		}
		ranges, err := PushRanges(repo, remote, strings.NewReader(s.Text()+"\n"))
		if err != nil {
			return nil, err
		}
		if len(ranges) > 0 {
			r.Range = ranges[0]
		}
		out = append(out, r)
	}
	return out, s.Err()
}

// answerTimeout: a push nobody answers is not approved.
const answerTimeout = 10 * time.Minute

// Push is what the person is asked to approve.
type Push struct {
	Repo, Remote string
	Refs         []Ref
	// Suspect: the docs these commits made suspect and nobody judged yet.
	Suspect []string
	// Docs has the docs judged and reviewed on the same terminal; it says
	// whether a docs commit was made, which this push can no longer carry.
	Docs func(in *bufio.Reader, out io.Writer) (committed bool)
}

// Approve asks a person, on their terminal, whether the push goes: the
// commits it sends are listed, `v` opens a page showing them in full. The
// terminal is opened here, never given: without one — an agent, an editor's
// button — no person can answer, and the push does not go.
func Approve(p Push, errOut io.Writer) bool {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintln(errOut, "workline: a person approves every push, and there is no terminal to ask on (an agent, or an editor's button): push stopped.")
		fmt.Fprintln(errOut, "  Push from a terminal, or turn this off in your own config: `approve-push: false`.")
		return false
	}
	defer tty.Close()
	tty.SetReadDeadline(time.Now().Add(answerTimeout))
	return ask(p, tty, tty, review.Open)
}

// ask lists the push on out and reads the answer from in; view opens a page.
func ask(p Push, in io.Reader, out io.Writer, view func(string) error) bool {
	n := 0
	for _, r := range p.Refs {
		if r.Delete {
			fmt.Fprintf(out, "workline: deletes %s on %s\n", r.Remote, p.Remote)
			continue
		}
		if r.Range == "" {
			continue
		}
		log, _ := git(p.Repo, "log", "--format=%h %s", r.Range)
		log = "  " + strings.ReplaceAll(log, "\n", "\n  ")
		stat, _ := git(p.Repo, "diff", "--shortstat", rangeBase(p.Repo, r.Range), rangeHead(r.Range))
		c := strings.Count(log, "\n") + 1
		n += c
		fmt.Fprintf(out, "workline: %d commit(s) to %s %s — %s\n%s\n", c, p.Remote, strings.TrimPrefix(r.Remote, "refs/heads/"), strings.TrimSpace(stat), log)
	}
	if n == 0 && !anyDelete(p.Refs) {
		return true // nothing leaves the machine
	}
	question := "Push? [y]es / [N]o / [v]iew "
	if len(p.Suspect) > 0 {
		fmt.Fprintf(out, "workline: %d doc(s) these commits made suspect, not judged yet: %s\n", len(p.Suspect), strings.Join(p.Suspect, ", "))
		if p.Docs != nil {
			question = "Push? [y]es / [N]o / [v]iew / [d]ocs "
		}
	}
	lines := bufio.NewReader(in)
	for {
		answer, ok := review.Choice(lines, out, question)
		if !ok {
			fmt.Fprintln(out, "\nworkline: no answer: push stopped.")
			return false
		}
		switch answer {
		case "y", "yes", "o", "oui":
			return true
		case "v", "view", "voir":
			page, err := pushPage(p)
			if err == nil {
				err = view(page)
			}
			if err != nil {
				fmt.Fprintf(out, "workline: the page could not be opened (%v); it is %s\n", err, page)
			} else {
				fmt.Fprintf(out, "workline: opened %s\n", page)
			}
		case "d", "docs":
			if p.Docs == nil || len(p.Suspect) == 0 {
				fmt.Fprintln(out, "workline: no doc to judge.")
				continue
			}
			if p.Docs(lines, out) {
				fmt.Fprintln(out, "workline: the docs are committed; this push can no longer carry them: push again.")
				return false
			}
		default:
			fmt.Fprintln(out, "workline: push stopped.")
			return false
		}
	}
}

// pushPage shows each commit of the push, its message and its changes.
func pushPage(p Push) (string, error) {
	var sections []review.Section
	for _, r := range p.Refs {
		if r.Delete {
			sections = append(sections, review.Section{Heading: "Deletes " + r.Remote})
			continue
		}
		if r.Range == "" {
			continue
		}
		shas, err := git(p.Repo, "rev-list", "--reverse", r.Range)
		if err != nil {
			return "", err
		}
		for _, sha := range strings.Fields(shas) {
			head, _ := git(p.Repo, "show", "-s", "--format=%h %s%n%an, %ad · "+r.Local+" → "+r.Remote, "--date=format:%Y-%m-%d %H:%M", sha)
			subject, meta, _ := strings.Cut(head, "\n")
			body, _ := git(p.Repo, "show", "-s", "--format=%b", sha)
			stat, _ := git(p.Repo, "show", "--format=", "--stat", sha)
			patch, _ := git(p.Repo, "show", "--format=", "--patch", sha)
			sections = append(sections, review.Section{Heading: subject, Meta: meta, Message: body, Stat: stat, Patch: patch})
		}
	}
	return review.Page(p.Repo, "push.html", "Push to "+p.Remote, "", sections)
}

func anyDelete(refs []Ref) bool {
	for _, r := range refs {
		if r.Delete {
			return true
		}
	}
	return false
}

// rangeBase and rangeHead split "a..b"; a range of one commit (the first of
// a history) is compared with the empty tree.
func rangeBase(repo, rng string) string {
	if base, _, ok := strings.Cut(rng, ".."); ok {
		return base
	}
	empty, _ := git(repo, "hash-object", "-t", "tree", "/dev/null")
	return empty
}

func rangeHead(rng string) string {
	if _, head, ok := strings.Cut(rng, ".."); ok {
		return head
	}
	return rng
}
