package main

import (
	"slices"
	"testing"
)

func TestGateRunCutFromTrunksCommand(t *testing.T) {
	for _, c := range []struct {
		name, linter string
		shims        []string
		cmd          command
		run          string
		onFiles      bool
	}{
		{"shellcheck", "shellcheck", []string{"shellcheck"}, command{Run: "shellcheck ${target} -f json --external-sources"}, "shellcheck", true},
		{"golangci-lint", "golangci-lint2", []string{"golangci-lint"}, command{Run: "golangci-lint run --output.sarif.path ${tmpfile} ${target}", Target: "${parent}"}, "golangci-lint run", false},
		{"checkov", "checkov", []string{"checkov"}, command{Run: "checkov.cmd -f ${target} -o sarif"}, "checkov -f", true},
		{"pyright", "pyright", []string{"pyright"}, command{Run: "pyright --outputjson ${target}"}, "pyright", true},
		{"clippy", "clippy", nil, command{Run: "cargo clippy --message-format json", Target: "${parent_with(Cargo.toml)}"}, "cargo clippy", false},
		{"pinact", "pinact", []string{"pinact"}, command{Run: "python3 ${plugin}/linters/pinact/pinact_run.py ${target}"}, "", false},
	} {
		run, onFiles := gateRun(c.cmd, c.name, c.linter, c.shims)
		if run != c.run || onFiles != c.onFiles {
			t.Errorf("%s: gateRun = %q, %v; want %q, %v", c.name, run, onFiles, c.run, c.onFiles)
		}
	}
}

func TestSeenAsNamesOfThreeLettersOrMore(t *testing.T) {
	if got := seenAs("golangci-lint", "golangci-lint2", []string{"golangci-lint"}, "golangci-lint run"); !slices.Equal(got, []string{"golangci-lint", "golangci-lint2", "golangci-lint run"}) {
		t.Errorf("golangci-lint: %v", got)
	}
	if got := seenAs("ty", "ty", []string{"ty"}, "ty check"); !slices.Equal(got, []string{"ty check"}) {
		t.Errorf("ty is seen only as its command: %v", got)
	}
}
