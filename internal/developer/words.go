package developer

// What its agent says beside its code (ADR-0056, #746): the worktree's
// diff is its code, so its words never cost it. An objection is written
// in a file of the worktree, read here and taken out of the diff, never
// parsed out of prose; closing words of which nothing read are posted as
// they came, so that an objection written there is not lost in silence.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/intent"
)

// ObjectionsFile is where, in its worktree, its agent writes its
// objections to frozen tests: a YAML list, one objection an item.
const ObjectionsFile = ".workline/objections.yaml"

// unreadMax bounds the words posted as they came.
const unreadMax = 3000

// agentWords are its agent's objections, read from its worktree's
// objections file, then removed so that its diff never holds it; and, as
// they came, its words that did not read: the file, its closing words.
func agentWords(runDir string) (objections []intent.Intention, unread []string, err error) {
	if tree := agent.Worktree(runDir); tree != "" {
		file := filepath.Join(tree, filepath.FromSlash(ObjectionsFile))
		data, err := os.ReadFile(file)
		switch {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			return nil, nil, fmt.Errorf("its agent's %s: %w", ObjectionsFile, err)
		default:
			if err := os.Remove(file); err != nil {
				return nil, nil, fmt.Errorf("its agent's %s: %w", ObjectionsFile, err)
			}
			read, ok := objectionsIn(data)
			if !ok {
				unread = append(unread, "Its objections file, `"+ObjectionsFile+"`, could not be read as YAML. As it came:\n\n"+quoted(string(data)))
			}
			objections = read
		}
	}
	if data, err := os.ReadFile(filepath.Join(runDir, "out", agent.ClosingUnread)); err == nil && strings.TrimSpace(string(data)) != "" {
		unread = append(unread, "Its agent's closing words could not be read as YAML: nothing of them was applied. As they came:\n\n"+quoted(string(data)))
	}
	return objections, unread, nil
}

// objectionsIn reads an objections file: a list, or one objection alone;
// an item an objection, or one written as the answer's `comment` or
// `objection` holding it. Each becomes the comment the bounds read.
func objectionsIn(data []byte) ([]intent.Intention, bool) {
	var v any
	if yaml.Unmarshal(data, &v) != nil {
		return nil, false
	}
	items, isList := v.([]any)
	if !isList {
		if v == nil {
			return nil, true
		}
		items = []any{v}
	}
	var out []intent.Intention
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			return nil, false
		}
		if c, ok := m["comment"].(map[string]any); ok && len(m) == 1 {
			m = c
		}
		if o, ok := m["objection"].(map[string]any); ok && len(m) == 1 {
			m = o
		}
		out = append(out, intent.Intention{Kind: "comment", Value: map[string]any{"objection": m}})
	}
	return out, true
}

// quoted is an agent's text as it came, in a fence it cannot close, cut
// at unreadMax, a record's marker in it defused: words posted are never
// read back as the engine's own.
func quoted(text string) string {
	text = strings.TrimSpace(text)
	if len(text) > unreadMax {
		text = strings.ToValidUTF8(text[:unreadMax], "") + "\n… (cut: whole in the run's folder)"
	}
	text = strings.ReplaceAll(text, "<!--", "<!- -")
	fence := "~~~"
	for strings.Contains(text, fence) {
		fence += "~"
	}
	return fence + "\n" + text + "\n" + fence + "\n"
}
