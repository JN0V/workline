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

// answerTimeout: a question nobody answers does not approve the push.
var answerTimeout = 10 * time.Minute

// Push is what the person is asked to approve.
type Push struct {
	Repo, Remote string
	Refs         []Ref
	// Suspect: the docs these commits made suspect and nobody judged yet.
	Suspect []string
	// Docs has the docs judged and reviewed on the same terminal; it says
	// whether a docs commit was made, which this push can no longer carry.
	Docs func(in *bufio.Reader, out io.Writer) (committed bool)
	// Via names the channels the person allows, among terminal, editor and
	// dialog, in the order to try them; empty, all of them, in that order.
	Via []string
}

// channels are those the person allows, in the order they are tried.
func (p Push) channels() []string {
	if len(p.Via) == 0 {
		return []string{"terminal", "editor", "dialog"}
	}
	return p.Via
}

// Approve asks a person whether the push goes, on the first channel that
// reaches one (ADR-0008): the terminal the hook opens, `d` included; else the
// editor window the push came from; else a desktop dialog. None given by
// the pushing process: without any, no person can answer, and the push does
// not go.
func Approve(p Push, errOut io.Writer) bool {
	for _, channel := range p.channels() {
		switch channel {
		case "terminal":
			if tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
				defer tty.Close()
				return ask(p, answerReader{tty}, tty, review.Open)
			}
		case "editor":
			if socket, ok := editorSocket("/proc", os.Getppid()); ok {
				fmt.Fprintln(errOut, "workline: asking in the editor window the push came from, at the top…")
				notify("A push waits for you", "Answer at the top of the editor window: y pushes, v shows the commits.")
				return askOnce(p, errOut, func(question, title string) (string, error) {
					return askEditor(socket, question, title, answerTimeout)
				}, review.Open)
			}
		case "dialog":
			if dialog, ok := desktopDialog(); ok {
				fmt.Fprintln(errOut, "workline: asking in a window on the desktop…")
				return askOnce(p, errOut, dialog, review.Open)
			}
		}
	}
	fmt.Fprintln(errOut, "workline: a person approves every push, and none could be asked here (no terminal, editor window or desktop): push stopped.")
	fmt.Fprintln(errOut, "  Push from a terminal or from the editor, or stop asking in your own config: remove `approve-push: true`.")
	return false
}

// answerReader gives the person answerTimeout for each answer, not for the
// whole push: judging the docs (`d`) can take longer than that, and the
// question after it must still wait for them.
type answerReader struct{ tty *os.File }

func (r answerReader) Read(b []byte) (int, error) {
	r.tty.SetReadDeadline(time.Now().Add(answerTimeout))
	return r.tty.Read(b)
}

// describe lists what the push sends, one line a ref then its commits, and
// counts the commits.
func describe(p Push) (string, int) {
	var b strings.Builder
	n := 0
	for _, r := range p.Refs {
		if r.Delete {
			fmt.Fprintf(&b, "workline: deletes %s on %s\n", r.Remote, p.Remote)
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
		fmt.Fprintf(&b, "workline: %d commit(s) to %s %s — %s\n%s\n", c, p.Remote, strings.TrimPrefix(r.Remote, "refs/heads/"), strings.TrimSpace(stat), log)
	}
	return b.String(), n
}

// ask lists the push on out and reads the answer from in; view opens a page.
func ask(p Push, in io.Reader, out io.Writer, view func(string) error) bool {
	listed, n := describe(p)
	fmt.Fprint(out, listed)
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
			showPage(p, out, view)
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

// askOnce asks with one question at a time, in a window: the editor's input
// box or a dialog. What leaves is written to out, where git shows it; the
// window says how many commits, where, and what to answer. Judging the docs
// needs a terminal: the question names them, and `workline docs`.
func askOnce(p Push, out io.Writer, question func(question, title string) (string, error), view func(string) error) bool {
	listed, n := describe(p)
	fmt.Fprint(out, listed)
	if n == 0 && !anyDelete(p.Refs) {
		return true
	}
	var where []string
	for _, r := range p.Refs {
		if r.Range != "" || r.Delete {
			where = append(where, strings.TrimPrefix(r.Remote, "refs/heads/"))
		}
	}
	title := fmt.Sprintf("workline: push %d commit(s) to %s %s?", n, p.Remote, strings.Join(where, ", "))
	if anyDelete(p.Refs) {
		title = fmt.Sprintf("workline: push to %s %s, deleting a branch?", p.Remote, strings.Join(where, ", "))
	}
	if len(p.Suspect) > 0 {
		title += fmt.Sprintf(" %d doc(s) made suspect, not judged yet: `workline docs` judges them.", len(p.Suspect))
		fmt.Fprintf(out, "workline: %d doc(s) these commits made suspect, not judged yet: %s\n", len(p.Suspect), strings.Join(p.Suspect, ", "))
	}
	for {
		answer, err := question("Type y then Enter to push, v to see the commits; Enter alone stops", title)
		if err != nil {
			fmt.Fprintf(out, "workline: no answer (%v): push stopped.\n", err)
			return false
		}
		switch strings.ToLower(answer) {
		case "y", "yes", "o", "oui":
			fmt.Fprintln(out, "workline: approved.")
			return true
		case "v", "view", "voir":
			showPage(p, out, view)
		case "":
			fmt.Fprintln(out, "workline: push stopped: nothing typed (Escape, or Enter on an empty box).")
			return false
		default:
			fmt.Fprintf(out, "workline: push stopped: %q is not y.\n", answer)
			return false
		}
	}
}

// showPage writes the page of the push and opens it.
func showPage(p Push, out io.Writer, view func(string) error) {
	page, err := pushPage(p)
	if err == nil {
		err = view(page)
	}
	if err != nil {
		fmt.Fprintf(out, "workline: the page could not be opened (%v); it is %s\n", err, page)
	} else {
		fmt.Fprintf(out, "workline: opened %s\n", page)
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
