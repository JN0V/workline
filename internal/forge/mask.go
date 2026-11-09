package forge

import (
	"cmp"
	"os"
	"slices"
	"strings"
)

// Tokens are the variables holding what writes to the forge or signs for
// workline: the forge's tokens, the App's key, the signing key, and what
// a CI job holds that mints or carries one (GitHub's OIDC request token,
// GitLab's job token in its registry password and clone URL). No agent
// is handed one (principle 6): it proposes, the engine acts.
var Tokens = []string{
	"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN",
	"GITLAB_TOKEN", "GLAB_TOKEN", "WORKLINE_GITLAB_TOKEN", "WORKLINE_GITLAB_READ_TOKEN", "CI_JOB_TOKEN",
	"FORGEJO_TOKEN", "WORKLINE_APP_PRIVATE_KEY", "WORKLINE_SIGNING_KEY",
	"ACTIONS_ID_TOKEN_REQUEST_TOKEN", "ACTIONS_RUNTIME_TOKEN",
	"CI_REGISTRY_PASSWORD", "CI_DEPLOY_PASSWORD", "CI_JOB_JWT", "CI_JOB_JWT_V1", "CI_JOB_JWT_V2", "CI_REPOSITORY_URL",
	"CI_BUILD_TOKEN", "CI_DEPENDENCY_PROXY_PASSWORD",
}

// secretSuffixes end the names of the variables read as a secret, case
// aside, besides Tokens.
var secretSuffixes = []string{"_TOKEN", "_KEY", "_SECRET", "_PASSWORD", "_PAT", "_CREDENTIALS"}

// minSecret is the fewest characters a value is masked from: a shorter
// one ("1", "true") would mask words of the output, not a secret.
const minSecret = 8

// Masked is what a secret's value is replaced with.
const Masked = "***"

// Secrets are the values in env a command's output must never show
// quoted (#1215): those of Tokens and of the variables named like a
// secret, each line of a value on several lines too (a private key),
// of minSecret characters or more, the longest first.
func Secrets(env []string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(v string) {
		if v = strings.TrimSpace(v); len(v) >= minSecret && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	for _, kv := range env {
		name, value, ok := strings.Cut(kv, "=")
		if !ok || !secretName(name) {
			continue
		}
		add(value)
		if strings.Contains(value, "\n") {
			for _, line := range strings.Split(value, "\n") {
				add(line)
			}
		}
	}
	slices.SortStableFunc(out, func(a, b string) int { return cmp.Compare(len(b), len(a)) })
	return out
}

func secretName(name string) bool {
	if slices.Contains(Tokens, name) {
		return true
	}
	upper := strings.ToUpper(name)
	for _, s := range secretSuffixes {
		if strings.HasSuffix(upper, s) {
			return true
		}
	}
	return false
}

// Mask is s with each secret of the process's environment (Secrets)
// replaced by Masked: a command's output before a reason, a finding or
// a comment quotes it, and before it is cut, so no cut leaves a part of
// one.
func Mask(s string) string {
	return MaskWith(s, Secrets(os.Environ()))
}

// MaskWith is s with each of secrets replaced by Masked. Every match of
// every value is found on s as it came, overlapping matches merged into
// one span: a value replaced first never leaves part of another one,
// which overlaps it, shown.
func MaskWith(s string, secrets []string) string {
	type span struct{ from, to int }
	var spans []span
	for _, v := range secrets {
		if v == "" {
			continue
		}
		for at := 0; ; {
			i := strings.Index(s[at:], v)
			if i < 0 {
				break
			}
			spans = append(spans, span{at + i, at + i + len(v)})
			at += i + 1
		}
	}
	if len(spans) == 0 {
		return s
	}
	slices.SortFunc(spans, func(a, b span) int { return cmp.Compare(a.from, b.from) })
	var b strings.Builder
	last, cur := 0, spans[0]
	flush := func() {
		b.WriteString(s[last:cur.from])
		b.WriteString(Masked)
		last = cur.to
	}
	for _, sp := range spans[1:] {
		if sp.from <= cur.to {
			cur.to = max(cur.to, sp.to)
			continue
		}
		flush()
		cur = sp
	}
	flush()
	b.WriteString(s[last:])
	return b.String()
}
