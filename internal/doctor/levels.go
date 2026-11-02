package doctor

import (
	"fmt"
	"slices"

	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
)

// levels says each role's autonomy level and where it comes from, once
// the project sets one (ADR-0060, §3): a role with levels, the level it
// runs at; one with none, that the level changes nothing for it; the
// merge, that it keeps its own. Nothing set anywhere: nothing said.
func levels(r *Report, o Options, cfg *role.ProjectConfig, line *routing.Config) {
	var names []string
	add := func(name string) {
		if name != "" && !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	for _, steps := range line.Events {
		for _, s := range steps {
			add(s)
		}
	}
	for _, s := range line.States {
		add(s.Role)
	}
	for name := range cfg.Roles {
		add(name)
	}
	slices.Sort(names)
	said := false
	for _, name := range names {
		own, set := cfg.Roles[name].Settings[role.ProjectDefault]
		set = set && own != nil
		if cfg.Autonomy == "" && !set {
			continue
		}
		rl, err := role.Load(o.RolesDir, name)
		if err != nil {
			continue // not a role: a gate's name, or said by the run that reads it
		}
		said = true
		if _, has := rl.Levels[role.ProjectDefault]; !has {
			level, rule := Note, "level-unused"
			msg := fmt.Sprintf("%s has no autonomy levels: the project's default, %s, changes nothing for it; it runs as before", name, cfg.Autonomy)
			if set {
				level = Warn
				msg = fmt.Sprintf("%s has no autonomy levels: its own `autonomy: %v` changes nothing", name, own)
			}
			r.add(Check{Area: "repository", Rule: rule, Where: name, Level: level, Message: msg,
				Fix: "docs/autonomy.md#roles-with-no-levels says which roles have levels, and why"})
			continue
		}
		lv, from := role.LevelOf(rl.MergedSettings(cfg), role.ProjectDefault)
		if _, ok := rl.Levels[role.ProjectDefault][lv]; !ok {
			r.add(Check{Area: "repository", Rule: "level-unknown", Where: name, Level: Error,
				Message: fmt.Sprintf("%s's autonomy is %q (%s), not one of its levels: its run stops", name, lv, from)})
			continue
		}
		r.add(Check{Area: "repository", Rule: "level", Where: name, Level: OK,
			Message: fmt.Sprintf("%s runs at %s (%s)", name, lv, from)})
	}
	if !said {
		return
	}
	switch {
	case cfg.Merge != nil:
		r.add(Check{Area: "repository", Rule: "level", Where: "merge", Level: OK,
			Message: fmt.Sprintf("the merge runs at %s, its own key (merge: {autonomy: %s}), whatever the project's default", cfg.Merge.Autonomy, cfg.Merge.Autonomy)})
	case cfg.Autonomy != "":
		r.add(Check{Area: "repository", Rule: "level-unused", Where: "merge", Level: Note,
			Message: fmt.Sprintf("no merge policy: the project's default, %s, turns no auto-merge on; the merge keeps its own key, `merge: {autonomy: …}`", cfg.Autonomy),
			Fix:     "docs/merge-policy.md"})
	}
}
