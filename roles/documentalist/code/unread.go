package documentalist

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// Docs not read: the files that read as docs and that the role does not
// read, so that none of them is ever found suspect. The role reads Markdown
// under the docs globs; pip's 68 `.rst` docs, backstage's 113 `.mdx` and its
// 241 package READMEs went unseen, unsaid
// (docs/research/documentalist-genericity.md). They are said, once a run,
// with how to include those the role can read.

// unreadKinds are the formats the role does not read, by extension: no
// header form is settled for them (`.rst` and `.adoc` have no frontmatter,
// MDX takes no HTML comment), nor the reading of their links and sections.
var unreadKinds = []string{".rst", ".mdx", ".adoc", ".asciidoc"}

// notDocFolders hold what is no doc, whatever it is named: dependencies,
// tests and their data, the forge's own files.
var notDocFolders = map[string]bool{"node_modules": true, "vendor": true, "third_party": true,
	"third-party": true, "testdata": true, "fixtures": true, "__fixtures__": true,
	"__snapshots__": true, "__tests__": true, "test": true, "tests": true}

// notDocText are text files of a docs folder that are no doc.
var notDocText = []string{"requirements*.txt", "robots.txt", "cmakelists.txt", "license*", "*.in.txt"}

// unreadDocs sorts the tracked files that read as docs and that the role
// does not read, by kind: a format it does not read, a `.txt` under a docs
// folder, a README outside the docs globs.
func unreadDocs(files map[string]bool, globs []string) map[string][]string {
	out := map[string][]string{}
	for f := range files {
		if kind := unreadKind(f, globs); kind != "" {
			out[kind] = append(out[kind], f)
		}
	}
	for _, list := range out {
		sort.Strings(list)
	}
	return out
}

func unreadKind(f string, globs []string) string {
	dirs := strings.Split(path.Dir(f), "/")
	inDocs := false
	for _, d := range dirs {
		if notDocFolders[strings.ToLower(d)] || strings.HasPrefix(d, ".") && d != "." {
			return ""
		}
		inDocs = inDocs || strings.EqualFold(d, "docs") || strings.EqualFold(d, "doc")
	}
	base := strings.ToLower(path.Base(f))
	ext := path.Ext(base)
	for _, k := range unreadKinds {
		if ext == k {
			return k
		}
	}
	switch {
	case ext == ".txt" && inDocs:
		for _, g := range notDocText {
			if ok, _ := path.Match(g, base); ok {
				return ""
			}
		}
		return ".txt"
	case base == "readme.md" && !matchAny(globs, f):
		return "README.md"
	}
	return ""
}

// unreadMessage says the docs not read, by kind, a few of each, and how to
// include them; "" when there are none.
func unreadMessage(byKind map[string][]string) string {
	kinds := make([]string, 0, len(byKind))
	total := 0
	for k, list := range byKind {
		kinds = append(kinds, k)
		total += len(list)
	}
	if total == 0 {
		return ""
	}
	sort.Strings(kinds)
	var parts []string
	for _, k := range kinds {
		list := byKind[k]
		what := fmt.Sprintf("%d %s", len(list), k)
		if k == "README.md" {
			what += " outside the docs globs"
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", what, sample(list, 5)))
	}
	how := ""
	if len(byKind["README.md"]) > 0 {
		how += ` Markdown is read once its globs are in the documentalist's ` + "`docs`" + ` setting: ` + `"**/README.md"` + ` for every README.`
	}
	for _, k := range kinds {
		if k != "README.md" {
			how += " " + strings.Join(without(kinds, "README.md"), ", ") + " files are not read yet: a person keeps them true, or turns them into Markdown."
			break
		}
	}
	return fmt.Sprintf("%d files read as docs that the documentalist does not read, so none is ever found suspect: %s.%s",
		total, strings.Join(parts, "; "), how)
}

func sample(list []string, n int) string {
	if len(list) <= n {
		return strings.Join(list, ", ")
	}
	return strings.Join(list[:n], ", ") + fmt.Sprintf(", and %d more", len(list)-n)
}

func without(list []string, x string) []string {
	var out []string
	for _, s := range list {
		if s != x {
			out = append(out, s)
		}
	}
	return out
}

// UnreadDocs is what the doctor says of the docs the role does not read: ""
// when there are none.
func UnreadDocs(repo string, globs []string) (string, error) {
	files, err := trackedFiles(repo)
	if err != nil {
		return "", err
	}
	return unreadMessage(unreadDocs(files, globs)), nil
}
