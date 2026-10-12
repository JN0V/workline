package reviewer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// lineComments are the findings on the change, each a comment on its line
// in the forge's own review (#277): the rules', on the lines the change
// added, then each lens's, one a finding — two on one line, two comments,
// each answered and resolved apart. A finding quoted from an issue has no
// line, and stays in the summary comment only.
func lineComments(repo string, s Settings, st state, related []Finding) []forge.LineComment {
	files := map[string][]string{}
	linesOf := func(path string) []string {
		if l, ok := files[path]; ok {
			return l
		}
		if text, ok := fileAt(repo, st.Head, path); ok {
			files[path] = strings.Split(text, "\n")
		} else {
			files[path] = nil // a file gone at the head: each line keyed by its number
		}
		return files[path]
	}
	var out []forge.LineComment
	for _, f := range st.Mechanical {
		path, n, ok := fileLine(f.Where)
		if !ok {
			continue
		}
		out = append(out, forge.LineComment{Path: path, Line: n, Key: lineKeyOf(linesOf(path), path, n, f.Rule), Body: ruleBody(f)})
	}
	for _, g := range related {
		for _, f := range members(g) {
			path, n, ok := fileLine(f.Where)
			if !ok {
				continue
			}
			if f.On > 0 {
				n = f.On
			}
			out = append(out, forge.LineComment{Path: path, Line: n, Key: lineKeyOf(linesOf(path), path, n, f.Lens), Body: lineBody(s, f)})
		}
	}
	return out
}

// fileLine reads a place, `path:line`; an issue's (`#4`) has none.
func fileLine(where string) (string, int, bool) {
	i := strings.LastIndex(where, ":")
	if i <= 0 || strings.HasPrefix(where, "#") {
		return "", 0, false
	}
	n, err := strconv.Atoi(where[i+1:])
	if err != nil || n < 1 {
		return "", 0, false
	}
	return where[:i], n, true
}

// lineKeyOf keys a finding's comment by what found it and the line it
// stands on: its file, its text spaces aside, and which of the lines
// reading so it is. A line moved by lines added above keeps its key; a
// line changed gets a new one. A line not read — a file gone at the head,
// past its end — is keyed by its number.
func lineKeyOf(lines []string, path string, n int, by string) string {
	text, nth := fmt.Sprintf("\x00line %d", n), 0
	if n <= len(lines) {
		text = norm(lines[n-1])
		for _, l := range lines[:n-1] {
			if norm(l) == text {
				nth++
			}
		}
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d", path, text, nth)))
	for strings.Contains(by, "--") { // a run of dashes would end the marker
		by = strings.ReplaceAll(by, "--", "-")
	}
	return strings.Join(strings.Fields(by), "-") + "/" + hex.EncodeToString(sum[:6])
}

// holds is what a finding does to the merge request: blocks, or warns.
func holds(blocks bool) string {
	if blocks {
		return "**blocks**"
	}
	return "warns"
}

// lineBody is a lens's finding on its line: what it holds and which lens
// found it, then the finding whole — title, why, fix, how it was verified.
func lineBody(s Settings, f Finding) string {
	blocks := s.AIFindings.Blocks(f.Lens) && f.Severity == "important" && strings.HasPrefix(f.Verified, "verified")
	sev := "Important"
	if f.Severity != "important" {
		sev = "Nit"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "**workline reviewer** · %s · %s\n\n**%s: %s**\n\n%s\n", holds(blocks), f.Lens, sev, f.Title, f.Why)
	if f.Claim != "" {
		fmt.Fprintf(&b, "\nThe author said: “%s”\n", oneLine(f.Claim, 300))
	}
	if f.Symptom != nil {
		fmt.Fprintf(&b, "\nIt shows at `%s`.\n", f.Symptom.Path)
	}
	if fix := fixLine(f); fix != "" {
		b.WriteString("\n" + fix + "\n")
	}
	if f.Verified != "" {
		fmt.Fprintf(&b, "\n_%s._\n", strings.TrimSuffix(f.Verified, "."))
	}
	return strings.TrimSpace(b.String())
}

// ruleBody is a rule's finding on its line.
func ruleBody(f verdict.Finding) string {
	return fmt.Sprintf("**workline reviewer** · %s · %s\n\n%s", holds(status([]verdict.Finding{f}) == verdict.Block), f.Rule, f.Message)
}

// shortMessage is a finding's row in the summary comment when it is on its
// line: its title, the why and the fix left to the comment there.
func shortMessage(f Finding) string {
	sev := "Important"
	if f.Severity != "important" {
		sev = "Nit"
	}
	m := fmt.Sprintf("%s: %s", sev, f.Title)
	for _, o := range f.Also {
		m += fmt.Sprintf("; also %s: %s", o.Lens, o.Title)
	}
	return m
}

// review is the review a summary comment carries: its comments on lines,
// posted at head (docs/spec/role-outcome.md, `comment`).
func reviewOf(st state, lines []forge.LineComment) map[string]any {
	comments := make([]any, 0, len(lines))
	for _, c := range lines {
		comments = append(comments, map[string]any{"path": c.Path, "line": c.Line, "key": c.Key, "body": c.Body})
	}
	return map[string]any{"head": st.Head, "comments": comments,
		"body": fmt.Sprintf("**workline reviewer** on %s: each finding on its line, what blocks marked. The summary comment holds the verdict; the reviewer never approves: the author fixes, a person merges.", short(st.Head))}
}
