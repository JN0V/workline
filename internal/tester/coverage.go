package tester

// The coverage the test command writes, where it does (coverage-report):
// Go's profile (`-coverprofile`) or LCOV (Jest, c8, coverage.py's
// `lcov`, cargo-llvm-cov), read for its total and its least covered
// packages; and whether a mutation tool exists for the project's
// language (ADR-0055: step 7's proof). Per-test coverage is not
// measured: a run per test (roles/tester/docs/curator.md).

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// coverageLeast is how many of the least covered packages are named.
const coverageLeast = 5

// cover is a package's statements (Go) or lines (LCOV): all, and covered.
type cover struct{ all, hit int }

// coverageLines say the coverage report's total and least covered
// packages; that none is set, or none was written, said too.
func coverageLines(tree, report string) []string {
	if strings.TrimSpace(report) == "" {
		return []string{"- **coverage**: not measured (no `coverage-report`)"}
	}
	f, err := os.Open(filepath.Join(tree, filepath.FromSlash(report)))
	if err != nil {
		return []string{fmt.Sprintf("- **coverage**: the test command wrote no %s", report)}
	}
	defer func() { _ = f.Close() }()
	pkgs, unit, err := readCoverage(bufio.NewScanner(f))
	if err != nil || len(pkgs) == 0 {
		return []string{fmt.Sprintf("- **coverage**: %s is neither Go's profile nor LCOV", report)}
	}
	var total cover
	var names []string
	for p, c := range pkgs {
		total.all += c.all
		total.hit += c.hit
		names = append(names, p)
	}
	pct := func(c cover) float64 { return 100 * float64(c.hit) / float64(max(c.all, 1)) }
	sort.Slice(names, func(i, j int) bool {
		if a, b := pct(pkgs[names[i]]), pct(pkgs[names[j]]); a != b {
			return a < b
		}
		return names[i] < names[j]
	})
	lines := []string{fmt.Sprintf("- **coverage**: %.1f%% of %d %s, %d package%s; the least covered:", pct(total), total.all, unit, len(pkgs), plural(len(pkgs)))}
	for _, p := range names[:min(coverageLeast, len(names))] {
		lines = append(lines, fmt.Sprintf("  - %s: %.1f%% of %d", p, pct(pkgs[p]), pkgs[p].all))
	}
	return lines
}

// readCoverage reads Go's profile or LCOV, by package: the folder of each
// file. A Go block counted in more than one package's run is covered when
// one of them covers it.
func readCoverage(sc *bufio.Scanner) (map[string]cover, string, error) {
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	pkgs := map[string]cover{}
	if !sc.Scan() {
		return nil, "", sc.Err()
	}
	if strings.HasPrefix(sc.Text(), "mode:") {
		blocks := map[string][2]int{} // file:range -> statements, hit
		for sc.Scan() {
			l := sc.Text()
			i := strings.LastIndexByte(l, ':')
			if i < 0 {
				continue
			}
			fields := strings.Fields(l[i+1:])
			if len(fields) != 3 {
				continue
			}
			stmts, _ := strconv.Atoi(fields[1])
			count, _ := strconv.Atoi(fields[2])
			key := l[:i] + ":" + fields[0]
			b := blocks[key]
			b[0] = stmts
			if count > 0 {
				b[1] = 1
			}
			blocks[key] = b
		}
		for key, b := range blocks {
			file := key[:strings.LastIndexByte(key, ':')]
			c := pkgs[path.Dir(file)]
			c.all += b[0]
			c.hit += b[0] * b[1]
			pkgs[path.Dir(file)] = c
		}
		return pkgs, "statements", sc.Err()
	}
	pkg := ""
	for ok := true; ok; ok = sc.Scan() {
		l := sc.Text()
		switch {
		case strings.HasPrefix(l, "SF:"):
			pkg = path.Dir(filepath.ToSlash(strings.TrimPrefix(l, "SF:")))
		case strings.HasPrefix(l, "DA:") && pkg != "":
			parts := strings.Split(strings.TrimPrefix(l, "DA:"), ",")
			if len(parts) < 2 {
				continue
			}
			c := pkgs[pkg]
			c.all++
			if n, _ := strconv.Atoi(parts[1]); n > 0 {
				c.hit++
			}
			pkgs[pkg] = c
		}
	}
	return pkgs, "lines", sc.Err()
}

// mutators is, per ecosystem, the file that tells it, its mutation tools,
// and the commands and config files that say one is set up here.
var mutators = []struct {
	lang, marker, tools string
	commands, configs   []string
}{
	{"Go", "go.mod", "Gremlins, go-mutesting", []string{"gremlins", "go-mutesting"}, []string{".gremlins.yaml"}},
	{"Rust", "Cargo.toml", "cargo-mutants", []string{"cargo-mutants"}, []string{".cargo/mutants.toml"}},
	{"Python", "pyproject.toml", "mutmut", []string{"mutmut"}, []string{"setup.cfg"}},
	{"Python", "setup.py", "mutmut", []string{"mutmut"}, []string{"setup.cfg"}},
	{"JavaScript/TypeScript", "package.json", "Stryker", nil, []string{"stryker.conf.js", "stryker.conf.json", "stryker.config.mjs", "stryker.config.json", "stryker.config.js"}},
	{"Java", "pom.xml", "PIT", nil, nil},
	{"Java", "build.gradle", "PIT", nil, nil},
}

// mutationLine says whether a mutation tool exists for the project's
// language, and whether one is set up here.
func mutationLine(tree string) string {
	for _, m := range mutators {
		if _, err := os.Stat(filepath.Join(tree, m.marker)); err != nil {
			continue
		}
		set := ""
		for _, c := range m.commands {
			if _, err := exec.LookPath(c); err == nil {
				set = "`" + c + "` on the PATH"
			}
		}
		for _, c := range m.configs {
			data, err := os.ReadFile(filepath.Join(tree, c))
			if err == nil && (c != "setup.cfg" || strings.Contains(string(data), "mutmut")) {
				set = "`" + c + "`"
			}
		}
		if set == "" && m.lang == "Java" {
			data, _ := os.ReadFile(filepath.Join(tree, m.marker))
			if strings.Contains(string(data), "pitest") {
				set = "pitest in `" + m.marker + "`"
			}
		}
		if set == "" && m.marker == "pyproject.toml" {
			data, _ := os.ReadFile(filepath.Join(tree, m.marker))
			if strings.Contains(string(data), "[tool.mutmut]") {
				set = "`[tool.mutmut]` in `pyproject.toml`"
			}
		}
		if set == "" {
			set = "none set up here"
		} else {
			set = "set up here: " + set
		}
		return fmt.Sprintf("- **mutation**: %s has %s; %s (step 7's proof)", m.lang, m.tools, set)
	}
	return "- **mutation**: no tool known for this project's language"
}
