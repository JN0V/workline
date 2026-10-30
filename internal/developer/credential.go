package developer

// No credential in reach (ADR-0056, step 2): the agent works in a worktree
// of the repository, runs commands there, and its git reads the
// repository's config. The engine already keeps the forge's tokens out of
// its environment (agent.ForgeTokens); here it refuses to start it while
// git would hand it one anyway: a header git sends (a checkout's persisted
// token), a credential helper, a URL holding a token, a file the `store`
// helper keeps. Checked, not trusted (principle 7): a refusal is a block,
// said, nothing claimed; its findings name a key or a file, never a value.

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/JN0V/workline/internal/verdict"
)

// ruleCredential is the rule of a credential in the agent's reach.
const ruleCredential = "credential-in-reach"

// userinfo is an http(s) URL's user part, where a token rides.
var userinfo = regexp.MustCompile(`(?i)(https?://)[^/@\s]+@`)

// credentialsInReach are the credentials git would hand an agent working
// in repo, as git reads its config there: system, global, the
// repository's, its includes, GIT_CONFIG_* in the environment. A
// config git cannot read is one: nothing is known to be out of reach.
func credentialsInReach(repo string) []verdict.Finding {
	var found []verdict.Finding
	add := func(where, why string) {
		found = append(found, verdict.Finding{Rule: ruleCredential, Where: where, Message: why})
	}
	cmd := exec.Command("git", "-C", repo, "config", "--list", "--includes", "-z")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		add("git config", "its git config cannot be read: "+strings.TrimSpace(stderr.String()))
		return found
	}
	// Multi-valued keys read in order, an empty value resetting the list
	// (git's rule for helpers and headers): only what is left counts.
	var keys []string
	left := map[string]bool{}
	for _, entry := range strings.Split(string(out), "\x00") {
		key, value, _ := strings.Cut(entry, "\n")
		if key == "" {
			continue
		}
		lower := strings.ToLower(key)
		switch {
		case strings.HasPrefix(lower, "http.") && strings.HasSuffix(lower, ".extraheader"),
			strings.HasPrefix(lower, "credential.") && strings.HasSuffix(lower, ".helper"):
			if _, seen := left[key]; !seen {
				keys = append(keys, key)
			}
			left[key] = strings.TrimSpace(value) != ""
		case strings.HasPrefix(lower, "url.") && (strings.HasSuffix(lower, ".insteadof") || strings.HasSuffix(lower, ".pushinsteadof")):
			if userinfo.MatchString(key) {
				add(redact(key), "a URL git rewrites to holds a user and a token")
			}
		case strings.HasSuffix(lower, ".url") || strings.HasSuffix(lower, ".pushurl"):
			if userinfo.MatchString(value) {
				add(key, "its URL holds a user and a token")
			}
		}
	}
	for _, key := range keys {
		if !left[key] {
			continue
		}
		if strings.HasSuffix(strings.ToLower(key), ".helper") {
			add(redact(key), "a credential helper hands a token to any git command that asks")
		} else {
			add(redact(key), "git sends this header with each request: a checkout's persisted token")
		}
	}
	for _, f := range credentialFiles() {
		if st, err := os.Stat(f); err == nil && st.Size() > 0 {
			add(f, "a file of saved credentials, readable by the agent")
		}
	}
	return found
}

// credentialFiles are where git's `store` helper keeps credentials.
func credentialFiles() []string {
	var files []string
	if home, err := os.UserHomeDir(); err == nil {
		files = append(files, filepath.Join(home, ".git-credentials"))
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		files = append(files, filepath.Join(x, "git", "credentials"))
	} else if home, err := os.UserHomeDir(); err == nil {
		files = append(files, filepath.Join(home, ".config", "git", "credentials"))
	}
	return files
}

// redact hides a URL's user part in a key, where a token may ride.
func redact(key string) string { return userinfo.ReplaceAllString(key, "${1}…@") }

// refuseCredentials is the block when a credential is in the agent's
// reach: its findings, and how to take it away. nil: none.
func refuseCredentials(repo string, findings []verdict.Finding) *verdict.Verdict {
	found := credentialsInReach(repo)
	if len(found) == 0 {
		return nil
	}
	for i := range found {
		found[i].Message += ": the agent is not started (principle 6). In CI, check out with `persist-credentials: false` (GitHub) or reset the remote's URL (GitLab); on a machine, `git config credential.helper ''` in the repository"
	}
	return &verdict.Verdict{Status: verdict.Block, Summary: "a credential is in the agent's reach: the agent is not started",
		Findings: append(findings, found...)}
}
