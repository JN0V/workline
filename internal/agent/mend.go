package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Mend fixes the slips that leave agents' answers unreadable or cut,
// before any YAML reader takes them, and says what it mended: code quoted
// with its tabs under a block scalar, when the answer fails on a tab (#157);
// a value opening on a quoted phrase and going on after it, when the answer
// fails there (#128); and a plain free text cut by ` #`, read whole (#156). The block's text and
// the line's are kept exactly as written; an answer that still does not read
// comes back as it came. Why and the rules: docs/spec/role-contract.md,
// "One run", step 3.
func Mend(answer string) (string, []string) {
	var said []string
	var probe any
	if err := yaml.Unmarshal([]byte(answer), &probe); err != nil {
		var fixed string
		var n int
		switch {
		case strings.Contains(err.Error(), "tab character"):
			fixed, n = untab(answer)
			said = append(said, fmt.Sprintf("%d block(s) of code indented with tabs (the YAML reader refuses a tab as indentation): indented with spaces, their text kept as written", n))
		case strings.Contains(err.Error(), "did not find expected key"):
			fixed, n = unquote(answer)
			said = append(said, fmt.Sprintf("%d value(s) opening on a quoted phrase and going on after it (the YAML reader takes the quotes for the whole value): read whole, as written", n))
		}
		if n == 0 || yaml.Unmarshal([]byte(fixed), &probe) != nil {
			return answer, nil
		}
		answer = fixed
	}
	if fixed, n := unhash(answer); n > 0 {
		answer = fixed
		said = append(said, fmt.Sprintf("%d plain text(s) holding ` #` (the YAML reader reads the rest as a comment and drops it): read whole, as written", n))
	}
	return answer, said
}

// quotedThenMore is a line whose value opens on a quoted phrase and goes
// on after its closing quote: `title: "Idle for 60 minutes" is unclear`.
// No comment after the quote: `x: "a" # note` reads already.
var quotedThenMore = regexp.MustCompile(`^( *(?:- +)*[A-Za-z][\w-]*: +)("[^"\\]*"|'[^']*')([ \t]*[^ \t#].*)$`)

// unquote reads whole the values quotedThenMore matches, rewritten
// double-quoted as the agent wrote them, quotes included, and says how many.
func unquote(answer string) (string, int) {
	lines := strings.Split(answer, "\n")
	n := 0
	for i, l := range lines {
		m := quotedThenMore.FindStringSubmatch(strings.TrimRight(l, " \t\r"))
		// Closed on the quote it opened with, it is one quoted text whose
		// inner quotes were left unescaped: the agent is asked again.
		if m == nil || strings.HasSuffix(m[3], m[2][:1]) {
			continue
		}
		var b bytes.Buffer
		enc := json.NewEncoder(&b)
		enc.SetEscapeHTML(false)
		enc.Encode(m[2] + m[3])
		lines[i] = m[1] + strings.TrimSpace(b.String())
		n++
	}
	return strings.Join(lines, "\n"), n
}

// blockHeader is a line opening a block scalar: `key: |`, `- key: >-`,
// `- |`, an indentation indicator or a comment after it.
var blockHeader = regexp.MustCompile(`^( *)((?:- +)*)([^\s#'"\-][^#]*?:[ \t]+)?([|>])([1-9]?[+-]?|[+-][1-9])[ \t]*(#.*)?$`)

// untab gives the block scalars of an answer whose lines are indented with
// tabs their indentation in spaces, and says how many it changed. A block's
// lines are those after its header indented more than its key, blank, or
// starting with a tab before its indentation is reached (code pasted as it
// is in the file): those are indented as its first line, their text kept.
func untab(answer string) (string, int) {
	lines := strings.Split(answer, "\n")
	changed := 0
	for i := 0; i < len(lines); i++ {
		m := blockHeader.FindStringSubmatchIndex(lines[i])
		if m == nil {
			continue
		}
		head := lines[i]
		parent := len(head[m[2]:m[3]]) + len(head[m[4]:m[5]]) // the key's column
		if m[6] < 0 && m[5] > m[4] {
			parent -= 2 // `- |`: the dash's column
		}
		if strings.ContainsAny(head[m[10]:m[11]], "123456789") {
			continue // its indentation said already
		}
		// The block's lines, and its indentation: its first line's, in
		// spaces, or two more than its key's.
		end, indent := i+1, -1
		for ; end < len(lines); end++ {
			l := lines[end]
			if strings.TrimSpace(l) == "" {
				continue
			}
			k := len(l) - len(strings.TrimLeft(l, " "))
			if k <= parent && l[k] != '\t' {
				break
			}
			if indent < 0 {
				indent = k
				if k <= parent {
					indent = parent + 2
				}
			}
		}
		if indent < 0 {
			continue
		}
		touched := false
		first := true
		for j := i + 1; j < end; j++ {
			l := lines[j]
			if strings.TrimSpace(l) == "" {
				if strings.Contains(l, "\t") {
					lines[j], touched = "", true
				}
				continue
			}
			k := len(l) - len(strings.TrimLeft(l, " "))
			if k < indent && l[k] == '\t' {
				l, touched = strings.Repeat(" ", indent)+l[k:], true
				lines[j] = l
			}
			if first && len(l) > indent && l[indent] == '\t' && indent-parent >= 1 && indent-parent <= 9 {
				// The reader finds a block's indentation on its first line
				// and stops at a tab: said, it reads the tab as text.
				at := m[9]
				lines[i] = head[:at] + fmt.Sprint(indent-parent) + head[at:]
				touched = true
			}
			first = false
		}
		if touched {
			changed++
		}
		i = end - 1
	}
	return strings.Join(lines, "\n"), changed
}

// unhash reads whole the plain values a ` #` cut, as Mend says, and says how
// many. The value is rewritten double-quoted, from its first character to
// the end of its line, so nothing else of the answer moves.
func unhash(answer string) (string, int) {
	var doc yaml.Node
	if yaml.Unmarshal([]byte(answer), &doc) != nil {
		return answer, 0
	}
	lines := strings.Split(answer, "\n")
	edits := map[int]int{} // a line: the column its value starts at, in runes
	var check func(n *yaml.Node)
	check = func(n *yaml.Node) {
		if n.Kind != yaml.ScalarNode || n.Style != 0 || n.Line < 1 || n.Line > len(lines) {
			return
		}
		line := []rune(lines[n.Line-1])
		if n.Column < 1 || n.Column > len(line) {
			return
		}
		rest := string(line[n.Column-1:])
		var after string
		at := n.Column - 1
		switch {
		case n.Value == "" && n.Tag == "!!null": // `title: #20 is…`: all of it read as a comment
			after = strings.TrimLeft(rest, " \t")
			at += len(rest) - len(after)
		case strings.HasPrefix(rest, n.Value):
			after = strings.TrimLeft(rest[len(n.Value):], " \t")
			if after == rest[len(n.Value):] {
				return // no space before the hash: not a comment, or not this line's
			}
		default:
			return // a value over several lines
		}
		if !strings.HasPrefix(after, "#") {
			return
		}
		next := strings.TrimPrefix(after, "#")
		if !strings.Contains(n.Value, " ") && (next == "" || next[0] == ' ' || next[0] == '\t') {
			return // `nit # a note`: a closed value, annotated
		}
		edits[n.Line-1] = at
	}
	var walk func(n *yaml.Node)
	walk = func(n *yaml.Node) {
		switch n.Kind {
		case yaml.MappingNode:
			if n.Style&yaml.FlowStyle != 0 {
				return
			}
			for i := 0; i+1 < len(n.Content); i += 2 {
				k, v := n.Content[i], n.Content[i+1]
				if v.Kind == yaml.ScalarNode && (v.Tag != "!!null" || v.Line == k.Line) {
					check(v)
				}
				walk(v)
			}
		case yaml.SequenceNode:
			if n.Style&yaml.FlowStyle != 0 {
				return
			}
			for _, c := range n.Content {
				if c.Kind == yaml.ScalarNode && c.Tag != "!!null" {
					check(c)
				}
				walk(c)
			}
		case yaml.DocumentNode:
			for _, c := range n.Content {
				walk(c)
			}
		}
	}
	walk(&doc)
	if len(edits) == 0 {
		return answer, 0
	}
	for at, col := range edits {
		line := []rune(lines[at])
		text := strings.TrimRight(string(line[col:]), " \t\r")
		var b bytes.Buffer
		enc := json.NewEncoder(&b)
		enc.SetEscapeHTML(false)
		enc.Encode(text)
		lines[at] = string(line[:col]) + strings.TrimSpace(b.String())
	}
	fixed := strings.Join(lines, "\n")
	var probe any
	if yaml.Unmarshal([]byte(fixed), &probe) != nil {
		return answer, 0
	}
	return fixed, len(edits)
}
