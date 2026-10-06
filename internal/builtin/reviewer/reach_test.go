package reviewer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := t.TempDir()
	for name, text := range files {
		p := filepath.Join(repo, name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(text), 0o644)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "start"}} {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	return repo
}

const calcGo = `package calc

// Mean is the average of every value.
func Mean(values []int) int {
	return sum(values) / len(values)
}

// Report says the mean.
func Report(values []int) string {
	return fmt.Sprint(Mean(values))
}
`

const sumGo = `package calc

// sum adds the values after the first.
func sum(values []int) int {
	total := 0
	for _, v := range values[1:] {
		total += v
	}
	return total
}

func unrelated() {}
`

// The judge's code: the cause's function whole, the function it calls in
// another file and the one calling it, whole, up to the cap; the rest named
// with where it lies; never a test, nor a function it does not reach.
func TestReach(t *testing.T) {
	repo := gitRepo(t, map[string]string{
		"calc/calc.go":      calcGo,
		"calc/sum.go":       sumGo,
		"calc/calc_test.go": "package calc\n\nfunc TestMean() { Mean(nil) }\n",
		"other/other.go":    "package other\n\nfunc sum(v []int) int { return 0 }\n\nfunc use() int { return sum(nil) }\n",
	})
	// An unexported Go function is called only from its own package.
	g := Finding{Cause: Quote{Path: "calc/sum.go", Quote: "total += v"}}
	if got := reach(repo, "HEAD", nil, g, 7, 0, 200); !strings.Contains(got, "`Mean`, calls `sum`") || strings.Contains(got, "other.go") {
		t.Errorf("the callers of sum:\n%s", got)
	}
	f := Finding{Title: "Mean leaves out the first value", Why: "sum skips one",
		Cause: Quote{Path: "calc/calc.go", Quote: "return sum(values) / len(values)"}}
	got := reach(repo, "HEAD", []string{"**/*_test.*"}, f, 5, 0, 200)
	for _, want := range []string{
		"### calc/calc.go, lines 3 to 6: `Mean`, the function the cause lies in",
		"### calc/sum.go, lines 3 to 10: `sum`, called by `Mean`",
		"6  \tfor _, v := range values[1:] {",
		"### calc/calc.go, lines 8 to 11: `Report`, calls `Mean`",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	for _, not := range []string{"unrelated", "TestMean", "Not shown"} {
		if strings.Contains(got, not) {
			t.Errorf("holds %q:\n%s", not, got)
		}
	}
	got = reach(repo, "HEAD", nil, f, 5, 0, 10)
	if !strings.Contains(got, "Not shown, past 10 lines: `sum` (calc/sum.go:3-10, called by `Mean`)") ||
		!strings.Contains(got, "`Report`, calls `Mean`") {
		t.Errorf("the cap: sum named, Report (4 lines) still fits:\n%s", got)
	}
	if got := reach(repo, "HEAD", nil, f, 5, 0, 0); !strings.Contains(got, "lines 1 to 11: the lines around the cause") || strings.Contains(got, "sum.go") {
		t.Errorf("judge-lines-max 0: the lines around only:\n%s", got)
	}
	// A function the finding names, in another language than the cause's.
	f.Why = "the result is handed to `Report` unchecked"
	f.Cause = Quote{Path: "run.sh", Quote: "echo"}
	repo = gitRepo(t, map[string]string{"calc/calc.go": calcGo, "calc/sum.go": sumGo, "run.sh": "#!/bin/sh\nset -e\necho mean\n"})
	got = reach(repo, "HEAD", nil, f, 3, 0, 200)
	if !strings.Contains(got, "no function found around it (shell)") || !strings.Contains(got, "`Report`, named by the finding") {
		t.Errorf("a function named across languages:\n%s", got)
	}
}

// A function's bounds and calls with no parser: braces, indentation, `end`.
func TestEnclosingByLanguage(t *testing.T) {
	files := map[string]string{
		"a.sh":   "#!/bin/sh\n# api calls the forge.\napi() {\n  curl \"$1\" | jq '{a: .b}'\n}\nall() {\n  api \"$1\" # {\n  echo done\n}\nall x\n",
		"a.py":   "import os\n\n\ndef load(\n    path,\n):\n    if path:\n        return parse(path)\n\n    return None\n\n\ndef parse(p):\n    return p\n",
		"a.js":   "const add = (a, b) => {\n  return sum(a, b);\n};\n\nfunction sum(a, b) {\n  if (a) {\n    return a + b;\n  }\n}\n",
		"a.rb":   "class A\n  def total(xs)\n    xs.each do |x|\n      puts x\n    end\n  end\nend\n",
		"a.java": "class A {\n  public static int twice(int x)\n  {\n    return helper(x) * 2;\n  }\n  int helper(int x) { return x; }\n}\n",
	}
	repo := gitRepo(t, files)
	r := &reacher{repo: repo, head: "HEAD", files: map[string][]string{}}
	for _, c := range []struct {
		file     string
		at       int
		name     string
		from, to int
		calls    string
	}{
		{"a.sh", 7, "all", 6, 9, "api"},
		{"a.sh", 4, "api", 2, 5, ""},
		{"a.py", 8, "load", 4, 10, "parse"},
		{"a.js", 7, "sum", 5, 9, ""},
		{"a.js", 2, "add", 1, 3, "sum"},
		{"a.rb", 4, "total", 2, 6, "puts"},
		{"a.java", 4, "twice", 2, 5, "helper"},
	} {
		f, ok := r.enclosing(c.file, c.at)
		if !ok || f.name != c.name || f.from != c.from || f.to != c.to || (c.calls != "" && !strings.Contains(" "+strings.Join(f.calls, " ")+" ", " "+c.calls+" ")) {
			t.Errorf("%s:%d = %+v, %v; want %s %d-%d calling %s", c.file, c.at, f, ok, c.name, c.from, c.to, c.calls)
		}
	}
	if f, ok := r.enclosing("a.sh", 10); ok {
		t.Errorf("a script's top level is no function: %+v", f)
	}
	if got := r.defs([]string{"api", "parse", "helper", "nowhere"}, nil); len(got["api"]) != 1 || len(got["parse"]) != 1 || len(got["helper"]) != 1 || len(got["nowhere"]) != 0 {
		t.Errorf("defs: %+v", got)
	}
}

func TestNamedIn(t *testing.T) {
	names, weak := namedIn("The `comments` operation emits no `bot` field, and Agreement accepts it; see load_all() and run(x). Report says readBlock. Last")
	if got := strings.Join(names, " "); got != "comments bot Agreement load_all run readBlock" {
		t.Errorf("namedIn = %q", got)
	}
	if got := strings.Join(weak, " "); got != "The Report Last" {
		t.Errorf("namedIn, weak = %q", got)
	}
}

// A brace in a string, a character, a comment is not the code's; a quote
// left open on the line, an apostrophe or a lifetime, hides nothing.
func TestStripCode(t *testing.T) {
	for _, c := range []struct {
		line  string
		shell bool
		want  string
	}{
		{`if c == '{' { // }`, false, `if c ==  { `},
		{`s := "a\\" + "}"; x {`, false, `s :=  + ; x {`},
		{"r := `\\` + f() {", false, "r :=  + f() {"},
		{`fn f<'a>(x: &'a str) {`, false, `fn f<a str) {`}, // a lifetime taken for a quote, its brace kept
		{`jq '{a: .b}' "$x" # }`, true, `jq   `},
		{`echo "${#x}" {`, true, `echo  {`},
		{`msg := "it goes on`, false, `msg := `},
	} {
		if got := stripCode(c.line, c.shell); got != c.want {
			t.Errorf("stripCode(%q) = %q, want %q", c.line, got, c.want)
		}
	}
}

// A symptom in the function its cause lies in is not shown again as lines
// around it.
func TestReachSymptomInTheCauseFunction(t *testing.T) {
	repo := gitRepo(t, map[string]string{"calc/calc.go": calcGo, "calc/sum.go": sumGo})
	f := Finding{Cause: Quote{Path: "calc/sum.go", Quote: "total := 0"}, Symptom: &Quote{Path: "calc/sum.go", Quote: "return total"}}
	got := reach(repo, "HEAD", nil, f, 5, 9, 8)
	if strings.Count(got, "### calc/sum.go") != 1 {
		t.Errorf("sum shown once:\n%s", got)
	}
}
