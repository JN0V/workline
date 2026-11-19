package doctor

import (
	"os"
	"path/filepath"

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
