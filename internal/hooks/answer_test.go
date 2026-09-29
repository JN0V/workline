package hooks

import (
	"bufio"
	"os"
	"testing"
	"time"
)

// Each answer gets its own time: two answers, each within the timeout, pass
// even when together they take longer than it.
func TestEachAnswerHasItsOwnTime(t *testing.T) {
	defer func(d time.Duration) { answerTimeout = d }(answerTimeout)
	answerTimeout = 200 * time.Millisecond
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	go func() {
		for _, a := range []string{"d\n", "y\n"} {
			time.Sleep(120 * time.Millisecond)
			w.WriteString(a)
		}
	}()
	in := bufio.NewReader(answerReader{r})
	for _, want := range []string{"d\n", "y\n"} {
		got, err := in.ReadString('\n')
		if err != nil || got != want {
			t.Fatalf("answer = %q, %v; want %q", got, err, want)
		}
	}
	if _, err := in.ReadString('\n'); !os.IsTimeout(err) {
		t.Fatalf("no answer: err = %v, want a timeout", err)
	}
}
