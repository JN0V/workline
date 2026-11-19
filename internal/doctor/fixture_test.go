package doctor

import (
	"os"
	"testing"

	"github.com/JN0V/workline/internal/contract"
)

// fixtureDocs is what the fixture documentalist answers the doctor:
// the docs it sorts, and the files it does not read.
var fixtureDocs struct {
	tracked, none, untracked []string
	unread                   string
}

// The doctor's tests run on a fixture role in place of the documentalist
// (docs/spec/layout.md, Tests): the real one is tried in its own package
// and, linked whole, by the conformance cases.
func TestMain(m *testing.M) {
	r := contract.NewRegistry()
	r.Join("fixture-docs", contract.Role{Docs: &contract.Docs{
		Coverage: func(string, []string) ([]string, []string, []string, error) {
			return fixtureDocs.tracked, fixtureDocs.none, fixtureDocs.untracked, nil
		},
		UnreadDocs: func(string, []string) (string, error) { return fixtureDocs.unread, nil },
	}})
	restore := contract.UseForTest(r)
	code := m.Run()
	restore()
	os.Exit(code)
}
