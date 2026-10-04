package agent

import (
	"testing"

	"github.com/JN0V/workline/internal/intent"
)

// An intention the engine applies but the agent is never shown the shape
// of is one no agent proposes: the product owner's refine went unproposed
// on DomoticsCore for that.
func TestEveryIntentionHasItsShape(t *testing.T) {
	for kind := range intent.Catalogue {
		if contracts[kind] == "" {
			t.Errorf("%s: no shape in the output contract (contracts, prompt.go)", kind)
		}
	}
}
