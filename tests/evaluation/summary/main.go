// Summary reads results.tsv and prints, per case and per models and effort,
// how many runs there were, the mean score and its range, and what a run used:
// one run says little, since a model answers differently from one run to the
// next.
//
//	go run ./tests/evaluation/summary [results.tsv]
package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

type key struct{ kase, models, effort string }

type group struct {
	scores              []float64 // share of the points earned
	tokensIn, tokensOut []float64
	seconds             []float64
	calls               []float64
}

func main() {
	path := "tests/evaluation/results.tsv"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma, r.LazyQuotes, r.FieldsPerRecord = '\t', true, -1
	head, err := r.Read()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	col := map[string]int{}
	for i, h := range head {
		col[h] = i
	}
	groups := map[key]*group{}
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		get := func(name string) string {
			if i, ok := col[name]; ok && i < len(row) {
				return row[i]
			}
			return ""
		}
		k := key{get("case"), get("models"), get("effort")}
		if k.models == "" {
			k.models = "(not recorded)"
		}
		g := groups[k]
		if g == nil {
			g = &group{}
			groups[k] = g
		}
		if got, total, ok := strings.Cut(get("score"), "/"); ok {
			if n, err1 := strconv.ParseFloat(got, 64); err1 == nil {
				if d, err2 := strconv.ParseFloat(total, 64); err2 == nil && d > 0 {
					g.scores = append(g.scores, n/d)
				}
			}
		}
		add(&g.tokensIn, get("tokens in"))
		add(&g.tokensOut, get("tokens out"))
		add(&g.seconds, get("seconds"))
		add(&g.calls, get("agent calls"))
	}
	keys := make([]key, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.kase != b.kase {
			return a.kase < b.kase
		}
		if a.models != b.models {
			return a.models < b.models
		}
		return a.effort < b.effort
	})
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "case\tmodels\teffort\truns\tscore\trange\tcalls\ttokens in\ttokens out\tseconds")
	for _, k := range keys {
		g := groups[k]
		lo, hi := bounds(g.scores)
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%.0f%%\t%.0f–%.0f%%\t%s\t%s\t%s\t%s\n", k.kase, k.models, k.effort, len(g.scores),
			100*mean(g.scores), 100*lo, 100*hi, show(g.calls, "%.1f"), show(g.tokensIn, "%.0f"), show(g.tokensOut, "%.0f"), show(g.seconds, "%.0f"))
	}
	w.Flush()
}

// add keeps a number, when the column holds one: older runs did not record
// every column.
func add(to *[]float64, s string) {
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		*to = append(*to, v)
	}
}

func mean(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	t := 0.0
	for _, x := range v {
		t += x
	}
	return t / float64(len(v))
}

func bounds(v []float64) (lo, hi float64) {
	for i, x := range v {
		if i == 0 || x < lo {
			lo = x
		}
		if i == 0 || x > hi {
			hi = x
		}
	}
	return lo, hi
}

// show gives a mean, or - when no run recorded it.
func show(v []float64, format string) string {
	if len(v) == 0 {
		return "-"
	}
	return fmt.Sprintf(format, mean(v))
}
