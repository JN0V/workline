package forge

import (
	"cmp"
	"os"
	"regexp"
	"strconv"
)

// At is the hidden line a role's sticky comment holds above its record:
// the head its run read, and that run's place among the project's CI
// runs, GitHub Actions' run id or GitLab's pipeline id, which grows with
// each push; none on a machine. An apply of an older run than the
// comment's keeps the comment's text, and merges only its record into it
// (Newer, #598): GitLab may apply two waiting runs in any order.
func At(head string) string {
	if run := cmp.Or(os.Getenv("GITHUB_RUN_ID"), os.Getenv("CI_PIPELINE_ID")); run != "" {
		return Marker("at head=" + head + " run=" + run)
	}
	return Marker("at head=" + head)
}

var atLine = regexp.MustCompile(`<!-- workline:at head=[0-9a-f]* run=(\d+) -->`)

// Newer says whether found, a sticky comment, was written by a later run
// than body: both name their run, found's the greater. One naming none,
// written on a machine or by an engine before it, is never newer.
func Newer(found, body string) bool {
	f, b := atLine.FindStringSubmatch(found), atLine.FindStringSubmatch(body)
	if f == nil || b == nil {
		return false
	}
	fr, err1 := strconv.ParseUint(f[1], 10, 64)
	br, err2 := strconv.ParseUint(b[1], 10, 64)
	return err1 == nil && err2 == nil && fr > br
}
