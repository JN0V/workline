package documentalist

import (
	"strings"
	"testing"
)

const goSrc = `package auth

// TokenTTL is how long a token lasts.
const TokenTTL = 3600

const (
	// A is first.
	A = 1
	B = 2
)

// Store keeps tokens.
type Store[T any] struct{ n int }

// Get finds a token.
func (s *Store[T]) Get() int { return s.n }

// Get is a function of the same name.
func Get() int { return 0 }
`

func TestGoPart(t *testing.T) {
	for anchor, want := range map[string]string{
		"TokenTTL":  "// TokenTTL is how long a token lasts.\nconst TokenTTL = 3600",
		"A":         "// A is first.\n\tA = 1",
		"B":         "B = 2",
		"Store":     "// Store keeps tokens.\ntype Store[T any] struct{ n int }",
		"Store.Get": "// Get finds a token.\nfunc (s *Store[T]) Get() int { return s.n }",
		"Get":       "// Get is a function of the same name.\nfunc Get() int { return 0 }",
	} {
		got, found, err := partOf("auth.go", goSrc, anchor)
		if err != nil || !found || got != want {
			t.Errorf("%s: got %q, %v, %v; want %q", anchor, got, found, err, want)
		}
	}
	if _, found, err := partOf("auth.go", goSrc, "Gone"); found || err != nil {
		t.Errorf("Gone: found %v, err %v", found, err)
	}
	if _, _, err := partOf("auth.go", "package", "A"); err == nil {
		t.Error("a file that does not parse: no error")
	}
}

const yamlSrc = `on: push
.hidden.job: &h
  image: go
jobs:
  test:
    runs-on: ubuntu
    steps:
      - run: go test

  # lint checks the style.
  lint:
    script: |
      golangci-lint run

      echo done
# the end
`

func TestYAMLPart(t *testing.T) {
	for anchor, want := range map[string]string{
		"on":                "on: push",
		".hidden.job":       ".hidden.job: &h\n  image: go",
		".hidden.job.image": "  image: go",
		"jobs.test":         "  test:\n    runs-on: ubuntu\n    steps:\n      - run: go test",
		"jobs.test.runs-on": "    runs-on: ubuntu",
		"jobs.lint":         "  # lint checks the style.\n  lint:\n    script: |\n      golangci-lint run\n\n      echo done\n# the end",
	} {
		got, found, err := partOf("ci.yml", yamlSrc, anchor)
		if err != nil || !found || got != want {
			t.Errorf("%s: got %q, %v, %v; want %q", anchor, got, found, err, want)
		}
	}
	for _, anchor := range []string{"jobs.build", "on.push", "nothing"} {
		if _, found, _ := partOf("ci.yml", yamlSrc, anchor); found {
			t.Errorf("%s: found", anchor)
		}
	}
}

func TestPartOfOtherFiles(t *testing.T) {
	if got, found, _ := partOf("x.sh", "echo", ""); !found || got != "echo" {
		t.Errorf("whole file: %q %v", got, found)
	}
	if _, _, err := partOf("x.sh", "echo", "main"); err == nil || !strings.Contains(err.Error(), "only in Markdown") {
		t.Errorf("a part of a shell script: %v", err)
	}
}

func TestRemovedLines(t *testing.T) {
	got := removedLines("a\nb\nb\nc", "b\nc\nd")
	if strings.Join(got, ",") != "a,b" {
		t.Errorf("got %q", got)
	}
}
