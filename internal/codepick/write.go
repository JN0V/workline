package codepick

import (
	"fmt"
	"strings"
)

// Style is how a role shows its pick.
type Style struct {
	Numbered bool                     // each line with its number in the file
	About    func(file string) string // said under a file's heading: its last commits
	Namer    string                   // what names the code: "the Scope", "the issue", "the spec"
	NoTest   bool                     // the test is the role's to show
}

// Write shows a pick: each file whole, by its first lines, or by the
// pieces the text names, the rest said; the test to write alike; what is
// defined in more places than given; the files one step away, as paths.
func (pk Pick) Write(b *strings.Builder, st Style) {
	namer := st.Namer
	if namer == "" {
		namer = "the issue"
	}
	budgetName := pk.Ask.BudgetName
	if budgetName == "" {
		budgetName = "the code budget"
	}
	var unread, above []string
	for _, f := range pk.Files {
		if f.Test {
			continue
		}
		if f.Above && len(f.Given) == 0 && len(f.Decls) == 0 {
			above = append(above, "`"+f.Path+"`")
			continue
		}
		if len(f.Given) == 0 && len(f.Decls) == 0 {
			if len(f.Unread) == 0 {
				// Named, never shown empty: a file given as 0 lines
				// read as an empty file to the agent (#801).
				unread = append(unread, fmt.Sprintf("- `%s` (%d lines)", f.Path, len(f.Lines)))
				continue
			}
		}
		pk.writeFile(b, f, st, namer, budgetName)
	}
	if len(above) > 0 {
		fmt.Fprintf(b, "\nGiven above, for another issue: %s.\n", strings.Join(above, ", "))
	}
	if len(unread) > 0 {
		fmt.Fprintf(b, "\nNot given, past %s (%d lines in all), not empty:\n\n%s\n", budgetName, pk.Ask.Budget, strings.Join(unread, "\n"))
	}
	if t := pk.Test(); t != nil && !st.NoTest && len(t.Given) > 0 {
		b.WriteString("\nA test already there, to write alike:\n")
		pk.writeFile(b, *t, st, namer, budgetName)
	}
	if len(pk.Many) > 0 {
		fmt.Fprintf(b, "\nDefined in more places than given: %s.\n", strings.Join(pk.Many, "; "))
	}
	if len(pk.Unshown) > 0 {
		fmt.Fprintf(b, "\nOther places naming them, not shown: %s.\n", strings.Join(pk.Unshown, ", "))
	}
	if len(pk.Linked) > 0 {
		fmt.Fprintf(b, "\nOne step away, not shown: %s.\n", strings.Join(pk.Linked, ", "))
	}
}

func (pk Pick) writeFile(b *strings.Builder, f File, st Style, namer, budgetName string) {
	about := ""
	if st.About != nil {
		about = st.About(f.Path)
	}
	n := len(f.Lines)
	switch {
	case len(f.Decls) > 0:
		var whole []string
		for _, p := range f.Given {
			whole = append(whole, fmt.Sprintf("lines %d to %d whole, %s", p.From, p.To, p.Why))
		}
		also := ""
		if len(whole) > 0 {
			also = "; " + strings.Join(whole, "; ")
		}
		fmt.Fprintf(b, "\n### `%s` (%d lines: its declarations, each body left out as `…`, %s%s)\n%s\n%s", f.Path, n, f.DeclsWhy, also, about, fence(f.Decls, 1, len(f.Decls), false))
	case f.Whole():
		fmt.Fprintf(b, "\n### `%s`\n%s\n%s", f.Path, about, fence(f.Lines, 1, n, st.Numbered))
	case len(f.Given) == 1 && f.Given[0].From == 1 && f.Given[0].Why == "its first lines":
		k := f.Given[0].To
		fmt.Fprintf(b, "\n### `%s`\n%s\n%s(cut: the first %d lines given)\n", f.Path, about, fence(f.Lines, 1, k, st.Numbered), k)
	default:
		var spans []string
		for _, p := range f.Given {
			spans = append(spans, fmt.Sprintf("%d to %d", p.From, p.To))
		}
		given := "none given"
		if len(spans) > 0 {
			given = "lines " + strings.Join(spans, ", ") + " given"
		}
		fmt.Fprintf(b, "\n### `%s` (%d lines: %s, what %s names; the rest not shown)\n%s", f.Path, n, given, namer, about)
		for _, p := range f.Given {
			fmt.Fprintf(b, "\nLines %d to %d, %s:\n\n%s", p.From, p.To, p.Why, fence(f.Lines, p.From, p.To, st.Numbered))
			if f.Cut > p.To && p == f.Given[len(f.Given)-1] {
				fmt.Fprintf(b, "(cut, past %s: what it names goes on to line %d)\n", budgetName, f.Cut)
			}
		}
		for _, p := range f.Unread {
			fmt.Fprintf(b, "\nNot given, past %s: lines %d to %d, %s.\n", budgetName, p.From, p.To, p.Why)
		}
	}
}

// fence shows lines from to of a file in a fence longer than any they hold.
func fence(lines []string, from, to int, numbered bool) string {
	var body strings.Builder
	for n := from; n <= to && n <= len(lines); n++ {
		if numbered {
			fmt.Fprintf(&body, "%4d  %s\n", n, lines[n-1])
		} else {
			body.WriteString(lines[n-1] + "\n")
		}
	}
	text := strings.TrimSuffix(body.String(), "\n")
	mark := "````"
	for strings.Contains(text, mark) {
		mark += "`"
	}
	return fmt.Sprintf("%s\n%s\n%s\n", mark, text, mark)
}
