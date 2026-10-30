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
		// Git's resolution decides, not the key: a reset clears every helper
		// before it whose pattern it matches, whatever their key.
		{name: "scoped helper reset by an unscoped one", global: "[credential \"https://gitlab.com\"]\n\thelper = store\n",
			config: [][]string{{"credential.helper", ""}}},
		{name: "helper reset for another host only", global: "[credential \"https://gitlab.com\"]\n\thelper = store\n",
			config: [][]string{{"credential.https://github.com.helper", ""}}, where: []string{"credential.https://gitlab.com.helper"}},
		{name: "helper set after its reset", global: "[credential]\n\thelper = store\n",
			config: [][]string{{"--add", "credential.helper", ""}, {"--add", "credential.helper", "cache"}}, where: []string{"credential.helper"}},
		{name: "host-wide reset, path-scoped helper after it", config: [][]string{{"credential.https://gitlab.com.helper", ""}, {"credential.https://gitlab.com/group.helper", "store"}},
			where: []string{"credential.https://gitlab.com/group.helper"}},
		// A header's most specific pattern wins: an unscoped reset leaves a
		// scoped header sent, a scoped reset takes it.
		{name: "scoped header past an unscoped reset", global: "[http \"https://github.com/\"]\n\textraheader = AUTHORIZATION: basic c2VjcmV0\n",
			config: [][]string{{"http.extraheader", ""}}, where: []string{"http.https://github.com/.extraheader"}},
		{name: "scoped header reset in its scope", global: "[http \"https://github.com/\"]\n\textraheader = AUTHORIZATION: basic c2VjcmV0\n",
			config: [][]string{{"http.https://github.com/.extraheader", ""}}},
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
