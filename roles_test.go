package workline

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// loaded names what the engine reads at a role's top level
// (docs/spec/role-contract.md, "Layout"): beside it, only README.md, for
// the role's users, and docs/, the pages about the role.
var loaded = map[string]bool{
	"role.yaml": true, "persona.md": true, "instruction.md": true, "policy.md": true,
	"pre": true, "post": true, "knowledge": true, "lenses": true, "kinds": true, "skills": true,
}

func TestRoleTopLevelHoldsOnlyWhatTheEngineLoads(t *testing.T) {
	roles, err := fs.ReadDir(Roles, "roles")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range roles {
		if !r.IsDir() {
			t.Errorf("roles/%s: a file beside the roles", r.Name())
			continue
		}
		entries, err := fs.ReadDir(Roles, "roles/"+r.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			name := e.Name()
			switch {
			case name == "README.md" && !e.IsDir(), name == "docs" && e.IsDir(), loaded[name]:
			default:
				t.Errorf("roles/%s/%s: the engine does not load it; a page about the role goes to roles/%s/docs/",
					r.Name(), name, r.Name())
			}
		}
	}
}

// Each shared skill is in the Agent Skills format: a SKILL.md whose
// frontmatter names it as its folder and says when to use it; each skill a
// role declares in context.skills is shipped.
func TestSharedSkillsFollowTheAgentSkillsFormat(t *testing.T) {
	name := regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	skills, err := fs.ReadDir(Skills, "skills")
	if err != nil {
		t.Fatal(err)
	}
	shipped := map[string]bool{}
	for _, d := range skills {
		data, err := fs.ReadFile(Skills, "skills/"+d.Name()+"/SKILL.md")
		if err != nil {
			t.Errorf("skills/%s: no SKILL.md", d.Name())
			continue
		}
		var front struct{ Name, Description string }
		head, _, _ := strings.Cut(strings.TrimPrefix(string(data), "---\n"), "\n---\n")
		if err := yaml.Unmarshal([]byte(head), &front); err != nil || front.Name != d.Name() || !name.MatchString(front.Name) ||
			front.Description == "" || len(front.Description) > 1024 {
			t.Errorf("skills/%s/SKILL.md: frontmatter %+v (%v): a name as its folder's, a description of 1 to 1024 characters", d.Name(), front, err)
		}
		shipped[d.Name()] = true
	}
	roles, _ := fs.ReadDir(Roles, "roles")
	for _, r := range roles {
		data, err := fs.ReadFile(Roles, "roles/"+r.Name()+"/role.yaml")
		if err != nil {
			continue
		}
		var role struct {
			Context struct{ Skills []string } `yaml:"context"`
		}
		if err := yaml.Unmarshal(data, &role); err != nil {
			t.Errorf("roles/%s/role.yaml: %v", r.Name(), err)
		}
		for _, s := range role.Context.Skills {
			if !shipped[s] {
				t.Errorf("roles/%s declares the skill %q, which is not shipped in skills/", r.Name(), s)
			}
		}
	}
}
