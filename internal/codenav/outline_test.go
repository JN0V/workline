package codenav

import (
	"strings"
	"testing"
)

// outline renders a source's outline, no line shown whole.
func outline(t *testing.T, file, src string) string {
	t.Helper()
	lines := strings.Split(strings.TrimRight(src, "\n"), "\n")
	o := Outline(file, lines)
	if o == nil {
		t.Fatalf("%s: no outline", file)
	}
	return strings.Join(o.Render(lines, func(int) bool { return false }), "\n")
}

// One rule for every language of a family: a container's members kept,
// every other block cut to its opening line and `…`, comments out.
func TestOutline(t *testing.T) {
	for _, c := range []struct {
		file, src string
		want      []string
		not       []string
	}{
		{"a.h", `#pragma once
#include "b.h"
/** A client. */
namespace net {
class Client : public Base {
public:
    // connects
    virtual bool connect(const char* id,
                         int port) = 0;
    bool up() const { return up_; }
    void reset() {
        up_ = false;
    }
private:
    bool up_ = false;
};
template <class T> T twice(T x) {
    return x + x;
}
} // namespace net
`, []string{"#include \"b.h\"", "class Client : public Base {", "    virtual bool connect(const char* id,\n                         int port) = 0;",
			"    bool up() const { return up_; }", "    void reset() { … }", "template <class T> T twice(T x) { … }", "} // namespace net"},
			[]string{"        up_ = false;", "return x + x", "A client", "connects"}},
		{"a.go", `package shop

import (
	"fmt"
)

// Cart holds lines.
type Cart struct {
	Lines []int
}

var names = map[string]int{
	"a": 1,
}

func (c Cart) Total() int {
	return len(c.Lines)
}
`, []string{"import (\n\t\"fmt\"\n)", "type Cart struct {\n\tLines []int\n}", "var names = map[string]int{ … }", "func (c Cart) Total() int { … }"},
			[]string{"return len", "Cart holds"}},
		{"a.py", `"""Doc."""
import os


class Cart:
    """Lines."""

    @property
    def total(self,
              x: int) -> int:
        # sum
        return 1


def main():
    print("body")
`, []string{"class Cart:", "    @property", "    def total(self,\n              x: int) -> int: …", "def main(): …"},
			[]string{"return 1", "body", "# sum"}},
		{"a.ts", `import { x } from './x';
/** An amount. */
export interface Amount {
  cents: number;
}
export type Pair = { a: number };
export class Cart {
  constructor(public lines: Amount[]) {}
  total(): number {
    return 0;
  }
}
export const add = (a: number) => {
  return a;
};
`, []string{"export interface Amount {\n  cents: number;\n}", "export type Pair = { a: number };", "export class Cart {",
			"  total(): number { … }", "export const add = (a: number) => { … };"},
			[]string{"return 0", "return a;", "An amount"}},
	} {
		got := outline(t, c.file, c.src)
		for _, w := range c.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: want %q in:\n%s", c.file, w, got)
			}
		}
		for _, w := range c.not {
			if strings.Contains(got, w) {
				t.Errorf("%s: %q not wanted in:\n%s", c.file, w, got)
			}
		}
	}
}

// A line shown whole stays whole, its body too; the other bodies are cut.
func TestOutlineFull(t *testing.T) {
	src := "void a() {\n    one();\n}\nvoid b() {\n    two();\n}\n"
	lines := strings.Split(strings.TrimRight(src, "\n"), "\n")
	got := strings.Join(Outline("a.c", lines).Render(lines, func(n int) bool { return n == 5 }), "\n")
	if want := "void a() { … }\nvoid b() {\n    two();\n}"; got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// A container word inside a name or before `=` opens no container: an
// initializer is a body (the reviewer on #1108).
func TestOutlineInitializers(t *testing.T) {
	got := outline(t, "a.js", "module.exports = {\n  secret: 1,\n};\nvar record = {\n  value: 2,\n};\n")
	if strings.Contains(got, "secret") || strings.Contains(got, "value") {
		t.Errorf("an initializer's members shown:\n%s", got)
	}
}

// A body whose opening line is shown whole is shown whole, never cut
// silently (the reviewer on #1108).
func TestOutlineFullOpening(t *testing.T) {
	src := "void a() {\n    one();\n    two();\n}\n"
	lines := strings.Split(strings.TrimRight(src, "\n"), "\n")
	got := strings.Join(Outline("a.c", lines).Render(lines, func(n int) bool { return n <= 2 }), "\n")
	if got != strings.TrimRight(src, "\n") {
		t.Errorf("got:\n%s", got)
	}
}

// Each language's row of the table decides: a container word before the
// parameters, what a header holds that says nothing, where a statement
// ends, which indented blocks are bodies (the review of #1108).
func TestOutlineByLanguage(t *testing.T) {
	for _, c := range []struct {
		file, src string
		want, not []string
	}{
		{"a.h", "struct __attribute__((packed)) S {\n    int a;\n};\nstruct alignas(8) T {\n    int b;\n};\ntemplate <class T> T twice(T x) {\n    return x + x;\n}\n",
			[]string{"    int a;", "    int b;", "template <class T> T twice(T x) { … }"}, []string{"return x + x"}},
		{"a.kt", "class P(val x: Int) : B() {\n    fun f(): Int {\n        return x\n    }\n}\n",
			[]string{"class P(val x: Int) : B() {\n    fun f(): Int { … }\n}"}, []string{"return x"}},
		{"A.java", "@Entity(name = \"a\")\npublic record P(int x) {\n    int twice() {\n        return 2 * x;\n    }\n}\n",
			[]string{"public record P(int x) {\n    int twice() { … }\n}"}, []string{"2 * x"}},
		{"a.go", "package a\n\ntype X = int\nvar tbl = []string{\n\t\"secret\",\n}\n",
			[]string{"type X = int", "var tbl = []string{ … }"}, []string{"secret"}},
		{"a.py", "\"\"\"Tool.\n\nUsage:\n    tool run\n\"\"\"\nfrom typing import TYPE_CHECKING\n\nif TYPE_CHECKING:\n    from shop.money import Amount\n\n\ndef f():\n    \"\"\"Doc.\n\nExample:\n    x\n\"\"\"\n    return 1\n",
			[]string{"Usage:\n    tool run", "if TYPE_CHECKING:\n    from shop.money import Amount", "def f(): …"}, []string{"return 1", "Example"}},
		{"make.c", "static struct foo *make(int x) {\n    return secret(x);\n}\nenum color pick(void) {\n    return hidden;\n}\n",
			[]string{"static struct foo *make(int x) { … }", "enum color pick(void) { … }"}, []string{"secret", "hidden"}},
		{"q.py", "def quote():\n    return '\"\"\"'\n\n\nif TYPE_CHECKING:\n    from shop.money import Amount\n\n\ndef f():\n    return 1\n",
			[]string{"def quote(): …", "if TYPE_CHECKING:\n    from shop.money import Amount", "def f(): …"}, []string{"return 1"}},
		{"lib.rs", "#[derive(Debug)]\nstruct S {\n    a: i32,\n}\nimpl Show for S {\n    fn show(&self) {\n        println!(\"x\");\n    }\n}\n",
			[]string{"struct S {\n    a: i32,\n}", "impl Show for S {\n    fn show(&self) { … }\n}"}, []string{"println"}},
	} {
		got := outline(t, c.file, c.src)
		for _, w := range c.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: want %q in:\n%s", c.file, w, got)
			}
		}
		for _, w := range c.not {
			if strings.Contains(got, w) {
				t.Errorf("%s: %q not wanted in:\n%s", c.file, w, got)
			}
		}
	}
}

// Braces that do not balance — a regular expression's, the branches of
// an `#if` — give no outline: the file whole or cut, never the rest of
// it hidden (the review of #1108).
func TestOutlineUnbalanced(t *testing.T) {
	for file, src := range map[string]string{
		"a.ts": "const open = /\\{/;\nexport function f() {\n  return 1;\n}\n",
		"a.c":  "#if A\nvoid f() {\n#else\nvoid f(int x) {\n#endif\n    g();\n}\nvoid h() {\n    k();\n}\n",
		"b.c":  "}\nvoid h() {\n}\n",
	} {
		if Outline(file, strings.Split(src, "\n")) != nil {
			t.Errorf("%s: an outline of unbalanced braces", file)
		}
	}
}

// The types a type derives from or embeds, by its language's pattern.
func TestBasesIn(t *testing.T) {
	for file, c := range map[string]struct{ line, want string }{
		"a.cs":    {"public class Repo : Base<int>, IDisposable {", "Base"},
		"a.kt":    {"class Repo(val x: Int) : Base() {", "Base"},
		"a.rs":    {"impl Display for Repo {", "Display"},
		"a.swift": {"class Repo: Base {", "Base"},
		"a.go":    {"\tBase", "Base"},
		"a.cpp":   {"class Repo : public Base {", "Base"},
		"a.java":  {"public class Repo extends Base implements Closeable {", "Base"},
		"a.py":    {"class Repo(models.Base):", "Base"},
	} {
		if got := Of(file).BasesIn(c.line); len(got) == 0 || got[0] != c.want {
			t.Errorf("%s %q: bases %v, want %s first", file, c.line, got, c.want)
		}
	}
}

// A language whose blocks end on a word has no outline: never a body
// shown for a declaration.
func TestOutlineUnknown(t *testing.T) {
	for _, f := range []string{"a.rb", "a.sh", "a.txt"} {
		if Outline(f, []string{"x"}) != nil {
			t.Errorf("%s: an outline", f)
		}
	}
}
