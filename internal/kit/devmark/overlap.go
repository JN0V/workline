package devmark

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/JN0V/workline/internal/work"
)

// OverlapWhy says why an issue is passed over: its Scope's path p
// overlaps issue n's, held.
func OverlapWhy(n int, p string) string {
	return fmt.Sprintf("its %s's `%s` overlaps #%d's, taken", work.Name(work.Scope), p, n)
}

// overlapSaid is OverlapWhy as the developer's comment says it.
var overlapSaid = regexp.MustCompile("'s `[^`]+` overlaps #\\d+'s, taken")

// PassedForOverlap says whether the developer's comment c says it passed
// its issue over for an overlap (OverlapWhy).
func PassedForOverlap(c string) bool {
	return strings.Contains(c, "waits on this issue") && overlapSaid.MatchString(c)
}
