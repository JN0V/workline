package documentalist

import "testing"

func TestSection(t *testing.T) {
	doc := "---\nchecked: abc\n---\n# Auth\n\nIntro.\n\n## Token refresh\n\nOne hour.\n\n### Detail\n\nMore.\n\n## Sign out\n\nBye.\n"
	got, ok := Section(doc, "token-refresh")
	if !ok || got != "## Token refresh\n\nOne hour.\n\n### Detail\n\nMore." {
		t.Fatalf("Section = %q, %v", got, ok)
	}
	if _, ok := Section(doc, "missing"); ok {
		t.Fatal("a missing anchor must be reported")
	}
	a, _ := Section("---\nchecked: abc\n---\n# T\n\nSame.\n", "")
	b, _ := Section("---\nchecked: def\n---\n# T\n\nSame.\n", "")
	if a != b {
		t.Fatal("changing only the frontmatter is not a change")
	}
}

func TestSplitSource(t *testing.T) {
	for _, c := range []struct{ in, repo, path, anchor string }{
		{"src/a.go", "", "src/a.go", ""},
		{"docs/tech/auth.md#token-refresh", "", "docs/tech/auth.md", "token-refresh"},
		{"api:docs/x.md#part", "api", "docs/x.md", "part"},
	} {
		r, p, a := splitSource(c.in)
		if r != c.repo || p != c.path || a != c.anchor {
			t.Errorf("splitSource(%q) = %q %q %q", c.in, r, p, a)
		}
	}
}

func TestParseDocKeepsCommitsAsWritten(t *testing.T) {
	for _, c := range []struct{ fm, repo, want string }{
		{"checked: 11180e1", "", "11180e1"}, // a number, read as YAML
		{"checked: 1234567", "", "1234567"},
		{"checked: {api: 0012e45}", "api", "0012e45"},
	} {
		d, err := ParseDoc("d.md", []byte("---\nsources: [a.go]\n"+c.fm+"\n---\n"))
		if err != nil || d.Checked[c.repo] != c.want {
			t.Errorf("%s: checked = %v, %v", c.fm, d.Checked, err)
		}
	}
}
