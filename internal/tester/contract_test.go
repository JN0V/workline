package tester

import (
	"reflect"
	"testing"

	"github.com/JN0V/workline/internal/contract"
)

// The tester joins the shipped registry offering every field of what the
// doctor asks of the project's tests: none left nil.
func TestTheTesterOffersEveryTestsField(t *testing.T) {
	tt, name := contract.Builtins().Tests()
	if tt == nil || name != "tester" {
		t.Fatalf("Tests = %v, %q: the tester offers none", tt, name)
	}
	v := reflect.ValueOf(*tt)
	for i := range v.NumField() {
		if v.Field(i).IsNil() {
			t.Errorf("the tester leaves contract.Tests.%s nil", v.Type().Field(i).Name)
		}
	}
}
