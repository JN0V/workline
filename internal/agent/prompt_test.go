package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/role"
)

// Every role's output contract says how to write code and quotes so that
// they read (#138), and an answer asked for again repeats it.
func TestPromptAsksForBlockScalars(t *testing.T) {
	dir, run := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(dir, "instruction.md"), []byte("Review."), 0o644)
	os.MkdirAll(filepath.Join(run, "in"), 0o755)
	os.MkdirAll(filepath.Join(run, "out"), 0o755)
	os.WriteFile(filepath.Join(run, "in", "task.md"), []byte("The change."), 0o644)
	req := Request{RunDir: run, Repo: t.TempDir(), Role: &role.Role{Name: "reviewer", Dir: dir, Intentions: []string{"finding"}}}
	_, user, err := Prompt(req)
	if err != nil || !strings.Contains(user, BlockScalars) {
		t.Errorf("the output contract does not ask for block scalars: %v\n%s", err, user)
	}
}

// An intention the engine applies but the agent is never shown the shape
// of is one no agent proposes: the product owner's refine went unproposed
// on DomoticsCore for that.
func TestEveryIntentionHasItsShape(t *testing.T) {
	for kind := range intent.Catalogue {
		if contracts[kind] == "" && !engineOnly[kind] {
			t.Errorf("%s: no shape in the output contract (contracts, prompt.go)", kind)
		}
	}
}

// A role's budget stops a prompt really over it (#235): estimated at four
// characters a token, a reviewer's call of 96.6k tokens passed a budget of
// 40000 as 29k. Code over the budget at the measured ratio is refused;
// prose within it is not.
func TestPromptBudgetCountsCodeAtTheMeasuredRatio(t *testing.T) {
	ask := func(task string) error {
		dir, run := t.TempDir(), t.TempDir()
		os.WriteFile(filepath.Join(dir, "instruction.md"), []byte("Review."), 0o644)
		os.MkdirAll(filepath.Join(run, "in"), 0o755)
		os.MkdirAll(filepath.Join(run, "out"), 0o755)
		os.WriteFile(filepath.Join(run, "in", "task.md"), []byte(task), 0o644)
		r := &role.Role{Name: "reviewer", Dir: dir, Intentions: []string{"note"}}
		r.Context.Budget = 4000
		_, _, err := Prompt(Request{RunDir: run, Repo: t.TempDir(), Role: r})
		return err
	}
	// About 5,000 characters of Go: 1,250 tokens at four a token, 4,800 measured.
	code := strings.Repeat("\tif n, err := f(x[i]); err != nil {\n\t\treturn 0, err\n\t}\n", 90)
	err := ask(code)
	if err == nil || !strings.Contains(err.Error(), "over the role's budget of 4000") || !strings.Contains(err.Error(), TokensRatio) {
		t.Errorf("code over the budget at the measured ratio is not refused, or not said: %v", err)
	}
	prose := strings.Repeat("The reviewer reads the change and the files it touches. ", 35)
	if err := ask(prose); err != nil {
		t.Errorf("prose within the budget is refused: %v", err)
	}
	if got := Tokens(117000); got < 96000 || got > 97200 {
		t.Errorf("a lens's call of 117k characters (96.6k tokens reported, #147) is estimated at %d", got)
	}
}

// A person's facets are read from the folder their config.yaml is read
// from, then from ~/.config/workline, where they were read before (#104):
// on macOS, Go's config folder is ~/Library/Application Support.
func TestFacetDirs(t *testing.T) {
	defer func(c, h func() (string, error)) { userConfigDir, userHomeDir = c, h }(userConfigDir, userHomeDir)
	req := Request{Repo: "/work/repo", Role: &role.Role{Name: "committer", Dir: "/roles/committer"}}
	for _, c := range []struct {
		name, cfg, home string
		want            []string
	}{
		{"macOS", "/Users/me/Library/Application Support", "/Users/me", []string{"/work/repo/.workline/roles/committer",
			"/Users/me/Library/Application Support/workline/roles/committer", "/Users/me/.config/workline/roles/committer", "/roles/committer"}},
		{"Linux, searched once", "/home/me/.config", "/home/me", []string{"/work/repo/.workline/roles/committer",
			"/home/me/.config/workline/roles/committer", "/roles/committer"}},
	} {
		userConfigDir = func() (string, error) { return c.cfg, nil }
		userHomeDir = func() (string, error) { return c.home, nil }
		if got := facetDirs(req); strings.Join(got, "\n") != strings.Join(c.want, "\n") {
			t.Errorf("%s: facets looked for in\n%s\nwant\n%s", c.name, strings.Join(got, "\n"), strings.Join(c.want, "\n"))
		}
	}
}

// The config folder's facet is used before the one in ~/.config, and one
// only in ~/.config is still found.
func TestFacetFromTheConfigFolderFirst(t *testing.T) {
	defer func(c, h func() (string, error)) { userConfigDir, userHomeDir = c, h }(userConfigDir, userHomeDir)
	cfg, home, own := t.TempDir(), t.TempDir(), t.TempDir()
	userConfigDir = func() (string, error) { return cfg, nil }
	userHomeDir = func() (string, error) { return home, nil }
	write := func(dir, name, text string) {
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644)
	}
	write(filepath.Join(cfg, "workline", "roles", "committer"), "policy.md", "from the config folder")
	write(filepath.Join(home, ".config", "workline", "roles", "committer"), "policy.md", "from ~/.config")
	write(filepath.Join(home, ".config", "workline", "roles", "committer"), "persona.md", "only in ~/.config")
	write(own, "policy.md", "the role's own")
	req := Request{Repo: t.TempDir(), Role: &role.Role{Name: "committer", Dir: own}}
	if got, _ := facet(req, "policy.md"); got != "from the config folder" {
		t.Errorf("policy.md read %q, want the config folder's", got)
	}
	if got, _ := facet(req, "persona.md"); got != "only in ~/.config" {
		t.Errorf("persona.md read %q, want the one in ~/.config", got)
	}
}

// A shared skill is given after the task when the run's pre names it and
// the role declares it; a project's copy wins over the shipped one; one
// the role does not declare is refused.
func TestPromptLoadsTheSkillsThePreNames(t *testing.T) {
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	ask := func(declared []string, project string) (string, error) {
		dir, run, repo := t.TempDir(), t.TempDir(), t.TempDir()
		must(os.WriteFile(filepath.Join(dir, "instruction.md"), []byte("Answer."), 0o644))
		must(os.MkdirAll(filepath.Join(run, "in"), 0o755))
		must(os.MkdirAll(filepath.Join(run, "out"), 0o755))
		must(os.WriteFile(filepath.Join(run, "in", "task.md"), []byte("The thread."), 0o644))
		must(os.WriteFile(filepath.Join(run, "in", SkillsFile), []byte("five-whys\n"), 0o644))
		if project != "" {
			must(os.MkdirAll(filepath.Join(repo, ".workline", "skills", "five-whys"), 0o755))
			must(os.WriteFile(filepath.Join(repo, ".workline", "skills", "five-whys", "SKILL.md"), []byte(project), 0o644))
		}
		r := &role.Role{Name: "tester", Dir: dir, Intentions: []string{"note"}}
		r.Context.Skills = declared
		_, user, err := Prompt(Request{RunDir: run, Repo: repo, Role: r})
		return user, err
	}
	user, err := ask([]string{"five-whys"}, "")
	if err != nil || !strings.Contains(user, "# Skill: five-whys") || !strings.Contains(user, "Root cause:") || strings.Contains(user, "description:") {
		t.Errorf("the shipped skill, its body only, is not given: %v\n%s", err, user)
	}
	if strings.Index(user, "# Skill") < strings.Index(user, "# Task") {
		t.Error("the skill comes before the task")
	}
	if user, _ := ask([]string{"five-whys"}, "---\nname: five-whys\ndescription: ours\n---\nOur own whys."); !strings.Contains(user, "Our own whys.") {
		t.Errorf("the project's copy is not the one given:\n%s", user)
	}
	if _, err := ask(nil, ""); err == nil || !strings.Contains(err.Error(), "context.skills") {
		t.Errorf("a skill the role does not declare is not refused: %v", err)
	}
}
