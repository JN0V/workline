package forge

// VariableScopes is a forge whose CI/CD variables are each given to the
// jobs of some environments only: GitLab's environment scope, on every
// tier for a project's variables (#520).
type VariableScopes interface {
	// VariableScopes gives the environment scopes of each of the
	// project's own variables, by its key: `*` for every job. A token
	// that may not list them (GitLab asks a Maintainer's) is an error.
	VariableScopes() (map[string][]string, error)
}

var _ VariableScopes = &gitlab{}

// VariableScopes reads the project's CI/CD variables. GitLab's answer
// holds their values too: only each key and scope is decoded, and an
// error says the call, never the answer's variables.
func (g *gitlab) VariableScopes() (map[string][]string, error) {
	out, err := g.api("--paginate", "projects/:id/variables?per_page=100")
	if err != nil {
		return nil, err
	}
	vars, err := pages[struct {
		Key   string `json:"key"`
		Scope string `json:"environment_scope"`
	}](out)
	if err != nil {
		return nil, err
	}
	scopes := map[string][]string{}
	for _, v := range vars {
		scopes[v.Key] = append(scopes[v.Key], v.Scope)
	}
	return scopes, nil
}
