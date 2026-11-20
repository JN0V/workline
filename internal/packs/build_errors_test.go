package packs

import (
	"encoding/json"
	"regexp"
	"testing"
)

// Java build error. Given only `kotlin-gradle` is listed, When a new Java
// test breaks the build with javac's `cannot find symbol`, Then the
// build-errors rows the pack gives the tester read it as a missing name,
// the name taken from javac's `symbol:` line; another javac error stays
// no missing name.
func TestKotlinGradleReadsJavacMissingName(t *testing.T) {
	listed := []Pack{Builtin()["kotlin-gradle"]}
	raw, err := json.Marshal(TesterDefaults(listed)["build-errors"])
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Error       string   `json:"error"`
		MissingName []string `json:"missing-name"`
		NameLine    string   `json:"name-line"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatalf("build-errors: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("kotlin-gradle alone gives the tester no build-errors rows")
	}

	// read applies the rows as the tester does: the first row whose error
	// pattern matches the line, its message a missing name, the name on the
	// symbol line after it.
	read := func(line, next string) (missing bool, name string) {
		for _, r := range rows {
			re, err := regexp.Compile(r.Error)
			if err != nil {
				t.Fatalf("error pattern %q: %v", r.Error, err)
			}
			m := re.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			msg := m[re.SubexpIndex("message")]
			for _, p := range r.MissingName {
				pre, err := regexp.Compile(p)
				if err != nil {
					t.Fatalf("missing-name pattern %q: %v", p, err)
				}
				if !pre.MatchString(msg) {
					continue
				}
				if r.NameLine != "" {
					nre, err := regexp.Compile(r.NameLine)
					if err != nil {
						t.Fatalf("name-line pattern %q: %v", r.NameLine, err)
					}
					if nm := nre.FindStringSubmatch(next); nm != nil && nre.SubexpIndex("name") >= 0 {
						name = nm[nre.SubexpIndex("name")]
					}
				}
				return true, name
			}
			return false, ""
		}
		t.Errorf("no row reads the error line %q", line)
		return false, ""
	}

	const file = "/tmp/workline-test-42/feature/onboarding/src/test/java/probe/ProbeJavaTest.java"
	missing, name := read(file+":9: error: cannot find symbol", "  symbol:   variable Speeds")
	if !missing || name != "Speeds" {
		t.Errorf("javac's cannot find symbol: missing %v, name %q, want a missing name Speeds", missing, name)
	}
	if missing, _ := read(file+":12: error: ';' expected", "  symbol:   method unrelated()"); missing {
		t.Error("javac's `';' expected` is no missing name")
	}
}
