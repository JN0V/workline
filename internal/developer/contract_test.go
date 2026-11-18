package developer

import (
	"reflect"
	"testing"

	"github.com/JN0V/workline/internal/contract"
)

// The developer joins the shipped registry offering every field of the
// code the engine and the line ask of it: none left nil, which they would
// call (contract.BuiltinCode).
func TestTheDeveloperOffersEveryCodeField(t *testing.T) {
	c, name := contract.Builtins().Code()
	if c == nil || name != "developer" {
		t.Fatalf("Code = %v, %q: the developer offers none", c, name)
	}
	v := reflect.ValueOf(*c)
	for i := range v.NumField() {
		if v.Field(i).IsNil() {
			t.Errorf("the developer leaves contract.Code.%s nil", v.Type().Field(i).Name)
		}
	}
}
