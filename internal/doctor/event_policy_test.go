package doctor

import (
  "os"
  "os/exec"
  "path/filepath"
  "strings"
  "testing"
)

// Person told before: Given a public repository whose line depends on
// pull_request_target with no event policy allowing it, When the person
// runs `workline doctor`, Then it names the blocked event and the setting
// to change, or says the line does not need it.
func TestDoctorTellsAPublicRepositoryOfTheBlockedPullRequestTarget(t *testing.T) {
  isolate(t)
  repo := t.TempDir()
  git := func(args ...string) {
    t.Helper()
    cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
    cmd.Env = append(os.Environ(),
      "GIT_AUTHOR_NAME=Tester", "GIT_AUTHOR_EMAIL=tester@example.com",
      "GIT_COMMITTER_NAME=Tester", "GIT_COMMITTER_EMAIL=tester@example.com")
    if out, err := cmd.CombinedOutput(); err != nil {
      t.Fatalf("git %v: %v: %s", args, err, out)
    }
  }
  git("init", "-q")
  git("remote", "add", "origin", "https://github.com/example/public-repo.git")

  if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
    t.Fatal(err)
  }
  if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte("forge: github\n"), 0o644); err != nil {
    t.Fatal(err)
  }
  dir := filepath.Join(repo, ".github", "workflows")
  if err := os.MkdirAll(dir, 0o755); err != nil {
    t.Fatal(err)
  }
  // The line's bounds check, run by pull_request_target as the templates have it.
  wf := "name: workline bounds\non:\n  pull_request_target: {types: [opened, synchronize, reopened]}\npermissions: {}\n" +
    "jobs:\n  bounds:\n    runs-on: ubuntu-latest\n    steps:\n      - run: workline route merge-request --forge github\n"
  if err := os.WriteFile(filepath.Join(dir, "workline-bounds.yml"), []byte(wf), 0o644); err != nil {
    t.Fatal(err)
  }

  // A gh that says the repository is public, and no event policy allowing the event.
  bin := t.TempDir()
  script := "#!/bin/sh\ncase \"$*\" in\n*rules/branches*|*rulesets*|*default_branch*) exit 1 ;;\n" +
    "*) echo '{\"visibility\":\"public\",\"private\":false,\"isPrivate\":false}' ;;\nesac\n"
  if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
    t.Fatal(err)
  }
  t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

  r := Run(Options{Repo: repo, RolesDir: rolesDir, Engine: "v0.71.0"})
  for _, c := range r.Checks {
    if strings.Contains(c.Message, "pull_request_target") && c.Fix != "" {
      return
    }
  }
  t.Errorf("no check names the blocked pull_request_target with the setting to change in its fix: %+v", r.Checks)
}
