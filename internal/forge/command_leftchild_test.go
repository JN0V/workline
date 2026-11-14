package forge

import "testing"

// A command that answers and exits 0 but leaves a child on its output is
// reachable, not failing: its answer is whole.
func TestCheckCommandLeftChildIsReachable(t *testing.T) {
	state, detail := CheckCommand(t.TempDir(), `cat >/dev/null; echo '{}'; sleep 5 & exit 0`)
	if state != ProbeReachable {
		t.Fatalf("state = %s (%s), want reachable", state, detail)
	}
}

func TestCallLeftChildKeepsAnswer(t *testing.T) {
	c := &command{repo: t.TempDir(), script: `cat >/dev/null; echo '{"id": 7}'; sleep 5 & exit 0`}
	var a idAnswer
	if err := c.call("open-issue", nil, &a); err != nil || a.ID != 7 {
		t.Fatalf("id = %d, err = %v", a.ID, err)
	}
}
