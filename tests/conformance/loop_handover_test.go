package conformance

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The loop's template, read for the handover tests: the project's own
// .github/workflows copy is CI the developer may not change.
const loopTemplate = "../../defaults/ci/github/workline-loop.yml"

// loopHandoverStart is the fake clock's first second, in a job.
const loopHandoverStart = int64(1_000_000)

// runLoopJob runs the beat's script as one whole loop job on a fake
// clock (a `date +%s` and a `sleep` that only move it), with a fresh
// RUNNER_TEMP, as GitHub gives each job, a tick that lists the merge job
// on every beat, and dispatching on. It returns the clock seconds at which
// the job's beats sent a `gh workflow run`, in order.
func runLoopJob(t *testing.T, script string, start int64, minutes, beatSeconds string) []int64 {
	t.Helper()
	for _, tool := range []string{"jq", "bash", "date"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " is not installed")
		}
	}
	realDate, _ := exec.LookPath("date")
	dir := t.TempDir()
	bin, tmp := filepath.Join(dir, "bin"), filepath.Join(dir, "tmp")
	for _, d := range []string{bin, tmp} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	clock, calls := filepath.Join(dir, "clock"), filepath.Join(dir, "gh-calls")
	if err := os.WriteFile(clock, []byte(strconv.FormatInt(start, 10)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeExec(t, filepath.Join(bin, "workline"), "#!/bin/bash\necho '[{\"job\":\"merge\",\"why\":\"due\",\"key\":\"workline merge\"}]'\n")
	writeExec(t, filepath.Join(bin, "gh"), "#!/bin/bash\necho \"$(cat \"$CLOCK\") $*\" >> \"$CALLS\"\n")
	writeExec(t, filepath.Join(bin, "date"), "#!/bin/bash\nif [ \"$1\" = +%s ]; then cat \"$CLOCK\"; else exec "+realDate+" \"$@\"; fi\n")
	writeExec(t, filepath.Join(bin, "sleep"), "#!/bin/bash\necho $(( $(cat \"$CLOCK\") + $1 )) > \"$CLOCK\"\n")
	runBeat(t, script, dir, bin, tmp, "DISPATCH=true", "HOLD_SECONDS=240",
		"LOOP_MINUTES="+minutes, "BEAT_SECONDS="+beatSeconds, "CLOCK="+clock, "CALLS="+calls)
	var at []int64
	b, _ := os.ReadFile(calls)
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if line == "" {
			continue
		}
		n, err := strconv.ParseInt(strings.Fields(line)[0], 10, 64)
		if err != nil {
			t.Fatalf("a gh call logged without its time: %q", line)
		}
		at = append(at, n)
	}
	return at
}

// Handover hold
// Given a job dispatched in the last beat of a loop whose run is not
// listed yet (the merge job, sent at second 0 of the loop before),
// When the next loop's first beat runs, in a new job with nothing kept
// of the one before, 5 s later, tick listing the job again,
// Then that job is not dispatched again: no `gh workflow run` is sent.
func TestLoopHoldsADispatchAtTheHandover(t *testing.T) {
	beat := loopStepNamed(t, readLoop(t, loopTemplate), loopTemplate, "The beat")
	last := runLoopJob(t, beat.Run, loopHandoverStart, "0", "0")
	if len(last) != 1 {
		t.Fatalf("%s: the last loop's beat sent %d dispatches; want 1", loopTemplate, len(last))
	}
	first := runLoopJob(t, beat.Run, loopHandoverStart+5, "0", "0")
	if len(first) != 0 {
		t.Errorf("%s: the next loop's first beat dispatched the job again, at seconds %v: a handover loses the hold", loopTemplate, first)
	}
}

// Hold ends
// Given the hold time has passed (a loop that took over 5 s after a
// dispatch, its beats 180 s apart for 10 minutes, a hold of 240 s),
// When a beat finds the job due,
// Then it is dispatched: the hold does not last for ever, and the first
// beat after the handover is not the one that sends it.
func TestLoopDispatchesAgainWhenTheHandoverHoldEnds(t *testing.T) {
	beat := loopStepNamed(t, readLoop(t, loopTemplate), loopTemplate, "The beat")
	if last := runLoopJob(t, beat.Run, loopHandoverStart, "0", "0"); len(last) != 1 {
		t.Fatalf("%s: the last loop's beat sent %d dispatches; want 1", loopTemplate, len(last))
	}
	startNext := loopHandoverStart + 5
	next := runLoopJob(t, beat.Run, startNext, "10", "180")
	if len(next) == 0 {
		t.Fatalf("%s: the next loop never dispatches the due job, though its hold has ended", loopTemplate)
	}
	if next[0] <= startNext {
		t.Errorf("%s: the next loop dispatched the job in its first beat (second %d, it took over at %d): the hold is lost", loopTemplate, next[0], startNext)
	}
}
