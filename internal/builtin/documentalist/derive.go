package documentalist

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/JN0V/workline/internal/verdict"
)

// derived matches a block of facts regenerated from the code: inline
// (`<!-- workline:derive name -->3<!-- workline:end -->`) or on lines of its own.
var derived = regexp.MustCompile(`(?s)<!-- workline:derive (\S+) -->(.*?)<!-- workline:end -->`)

// Derive regenerates the derived blocks of the docs from the commands the
// project declares by name (`settings.derive`). Commands live in the config,
// which only people change, never in a doc: a doc names one. Markers shown in
// code — a fenced block or a code span — are examples, not blocks. It returns
// what it found, and the new content of each doc whose blocks were stale.
func Derive(repo string, docs map[string]string, commands map[string]string) ([]verdict.Finding, map[string]string) {
	d := deriver{repo: repo, commands: commands, outputs: map[string]string{}, failed: map[string]error{}}
	fixed := map[string]string{}
	paths := make([]string, 0, len(docs))
	for p := range docs {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		content := docs[p]
		code := codeRanges(content)
		var now strings.Builder
		last, stale := 0, 0
		for _, loc := range derived.FindAllStringSubmatchIndex(content, -1) {
			if inRanges(code, loc[0]) {
				continue
			}
			block := d.block(p, content[loc[2]:loc[3]], content[loc[4]:loc[5]])
			if block == "" {
				continue // could not be regenerated; left as it is, and reported
			}
			if block != content[loc[0]:loc[1]] {
				stale++
			}
			now.WriteString(content[last:loc[0]] + block)
			last = loc[1]
		}
		now.WriteString(content[last:])
		if stale > 0 {
			fixed[p] = now.String()
			d.findings = append(d.findings, verdict.Finding{Rule: "derived-stale", Where: p,
				Message: fmt.Sprintf("%d derived block(s) no longer match what their command gives; regenerated", stale)})
		}
	}
	return d.findings, fixed
}

type deriver struct {
	repo     string
	commands map[string]string
	outputs  map[string]string // each command runs once per run
	failed   map[string]error
	findings []verdict.Finding
}

// block returns a derived block as it should read, or "" when it cannot be
// regenerated.
func (d *deriver) block(doc, name, inner string) string {
	cmd, ok := d.commands[name]
	if !ok {
		d.findings = append(d.findings, verdict.Finding{Rule: "derive-unknown", Where: doc, Level: "block",
			Message: fmt.Sprintf("%q is not declared in the documentalist's `derive` settings, so this fact cannot be regenerated", name)})
		return ""
	}
	out, seen := d.outputs[name]
	if !seen && d.failed[name] == nil {
		var err error
		if out, err = run(d.repo, cmd); err != nil {
			d.failed[name] = err
		} else {
			d.outputs[name] = out
		}
	}
	if err := d.failed[name]; err != nil {
		d.findings = append(d.findings, verdict.Finding{Rule: "derive-failed", Where: doc, Level: "block",
			Message: fmt.Sprintf("%q (%s) failed: %v", name, cmd, err)})
		return ""
	}
	if strings.HasPrefix(inner, "\n") {
		out = "\n" + out + "\n" // a block on lines of its own stays so
	}
	return "<!-- workline:derive " + name + " -->" + out + "<!-- workline:end -->"
}

// codeRanges returns the byte ranges of a doc that are code: fenced blocks,
// and code spans on the other lines.
func codeRanges(content string) [][2]int {
	var out [][2]int
	fence, offset := "", 0
	for _, l := range strings.SplitAfter(content, "\n") {
		t := strings.TrimSpace(l)
		switch {
		case fence != "":
			out = append(out, [2]int{offset, offset + len(l)})
			if strings.HasPrefix(t, fence) {
				fence = ""
			}
		case strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~"):
			fence = t[:3]
			out = append(out, [2]int{offset, offset + len(l)})
		default:
			for _, m := range codeSpan.FindAllStringIndex(l, -1) {
				out = append(out, [2]int{offset + m[0], offset + m[1]})
			}
		}
		offset += len(l)
	}
	return out
}

func inRanges(ranges [][2]int, at int) bool {
	for _, r := range ranges {
		if at >= r[0] && at < r[1] {
			return true
		}
	}
	return false
}

func run(repo, command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = repo
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%v: %s", err, strings.TrimSpace(errOut.String()))
	}
	return strings.TrimSpace(out.String()), nil
}
