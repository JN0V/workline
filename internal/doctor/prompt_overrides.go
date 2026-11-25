package doctor

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/JN0V/workline/internal/role"
)

// flatPrompts names each prompt override left at its old path, in the
// project's and the person's folders, for a role whose prompts are under
// prompts/: such a file is no longer read.
func flatPrompts(r *Report, o Options) {
	entries, err := os.ReadDir(o.RolesDir)
	if err != nil {
		return
	}
	folders := []string{filepath.Join(o.Repo, ".workline", "roles")}
	if cfg, err := os.UserConfigDir(); err == nil {
		folders = append(folders, filepath.Join(cfg, "workline", "roles"))
	}
	// ~/.config/workline, where the run also reads them when it differs.
	if home, err := os.UserHomeDir(); err == nil {
		if d := filepath.Join(home, ".config", "workline", "roles"); d != folders[len(folders)-1] {
			folders = append(folders, d)
		}
	}
	for _, e := range entries {
		roleDir := filepath.Join(o.RolesDir, e.Name())
		for _, base := range folders {
			for _, f := range role.FlatPrompts(roleDir, filepath.Join(base, e.Name())) {
				r.add(Check{Area: "repository", Rule: "prompt-override-moved", Where: f.Old, Level: Error,
					Message: "this override is no longer read: the " + e.Name() + "'s prompts are under prompts/, and the shipped one runs in its place",
					Fix:     "mv " + f.Old + " " + f.New})
			}
		}
	}
}

// auditorKindRole is the role whose acts a kind of the auditor's questions
// judges, as the weekly sample's kindOf draws them.
var auditorKindRole = map[string]string{
	"finding": "reviewer", "rewrite": "committer", "code": "developer",
	"tests": "tester", "kept": "tester", "rewritten": "tester", "whys": "tester", "removal": "tester",
	"ready": "product-owner", "refine": "product-owner", "sources": "product-owner", "ask": "product-owner",
	"assume": "product-owner", "kind": "product-owner", "size": "product-owner", "split": "product-owner",
	"order": "product-owner", "closing": "product-owner",
}

// oldAuditorKinds names each auditor question left under kinds/, which is no
// longer read: it goes to criteria/<role>/.
func oldAuditorKinds(r *Report, o Options) {
	dir := filepath.Join(o.Repo, ".workline", "roles", "auditor", "kinds")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".md" {
			continue
		}
		old := filepath.Join(dir, name)
		kind := strings.TrimSuffix(name, ".md")
		owner, ok := auditorKindRole[kind]
		if !ok {
			r.add(Check{Area: "repository", Rule: "auditor-kind-moved", Where: old, Level: Error,
				Message: "this auditor question is no longer read, and no role has a kind " + kind,
				Fix:     "remove " + old})
			continue
		}
		dest := filepath.Join(o.Repo, ".workline", "roles", "auditor", "criteria", owner, name)
		if _, err := os.Stat(dest); err == nil {
			r.add(Check{Area: "repository", Rule: "auditor-kind-moved", Where: old, Level: Error,
				Message: "this auditor question is no longer read: its copy is already at " + dest,
				Fix:     "remove " + old})
			continue
		}
		r.add(Check{Area: "repository", Rule: "auditor-kind-moved", Where: old, Level: Error,
			Message: "this auditor question is no longer read: the questions are under criteria/<role>/",
			Fix:     "mkdir -p " + filepath.Dir(dest) + " && mv " + old + " " + dest})
	}
}
