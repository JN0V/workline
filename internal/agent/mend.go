package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Mend fixes, before any YAML reader takes an answer, the two slips that
// made agents' answers unreadable or cut, whatever the role
// (docs/spec/role-contract.md, "One run", step 3), and says what it mended:
//
//   - code quoted with its tabs under a block scalar (`|`), its first line
//     starting with a tab or its lines not indented under the key: the
//     reader takes the tab for indentation and refuses the whole answer
//     (#157). Only when the answer does not read for that reason, the block
//     is given its indentation in spaces — an explicit indentation
//     indicator, `|2`, when its first line starts with a tab — and the text
//     of the block is kept as written, tabs included.
//   - a text written plain holding ` #`: the reader takes the rest of the
//     line for a comment and drops it without a word (#156). A plain value
//     followed on its line by ` #…` is read whole, as written, when it is
//     free text: several words, or a `#` stuck to what follows (`#20`). A
//     single word followed by `# a note` stays a comment: a closed value
//     (a severity, a path) the agent annotated.
//
// An answer that still does not read is returned as it came, nothing said:
// the reader's own error is what the agent is told when asked again.
func Mend(answer string) (string, []string) {
	var said []string
	var probe any
	if err := yaml.Unmarshal([]byte(answer), &probe); err != nil {
		if !strings.Contains(err.Error(), "tab character") {
			return answer, nil
		}
		fixed, n := untab(answer)
		if n == 0 || yaml.Unmarshal([]byte(fixed), &probe) != nil {
			return answer, nil
		}
		answer = fixed
		said = append(said, fmt.Sprintf("%d block(s) of code indented with tabs (the YAML reader refuses a tab as indentation): indented with spaces, their text kept as written", n))
	}
	if fixed, n := unhash(answer); n > 0 {
		answer = fixed
		said = append(said, fmt.Sprintf("%d plain text(s) holding ` #` (the YAML reader reads the rest as a comment and drops it): read whole, as written", n))
	}
	return answer, said
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
