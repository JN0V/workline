package work

// An issue's sections are read by meaning (ADR-0039): a heading counts for
// what it says, through one table — the one place in the engine that names
// a section's heading. Each meaning lists its headings: the first is the
// one the role writes, every one is read, so an issue written under a
// heading is read under it forever, and a new name goes in front.

// Meaning is what a section of an issue says, whatever heading it is under.
// Its id is the key a refine gives the section, and never changes.
type Meaning string

// The meanings of an issue's sections.
const (
	Need         Meaning = "need"
	Example      Meaning = "example"
	Steps        Meaning = "steps"
	Validation   Meaning = "validation"
	Verification Meaning = "verification"
	Scope        Meaning = "scope"
)

// Row is a meaning and its headings, the first the one written. A
// builder's section is below the line between the person's part of an
// issue and the builder's; above it, a person's, which the role drafts for
// a person to make theirs.
type Row struct {
	Meaning  Meaning
	Headings []string
	Builder  bool
}

// Table holds each meaning once, in the order the role writes an issue's
// sections in, for its reader: the person's part — the need, a real
// example of it or a bug's steps to reproduce, how a person accepts it —
// then the builder's — how the machine proves it, the code it touches.
type Table []Row

// Headings is the table the engine reads and writes issues with. A heading
// never leaves it: issues were written under it.
var Headings = Table{
	{Meaning: Need, Headings: []string{"Need"}},
	{Meaning: Example, Headings: []string{"Example"}},
	{Meaning: Steps, Headings: []string{"Steps to reproduce"}},
	{Meaning: Validation, Headings: []string{"Validation"}},
	{Meaning: Verification, Headings: []string{"Verification"}, Builder: true},
	{Meaning: Scope, Headings: []string{"Scope"}, Builder: true},
}

// Required are the meanings an item needs before it is ready, in the order
// a check says what is missing.
var Required = []Meaning{Need, Verification, Validation, Scope}

// Meaning is the meaning a heading names; "" when it names none.
func (t Table) Meaning(heading string) Meaning {
	for _, r := range t {
		for _, h := range r.Headings {
			if same(h, heading) {
				return r.Meaning
			}
		}
	}
	return ""
}

// same says whether a heading read is one of the table's: written as the
// table has it.
func same(table, read string) bool { return table == read }

// Name is the heading a meaning is written under.
func (t Table) Name(m Meaning) string {
	for _, r := range t {
		if r.Meaning == m && len(r.Headings) > 0 {
			return r.Headings[0]
		}
	}
	return ""
}

// Of are the headings a meaning is read under, the one written first.
func (t Table) Of(m Meaning) []string {
	for _, r := range t {
		if r.Meaning == m {
			return r.Headings
		}
	}
	return nil
}

// Meanings are the table's meanings, in the order the role writes them.
func (t Table) Meanings() []Meaning {
	out := make([]Meaning, 0, len(t))
	for _, r := range t {
		out = append(out, r.Meaning)
	}
	return out
}

// Builder says whether a meaning's section is the builder's, below the line.
func (t Table) Builder(m Meaning) bool {
	for _, r := range t {
		if r.Meaning == m {
			return r.Builder
		}
	}
	return false
}

// With is the table with written as the heading m is written under, those
// it had still read after it: how a meaning takes a new name.
func (t Table) With(m Meaning, written string) Table {
	out := make(Table, len(t))
	for i, r := range t {
		out[i] = r
		if r.Meaning == m {
			out[i].Headings = append([]string{written}, r.Headings...)
		}
	}
	return out
}

// Section is a body's section read by its meaning: the heading it is under
// there, the first when several hold it, and the text under all of them.
type Section struct {
	Heading string
	Text    string
}

// Read is a body's sections by meaning, as Sections reads them by heading:
// a meaning under two headings holds the text under both, the first one's
// first. A heading the table does not hold is in none.
func (t Table) Read(body string) map[Meaning]Section {
	out := map[Meaning]Section{}
	for _, p := range parts(body) {
		m := t.Meaning(p.heading)
		if m == "" {
			continue
		}
		s, ok := out[m]
		if !ok {
			s.Heading = p.heading
		}
		s.Text = join(s.Text, p.text)
		out[m] = s
	}
	return out
}

// In is the first of names, headings, that names meaning m; ok is false
// when none does.
func (t Table) In(names []string, m Meaning) (string, bool) {
	if m == "" {
		return "", false
	}
	for _, n := range names {
		if t.Meaning(n) == m {
			return n, true
		}
	}
	return "", false
}

// MeaningOf is the meaning a heading names in the engine's table; "" for
// none.
func MeaningOf(heading string) Meaning { return Headings.Meaning(heading) }

// Name is the heading the engine writes a meaning under.
func Name(m Meaning) string { return Headings.Name(m) }

// ByMeaning is a body's sections by meaning, read with the engine's table.
func ByMeaning(body string) map[Meaning]Section { return Headings.Read(body) }

// Has says whether names, headings, hold one of meaning m.
func Has(names []string, m Meaning) bool {
	_, ok := Headings.In(names, m)
	return ok
}
