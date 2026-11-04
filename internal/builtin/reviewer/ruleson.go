package reviewer

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/verdict"
)

// Name is the reviewer's role, as a project routes it.
const Name = "reviewer"

// RulesOn runs the reviewer's rules, no agent, on what head adds to base
// in repo, as the project sets them: the findings that block. Another
// role runs them on what it writes before the reviewer does, so that a
// frozen file never holds a rule's finding nobody may fix (#1040).
func RulesOn(repo, base, head string) ([]verdict.Finding, error) {
	r, err := role.Load(os.Getenv("WORKLINE_ROLES_DIR"), Name)
	if err != nil {
		return nil, err
	}
	cfg, err := role.LoadProjectConfig(repo)
	if err != nil {
		return nil, err
	}
	var s Settings
	data, _ := json.Marshal(r.MergedSettings(cfg))
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("the reviewer's settings: %w", err)
	}
	files, err := codeFiles(repo, s, base, head)
	if err != nil || len(files) == 0 {
		return nil, err
	}
	change, err := readChange(repo, base, head, files)
	if err != nil {
		return nil, err
	}
	codes, err := committerSettings(repo)
	if err != nil {
		return nil, err
	}
	found, err := mechanical(change, s, codes)
	if err != nil {
		return nil, err
	}
	var blocks []verdict.Finding
	for _, f := range verdict.Levels(found, cfg.Roles[Name].Enforce) {
		if f.Blocks() {
			blocks = append(blocks, f)
		}
	}
	return blocks, nil
}
