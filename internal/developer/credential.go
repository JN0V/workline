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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
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
	// Helpers and headers are multi-valued, scoped by URL, an empty value
	// resetting what came before: git's rules decide, at each URL they
	// could serve.
	var helpers, headers []scoped
	for _, entry := range strings.Split(string(out), "\x00") {
		key, value, _ := strings.Cut(entry, "\n")
		if key == "" {
			continue
		}
		lower := strings.ToLower(key)
		switch {
		case strings.HasPrefix(lower, "http.") && strings.HasSuffix(lower, ".extraheader"):
			headers = append(headers, scoped{key: key, pattern: middle(key, "http.", ".extraheader"), set: strings.TrimSpace(value) != ""})
		case strings.HasPrefix(lower, "credential.") && strings.HasSuffix(lower, ".helper"):
			helpers = append(helpers, scoped{key: key, pattern: middle(key, "credential.", ".helper"), set: strings.TrimSpace(value) != ""})
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
	keys, err := helpersLeft(repo, helpers)
	if err != nil {
		add("git config", "its credential helpers cannot be resolved: "+err.Error())
	}
	for _, key := range keys {
		add(redact(key), "a credential helper hands a token to any git command that asks")
	}
	keys, err = headersLeft(repo, headers)
	if err != nil {
		add("git config", "its http headers cannot be resolved: "+err.Error())
	}
	for _, key := range keys {
		add(redact(key), "git sends this header with each request: a checkout's persisted token")
	}
	for _, f := range credentialFiles() {
		if st, err := os.Stat(f); err == nil && st.Size() > 0 {
			add(f, "a file of saved credentials, readable by the agent")
		}
	}
	return found
}

// scoped is a helper's or a header's entry, in config order: its key,
// the URL pattern it is scoped to ("" for all), whether it sets a value
// (false: an empty value, which resets).
type scoped struct {
	key, pattern string
	set          bool
}

// middle is a key's subsection, between its section and its name: ""
// when it has none.
func middle(key, section, name string) string {
	if len(key) <= len(section)+len(name) {
		return ""
	}
	return key[len(section) : len(key)-len(name)]
}

// probeURL is a pattern as a URL it matches.
func probeURL(pattern string) string {
	u := strings.ReplaceAll(pattern, "*", "workline-probe")
	if !strings.Contains(u, "://") {
		u = "https://" + u
	}
	return u
}

// probes are the URLs a helper or header could serve: one no pattern
// matches, and the pattern of each entry that sets a value. A value left
// at its own pattern's URL is in reach; a value reset there is reset at
// every URL below it too, a reset's pattern being as wide as the URL's.
func probes(entries []scoped) []string {
	urls := []string{"https://workline.invalid/"}
	for _, e := range entries {
		if u := probeURL(e.pattern); e.set && e.pattern != "" && !slices.Contains(urls, u) {
			urls = append(urls, u)
		}
	}
	return urls
}

// helpersLeft are the keys of the helpers git would run for some URL. Git
// runs every helper whose pattern matches the URL, in config order, an
// empty value clearing those before it, whatever their scope; which
// pattern matches a URL, git says. A pattern that is no URL git may still
// match (a host alone): its helper counts, its reset does not.
func helpersLeft(repo string, helpers []scoped) ([]string, error) {
	var keys []string
	for _, u := range probes(helpers) {
		matched, err := patternsMatching(repo, helpers, u)
		if err != nil {
			return keys, err
		}
		var list []string
		for i, h := range helpers {
			url := h.pattern == "" || strings.Contains(h.pattern, "://")
			switch {
			case url && h.pattern != "" && !matched[i]:
			case h.set:
				list = append(list, h.key)
			case url:
				list = nil
			}
		}
		for _, k := range list {
			if !slices.Contains(keys, k) {
				keys = append(keys, k)
			}
		}
	}
	return keys, nil
}

// patternsMatching says, by git, which entries' patterns match url: each
// pattern is given a probe key, and `git config --get-urlmatch` lists the
// probes whose pattern matches.
func patternsMatching(repo string, entries []scoped, url string) (map[int]bool, error) {
	args := []string{"-C", repo}
	for i, e := range entries {
		if e.pattern != "" && !strings.Contains(e.pattern, "=") {
			args = append(args, "-c", fmt.Sprintf("credential.%s.worklineprobe%d=1", e.pattern, i))
		}
	}
	out, err := gitConfigOut(append(args, "config", "--get-urlmatch", "credential", url)...)
	if err != nil {
		return nil, err
	}
	matched := map[int]bool{}
	for _, line := range strings.Split(out, "\n") {
		key, _, _ := strings.Cut(line, " ")
		if n, ok := strings.CutPrefix(key, "credential.worklineprobe"); ok {
			if i, err := strconv.Atoi(n); err == nil {
				matched[i] = true
			}
		}
	}
	return matched, nil
}

// headersLeft are the keys of the headers git would send to some URL. Git
// resolves them: `git config --get-urlmatch http.extraheader` is the last
// value of the most specific pattern, empty when they were reset.
func headersLeft(repo string, headers []scoped) ([]string, error) {
	var keys []string
	for _, u := range probes(headers) {
		out, err := gitConfigOut("-C", repo, "config", "--get-urlmatch", "http.extraheader", u)
		if err != nil {
			return keys, err
		}
		if strings.TrimSpace(out) == "" {
			continue
		}
		key := "http.extraheader" // named: the last one setting a value at this URL
		for _, h := range headers {
			if h.set && (h.pattern == "" || probeURL(h.pattern) == u) {
				key = h.key
			}
		}
		if !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

// gitConfigOut is git's output; exit status 1 with nothing said (no value
// found) is no error.
func gitConfigOut(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 && stderr.Len() == 0 {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
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
