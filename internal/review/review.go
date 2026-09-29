// Package review shows a person what the machine proposes, for them to
// accept or refuse: a page any browser opens, and a question on the terminal.
package review

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
)

// Section is one part of a page: a commit, or a doc changed.
type Section struct {
	Heading, Meta, Message, Stat, Patch string
}

// Page writes one self-contained page, in the repository's git folder, and
// returns its path. It needs nothing but a browser.
func Page(repo, name, title, intro string, sections []Section) (string, error) {
	var b strings.Builder
	b.WriteString(`<!doctype html><html><head><meta charset="utf-8"><title>workline — ` + html.EscapeString(title) + `</title><style>
:root{--bg:#fff;--fg:#1f2328;--mute:#59636e;--add:#dafbe1;--del:#ffebe9;--hunk:#ddf4ff;--line:#d1d9e0}
@media (prefers-color-scheme:dark){:root{--bg:#0d1117;--fg:#e6edf3;--mute:#9198a1;--add:#12361f;--del:#3d1519;--hunk:#0c2d4a;--line:#3d444d}}
body{background:var(--bg);color:var(--fg);font:14px/1.45 system-ui,sans-serif;margin:0 auto;max-width:1100px;padding:16px}
h1{font-size:20px}h2{font-size:16px;margin:28px 0 4px}.meta,.stat{color:var(--mute);white-space:pre-wrap;font-family:ui-monospace,monospace;font-size:12px}
.msg{white-space:pre-wrap;margin:6px 0 10px}.file{border:1px solid var(--line);border-radius:6px;margin:10px 0;overflow-x:auto}
.name{padding:6px 10px;border-bottom:1px solid var(--line);font-weight:600;font-family:ui-monospace,monospace}
pre{margin:0;font:12px/1.5 ui-monospace,monospace}pre span{display:block;padding:0 10px;white-space:pre}
.a{background:var(--add)}.d{background:var(--del)}.h{background:var(--hunk);color:var(--mute)}
</style></head><body>`)
	fmt.Fprintf(&b, "<h1>%s</h1>", html.EscapeString(title))
	if intro != "" {
		fmt.Fprintf(&b, "<p class=meta>%s</p>", html.EscapeString(intro))
	}
	for _, s := range sections {
		fmt.Fprintf(&b, "<h2>%s</h2>", html.EscapeString(s.Heading))
		if s.Meta != "" {
			fmt.Fprintf(&b, "<div class=meta>%s</div>", html.EscapeString(s.Meta))
		}
		if s.Message != "" {
			fmt.Fprintf(&b, "<div class=msg>%s</div>", html.EscapeString(s.Message))
		}
		if s.Stat != "" {
			fmt.Fprintf(&b, "<div class=stat>%s</div>", html.EscapeString(s.Stat))
		}
		writePatch(&b, s.Patch)
	}
	b.WriteString("</body></html>\n")
	dir, err := gitOut(repo, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "workline", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	return path, os.WriteFile(path, []byte(b.String()), 0o644)
}

// Choice asks a question on out and reads a line from in: the lower-cased
// answer, or "" with no answer.
func Choice(in *bufio.Reader, out io.Writer, question string) (string, bool) {
	fmt.Fprint(out, question)
	answer, err := in.ReadString('\n')
	if err != nil && answer == "" {
		return "", false
	}
	return strings.ToLower(strings.TrimSpace(answer)), true
}

func gitOut(repo string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// Open opens a file with the system's default program for it.
func Open(path string) error {
	opener := "xdg-open"
	if runtime.GOOS == "darwin" {
		opener = "open"
	}
	cmd := exec.Command(opener, path)
	cmd.Stdout, cmd.Stderr = nil, nil
	return cmd.Start()
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

// Docs has a person keep or drop each doc changed in the working tree, one at
// a time, a page showing the change on `v`; those kept are committed alone,
// in one `docs:` commit, whatever else is staged. It says whether it
// committed.
func Docs(repo string, docs []string, in *bufio.Reader, out io.Writer, view func(string) error) (bool, error) {
	var kept []string
	for _, d := range docs {
		stat, _ := gitOut(repo, "diff", "--stat", "--", d)
		fmt.Fprintf(out, "%s\n", strings.TrimSpace(stat))
	ask:
		answer, ok := Choice(in, out, fmt.Sprintf("Keep this change to %s? [y]es / [n]o / [v]iew ", d))
		switch {
		case !ok:
			fmt.Fprintln(out, "\nworkline: no answer: the changes stay in your working tree, not committed.")
			return false, nil
		case answer == "y" || answer == "yes" || answer == "o" || answer == "oui":
			kept = append(kept, d)
		case answer == "v" || answer == "view" || answer == "voir":
			patch, _ := gitOut(repo, "diff", "--", d)
			page, err := Page(repo, "docs.html", "A change to "+d, "proposed by the documentalist, from what the code changed", []Section{{Heading: d, Stat: stat, Patch: patch}})
			if err == nil {
				err = view(page)
			}
			if err != nil {
				fmt.Fprintf(out, "workline: the page could not be opened (%v); it is %s\n", err, page)
			}
			goto ask
		default:
			if _, err := gitOut(repo, "checkout", "--", d); err != nil {
				return false, err
			}
			fmt.Fprintf(out, "workline: %s left as it was\n", d)
		}
	}
	if len(kept) == 0 {
		return false, nil
	}
	msg := fmt.Sprintf("docs: bring %d docs up to date with the code", len(kept))
	if one := "docs: bring " + kept[0] + " up to date with the code"; len(kept) == 1 && len(one) <= 72 {
		msg = one
	}
	cmd := exec.Command("git", append([]string{"-C", repo, "commit", "-q", "-m", msg, "--only", "--"}, kept...)...)
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("committing the docs: %w", err)
	}
	fmt.Fprintf(out, "workline: committed %s\n", msg)
	return true, nil
}
