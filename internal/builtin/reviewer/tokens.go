package reviewer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// tokensSpent is what the engine says of a call it did not make, the run
// having spent its ai-max-tokens.
const tokensSpent = "the run spent its ai-max-tokens"

// spent is what each call of the run used, as the engine kept it
// (out/calls.jsonl): a lens's part, or a judge's question, each told by the
// lenses it was for (#147).
func spent(runDir string, st state, c candidates) []Spent {
	f, err := os.Open(filepath.Join(runDir, "out", "calls.jsonl"))
	if err != nil {
		return nil
	}
	defer f.Close()
	judged := map[string]Finding{} // a judge's question, by its key
	for _, a := range c.Asked {
		i := strings.LastIndex(a, "\x00")
		if i < 0 {
			continue
		}
		parts := strings.Split(a[:i], "\x00") // JudgeKey: where, lens, …
		if len(parts) >= 2 {
			judged[a[i+1:]] = Finding{Where: parts[0], Lens: parts[1]}
		}
	}
	var out []Spent
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		var call struct {
			For       string `json:"for"`
			Model     string `json:"model"`
			TokensIn  int    `json:"tokens-in"`
			TokensOut int    `json:"tokens-out"`
		}
		if json.Unmarshal(sc.Bytes(), &call) != nil || call.For == "" {
			continue
		}
		s := Spent{For: call.For, Model: call.Model, TokensIn: call.TokensIn, TokensOut: call.TokensOut}
		if key, ok := strings.CutPrefix(call.For, "judge/"); ok {
			f := judged[key]
			s.Lenses, s.Where = f.Lens, f.Where
		} else if st.Together {
			s.Lenses = strings.Join(st.Lenses, ", ")
		} else {
			_, s.Lenses, _ = strings.Cut(call.For, "-")
		}
		out = append(out, s)
	}
	return out
}

// tokensLine says, after the summary, what the review's calls used, all
// together, against what a run may spend.
func tokensLine(calls []Spent, max int) string {
	in, out := 0, 0
	for _, c := range calls {
		in, out = in+c.TokensIn, out+c.TokensOut
	}
	s := fmt.Sprintf("; tokens: %d in, %d out (%d %s)", in, out, len(calls), plural(len(calls), "call", "calls"))
	if max > 0 {
		return s + fmt.Sprintf(", of %d a run (ai-max-tokens)", max)
	}
	return s + ", no ai-max-tokens"
}
