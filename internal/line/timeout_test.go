package line

import (
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/backlog"
)

// Given drafts or a question the forge does not date, at cautious
// (acts.ask.wait: never), when the line says what waits, then it names
// the setting that keeps them waiting, not the missing date (#1195).
func TestNeverSaysTheSettingUndated(t *testing.T) {
	never := backlog.Delay{Never: true}
	for name, said := range map[string]string{
		"drafts":   draftsWait(never, time.Time{}, false, 3),
		"question": questionWaits(never, backlog.Silence{}, 3),
	} {
		if !strings.Contains(said, "(acts.ask.wait: never)") || strings.Contains(said, "does not") {
			t.Errorf("%s: want the setting named, got %q", name, said)
		}
	}
}
