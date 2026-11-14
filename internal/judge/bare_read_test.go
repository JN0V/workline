package judge

import (
	"reflect"
	"testing"
)

// Bare search: given a judge's answer `- grep: "expandHome"`, when workline
// reads it, then it is a read of the lines naming expandHome, and the judge
// is asked again.
func TestAnswerReadingBareGrep(t *testing.T) {
	got, err := answerReading(Answer{}, []byte("- grep: \"expandHome\"\n"))
	if err != nil {
		t.Fatalf("a bare grep is a read, not an error: %v", err)
	}
	if want := []Read{{Grep: "expandHome"}}; !reflect.DeepEqual(got.Reads, want) {
		t.Errorf("reads = %+v, want %+v", got.Reads, want)
	}
	if got.Yes || got.Unsure {
		t.Errorf("a read is no verdict: %+v", got)
	}
}
