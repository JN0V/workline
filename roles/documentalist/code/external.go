package documentalist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// ExternalLinks checks the docs' links to other sites with lychee
// (https://lychee.cli.rs), which reads its own lychee.toml and .lycheeignore
// in the repository. A link answered by an HTTP error is broken; one that
// could not be reached (no network, a timeout) is only counted, so a run
// offline is not a flood of findings. ok is false when lychee is not
// installed: the links stay unchecked, and the caller says why.
func ExternalLinks(repo string, docs []string) (problems []Problem, ok bool, err error) {
	if _, err := exec.LookPath("lychee"); err != nil {
		return nil, false, nil
	}
	if len(docs) == 0 {
		return nil, true, nil
	}
	sort.Strings(docs)
	args := append([]string{"--no-progress", "--format", "json", "--scheme", "https", "--scheme", "http"}, docs...)
	cmd := exec.Command("lychee", args...)
	cmd.Dir = repo
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		if x, isExit := err.(*exec.ExitError); !isExit || x.ExitCode() != 2 { // 2: some links failed
			return nil, true, fmt.Errorf("lychee failed: %v: %s", err, strings.TrimSpace(errOut.String()))
		}
	}
	type entry struct {
		URL    string `json:"url"`
		Status struct {
			Text string `json:"text"`
			Code int    `json:"code"`
		} `json:"status"`
		Span struct {
			Line int `json:"line"`
		} `json:"span"`
	}
	var report struct {
		Errors   map[string][]entry `json:"error_map"`
		Timeouts map[string][]entry `json:"timeout_map"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		return nil, true, fmt.Errorf("lychee answered no report: %v", err)
	}
	unreached := map[string]int{}
	for doc, entries := range report.Errors {
		for _, e := range entries {
			if e.Status.Code == 0 {
				unreached[doc]++
				continue
			}
			where := fmt.Sprintf("%s:%d", doc, e.Span.Line)
			problems = append(problems, Problem{Rule: "external-link-broken", Where: where, Key: "external-link-broken " + doc + " " + e.URL,
				Message: fmt.Sprintf("line %d links to %s, which answers %d: fix the link, or remove it", e.Span.Line, e.URL, e.Status.Code)})
		}
	}
	for doc, entries := range report.Timeouts {
		unreached[doc] += len(entries)
	}
	for doc, n := range unreached {
		problems = append(problems, Problem{Rule: "links-not-checked", Where: doc, Key: "links-not-checked " + doc, Size: n,
			Message: fmt.Sprintf("%d link(s) to other sites could not be reached (no network, or a timeout); they are checked again on the next run", n)})
	}
	sort.SliceStable(problems, func(i, j int) bool { return problems[i].Where < problems[j].Where })
	return problems, true, nil
}
