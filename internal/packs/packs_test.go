package packs

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Every built-in pack reads strictly, the generic one among them.
func TestBuiltinRead(t *testing.T) {
	b := Builtin()
	for _, n := range []string{Generic, "go", "kotlin-gradle", "java-maven", "python-pytest", "js-jest"} {
		p, ok := b[n]
		if !ok {
			t.Fatalf("no built-in pack %s", n)
		}
		if len(p.TestPaths) == 0 {
			t.Errorf("%s: no test-paths", n)
		}
		if n != Generic && len(p.Detect.Files) == 0 {
			t.Errorf("%s: nothing detects it", n)
		}
	}
	if len(b[Generic].Detect.Files) > 0 {
		t.Error("the generic pack is never detected: it is read where none is listed")
	}
}

// A field workline does not read is refused, never ignored; a project's
// file replaces a built-in one of its name whole, or adds one.
func TestProjectPacks(t *testing.T) {
	repo := t.TempDir()
	dir := filepath.Join(repo, ".workline", "packs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.yaml", "test-paths: [\"**/*_check.go\"]\n")
	write("dart.yaml", "detect: {files: [pubspec.yaml], extensions: [.dart]}\ntest-paths: [\"test/**\"]\n")
	all, err := Available(repo)
	if err != nil {
		t.Fatal(err)
	}
	if g := all["go"]; !slices.Equal(g.TestPaths, []string{"**/*_check.go"}) || len(g.Detect.Files) > 0 || g.Source != ".workline/packs/go.yaml" {
		t.Errorf("go replaced whole: %+v", g)
	}
	if d := all["dart"]; !slices.Equal(d.Detect.Files, []string{"pubspec.yaml"}) {
		t.Errorf("dart added: %+v", d)
	}
	write("bad.yaml", "test-path: [\"x/**\"]\n")
	if _, err := Available(repo); err == nil || !strings.Contains(err.Error(), "test-path") {
		t.Errorf("a field nothing reads: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, "bad.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, err := Listed(repo, []string{"go", "cobol"}); err == nil || !strings.Contains(err.Error(), `"cobol"`) {
		t.Errorf("an unknown pack listed: %v", err)
	}
}

// The listed packs' test fields add up, each once; a field none gives is
// left to the generic pack.
func TestTesterDefaults(t *testing.T) {
	b := Builtin()
	d := TesterDefaults([]Pack{b["go"], b["kotlin-gradle"], b["java-maven"]})
	paths, _ := d["test-paths"].([]any)
	if len(paths) != 1+len(b["kotlin-gradle"].TestPaths) || paths[0] != "**/*_test.go" {
		t.Errorf("test-paths: %v", paths)
	}
	if homes, _ := d["test-homes"].([]any); len(homes) != len(b["kotlin-gradle"].TestHomes) {
		t.Errorf("test-homes, src/test → src/main once: %v", homes)
	}
	if d := TesterDefaults([]Pack{b["go"]}); d["test-homes"] != nil {
		t.Errorf("go gives no homes, left to the generic pack: %v", d)
	}
}

// Only the listed packs are read: a broken draft the project does not
// list stops nothing.
func TestListedReadsListedOnly(t *testing.T) {
	repo := t.TempDir()
	dir := filepath.Join(repo, ".workline", "packs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dart.yaml"), []byte("aside: [x]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if l, err := Listed(repo, []string{"go"}); err != nil || len(l) != 1 || l[0].Source != "built-in" {
		t.Errorf("go listed, dart a draft: %v %v", l, err)
	}
	if _, err := Listed(repo, []string{"dart"}); err == nil || !strings.Contains(err.Error(), "dart") {
		t.Errorf("dart listed, broken: %v", err)
	}
}

// Marker files outside test and aside folders call for a pack; the
// languages no listed pack knows are counted.
func TestDetectedAndUnknown(t *testing.T) {
	files := []string{"go.mod", "main.go", "app/build.gradle.kts", "app/src/main/kotlin/A.kt", "tests/fixture/pom.xml", "lib/a.dart", "lib/b.dart"}
	aside := func(f string) bool { return strings.HasPrefix(f, "tests/") }
	got := Detected(files, Builtin(), aside)
	if len(got) != 2 || got["go"] != "go.mod" || got["kotlin-gradle"] != "app/build.gradle.kts" {
		t.Errorf("Detected = %v", got)
	}
	unknown := Unknown(files, []Pack{Builtin()["go"]}, aside)
	var names []string
	for _, c := range unknown {
		names = append(names, c.Name)
	}
	if !slices.Equal(names, []string{"Dart", "Kotlin"}) || unknown[0].Files != 2 {
		t.Errorf("Unknown = %+v", unknown)
	}
	if k := Knowing(unknown[1].Language, Builtin()); !slices.Equal(k, []string{"kotlin-gradle"}) {
		t.Errorf("Knowing(Kotlin) = %v", k)
	}
}

// A pack never decides a verdict: its fields reach a role only as the
// defaults of its settings (internal/role) and the test-file definition
// (internal/testdef), and the doctor and init read them; no package that
// judges imports this one.
func TestNoVerdictReadsAPack(t *testing.T) {
	allowed := []string{"internal/testdef", "internal/role", "internal/doctor", "cmd/workline"}
	root := filepath.Join("..", "..")
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "testdata") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), p, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, filepath.Dir(p))
		rel = filepath.ToSlash(rel)
		for _, im := range f.Imports {
			path, _ := strconv.Unquote(im.Path.Value)
			if path == "github.com/JN0V/workline/internal/packs" && !slices.Contains(allowed, rel) {
				t.Errorf("%s imports internal/packs: a pack's fields reach a role through its settings, never a verdict's code", p)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
