package commitmsg

import "testing"

// Punctuation entry: Given the allow entry `C++`, When a subject says `C++`,
// Then the check passes and does not fail to compile.
func TestAllowEntryWithPunctuation(t *testing.T) {
	s := defaults
	s.InternalCodesAllow = []string{"C++"}

	got, err := Check("fix: support C++ in the parser", s)
	if err != nil {
		t.Fatalf("the allow entry C++ must not fail to compile: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("findings = %v, want none", got)
	}
}

// Dot literal: Given the allow entry `F.1`, When a subject holds `FX1`,
// Then `FX1` is not allowed.
func TestAllowEntryDotIsLiteral(t *testing.T) {
	s := defaults
	s.InternalCodes = []string{`\bF[A-Z]1\b`}
	s.InternalCodesAllow = []string{"F.1"}

	got, err := Check("fix: handle FX1 in the parser", s)
	if err != nil {
		t.Fatal(err)
	}
	var rules []string
	for _, f := range got {
		rules = append(rules, f.Rule)
	}
	if len(rules) != 1 || rules[0] != "internal-code" {
		t.Fatalf("rules = %v, want [internal-code]", rules)
	}

	// and the entry itself still lets F.1 through
	got, err = Check("fix: handle F.1 in the parser", s)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("findings for F.1 = %v, want none", got)
	}
}
