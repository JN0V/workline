package devmark

// Widen is a round's wait, for a person or the tester, as the record
// keeps it.
type Widen struct {
	At    string `json:"at"`    // the branch's head it waited at
	Scope string `json:"scope"` // the bounds' Scope then, hashed
	// Held: no file asked, a project check red its round did not fix
	// (#938): blocked for a person, who sends it back to resume.
	Held bool `json:"held,omitempty"`
	// Back: a round ran after the issue was sent back, by a person.
	Back bool `json:"back,omitempty"`
	// Objected: the round objected to a frozen test at At, and its
	// issue waits for the tester's answer (#987): a rewrite is taken by
	// the next round, a test kept asks the round again. Kept: the
	// tester's answers that kept a test on its Draft then; one more is
	// its answer.
	Objected bool `json:"objected,omitempty"`
	Draft    int  `json:"draft,omitempty"` // the tester's Draft it objected on
	Kept     int  `json:"kept,omitempty"`
	// Review: the round held a fix of the review's findings off a green
	// head (#987): they are asked again when the round resumes. Sent
	// back, the findings its last round asked are asked again in any
	// case, from the record (#1022).
	Review bool `json:"review,omitempty"`
}
