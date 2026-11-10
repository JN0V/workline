package doctor

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/packs"
	"github.com/JN0V/workline/internal/role"
)

// packsFix is how a project lists its packs.
const packsFix = "list them in .workline/config.yaml, as `packs: [go, kotlin-gradle]`, or run `workline init`, which writes the packs it detects; a language with no pack built in takes a file of yours, .workline/packs/<name>.yaml   # docs/packs.md"

// languagePacks says which language packs the project lists (ADR-0065),
// the languages its files hold that none of them knows, and drift
// between the packs its marker files call for and those listed.
func languagePacks(r *Report, o Options, cfg *role.ProjectConfig) {
	all, err := packs.Available(o.Repo)
	if err != nil {
		r.add(Check{Area: "repository", Rule: "pack-invalid", Where: packs.Dir, Level: Error, Message: err.Error()})
		return
	}
	files, err := packs.Files(o.Repo)
	if err != nil {
		r.add(Check{Area: "repository", Rule: "packs-unread", Where: o.Repo, Level: Warn,
			Message: fmt.Sprintf("the packs the repository calls for were not looked for: %v", err)})
		return
	}
	tests, _ := role.TestDef(o.RolesDir, o.Repo) // unread: the generic one, said by testFiles
	aside := tests.Aside
	detected := packs.Detected(files, all, aside)
	listed, _ := packs.Listed(o.Repo, cfg.Packs) // read whole by LoadProjectConfig already
	if len(cfg.Packs) == 0 {
		msg := "no language pack listed: every role reads the generic pack"
		if len(detected) > 0 {
			msg += fmt.Sprintf("; the repository calls for %s", calledFor(detected))
		}
		r.add(Check{Area: "repository", Rule: "packs-none", Where: ".workline/config.yaml", Level: Note, Message: msg, Fix: packsFix})
	} else {
		r.add(Check{Area: "repository", Rule: "packs-listed", Where: ".workline/config.yaml", Level: OK,
			Message: "language packs listed: " + strings.Join(cfg.Packs, ", ")})
		for _, n := range packs.Names(detected) {
			if !slices.Contains(cfg.Packs, n) {
				r.add(Check{Area: "repository", Rule: "pack-not-listed", Where: detected[n], Level: Warn,
					Message: fmt.Sprintf("%s is detected but not listed: %s calls for it, and no role reads it", n, detected[n]), Fix: packsFix})
			}
		}
		for _, n := range cfg.Packs {
			if _, ok := detected[n]; !ok && len(all[n].Detect.Files) > 0 {
				r.add(Check{Area: "repository", Rule: "pack-not-detected", Where: ".workline/config.yaml", Level: Note,
					Message: fmt.Sprintf("%s is listed, and no file of the repository calls for it (%s)", n, strings.Join(all[n].Detect.Files, ", "))})
			}
		}
	}
	for _, c := range packs.Unknown(files, listed, aside) {
		msg := fmt.Sprintf("%s (%d %s) has no pack listed: ", c.Name, c.Files, map[bool]string{true: "file", false: "files"}[c.Files == 1])
		if len(listed) == 0 {
			msg += "every role reads the generic pack's test files for it"
		} else {
			msg += "a test of it is a file the test paths every role reads name, said below"
		}
		fix := fmt.Sprintf("write a pack of yours for it, .workline/packs/%s.yaml, and list it in packs:   # docs/packs.md", c.File)
		if k := packs.Knowing(c.Language, all); len(k) > 0 {
			fix = fmt.Sprintf("list %s in packs: in .workline/config.yaml, if it fits   # docs/packs.md", strings.Join(k, " or "))
		}
		r.add(Check{Area: "repository", Rule: "language-without-pack", Where: o.Repo, Level: Note, Message: msg, Fix: fix})
	}
}

// calledFor is the detected packs, each with the file calling for it.
func calledFor(detected map[string]string) string {
	var out []string
	for _, n := range packs.Names(detected) {
		out = append(out, fmt.Sprintf("%s (%s)", n, detected[n]))
	}
	return strings.Join(out, ", ")
}
