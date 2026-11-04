package productowner

import (
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/codenav"
	"github.com/JN0V/workline/internal/forge"
)

const (
	// definedMax is the files given for a function an issue names that
	// more than two files hold: those defining it, each copy (#1057).
	definedMax = 8
	// definedScan is the files holding a name read for its definitions.
	definedScan = 40
	// pastAround is the lines given either side of a line past a file's
	// cut that names what the issue quotes, no function found around it.
	pastAround = 3
	// pastMax is the lines given past a file's cut, whatever the budget
	// left: a name on every line of a long file gives a few, not the file
	// (#1063).
	pastMax = 120
)

// defining are the files of files that define name as a function: the
// project's own first, then its tests' and examples' copies, each in
// their order; none when too many hold it to read.
func defining(repo, name string, files []string) []string {
	if len(files) > definedScan {
		return nil
	}
	key := repo + "\x00" + name
	if out, ok := definedSeen[key]; ok {
		return out
	}
	out := findDefining(repo, name, files)
	definedSeen[key] = out
	return out
}

// definedSeen keeps what defining found in a run: named for each issue,
// then what it left out said (definedRest).
var definedSeen = map[string][]string{}

// definedRest tells the agent, for each function an issue names that more
// files define than are given, the files left out, by path: it names the
// ones it needs as `sources` (#1057). "" for none.
func definedRest(repo string, is forge.Issue, comments, given []string) string {
	var out strings.Builder
	for _, name := range quoted(is, comments) {
		files, defs := holding(repo, name)
		if !defs {
			continue
		}
		var rest []string
		aside := 0
		for _, f := range files {
			switch {
			case slices.Contains(given, f):
			case asideRank(f) > 0:
				aside++ // a test's or an example's copy: counted, not listed
			default:
				rest = append(rest, f)
			}
		}
		if len(rest) > 0 || aside > 0 {
			said := strings.Join(rest, ", ")
			if aside > 0 {
				said = strings.TrimPrefix(fmt.Sprintf("%s, and %d in tests, examples or tools", said, aside), ", ")
			}
			fmt.Fprintf(&out, "`%s` is defined in %d files; not given here: %s. Name those it is about with `sources`.\n", name, len(files), said)
		}
	}
	return out.String()
}

func findDefining(repo, name string, files []string) []string {
	var out []string
	files = slices.Clone(files)
	slices.SortStableFunc(files, func(a, b string) int { return asideRank(a) - asideRank(b) })
	for _, f := range files {
		if codenav.Of(f) == nil {
			continue
		}
		data, err := exec.Command("git", "-C", repo, "show", "HEAD:"+f).Output()
		if err != nil {
			continue
		}
		if len(codenav.Named(f, strings.Split(string(data), "\n"), name)) > 0 {
			out = append(out, f)
		}
	}
	return out
}

// asideFolder is a folder of code beside the project's own: its tests,
// examples and tools.
var asideFolder = regexp.MustCompile(`(?i)(^|/)(tests?|examples?|tools|mocks?|testdata)/`)

// asideRank is 1 for a file in such a folder, 0 for the project's own.
func asideRank(f string) int {
	if asideFolder.MatchString(f) {
		return 1
	}
	return 0
}

// quoted are the names of the code an issue and a person's comments
// quote, as symbol reads them, `A::b` as `b`.
func quoted(is forge.Issue, comments []string) []string {
	text := is.Title + "\n" + is.Body
	for _, c := range comments {
		if !strings.Contains(c, "<!-- workline:") {
			text += "\n" + c
		}
	}
	var out []string
	for _, m := range symbol.FindAllStringSubmatch(text, -1) {
		name := m[1]
		if i := strings.LastIndex(name, "::"); i >= 0 {
			name = name[i+2:]
		}
		if len(name) >= 4 && !slices.Contains(out, name) {
			out = append(out, name)
		}
	}
	return out
}

// namedPast gives, of a file cut after its first given lines, the lines
// past the cut that name what the issues quote (#1056): the function
// around each, else a few lines either side, numbered as in the file,
// within budget. It returns what is written after the file and the
// budget left.
func namedPast(file string, lines []string, given int, names []string, budget int) (string, int) {
	if len(names) == 0 || budget <= 0 {
		return "", budget
	}
	left := budget - min(budget, pastMax) // given back after
	budget = min(budget, pastMax)
	var words []string
	for _, n := range names {
		words = append(words, regexp.QuoteMeta(n))
	}
	re := regexp.MustCompile(`\b(?:` + strings.Join(words, "|") + `)\b`)
	var spans [][2]int
	for i := given; i < len(lines); i++ {
		if !re.MatchString(lines[i]) {
			continue
		}
		n := i + 1
		if f, ok := codenav.Enclosing(file, lines, n); ok && f.From > given && f.To-f.From < codeLines {
			spans = append(spans, [2]int{f.From, f.To})
			continue
		}
		spans = append(spans, [2]int{max(n-pastAround, given+1), min(n+pastAround, len(lines))})
	}
	slices.SortFunc(spans, func(a, b [2]int) int { return a[0] - b[0] })
	var merged [][2]int
	for _, s := range spans {
		if k := len(merged) - 1; k >= 0 && s[0] <= merged[k][1]+1 {
			merged[k][1] = max(merged[k][1], s[1])
			continue
		}
		merged = append(merged, s)
	}
	var b strings.Builder
	for _, s := range merged {
		if budget <= 0 {
			break
		}
		to := min(s[1], s[0]+budget-1)
		if b.Len() > 0 {
			b.WriteString("   …\n")
		}
		for n := s[0]; n <= to; n++ {
			fmt.Fprintf(&b, "%4d  %s\n", n, lines[n-1])
		}
		budget -= to - s[0] + 1
	}
	if b.Len() == 0 {
		return "", budget + left
	}
	return fmt.Sprintf("\n\nPast them, the lines naming %s:\n\n```\n%s```", "`"+strings.Join(names, "`, `")+"`", b.String()), budget + left
}
