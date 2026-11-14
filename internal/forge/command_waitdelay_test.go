package forge

import "testing"

// A command that answers, exits 0 and leaves a child on its output is
// reachable: the child is not its failure.
func TestCheckCommandChildLeftBehindIsReachable(t *testing.T) {
	state, detail := CheckCommand(t.TempDir(), `echo '{}'; sleep 5 &`)
	if state != ProbeReachable {
		t.Fatalf("state = %s (%s), want %s", state, detail, ProbeReachable)
	}
}

func TestCallChildLeftBehindKeepsAnswer(t *testing.T) {
	c := &command{repo: t.TempDir(), script: `echo '{"id": 7}'; sleep 5 &`}
	var a idAnswer
	if err := c.call("open-issue", nil, &a); err != nil || a.ID != 7 {
		t.Fatalf("id = %d, err = %v; want 7, nil", a.ID, err)
	}
}
