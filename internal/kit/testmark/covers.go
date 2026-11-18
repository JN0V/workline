package testmark

import (
	"slices"
	"strings"
)

// Bare is a report's id without its parameters: a parametrised test's row
// (pytest's `test_x[1]`, JUnit 5's `addsItem(int)[1]`) is its test, and
// `addsItem()` is `addsItem`.
func Bare(id string) string {
	if i := strings.Index(id, "["); i > 0 && id[i-1] != ' ' && strings.HasSuffix(id, "]") {
		id = id[:i]
		if j := strings.LastIndex(id, "("); j > 0 && strings.HasSuffix(id, ")") {
			id = id[:j]
		}
	}
	return strings.TrimSuffix(id, "()")
}

// CoveredBy is the scenario a test id is declared for, by a record's
// covers: the one whose ids hold it, whole or as a whole dotted part
// either way; "" when none, or more than one.
func CoveredBy(covers map[string][]string, id string) string {
	if all := Covering(covers, id); len(all) == 1 {
		return all[0]
	}
	return ""
}

// Covering are the scenarios a test id is declared for, sorted.
func Covering(covers map[string][]string, id string) []string {
	id = strings.Trim(strings.TrimSpace(id), "`")
	if id == "" {
		return nil
	}
	var out []string
	for sc, ids := range covers {
		if slices.ContainsFunc(ids, func(t string) bool { return SameTest(t, id) }) {
			out = append(out, sc)
		}
	}
	slices.Sort(out)
	return out
}

// SameTest says whether two ids name the same test: equal, or one the
// other's whole dotted or file-named tail (`CartTest.addsItem` and
// `shop.CartTest.addsItem`; `addsItem` and `src/CartTest.kt:addsItem`),
// a parametrised row's parameters aside (`test_x` and `tests.test_x[1]`).
func SameTest(a, b string) bool {
	a, b = Bare(a), Bare(b)
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	tail := func(long, short string) bool {
		return strings.HasSuffix(long, "."+short) || strings.HasSuffix(long, ":"+short)
	}
	return tail(a, b) || tail(b, a)
}
