package tester

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadCoverageLCOVByFolder(t *testing.T) {
	lcov := "TN:\nSF:src/a/x.js\nDA:1,1\nDA:2,0\nend_of_record\nSF:src/b/y.js\nDA:1,3\nend_of_record\n"
	pkgs, unit, err := readCoverage(bufio.NewScanner(strings.NewReader(lcov)))
	if err != nil || unit != "lines" {
		t.Fatalf("unit %q, err %v", unit, err)
	}
	if pkgs["src/a"] != (cover{2, 1}) || pkgs["src/b"] != (cover{1, 1}) {
		t.Fatalf("got %+v", pkgs)
	}
}

func TestReadTimesSuiteTimeNotSubtests(t *testing.T) {
	file := filepath.Join(t.TempDir(), "r.xml")
	xml := `<testsuites><testsuite name="p" time="3"><testcase classname="p" name="TestA" time="2"/>` +
		`<testcase classname="p" name="TestA/one" time="2"/><testcase classname="p" name="TestB" time="1"><failure/></testcase></testsuite></testsuites>`
	if err := os.WriteFile(file, []byte(xml), 0o644); err != nil {
		t.Fatal(err)
	}
	out := map[string]timed{}
	total, err := readTimes(file, out)
	if err != nil || total != 3 {
		t.Fatalf("total %v, err %v", total, err)
	}
	if out["p.TestB"].res != failed || out["p.TestA"].seconds != 2 {
		t.Fatalf("got %+v", out)
	}
	parents := parentsOf([]string{"p.TestA", "p.TestA/one/two", "p.TestB"})
	if !parents["p.TestA"] || !parents["p.TestA/one"] || parents["p.TestB"] {
		t.Fatal("subtests not told")
	}
}
