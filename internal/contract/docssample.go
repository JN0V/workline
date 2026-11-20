package contract

import (
	"fmt"

	"github.com/JN0V/workline/internal/kit/docmark"
)

// DocsSample is what the weekly sample of the docs vouched for asks of the
// role that keeps the docs (ADR-0014, step 4): a doc's sources at the commit it was checked at, a quote looked for in them,
// a name looked for in the code, and its settings, as the role reads them.
type DocsSample struct {
	// Settings reads the role's settings, merged, as the sample reads them.
	Settings func(merged map[string]any) (docmark.SampleView, error)
	// SourcesAt reads a doc's sources at the commit each repository's
	// `checked` names, at most limit characters; SourceFileAt one of them.
	SourcesAt    func(repo string, d *docmark.Doc, limit int) (evidence []string, fits bool, err error)
	SourceFileAt func(repo string, d *docmark.Doc, p string) (content, commit string, ok bool)
	// QuoteIn says whether quote is in content, the file p, and in its code
	// rather than a comment, and at which line; InCodeAt whether name is
	// written in the code at rev.
	QuoteIn  func(p, content, quote string) (found, code bool, line int)
	InCodeAt func(repo, rev, name string) (bool, error)
}

// whole refuses a DocsSample missing a function.
func (d *DocsSample) whole(name string) {
	if d.Settings == nil || d.SourcesAt == nil ||
		d.SourceFileAt == nil || d.QuoteIn == nil || d.InCodeAt == nil {
		panic(fmt.Sprintf("contract: the role %q offers the docs' sample missing a function", name))
	}
}

// DocsSample is what the role keeping the docs offers the sample, and its
// name; nil when none joined offers it.
func (r *Registry) DocsSample() (*DocsSample, string) {
	return offered(r, func(role Role) *DocsSample { return role.DocsSample })
}

// BuiltinDocsSample is what the compiled-in role keeping the docs offers
// the sample; none joined is a programming error.
func BuiltinDocsSample() *DocsSample {
	d, _ := builtins.DocsSample()
	if d == nil {
		panic("contract: no compiled-in role offers the docs' sample")
	}
	return d
}
