package main_test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode"
)

// lotPages are the six user pages on triggers, the tick, troubleshooting,
// the local line and the packs.
var lotPages = []string{
	"docs/triggers.md",
	"docs/tick.md",
	"docs/troubleshooting.md",
	"docs/troubleshooting-product-owner.md",
	"docs/local-line.md",
	"docs/packs.md",
}

var (
	lotLink    = regexp.MustCompile(`\]\(([^)\s#]+\.md)(#[^)\s]*)?\)`)
	lotHeading = regexp.MustCompile(`^#{1,6}\s+(.*?)\s*#*\s*$`)
	lotMdLink  = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
)

// lotSourceFiles returns the distinct files a page's `sources` name, the
// part before the `#`, each read from the repository.
func lotSourceFiles(t *testing.T, page string) []string {
	t.Helper()
	sources, _, _ := userPageRead(t, page)
	var files []string
	for _, s := range sources {
		path, _, _ := strings.Cut(s, "#")
		if path != "" && !slices.Contains(files, path) {
			files = append(files, path)
		}
	}
	return files
}

// lotAnchors returns the anchors of a markdown file's headings, as GitHub
// writes them, those inside fenced code blocks left out.
func lotAnchors(t *testing.T, file string) []string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	fenced := false
	for _, l := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			fenced = !fenced
			continue
		}
		m := lotHeading.FindStringSubmatch(l)
		if fenced || m == nil {
			continue
		}
		title := strings.ReplaceAll(lotMdLink.ReplaceAllString(m[1], "$1"), "`", "")
		var b strings.Builder
		for _, r := range strings.ToLower(title) {
			switch {
			case r == ' ':
				b.WriteByte('-')
			case r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
				b.WriteRune(r)
			}
		}
		out = append(out, b.String())
	}
	return out
}

// Lot not suspect
//
// Given the merge commit of this change,
// When `workline docs --ai none` runs,
// Then none of the six pages (triggers, tick, troubleshooting,
// troubleshooting-product-owner, local-line, packs) is listed as suspect.
func TestLotPagesNoneSuspect(t *testing.T) {
	out := userPageDocsRun(t, false)
	for _, m := range userSuspect.FindAllStringSubmatch(out, -1) {
		if slices.Contains(lotPages, m[1]) {
			t.Errorf("%s is listed as suspect:\n%s", m[1], m[0])
		}
	}
}

// Names exist in the code
//
// Given each of the six pages,
// When each command, flag and finding name written in backticks on the page
// is searched in the files its `sources` name,
// Then each one is found in at least one of them.
func TestLotPagesNamesAreInTheirSources(t *testing.T) {
	for _, page := range lotPages {
		var corpus strings.Builder
		for _, f := range lotSourceFiles(t, page) {
			data, err := os.ReadFile(filepath.Join(userRepo, f))
			if err != nil {
				t.Errorf("%s: source %s cannot be read: %v", page, f, err)
				continue
			}
			corpus.Write(data)
			corpus.WriteByte('\n')
		}
		code := corpus.String()
		_, _, prose := userPageRead(t, page)
		seen := map[string]bool{}
		check := func(kind, name, span string) {
			if seen[kind+name] {
				return
			}
			seen[kind+name] = true
			if !strings.Contains(code, name) {
				t.Errorf("%s: the %s %q (in `%s`) is in none of its sources", page, kind, name, span)
			}
		}
		for _, m := range userSpan.FindAllStringSubmatch(prose, -1) {
			span := strings.Join(strings.Fields(m[1]), " ")
			fields := strings.Fields(span)
			if len(fields) == 0 {
				continue
			}
			if fields[0] == "workline" && len(fields) > 1 && userWord.MatchString(fields[1]) {
				check("command", fields[1], span)
			}
			if !userExternal[fields[0]] {
				for _, f := range fields {
					name, ok := strings.CutPrefix(f, "--")
					if !ok {
						continue
					}
					name, _, _ = strings.Cut(name, "=")
					name = strings.TrimRight(name, ",;.)>")
					if userFlagName.MatchString(name) {
						check("flag", name, span)
					}
				}
			}
			if userBare.MatchString(span) {
				check("finding", span, span)
			}
		}
	}
}

// Narrow sources
//
// Given each of the six pages,
// When the entries of its `sources` are counted,
// Then there are at most 15, and the file of each entry is named in the
// page text.
func TestLotPagesSourcesAreNarrow(t *testing.T) {
	for _, page := range lotPages {
		sources, _, _ := userPageRead(t, page)
		if len(sources) > 15 {
			t.Errorf("%s: %d sources, at most 15", page, len(sources))
		}
		data, err := os.ReadFile(filepath.Join(userRepo, page))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if strings.HasPrefix(text, "---\n") {
			_, text, _ = strings.Cut(text[4:], "\n---\n")
		}
		for _, f := range lotSourceFiles(t, page) {
			if !strings.Contains(text, f) && !strings.Contains(text, filepath.Base(f)) {
				t.Errorf("%s: the file %s of a source is not named in the page text", page, f)
			}
		}
	}
}

// Links the specification
//
// Given each of the six pages,
// When its text is searched for links to `docs/spec/`,
// Then at least one is found, and each linked file and its anchor exist.
func TestLotPagesLinkTheSpecification(t *testing.T) {
	for _, page := range lotPages {
		data, err := os.ReadFile(filepath.Join(userRepo, page))
		if err != nil {
			t.Fatal(err)
		}
		found := 0
		for _, m := range lotLink.FindAllStringSubmatch(string(data), -1) {
			target := filepath.Join(userRepo, filepath.Dir(page), m[1])
			if strings.HasPrefix(m[1], "docs/") {
				target = filepath.Join(userRepo, m[1])
			}
			if !strings.Contains(filepath.ToSlash(target), "docs/spec/") {
				continue
			}
			found++
			if _, err := os.Stat(target); err != nil {
				t.Errorf("%s: the link %s leads to no file", page, m[1])
				continue
			}
			if anchor := strings.TrimPrefix(m[2], "#"); anchor != "" && !slices.Contains(lotAnchors(t, target), anchor) {
				t.Errorf("%s: the link %s%s leads to no heading", page, m[1], m[2])
			}
		}
		if found == 0 {
			t.Errorf("%s: no link to docs/spec/", page)
		}
	}
}

// No internals
//
// Given each of the six pages,
// When its text outside code blocks is searched for the Go function and
// type names its `sources` give after the `#` sign (the common words and
// YAML keys, such as `on`, `jobs`, `Result` or `Due`, left out: only the
// names with a capital inside them are searched),
// Then none is found.
func TestLotPagesNameNoInternals(t *testing.T) {
	for _, page := range lotPages {
		sources, _, prose := userPageRead(t, page)
		for _, s := range sources {
			path, name, ok := strings.Cut(s, "#")
			if !ok || !strings.HasSuffix(path, ".go") {
				continue
			}
			for _, part := range strings.Split(name, ".") {
				inner := false
				for i, r := range part {
					if i > 0 && unicode.IsUpper(r) {
						inner = true
					}
				}
				if !inner {
					continue
				}
				if regexp.MustCompile(`\b` + regexp.QuoteMeta(part) + `\b`).MatchString(prose) {
					t.Errorf("%s: the prose names the Go identifier %s (source %s)", page, part, s)
				}
			}
		}
	}
}

// Judged fields kept
//
// Given docs/troubleshooting-product-owner.md,
// When the change is done,
// Then `checked` is changed in its header, and `judged` and
// `judged-in-parts` are left as they are.
func TestLotProductOwnerTroubleshootingKeepsItsJudgedFields(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(userRepo, "docs/troubleshooting-product-owner.md"))
	if err != nil {
		t.Fatal(err)
	}
	head, _, _ := strings.Cut(strings.TrimPrefix(string(data), "---\n"), "\n---\n")
	fields := map[string]string{}
	for _, l := range strings.Split(head, "\n") {
		if k, v, ok := strings.Cut(l, ":"); ok {
			fields[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	if fields["checked"] == "" || fields["checked"] == "cee5a7af" {
		t.Errorf("checked is %q: it is to be set to the current commit", fields["checked"])
	}
	if fields["judged"] != "5c1573b" {
		t.Errorf("judged is %q, left as it was: 5c1573b", fields["judged"])
	}
	if fields["judged-in-parts"] != "5c1573b" {
		t.Errorf("judged-in-parts is %q, left as it was: 5c1573b", fields["judged-in-parts"])
	}
}
