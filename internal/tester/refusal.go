package tester

// A refusal that comes back unchanged (#1666): the tester's comment on an
// issue keeps a hidden record of the findings of its last refusal and how
// many runs in a row gave them. A run refused for the same findings is a
// loop an agent asked again does not leave: it blocks at once, said so.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/verdict"
)

// refusalMark marks, in the tester's comment, the findings of its last
// refusal (a digest) and the runs in a row that gave them.
const refusalMark = "tester-refusal"

var refusalRecord = regexp.MustCompile(`<!-- workline:` + refusalMark + ` key=([0-9a-f]+) n=([0-9]+) -->`)

// refusalKey is a digest of what a refusal names: each finding's rule,
// place and message, in order.
func refusalKey(fs []verdict.Finding) string {
	h := sha256.New()
	for _, f := range fs {
		// A hash's Write never returns an error.
		_, _ = h.Write([]byte(f.Rule + "\x00" + f.Where + "\x00" + f.Message + "\x00"))
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// lastRefusal reads, in the issue's comments, the newest record of a
// refusal: its key and the runs that gave it; "", 0 when there is none.
func lastRefusal(comments []string) (string, int) {
	for i := len(comments) - 1; i >= 0; i-- {
		if m := refusalRecord.FindStringSubmatch(comments[i]); m != nil {
			n, _ := strconv.Atoi(m[2])
			return m[1], n
		}
	}
	return "", 0
}

// repeatedRefusal says how many runs in a row, this one included, gave v's
// findings, and whether this is a repeat; the record to write in the
// comment. A refusal that mends (`again`) is the one that loops.
func repeatedRefusal(t task, v verdict.Verdict) (times int, mark string) {
	key := refusalKey(v.Findings)
	times = 1
	if t.PriorRefusal != "" && t.PriorRefusal == key {
		times = t.PriorTimes + 1
	}
	return times, fmt.Sprintf("\n<!-- workline:%s key=%s n=%d -->", refusalMark, key, times)
}

// unchangedWhy is what the comment says of a finding that came back as it
// was.
func unchangedWhy(times int, why string) string {
	return fmt.Sprintf("the same finding came back unchanged, %d runs in a row, so the issue is blocked: %s", times, strings.TrimSpace(why))
}
