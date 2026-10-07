package agent

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// finding reads the first proposal's finding of an answer as proposalsFrom
// leaves it, and what was mended.
func finding(t *testing.T, answer string) (map[string]any, []string) {
	t.Helper()
	got, mended, err := proposalsFrom(answer)
	if err != nil {
		t.Fatalf("did not read: %v\n%s", err, answer)
	}
	var list []map[string]map[string]any
	if err := yaml.Unmarshal(got, &list); err != nil {
		t.Fatal(err)
	}
	return list[0]["finding"], mended
}

// Go code quoted with its tabs, as lenses wrote it on PR #157: the reader
// refuses the whole answer; mended, the code reads exactly as in the file.
func TestMendTabs(t *testing.T) {
	code := "if v := value(m, key); v != nil {\n\t\treturn v\n\t}\n"
	for _, c := range []struct{ name, quote, want string }{
		// The first line starting with a tab: the reader stops there to
		// find the block's indentation.
		{"first line a tab", "      quote: |\n        \tif v := value(m, key); v != nil {\n        \t\treturn v\n        \t}\n", "\t" + code},
		// The lines after the first pasted as they are in the file, at
		// the start of the line.
		{"lines not indented", "      quote: |\n        if v := value(m, key); v != nil {\n\t\treturn v\n\t}\n", code},
		// The whole block pasted at the start of the line.
		{"block not indented", "      quote: |\n\tif v := value(m, key); v != nil {\n\t\treturn v\n\t}\n", "\t" + code},
	} {
		answer := "- finding:\n    severity: important\n    title: A node not a mapping is returned\n    cause:\n      path: internal/routing/adopt.go\n" +
			c.quote + "    fix: Replace it with a mapping node.\n"
		var probe any
		if err := yaml.Unmarshal([]byte(answer), &probe); err == nil || !strings.Contains(err.Error(), "tab character") {
			t.Fatalf("%s: the answer is not the one that broke: %v", c.name, err)
		}
		f, mended := finding(t, answer)
		if q := f["cause"].(map[string]any)["quote"]; q != c.want {
			t.Errorf("%s: the quote moved: %q, want %q", c.name, q, c.want)
		}
		if f["fix"] != "Replace it with a mapping node." {
			t.Errorf("%s: what follows the block moved: %q", c.name, f["fix"])
		}
		if len(mended) != 1 || !strings.Contains(mended[0], "tab") {
			t.Errorf("%s: not said: %q", c.name, mended)
		}
	}
	// A block already read is never touched: its tabs after the
	// indentation are text.
	ok := "- finding:\n    quote: |\n      func f() {\n      \treturn x\n      }\n"
	if got, mended := Mend(ok); got != ok || mended != nil {
		t.Errorf("an answer that reads was mended: %q %q", got, mended)
	}
	// Tabs indenting the YAML itself are not guessed at: the agent is
	// asked again with what the reader said.
	if _, _, err := proposalsFrom("- finding:\n\tseverity: nit\n"); err == nil {
		t.Error("an answer indented with tabs read")
	}
}

// Issue #156: a reason written plain holding ` #20` lost all after it.
func TestMendHash(t *testing.T) {
	answer := "- finding:\n" +
		"    severity: nit # minor\n" +
		"    title: #20 is already covered\n" +
		"    why: fixes #20 and #21\n" +
		"    fix: The lifetime half is already covered by #20.   \n" +
		"    cause:\n" +
		"      path: calc/calc.go # the file\n" +
		"      quote: |\n" +
		"        x := 1 # not a comment: code\n" +
		"      lines: [\"a # b\"]\n"
	f, mended := finding(t, answer)
	for k, want := range map[string]string{
		"severity": "nit",
		"title":    "#20 is already covered",
		"why":      "fixes #20 and #21",
		"fix":      "The lifetime half is already covered by #20.",
	} {
		if f[k] != want {
			t.Errorf("%s: %q, want %q", k, f[k], want)
		}
	}
	cause := f["cause"].(map[string]any)
	if cause["path"] != "calc/calc.go" || cause["quote"] != "x := 1 # not a comment: code\n" {
		t.Errorf("the cause moved: %q", cause)
	}
	if len(mended) != 1 || !strings.Contains(mended[0], "3 plain") {
		t.Errorf("not said: %q", mended)
	}
	// A quote written plain holding ` # ` is read as the agent wrote it,
	// to be found again in the code.
	f, _ = finding(t, "- finding:\n    cause:\n      quote: total += v # the sum so far\n")
	if q := f["cause"].(map[string]any)["quote"]; q != "total += v # the sum so far" {
		t.Errorf("a plain quote holding ` # `: %q", q)
	}
	// In a list, too.
	got, _, err := proposalsFrom("- refine:\n    sections:\n      - Need: see #12 for the rest\n      - one two #3\n")
	if err != nil || !strings.Contains(string(got), `"see #12 for the rest"`) || !strings.Contains(string(got), `"one two #3"`) {
		t.Errorf("in a list: %q %v", got, err)
	}
}

// A title opening on a quoted phrase and going on after it, as a spec lens
// wrote it on workline-sandbox#43 (#128): the reader refuses the whole
// answer; mended, the title reads as written, its quotes kept.
func TestMendQuotedThenMore(t *testing.T) {
	answer := "- finding:\n    lens: ambiguous\n    severity: nit\n    title: \"Idle for 60 minutes\" boundary is unclear\n    why: 'it' can be read two ways\n    cause:\n      path: \"#43\"\n      quote: \"so a session should last 60 minutes idle\"\n"
	f, mended := finding(t, answer)
	if f["title"] != `"Idle for 60 minutes" boundary is unclear` || f["why"] != `'it' can be read two ways` || len(mended) != 1 {
		t.Errorf("title %q, why %q, mended %v", f["title"], f["why"], mended)
	}
	// One quoted text whose inner quotes were left unescaped is asked
	// again, not mended: its outer quotes are not the agent's words.
	broken := "- note: \"it returns `\"one hour\"` when unset.\"\n"
	if got, said := Mend(broken); got != broken || said != nil {
		t.Errorf("a quoted text with inner quotes was mended: %q, %v", got, said)
	}
	// One that reads is left as it is: a quoted value and a comment.
	if got, said := Mend("- note: \"a\" # said\n"); got != "- note: \"a\" # said\n" || said != nil {
		t.Errorf("a quoted value with a comment was mended: %q, %v", got, said)
	}
}
