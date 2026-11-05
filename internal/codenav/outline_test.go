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

// A language whose blocks end on a word has no outline: never a body
// shown for a declaration.
func TestOutlineUnknown(t *testing.T) {
	for _, f := range []string{"a.rb", "a.sh", "a.txt"} {
		if Outline(f, []string{"x"}) != nil {
			t.Errorf("%s: an outline", f)
		}
	}
}
