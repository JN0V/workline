package reviewer

import (
	"fmt"
	"strings"
)

// contextLines is the context around a hunk, as `git diff -U5` gives it.
const contextLines = 5

// withFunctions is the diff of a file past code-lines-max, each hunk
// widened to the function it is in (#1334): the outermost block around
// a changed line that holds at most max lines, found by indentation, so a
// long ViewModel is read by whole functions, not by 5 lines around the
// change. ok is false when the file's diff cannot be made so.
func withFunctions(repo, from, to string, f File, max int) (string, bool) {
	if max <= 0 {
		return "", false
	}
	out, err := git(repo, append([]string{"diff", "--unified=1000000000", "--no-color", "--no-ext-diff", "--find-renames", from, to, "--"}, pathspec([]File{f})...)...)
	if err != nil {
		return "", false
	}
	head, rest, ok := strings.Cut(out, "\n@@")
	if !ok {
		return "", false
	}
	_, rest, ok = strings.Cut(rest, "\n")
	if !ok {
		return "", false
	}
	body := strings.Split(strings.TrimSuffix(rest, "\n"), "\n")
	n := len(body)
	keep := make([]bool, n)
	changed := false
	for i, l := range body {
		if l == "" || (l[0] != '+' && l[0] != '-') {
			continue
		}
		changed = true
		for j := max0(i - contextLines); j <= min(n-1, i+contextLines); j++ {
			keep[j] = true
		}
		if s, e, found := enclosing(body, i, max); found {
			for j := s; j <= e; j++ {
				keep[j] = true
			}
		}
	}
	if !changed {
		return "", false
	}
	var b strings.Builder
	b.WriteString(head + "\n")
	// old and new line numbers of each body line
	oldAt, newAt := make([]int, n+1), make([]int, n+1)
	o, nw := 1, 1
	for i, l := range body {
		oldAt[i], newAt[i] = o, nw
		switch {
		case l == "" || l[0] == ' ':
			o++
			nw++
		case l[0] == '-':
			o++
		case l[0] == '+':
			nw++
		}
	}
	for i := 0; i < n; {
		if !keep[i] {
			i++
			continue
		}
		j := i
		for j < n && (keep[j] || strings.HasPrefix(body[j], `\`)) {
			j++
		}
		oc, nc := 0, 0
		for _, l := range body[i:j] {
			switch {
			case l == "" || l[0] == ' ':
				oc++
				nc++
			case l[0] == '-':
				oc++
			case l[0] == '+':
				nc++
			}
		}
		os, ns := oldAt[i], newAt[i]
		if oc == 0 {
			os--
		}
		if nc == 0 {
			ns--
		}
		fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", os, oc, ns, nc)
		b.WriteString(strings.Join(body[i:j], "\n") + "\n")
		i = j
	}
	return b.String(), true
}

func max0(i int) int { return max(i, 0) }

// indentOf is the width of a diff body line's indentation, its first
// character being the diff's mark; -1 for a blank line.
func indentOf(l string) int {
	if l == "" {
		return -1
	}
	t := l[1:]
	if strings.TrimSpace(t) == "" {
		return -1
	}
	w := 0
	for _, c := range t {
		switch c {
		case ' ':
			w++
		case '\t':
			w += 4
		default:
			return w
		}
	}
	return w
}

// enclosing is the outermost block around line i of at most max lines:
// from a line less indented than its inside, to the line that closes it.
func enclosing(body []string, i, max int) (start, end int, found bool) {
	cur := indentOf(body[i])
	if cur < 0 {
		return 0, 0, false
	}
	for j := i - 1; j >= 0 && cur > 0; j-- {
		d := indentOf(body[j])
		if d < 0 || d >= cur {
			continue
		}
		t := strings.TrimSpace(body[j][1:])
		if t == "" || strings.ContainsAny(t[:1], "})]") {
			continue
		}
		cur = d
		e := j
		for k := j + 1; k < len(body); k++ {
			dk := indentOf(body[k])
			if dk < 0 {
				continue
			}
			if dk <= d {
				if dk == d && strings.ContainsAny(strings.TrimSpace(body[k][1:])[:1], "})]") {
					e = k
				}
				break
			}
			e = k
		}
		if e < i {
			continue
		}
		if e-j+1 > max {
			break
		}
		start, end, found = j, e, true
	}
	return
}
