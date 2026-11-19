package contract

import (
	"testing"
)

// refused says whether join panics.
func refused(join func()) (panicked bool) {
	defer func() { panicked = recover() != nil }()
	join()
	return false
}

// Review and Docs: one role offers each, every function set; none
// offered, the shipped registry's lookup is a programming error.
func TestReviewAndDocsAreOfferedOnceAndWhole(t *testing.T) {
	reviewed := func(string, string, string, string) (bool, error) { return true, nil }
	coverage := func(string, []string) ([]string, []string, []string, error) { return nil, nil, nil, nil }
	unread := func(string, []string) (string, error) { return "", nil }
	r := NewRegistry()
	r.Join("reviews", Role{Review: &Review{ReviewedAll: reviewed}, Docs: &Docs{Coverage: coverage, UnreadDocs: unread}})
	if v, n := r.Review(); v == nil || n != "reviews" {
		t.Errorf("Review = %v, %q", v, n)
	}
	if d, n := r.Docs(); d == nil || n != "reviews" {
		t.Errorf("Docs = %v, %q", d, n)
	}
	for what, join := range map[string]func(){
		"a second review":              func() { r.Join("two", Role{Review: &Review{ReviewedAll: reviewed}}) },
		"a second docs":                func() { r.Join("three", Role{Docs: &Docs{Coverage: coverage, UnreadDocs: unread}}) },
		"a review missing ReviewedAll": func() { NewRegistry().Join("x", Role{Review: &Review{}}) },
		"docs missing Coverage":        func() { NewRegistry().Join("x", Role{Docs: &Docs{UnreadDocs: unread}}) },
		"docs missing UnreadDocs":      func() { NewRegistry().Join("x", Role{Docs: &Docs{Coverage: coverage}}) },
		"a record missing Into": func() {
			NewRegistry().Join("x", Role{Record: &Record{Merge: func(b, _, _ string) string { return b }}})
		},
		"a record missing Merge": func() {
			NewRegistry().Join("x", Role{Record: &Record{Into: func(f, _, _, _ string) string { return f }}})
		},
	} {
		if !refused(join) {
			t.Errorf("%s is taken", what)
		}
	}
	restore := UseForTest(NewRegistry())
	defer restore()
	if !refused(func() { BuiltinReview() }) || !refused(func() { BuiltinDocs() }) {
		t.Error("none offered: the shipped lookup answers")
	}
}
