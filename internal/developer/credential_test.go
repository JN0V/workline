package developer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Each way git would hand the agent a credential is found, by its key or
// its file, never its value; a helper reset by an empty value is not one.
func TestCredentialsInReach(t *testing.T) {
	cases := []struct {
		name   string
		config [][]string // git config calls, in the repository
		global string     // the global config's text
		file   string     // a .git-credentials in HOME
		where  []string   // the findings' wheres; none: nothing found
	}{
		{name: "nothing"},
		{name: "persisted header", config: [][]string{{"http.https://github.com/.extraheader", "AUTHORIZATION: basic c2VjcmV0"}},
			where: []string{"http.https://github.com/.extraheader"}},
		{name: "header reset", config: [][]string{{"--add", "http.extraheader", "AUTHORIZATION: basic c2VjcmV0"}, {"--add", "http.extraheader", ""}}},
		{name: "global helper", global: "[credential \"https://gitlab.com\"]\n\thelper = !glab auth git-credential\n",
			where: []string{"credential.https://gitlab.com.helper"}},
		{name: "helper reset in the repository", global: "[credential]\n\thelper = store\n", config: [][]string{{"credential.helper", ""}}},
		{name: "token in the remote", config: [][]string{{"remote.origin.url", "https://gitlab-ci-token:c2VjcmV0@gitlab.example/p.git"}},
			where: []string{"remote.origin.url"}},
		{name: "token in a rewrite", config: [][]string{{"url.https://x:c2VjcmV0@github.com/.insteadOf", "https://github.com/"}},
			where: []string{"url.https://…@github.com/.insteadof"}},
		{name: "saved credentials", file: "https://u:c2VjcmV0@github.com\n", where: []string{".git-credentials"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			home, repo := t.TempDir(), t.TempDir()
			global := filepath.Join(home, "gitconfig")
			if err := os.WriteFile(global, []byte(c.global), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Setenv("HOME", home)
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
			t.Setenv("GIT_CONFIG_GLOBAL", global)
			t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
			if c.file != "" {
				if err := os.WriteFile(filepath.Join(home, ".git-credentials"), []byte(c.file), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
				t.Fatal(err, string(out))
			}
			for _, args := range c.config {
				if out, err := exec.Command("git", append([]string{"-C", repo, "config"}, args...)...).CombinedOutput(); err != nil {
					t.Fatal(err, string(out))
				}
			}
			found := credentialsInReach(repo)
			if len(found) != len(c.where) {
				t.Fatalf("found %v, want %v", found, c.where)
			}
			for i, f := range found {
				if f.Rule != ruleCredential || !strings.HasSuffix(f.Where, c.where[i]) {
					t.Errorf("finding %d: %+v, want where %s", i, f, c.where[i])
				}
				if strings.Contains(f.Where+f.Message, "c2VjcmV0") {
					t.Errorf("finding %d tells the credential: %+v", i, f)
				}
			}
		})
	}
}
