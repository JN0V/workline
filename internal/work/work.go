// Package work reads and moves work items kept as files in .workline/work/,
// for projects without a forge (docs/spec/routing.md, "Work starts from a
// clear need").
package work

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/JN0V/workline/internal/verdict"
)

// Required are the sections an item needs before it is ready.
var Required = []string{"Need", "Verification", "Validation", "Scope"}

// Path is where item id lives.
func Path(repo, id string) string {
	return filepath.Join(repo, ".workline", "work", id+".md")
}

var stateLine = regexp.MustCompile(`(?m)^state: .*$`)

// Sections returns the text under each "## " heading.
func Sections(content string) map[string]string {
	out := map[string]string{}
	name := ""
	for _, l := range strings.Split(content, "\n") {
		if h, ok := strings.CutPrefix(l, "## "); ok {
			name = strings.TrimSpace(h)
			out[name] = ""
			continue
		}
		if name != "" {
			out[name] += l + "\n"
		}
	}
	return out
}

// Ready moves an item to ready if its four sections are written, and says
// which are missing otherwise. It checks presence, not quality: that is the
// person's call.
func Ready(repo, id string) *verdict.Verdict {
	path := Path(repo, id)
	data, err := os.ReadFile(path)
	if err != nil {
		return &verdict.Verdict{Status: verdict.Block, Summary: "no such work item",
			Findings: []verdict.Finding{{Rule: "no-item", Where: id, Message: err.Error()}}}
	}
	sections := Sections(string(data))
	var missing []verdict.Finding
	for _, r := range Required {
		if strings.TrimSpace(sections[r]) == "" {
			missing = append(missing, verdict.Finding{Rule: "not-ready", Where: r,
				Message: fmt.Sprintf("## %s is missing or empty", r)})
		}
	}
	if len(missing) > 0 {
		return &verdict.Verdict{Status: verdict.Block, Summary: "the item stays in to-refine", Findings: missing}
	}
	content := string(data)
	if stateLine.MatchString(content) {
		content = stateLine.ReplaceAllString(content, "state: ready")
	} else {
		content = "state: ready\n" + content
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return &verdict.Verdict{Status: verdict.Block, Findings: []verdict.Finding{{Rule: "engine-error", Message: err.Error()}}}
	}
	return &verdict.Verdict{Status: verdict.Pass, Summary: "item " + id + " is ready"}
}
