package conformance

import (
	"os/exec"
	"strings"
	"testing"
)

// Every command `workline --help` lists answers `workline <command> --help`
// with its own usage on stdout, and exits 0: a person or a script holding
// only the binary finds each command's arguments and options.
func TestEveryCommandHasHelp(t *testing.T) {
	out, err := exec.Command(engineBin, "--help").Output()
	if err != nil {
		t.Fatalf("workline --help: %v", err)
	}
	_, list, ok := strings.Cut(string(out), "\nCommands:\n")
	if !ok {
		t.Fatalf("workline --help lists no commands:\n%s", out)
	}
	list, _, _ = strings.Cut(list, "\n\n")
	n := 0
	for _, l := range strings.Split(list, "\n") {
		name := strings.Fields(l)[0]
		n++
		cmd := exec.Command(engineBin, name, "--help")
		cmd.Env = hermeticEnv()
		got, err := cmd.Output()
		if err != nil || !strings.HasPrefix(string(got), "usage: workline ") {
			t.Errorf("workline %s --help: %v\n%s", name, err, got)
		}
	}
	if n < 10 {
		t.Errorf("workline --help lists %d commands", n)
	}
}
