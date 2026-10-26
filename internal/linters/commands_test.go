package linters

import (
	"slices"
	"testing"
)

func TestCommandsRunOnlyWhatTheyRun(t *testing.T) {
	for text, want := range map[string][]string{
		"python -m pip install yamllint":                    nil,
		"python3.12 -m pip install --user ruff":             nil,
		"python -m mypy .":                                  {"python -m mypy ."},
		"sudo -E apt-get -y install shellcheck":             nil,
		"pip --quiet install yamllint":                      nil,
		"npm --global install eslint && npx eslint .":       {"npx eslint ."},
		"rustup component add clippy; cargo clippy":         {"cargo clippy"},
		"echo 'later; hadolint Dockerfile'":                 nil,
		`bash -c "shellcheck a.sh"`:                         {"bash -c shellcheck a.sh"},
		"golangci-lint run # and shellcheck, later":         {"golangci-lint run"},
		"curl -sSfLO https://x/shellcheck.tar.xz | tar xJ":  nil,
		"git ls-files -z '*.sh' | xargs -0 -r shellcheck":   {"git ls-files -z *.sh", "xargs -0 -r shellcheck"},
		"\t@-golangci-lint run ./...":                       {"golangci-lint run ./..."},
		"FOO=1 env -i shellcheck \\\n  a.sh":                {"FOO=1 env -i shellcheck a.sh"},
		"docker pull hadolint/hadolint":                     nil,
		"docker run --rm -i hadolint/hadolint < Dockerfile": {"docker run --rm -i hadolint/hadolint < Dockerfile"},
	} {
		if got := commands(text); !slices.Equal(got, want) {
			t.Errorf("commands(%q) = %q, want %q", text, got, want)
		}
	}
}

func TestYAMLReadAsYAML(t *testing.T) {
	flow := `jobs: {lint: {steps: [{name: "Lint with hadolint"}, {run: "echo 'x; hadolint'"}, {run: 'shellcheck a.sh'}]}}`
	block := "jobs:\n  lint:\n    steps:\n      - name: >\n          Lint with\n          yamllint\n        run: |\n          python -m pip install ruff\n          ruff check .\n"
	for doc, want := range map[string][]string{flow: {"shellcheck a.sh"}, block: {"ruff check ."}} {
		if got := commandsOf(".github/workflows/lint.yml", []byte(doc)); !slices.Equal(got, want) {
			t.Errorf("commandsOf(%q) = %q, want %q", doc, got, want)
		}
	}
	hooks := "repos:\n  - repo: https://github.com/koalaman/shellcheck-precommit\n    hooks:\n      - id: shellcheck\n"
	if got := commandsOf(".pre-commit-config.yaml", []byte(hooks)); !slices.Equal(got, []string{"shellcheck"}) {
		t.Errorf("pre-commit: a hook's id is the tool it runs: %q", got)
	}
}
