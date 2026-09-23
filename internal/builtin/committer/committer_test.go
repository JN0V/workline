package committer

import "testing"

var defaults = Settings{
	SubjectMax:         72,
	BodyMaxLines:       12,
	Types:              []string{"feat", "fix", "docs", "test", "refactor", "perf", "build", "ci", "chore", "revert"},
	InternalCodes:      []string{`\b[A-Z]{1,4}(-NEW)?-[0-9]+\b`},
	InternalCodesAllow: []string{"UTF-8", "UTF-16", "SHA-1", "SHA-256", "SHA-512", "ISO-8601", "RFC-[0-9]+", "CVE-[0-9]+"},
}

func TestCheck(t *testing.T) {
	cases := []struct {
		name  string
		msg   string
		rules []string
	}{
		{"plain and clear", "fix: stop crashing on an empty input file", nil},
		{"scope and breaking mark", "feat(export)!: export data as CSV", nil},
		{"internal code in subject", "fix: AC-3", []string{"internal-code"}},
		{"several codes", "refactor: phase-4 fixes (F-NEW-1, L-019)", []string{"internal-code", "internal-code"}},
		{"standard identifiers", "fix: read names as UTF-8 and hash with SHA-256", nil},
		{"code in trailer", "fix: stop crashing on an empty input file\n\nRefs: AC-3", nil},
		{"trailers do not count as body", "fix: x y\n\nRefs: AC-3\nCo-Authored-By: A <a@b.c>", nil},
		{"no type", "stop crashing", []string{"format"}},
		{"unknown type", "wip: stuff", []string{"format"}},
		{"subject too long", "fix: " + string(make([]byte, 80)), []string{"subject-length"}},
		{"body too long", "fix: x\n\n1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n13", []string{"body-length"}},
		{"git comments ignored", "fix: x\n# Please enter the commit message", nil},
		{"merge commits skipped", "Merge branch 'main' into feature", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Check(c.msg, defaults)
			if err != nil {
				t.Fatal(err)
			}
			var rules []string
			for _, f := range got {
				rules = append(rules, f.Rule)
			}
			if len(rules) != len(c.rules) {
				t.Fatalf("rules = %v, want %v", rules, c.rules)
			}
			for i := range rules {
				if rules[i] != c.rules[i] {
					t.Fatalf("rules = %v, want %v", rules, c.rules)
				}
			}
		})
	}
}
