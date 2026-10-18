package linters

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repo is a repository holding the files, added to the index: what git
// tracks is what is read.
func repo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, text := range files {
		p := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-f", "."}} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	return dir
}

func read(t *testing.T, files map[string]string) map[string]Language {
	t.Helper()
	r, err := Read(repo(t, files))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]Language{}
	for _, l := range r.Languages {
		out[l.Name] = l
	}
	return out
}

func TestTableIsTrunksWithItsNotice(t *testing.T) {
	k, err := load()
	if err != nil {
		t.Fatal(err)
	}
	var tab table
	if err := json.Unmarshal(trunkJSON, &tab); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(tab.Licence, "MIT License") || !strings.Contains(tab.Licence, "Trunk Technologies") {
		t.Errorf("trunk-io/plugins' notice is not kept: %.60q", tab.Licence)
	}
	if !strings.HasPrefix(tab.Source, "https://github.com/trunk-io/plugins/tree/") {
		t.Errorf("source = %q", tab.Source)
	}
	byName := map[string]Tool{}
	for _, x := range k.tools {
		byName[x.Name] = x
	}
	for name, want := range map[string]string{"shellcheck": "files", "eslint": "config", "clang-tidy": "config", "golangci-lint": "files"} {
		if got := byName[name].Suggest; got != want {
			t.Errorf("%s: suggest = %q, want %q", name, got, want)
		}
	}
	if got := byName["mypy"].Reads; len(got[Lint]) != 0 || len(got[Types]) == 0 {
		t.Errorf("mypy checks types, not lint: %v", got)
	}
}

func TestLanguagesByExtensionNameAndShebang(t *testing.T) {
	got := read(t, map[string]string{
		"main.go":         "package main\n",
		"tools/release":   "#!/usr/bin/env -S bash -e\necho\n",
		"bin/serve":       "#!/usr/bin/python3.12\nprint()\n",
		"notes":           "no #! line\n",
		"Dockerfile":      "FROM scratch\n",
		"src/a.cpp":       "int a;\n",
		"src/a.h":         "int a;\n",
		"assets/logo.png": "png",
	})
	if l := got["shell"]; len(l.Files) != 1 || l.Files[0] != "tools/release" || l.Shebang != 1 {
		t.Errorf("shell = %+v", l)
	}
	if l := got["python"]; len(l.Files) != 1 || l.Files[0] != "bin/serve" {
		t.Errorf("python = %+v", l)
	}
	if l := got["c/c++"]; len(l.Files) != 2 {
		t.Errorf("a source and a header are one language, c/c++: %+v", got)
	}
	if _, ok := got["png"]; ok {
		t.Errorf("an image no tool reads is not a language")
	}
	if _, ok := got["docker"]; !ok {
		t.Errorf("Dockerfile is told by its name: %v", got)
	}
}

func TestVendoredAndGeneratedLeftOut(t *testing.T) {
	dir := repo(t, map[string]string{
		"vendor/lib/a.sh":     "echo\n",
		"node_modules/x/b.sh": "echo\n",
		"gen/c.sh":            "echo\n",
		"vendor/ours/d.sh":    "echo\n",
		".gitattributes":      "gen/** linguist-generated\nvendor/ours/** -linguist-vendored\n",
	})
	r, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	if r.Skipped != 3 {
		t.Errorf("skipped = %d, want 3", r.Skipped)
	}
	for _, l := range r.Languages {
		if l.Name == "shell" && (len(l.Files) != 1 || l.Files[0] != "vendor/ours/d.sh") {
			t.Errorf("only the file .gitattributes takes back counts: %v", l.Files)
		}
	}
}

func TestWhatTheProjectRunsCounts(t *testing.T) {
	got := read(t, map[string]string{
		"deploy.sh":  "echo\n",
		"Dockerfile": "FROM scratch\n",
		"a.py":       "print()\n",
		"main.go":    "package main\n",
		".github/workflows/lint.yml": "jobs:\n  lint:\n    steps:\n" +
			"      - uses: ludeeus/action-shellcheck@2.0.0\n" +
			"      - run: cp .shellcheckrc x   # hadolint, later\n",
		".pre-commit-config.yaml": "repos:\n  - repo: https://github.com/dnephin/pre-commit-golang\n    hooks:\n      - id: go-vet\n",
		"package.json":            `{"scripts": {"lint": "ruff check ."}}`,
	})
	if s := got["shell"].Running[Lint]; len(s) != 1 || s[0].Tool != "shellcheck" || s[0].Where != ".github/workflows/lint.yml" {
		t.Errorf("shell: an action running shellcheck counts: %+v", got["shell"])
	}
	if len(got["docker"].Running) != 0 || got["docker"].Missing[Lint][0].Tool != "hadolint" {
		t.Errorf("docker: a comment runs nothing: %+v", got["docker"])
	}
	if s := got["go"].Running[Lint]; len(s) != 1 || s[0].Tool != "go vet" {
		t.Errorf("go: a pre-commit hook counts: %+v", got["go"])
	}
	if s := got["python"].Running[Lint]; len(s) != 1 || s[0].Tool != "ruff" || s[0].Where != "package.json" {
		t.Errorf("python: package.json's scripts count: %+v", got["python"])
	}
}

func TestGateChecksCountFirst(t *testing.T) {
	got := read(t, map[string]string{
		"main.go":                  "package main\n",
		".github/workflows/ci.yml": "jobs: {lint: {steps: [{run: golangci-lint run}]}}\n",
		".workline/config.yaml":    "gates:\n  lint:\n    checks:\n      - {id: go, run: golangci-lint run, output: exit}\n",
	})
	if s := got["go"].Running[Lint]; len(s) != 1 || s[0].Where != ".workline/config.yaml, gate lint" {
		t.Errorf("the gate check is said first: %+v", s)
	}
	if s := got["go"].Running[Types]; len(s) != 1 || s[0].Tool != "golangci-lint" {
		t.Errorf("golangci-lint type-checks as it lints: %+v", got["go"].Running)
	}
}

func TestSuggestedInOrder(t *testing.T) {
	got := read(t, map[string]string{"main.go": "package main\n"})
	m := got["go"].Missing[Lint]
	if len(m) < 2 || m[0].Tool != "go vet" || m[1].Tool != "golangci-lint" {
		t.Fatalf("go's own tool first, then trunk's: %+v", m)
	}
	if m[0].Check != `{id: go-vet, run: "go vet ./...", output: exit}` {
		t.Errorf("check = %s", m[0].Check)
	}
	got = read(t, map[string]string{"a.sh": "echo\n", "b.bash": "echo\n"})
	if c := got["shell"].Missing[Lint][0].Check; c != `{id: shellcheck, run: "git ls-files -z -- '*.bash' '*.sh' | xargs -0 -r shellcheck", output: exit}` {
		t.Errorf("check = %s", c)
	}
}

func TestProjectDependentOnlyWithItsConfig(t *testing.T) {
	got := read(t, map[string]string{"app.js": "1\n"})
	js := got["javascript"]
	if len(js.Missing) != 0 || len(js.Running) != 0 {
		t.Fatalf("no ESLint config, no suggestion: %+v", js)
	}
	found := false
	for _, s := range js.Unsuggested {
		found = found || (s.Tool == "eslint" && s.Config == "eslint.config.js")
	}
	if !found {
		t.Errorf("eslint is named with the config it needs: %+v", js.Unsuggested)
	}
	got = read(t, map[string]string{"web/app.js": "1\n", "web/.eslintrc.json": "{}\n"})
	if m := got["javascript"].Missing[Lint]; len(m) == 0 || m[0].Tool != "eslint" || m[0].Config != "web/.eslintrc.json" {
		t.Errorf("with its config, eslint is suggested first: %+v", got["javascript"])
	}
}

func TestEachFolderWhenTheConfigIsNotAtTheRoot(t *testing.T) {
	got := read(t, map[string]string{"lib/src/a.cpp": "int a;\n", "lib/platformio.ini": "[env]\n"})
	m := got["c/c++"].Missing[Lint]
	if len(m) == 0 || m[0].Tool != "pio check" || m[0].Each != "platformio.ini" {
		t.Errorf("pio check, from each folder holding a platformio.ini: %+v", m)
	}
	got = read(t, map[string]string{"src/a.cpp": "int a;\n", "platformio.ini": "[env]\n"})
	if m := got["c/c++"].Missing[Lint]; len(m) == 0 || m[0].Each != "" {
		t.Errorf("at the root, one check: %+v", m)
	}
}
