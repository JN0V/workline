package role

import (
	"strings"
	"testing"
)

func TestCheckConfig(t *testing.T) {
	for _, c := range []struct{ config, want string }{
		{"ai: claude\nforge: github\nroles:\n  committer:\n    settings: {anything: 1}\n    enforce: {internal-code: warn}\n", ""},
		{"routing:\n  events: {merge-request: [committer]}\n  handoffs: [{from: a, to: b}]\n", ""},
		{"gates:\n  release:\n    checks: [{id: x, run: y, output: sarif, max: {error: 0}, timeout: 5m}]\n", ""},
		{"repos:\n  api: {url: ../api, branch: main}\n", ""},
		{"", ""},
		{"rolse: {}\n", `line 1: unknown key "rolse" at the top level (known: ai, forge, gates, repos, roles, routing)`},
		{"roles:\n  committer:\n    from: https://x\n", `unknown key "from" in roles.committer: a role taken from elsewhere`},
		{"routing:\n  handoffs: [{from: a, too: b}]\n", `unknown key "too" in routing.handoffs`},
		{"gates:\n  release:\n    checks: [{id: x, timout: 5m}]\n", `unknown key "timout" in gates.release.checks`},
	} {
		err := CheckConfig([]byte(c.config))
		switch {
		case c.want == "" && err != nil:
			t.Errorf("%q: %v", c.config, err)
		case c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)):
			t.Errorf("%q: got %v, want %q", c.config, err, c.want)
		}
	}
}
