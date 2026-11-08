package simulator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/clock"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// TestBackstopStopsRounds: the review rounds on the developer's pull
// request cross the issue's backstop (ADR-0064, §5; the panel of
// 2026-11-08). #35 goes through the line at 950k fresh tokens, its pull
// request open; then the reviewer asks a fix each tick, and each round
// runs as CI runs it, `workline develop --merge-request`, its agent
// spending 300k, counted on the issue. Once the issue's runs reach its
// backstop, 1.6M, the line starts no round more and blocks #35, the pull
// request kept open; the round's own job refuses the same way.
func TestBackstopStopsRounds(t *testing.T) {
	tr := trace{name: "trial-35-rounds-cross", file: "trial-35", fresh: true,
		tokens: map[string]int{"product-owner": 350000, "tester": 350000, "developer": 250000}}
	w := newWorld(t, tr, &simAgent{})
	var log strings.Builder
	open := func() *forge.FakeItem {
		for _, m := range w.state(t).MergeRequests {
			if strings.HasPrefix(m.Branch, "workline/developer/") && !m.Closed {
				return &m
			}
		}
		return nil
	}
	// The line, no merge between ticks: the pull request waits for review.
	i := 0
	for ; i < 6 && open() == nil; i++ {
		tick = i
		res := w.run(start.Add(time.Duration(i) * period))
		fmt.Fprintf(&log, "tick %d: %s\n", i, res.Summary)
	}
	mr := open()
	if mr == nil {
		t.Fatalf("no developer's pull request open\n%s", log.String())
	}
	stub := filepath.Join(w.agent.work, "round-agent.sh")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\ncat > /dev/null\nprintf 'tokens-in: 300000\\ntokens-out: 0\\n' > \"$WORKLINE_CALL\"\necho '[]'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: w.forge}
	round := func(at time.Time) (string, error) {
		if err := gitIn(w.repo, "fetch", "-q", "origin", mr.Branch); err != nil {
			return "", err
		}
		if err := gitIn(w.repo, "checkout", "-q", "-B", mr.Branch, "FETCH_HEAD"); err != nil {
			return "", err
		}
		defer func() { _ = gitIn(w.repo, "checkout", "-q", "-") }()
		cmd := exec.Command(engine.Self(), "develop", "--merge-request", fmt.Sprint(mr.ID), "--roles", w.roles,
			"--ai", "cmd:sh "+stub, "--forge", "fake:"+w.forge)
		cmd.Dir = w.repo
		cmd.Env = append(os.Environ(), clock.Fixed+"="+at.Format(time.RFC3339))
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	for n := 1; n <= 6; n++ {
		at := start.Add(time.Duration(i+n) * period)
		// The reviewer asks a fix, a new review on the pull request.
		body := fmt.Sprintf("**workline reviewer**\n\n| | | Where | Finding |\n|---|---|---|---|\n| **blocks** | correctness | `lib.sh:2` | Important: review %d still asks a fix. |\n\n<!-- workline:record reviewed=1111111 runs=%d -->", n, n+1)
		if err := f.Sticky(forge.Target{Kind: "merge-request", ID: mr.ID}, body, forge.Marker("sticky=reviewer/summary"), true); err != nil {
			t.Fatal(err)
		}
		tick = i + n
		res := w.run(at)
		fmt.Fprintf(&log, "round tick %d: %s\n", n, res.Summary)
		stopped := slices.ContainsFunc(res.Findings, func(x verdict.Finding) bool {
			return x.Rule == "backstop-crossed" && x.Where == fmt.Sprintf("merge request %d", mr.ID)
		})
		due := slices.ContainsFunc(res.Findings, func(x verdict.Finding) bool {
			return x.Rule == "review-round-due" && x.Where == fmt.Sprintf("merge request %d", mr.ID)
		})
		if stopped {
			is := issueOf(w.state(t), 35)
			if !slices.Contains(is.Labels, "workline:blocked") || !atBackstop(*is) {
				t.Fatalf("round %d stopped, #35 not blocked at its backstop: %s\n%s", n, trail(*is), log.String())
			}
			for _, m := range w.state(t).MergeRequests {
				if m.ID == mr.ID && m.Closed {
					t.Fatalf("the pull request was closed\n%s", log.String())
				}
			}
			// The round's own job, as CI would start it: refused alike.
			out, _ := round(at)
			if !strings.Contains(out, "at its backstop") {
				t.Fatalf("`workline develop --merge-request %d` ran a round past the backstop:\n%s\n%s", mr.ID, out, log.String())
			}
			_, hops, _ := record(*is)
			t.Logf("stopped at round %d, #35 blocked, %d hops\n%s", n, hops, log.String())
			return
		}
		if !due {
			t.Fatalf("round %d: neither due nor stopped\n%s", n, log.String())
		}
		out, err := round(at)
		fmt.Fprintf(&log, "  round %d: %v %.300s\n", n, err, out)
	}
	t.Fatalf("the rounds never reached the backstop\n%s", log.String())
}
