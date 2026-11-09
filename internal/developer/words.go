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
	"path"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/intent"
)

// ObjectionsFile is where, in its worktree, its agent writes its
// objections to frozen tests: a YAML list, one objection an item.
const ObjectionsFile = ".workline/objections.yaml"

// BeyondFile is where, in its worktree, its agent says why it changed
// each file beyond the issue's Scope (ADR-0065, change 2): YAML, a path
// to its one-line reason.
const BeyondFile = ".workline/beyond.yaml"

// unreadMax bounds the words posted as they came.
const unreadMax = 3000

// agentWords are its agent's objections and its reasons for the files
// beyond the Scope, read from its worktree's files, then removed so that
// its diff never holds them; and, as they came, its words that did not
// read: a file, its closing words.
func agentWords(runDir string) (objections []intent.Intention, beyond map[string]string, unread []string, err error) {
	if tree := agent.Worktree(runDir); tree != "" {
		data, said, err := wordsFile(tree, BeyondFile, "reasons for the files beyond the "+scopeName)
		if err != nil {
			return nil, nil, nil, err
		}
		unread = append(unread, said...)
		if data != nil {
			read, ok := beyondIn(data)
			if !ok {
				unread = append(unread, "Its file of reasons for the files beyond the "+scopeName+", `"+BeyondFile+"`, could not be read as YAML. As it came:\n\n"+quoted(string(data)))
			}
			beyond = read
		}
		// A link is never followed: what it leads to is not its agent's
		// to post. Removed, a folder whole, it reaches no branch either.
		data, said, err = wordsFile(tree, ObjectionsFile, "objections file")
		if err != nil {
			return nil, nil, nil, err
		}
		unread = append(unread, said...)
		if data != nil {
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
	return objections, beyond, unread, nil
}

// wordsFile reads, then removes, a file its agent writes in its worktree
// beside its code: never followed when a link, nor read when not a plain
// file (said, removed); nil, nothing said, when absent.
func wordsFile(tree, name, what string) (data []byte, said []string, err error) {
	file := filepath.Join(tree, filepath.FromSlash(name))
	st, err := os.Lstat(file)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil, nil
	case err != nil:
		return nil, nil, fmt.Errorf("its agent's %s: %w", name, err)
	case !st.Mode().IsRegular():
		if err := os.RemoveAll(file); err != nil {
			return nil, nil, fmt.Errorf("its agent's %s: %w", name, err)
		}
		return nil, []string{"Its " + what + ", `" + name + "`, is not a plain file (a link, or a folder): not followed, not read, removed."}, nil
	}
	if data, err = os.ReadFile(file); err != nil {
		return nil, nil, fmt.Errorf("its agent's %s: %w", name, err)
	}
	if err := os.Remove(file); err != nil {
		return nil, nil, fmt.Errorf("its agent's %s: %w", name, err)
	}
	return data, nil, nil
}

// beyondIn reads a file of reasons: a map, path to reason, or a list of
// such maps, or of {file, why}. Each path cleaned, its reason one line.
func beyondIn(data []byte) (map[string]string, bool) {
	var v any
	if yaml.Unmarshal(data, &v) != nil {
		return nil, false
	}
	out := map[string]string{}
	put := func(file, why any) bool {
		f, ok := file.(string)
		if !ok || strings.TrimSpace(f) == "" {
			return false
		}
		out[path.Clean(filepath.ToSlash(strings.Trim(strings.TrimSpace(f), "`")))] = strings.Join(strings.Fields(fmt.Sprint(why)), " ")
		return true
	}
	var items []any
	switch x := v.(type) {
	case nil:
		return out, true
	case []any:
		items = x
	default:
		items = []any{x}
	}
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			return nil, false
		}
		if f, ok := m["file"]; ok {
			if !put(f, m["why"]) {
				return nil, false
			}
			continue
		}
		for f, why := range m {
			if !put(f, why) {
				return nil, false
			}
		}
	}
	return out, true
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
