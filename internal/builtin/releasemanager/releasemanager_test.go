package releasemanager

import (
	"strings"
	"testing"
	"time"
)

func TestBump(t *testing.T) {
	cases := []struct {
		name     string
		subjects []string
		body     string
		want     string
	}{
		{"fix only", []string{"fix: a"}, "", "patch"},
		{"feature wins over fix", []string{"fix: a", "feat: b"}, "", "minor"},
		{"breaking mark", []string{"feat!: drop the old API"}, "", "major"},
		{"breaking footer", []string{"refactor: rename"}, "BREAKING CHANGE: the flag is gone", "major"},
		{"docs and chores release nothing", []string{"docs: a", "chore: b"}, "", ""},
		{"non-conventional releases nothing", []string{"Update README"}, "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var commits []Commit
			for _, s := range c.subjects {
				commits = append(commits, Parse(s, c.body))
			}
			if got := Bump(commits); got != c.want {
				t.Fatalf("Bump = %q, want %q", got, c.want)
			}
		})
	}
}

func TestNextSemver(t *testing.T) {
	for _, c := range []struct{ cur, bump, want string }{
		{"1.2.0", "patch", "1.2.1"}, {"1.2.3", "minor", "1.3.0"}, {"1.2.3", "major", "2.0.0"}, {"0.0.0", "minor", "0.1.0"},
	} {
		if got, err := NextSemver(c.cur, c.bump); err != nil || got != c.want {
			t.Errorf("NextSemver(%s, %s) = %s, %v; want %s", c.cur, c.bump, got, err, c.want)
		}
	}
	if _, err := NextSemver("1.2", "patch"); err == nil {
		t.Error("a malformed version must be refused")
	}
}

func TestNextCalver(t *testing.T) {
	d := time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)
	for _, c := range []struct{ cur, want string }{
		{"", "2026.09.0"}, {"2026.09.0", "2026.09.1"}, {"2026.08.4", "2026.09.0"},
	} {
		if got := NextCalver(c.cur, d); got != c.want {
			t.Errorf("NextCalver(%q) = %s, want %s", c.cur, got, c.want)
		}
	}
}

func TestSectionAndInsert(t *testing.T) {
	d := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	s := Section("v1.3.0", d, []Commit{
		Parse("feat(export): let users export CSV", ""),
		Parse("fix: stop crashing on empty input", ""),
		Parse("chore: tidy", ""),
	})
	for _, want := range []string{"## v1.3.0 — 2026-01-03", "### Features", "- **export:** let users export CSV", "### Fixes"} {
		if !strings.Contains(s, want) {
			t.Errorf("section lacks %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "tidy") {
		t.Error("chores must not reach the changelog")
	}
	got := Insert("# Changelog\n\n## v1.2.0\n\n- old\n", s)
	if strings.Index(got, "v1.3.0") > strings.Index(got, "v1.2.0") {
		t.Errorf("the new section must come first:\n%s", got)
	}
	if !strings.HasPrefix(Insert("", s), "# Changelog\n\n## v1.3.0") {
		t.Error("an empty changelog gets a title")
	}
}
