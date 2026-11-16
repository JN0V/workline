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

// eventPolicyRepo makes a GitHub-remote repository whose bounds workflow
// runs on pull_request_target.
func eventPolicyRepo(t *testing.T) string {
  t.Helper()
  repo := t.TempDir()
  cmd := exec.Command("git", "-C", repo, "init", "-q")
  if out, err := cmd.CombinedOutput(); err != nil {
    t.Fatalf("git init: %v: %s", err, out)
  }
  cmd = exec.Command("git", "-C", repo, "remote", "add", "origin", "https://github.com/example/public-repo.git")
  if out, err := cmd.CombinedOutput(); err != nil {
    t.Fatalf("git remote add: %v: %s", err, out)
  }
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
  wf := "name: workline bounds\non:\n  pull_request_target: {types: [opened, synchronize, reopened]}\npermissions: {}\n" +
    "jobs:\n  bounds:\n    runs-on: ubuntu-latest\n    steps:\n      - run: workline route merge-request --forge github\n"
  if err := os.WriteFile(filepath.Join(dir, "workline-bounds.yml"), []byte(wf), 0o644); err != nil {
    t.Fatal(err)
  }
  return repo
}

// fakeGh puts a gh first on PATH that runs the given shell script body.
func fakeGh(t *testing.T, body string) {
  t.Helper()
  bin := t.TempDir()
  if err := os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\n"+body), 0o755); err != nil {
    t.Fatal(err)
  }
  t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// Person told before: Given a public repository whose line depends on
// pull_request_target and an Actions policy naming that event, When the
// person runs `workline doctor`, Then it reports the event as allowed (OK)
// and warns of no block.
func TestDoctorAcceptsAPublicRepositoryWhosePolicyNamesPullRequestTarget(t *testing.T) {
  isolate(t)
  repo := eventPolicyRepo(t)
  fakeGh(t, "case \"$*\" in\n*actions/policies*) echo '{\"total_count\":1,\"policies\":[{\"id\":1,\"name\":\"workline\",\"target\":\"actions\",\"source_type\":\"Repository\",\"source\":\"example/public-repo\",\"enforcement\":\"active\",\"conditions\":{\"workflow_path\":{\"include\":[\".github/workflows/workline-bounds.yml\"],\"exclude\":[]}},\"rules\":[{\"type\":\"restrict_action_events\",\"parameters\":{\"allowed_events\":[\"pull_request_target\"]}}]}]}' ;;\n"+
    "*rules/branches*|*rulesets*|*default_branch*) exit 1 ;;\n"+
    "*) echo '{\"visibility\":\"public\",\"private\":false}' ;;\nesac\n")

  r := Run(Options{Repo: repo, RolesDir: rolesDir, Engine: "v0.71.0"})
  found := false
  for _, c := range r.Checks {
    if !strings.Contains(c.Message, "pull_request_target") {
      continue
    }
    if c.Level != OK {
      t.Errorf("a policy names pull_request_target, yet a check about it is not OK: %+v", c)
    } else {
      found = true
    }
  }
  if !found {
    t.Errorf("no OK check says the policy allows pull_request_target: %+v", r.Checks)
  }
}

// Person told before: Given a repository whose line depends on
// pull_request_target and whose visibility gh cannot read, When the person
// runs `workline doctor`, Then it notes that the repository could not be
// read and points to the docs.
func TestDoctorNotesWhenTheRepositoryCannotBeRead(t *testing.T) {
  isolate(t)
  repo := eventPolicyRepo(t)
  fakeGh(t, "exit 1\n")

  r := Run(Options{Repo: repo, RolesDir: rolesDir, Engine: "v0.71.0"})
  for _, c := range r.Checks {
    if strings.Contains(c.Message, "pull_request_target") && strings.Contains(c.Message, "could not be read") {
      if c.Level != Note {
        t.Errorf("unreadable repository is level %v, want Note: %+v", c.Level, c)
      }
      if !strings.Contains(c.Fix, "pull-request-target.md") {
        t.Errorf("fix does not point to the docs: %q", c.Fix)
      }
      return
    }
  }
  t.Errorf("no check says the repository could not be read: %+v", r.Checks)
}

// Person told before: Given a public repository whose line depends on
// pull_request_target and whose Actions policies gh cannot read, When the
// person runs `workline doctor`, Then it notes the policies could not be
// read and names the setting to change.
func TestDoctorNotesWhenTheActionsPoliciesCannotBeRead(t *testing.T) {
  isolate(t)
  repo := eventPolicyRepo(t)
  fakeGh(t, "case \"$*\" in\n*actions/policies*|*rules/branches*|*rulesets*|*default_branch*) exit 1 ;;\n"+
    "*) echo '{\"visibility\":\"public\",\"private\":false}' ;;\nesac\n")

  r := Run(Options{Repo: repo, RolesDir: rolesDir, Engine: "v0.71.0"})
  for _, c := range r.Checks {
    if strings.Contains(c.Message, "pull_request_target") && strings.Contains(c.Message, "policies could not be read") {
      if c.Level != Note {
        t.Errorf("unreadable policies is level %v, want Note: %+v", c.Level, c)
      }
      if !strings.Contains(c.Fix, "Settings") || !strings.Contains(c.Fix, "Policies") {
        t.Errorf("fix does not name the setting to change: %q", c.Fix)
      }
      return
    }
  }
  t.Errorf("no check says the Actions policies could not be read: %+v", r.Checks)
}
