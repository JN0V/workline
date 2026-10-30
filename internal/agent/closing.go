package agent

import (
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// ClosingUnread, in a run's out/, holds the closing words of an agent that
// worked in a worktree when nothing of them read: its role says so and
// posts them as they came, for a person (#746).
const ClosingUnread = "closing-unread.txt"

// closingWords reads what an agent that worked in a worktree ends on
// (ADR-0056). Its edits are its code, so these words never fail the call,
// nor ask it again (a whole run of turns): read whole when they are a
// YAML list; else the YAML lists found among its prose, fenced or not,
// read alone; else nothing read, the words kept in out/ClosingUnread for
// its role to post. Empty or `[]`, it proposes nothing beside its edits.
func closingWords(runDir, answer string, kinds []string) ([]byte, []string) {
	unread := filepath.Join(runDir, "out", ClosingUnread)
	_ = os.Remove(unread) // an earlier attempt's
	s := strings.TrimSpace(answer)
	if s == "" {
		return []byte("[]\n"), nil
	}
	if p, mended, err := proposalsFrom(s, kinds...); err == nil {
		return p, mended
	}
	var list []map[string]any
	var mended []string
	for _, block := range yamlBlocks(s) {
		p, m, err := proposalsFrom(block, kinds...)
		var items []map[string]any
		if err != nil || yaml.Unmarshal(p, &items) != nil {
			continue
		}
		list, mended = append(list, items...), append(mended, m...)
	}
	if len(list) > 0 {
		out, err := yaml.Marshal(list)
		if err == nil {
			return out, append(mended, "its closing words held prose around their YAML: the YAML read alone, the prose left out (out/agent-answer.txt)")
		}
	}
	if lines := strings.Split(s, "\n"); strings.TrimSpace(lines[len(lines)-1]) == "[]" {
		return []byte("[]\n"), []string{"its closing words held prose before their `[]`: read as proposing nothing beyond its edits (out/agent-answer.txt)"}
	}
	_ = os.WriteFile(unread, []byte(answer), 0o644)
	return []byte("[]\n"), nil
}

// yamlBlocks are the parts of closing words that may be a YAML list: each
// fenced block's lines, and each run of lines opening on `- ` at the
// start of a line, with the lines indented under them.
func yamlBlocks(s string) []string {
	var blocks []string
	lines := strings.Split(s, "\n")
	fence := ""
	var cur []string
	flush := func() {
		if len(cur) > 0 {
			blocks = append(blocks, strings.Join(cur, "\n"))
		}
		cur = nil
	}
	for _, l := range lines {
		t := strings.TrimSpace(l)
		switch {
		case fence != "":
			if strings.HasPrefix(t, fence) && strings.Trim(t, "`~") == "" {
				fence = ""
				flush()
				continue
			}
			cur = append(cur, l)
		case strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~"):
			flush()
			fence = t[:3]
		case strings.HasPrefix(l, "- "):
			cur = append(cur, l)
		case len(cur) > 0 && (strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t") || t == ""):
			cur = append(cur, l)
		default:
			flush()
		}
	}
	flush()
	return blocks
}
