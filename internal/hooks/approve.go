package hooks

import (
	"bufio"
	"fmt"
	"html"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
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

// Approve asks a person, on their terminal, whether the push goes: the
// commits it sends are listed, `v` opens a page showing them in full. The
// terminal is opened here, never given: without one — an agent, an editor's
// button — no person can answer, and the push does not go.
func Approve(repo, remote string, refs []Ref, errOut io.Writer) bool {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintln(errOut, "workline: a person approves every push, and there is no terminal to ask on (an agent, or an editor's button): push stopped.")
		fmt.Fprintln(errOut, "  Push from a terminal, or turn this off in your own config: `approve-push: false`.")
		return false
	}
	defer tty.Close()
	return ask(repo, remote, refs, tty, tty, openPage)
}

// ask lists the push on out and reads the answer from in; view opens a page.
func ask(repo, remote string, refs []Ref, in io.Reader, out io.Writer, view func(string) error) bool {
	n := 0
	for _, r := range refs {
		if r.Delete {
			fmt.Fprintf(out, "workline: deletes %s on %s\n", r.Remote, remote)
			continue
		}
		if r.Range == "" {
			continue
		}
		log, _ := git(repo, "log", "--format=  %h %s", r.Range)
		stat, _ := git(repo, "diff", "--shortstat", rangeBase(repo, r.Range), rangeHead(r.Range))
		c := strings.Count(log, "\n") + 1
		n += c
		fmt.Fprintf(out, "workline: %d commit(s) to %s %s — %s\n%s\n", c, remote, strings.TrimPrefix(r.Remote, "refs/heads/"), strings.TrimSpace(stat), log)
	}
	if n == 0 && !anyDelete(refs) {
		return true // nothing leaves the machine
	}
	lines := bufio.NewReader(in)
	for {
		fmt.Fprint(out, "Push? [y]es / [N]o / [v]iew ")
		if f, ok := in.(*os.File); ok {
			f.SetReadDeadline(time.Now().Add(answerTimeout))
		}
		answer, err := lines.ReadString('\n')
		if err != nil {
			fmt.Fprintln(out, "\nworkline: no answer: push stopped.")
			return false
		}
		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "y", "yes", "o", "oui":
			return true
		case "v", "view", "voir":
			page, err := writePage(repo, remote, refs)
			if err == nil {
				err = view(page)
			}
			if err != nil {
				fmt.Fprintf(out, "workline: the page could not be opened (%v); it is %s\n", err, page)
			} else {
				fmt.Fprintf(out, "workline: opened %s\n", page)
			}
		default:
			fmt.Fprintln(out, "workline: push stopped.")
			return false
		}
	}
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

// openPage opens a file with the system's default program for it.
func openPage(path string) error {
	opener := "xdg-open"
	if runtime.GOOS == "darwin" {
		opener = "open"
	}
	cmd := exec.Command(opener, path)
	cmd.Stdout, cmd.Stderr = nil, nil
	return cmd.Start()
}

// writePage writes one self-contained page showing the push: each commit,
// its message, and its changes, file by file. It needs nothing but a browser.
func writePage(repo, remote string, refs []Ref) (string, error) {
	var b strings.Builder
	b.WriteString(`<!doctype html><html><head><meta charset="utf-8"><title>workline — push</title><style>
:root{--bg:#fff;--fg:#1f2328;--mute:#59636e;--add:#dafbe1;--del:#ffebe9;--hunk:#ddf4ff;--line:#d1d9e0}
@media (prefers-color-scheme:dark){:root{--bg:#0d1117;--fg:#e6edf3;--mute:#9198a1;--add:#12361f;--del:#3d1519;--hunk:#0c2d4a;--line:#3d444d}}
body{background:var(--bg);color:var(--fg);font:14px/1.45 system-ui,sans-serif;margin:0 auto;max-width:1100px;padding:16px}
h1{font-size:20px}h2{font-size:16px;margin:28px 0 4px}.meta,.stat{color:var(--mute);white-space:pre-wrap;font-family:ui-monospace,monospace;font-size:12px}
.msg{white-space:pre-wrap;margin:6px 0 10px}.file{border:1px solid var(--line);border-radius:6px;margin:10px 0;overflow-x:auto}
.name{padding:6px 10px;border-bottom:1px solid var(--line);font-weight:600;font-family:ui-monospace,monospace}
pre{margin:0;font:12px/1.5 ui-monospace,monospace}pre span{display:block;padding:0 10px;white-space:pre}
.a{background:var(--add)}.d{background:var(--del)}.h{background:var(--hunk);color:var(--mute)}
</style></head><body>`)
	fmt.Fprintf(&b, "<h1>Push to %s</h1>", html.EscapeString(remote))
	for _, r := range refs {
		if r.Delete {
			fmt.Fprintf(&b, "<h2>Deletes %s</h2>", html.EscapeString(r.Remote))
			continue
		}
		if r.Range == "" {
			continue
		}
		fmt.Fprintf(&b, "<p class=meta>%s → %s</p>", html.EscapeString(r.Local), html.EscapeString(r.Remote))
		shas, err := git(repo, "rev-list", "--reverse", r.Range)
		if err != nil {
			return "", err
		}
		for _, sha := range strings.Fields(shas) {
			head, _ := git(repo, "show", "-s", "--format=%h %s%n%an, %ad", "--date=format:%Y-%m-%d %H:%M", sha)
			subject, meta, _ := strings.Cut(head, "\n")
			body, _ := git(repo, "show", "-s", "--format=%b", sha)
			stat, _ := git(repo, "show", "--format=", "--stat", sha)
			patch, _ := git(repo, "show", "--format=", "--patch", sha)
			fmt.Fprintf(&b, "<h2>%s</h2><div class=meta>%s</div>", html.EscapeString(subject), html.EscapeString(meta))
			if body != "" {
				fmt.Fprintf(&b, "<div class=msg>%s</div>", html.EscapeString(body))
			}
			fmt.Fprintf(&b, "<div class=stat>%s</div>", html.EscapeString(stat))
			writePatch(&b, patch)
		}
	}
	b.WriteString("</body></html>\n")
	dir, err := git(repo, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "workline", "push.html")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	return path, os.WriteFile(path, []byte(b.String()), 0o644)
}

// writePatch renders a patch file by file, lines coloured as they change.
func writePatch(b *strings.Builder, patch string) {
	open := false
	for _, l := range strings.Split(patch, "\n") {
		switch {
		case strings.HasPrefix(l, "diff --git "):
			if open {
				b.WriteString("</pre></div>")
			}
			name := l[strings.LastIndex(l, " b/")+3:]
			fmt.Fprintf(b, "<div class=file><div class=name>%s</div><pre>", html.EscapeString(name))
			open = true
		case !open || strings.HasPrefix(l, "index ") || strings.HasPrefix(l, "--- ") || strings.HasPrefix(l, "+++ "):
		case strings.HasPrefix(l, "@@"):
			fmt.Fprintf(b, "<span class=h>%s</span>", html.EscapeString(l))
		case strings.HasPrefix(l, "+"):
			fmt.Fprintf(b, "<span class=a>%s</span>", html.EscapeString(l))
		case strings.HasPrefix(l, "-"):
			fmt.Fprintf(b, "<span class=d>%s</span>", html.EscapeString(l))
		default:
			fmt.Fprintf(b, "<span>%s</span>", html.EscapeString(l))
		}
	}
	if open {
		b.WriteString("</pre></div>")
	}
}
