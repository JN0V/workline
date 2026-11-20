package sample

import (
	"os"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/contract"
	"github.com/JN0V/workline/internal/kit/docmark"
)

// The sample's tests run beside a fixture documentalist (docs/spec/layout.md,
// Tests): its sources read nothing, a quote is found on the first line
// holding it that is not a `//` comment, a name is never in the code. The
// real documentalist is tried in its own package and, linked whole, by the
// conformance cases.
func TestMain(m *testing.M) {
	contract.Join("fixture-docs", contract.Role{DocsSample: &contract.DocsSample{
		Settings:     func(map[string]any) (docmark.SampleView, error) { return docmark.SampleView{}, nil },
		SourcesAt:    func(string, *docmark.Doc, int) ([]string, bool, error) { return nil, true, nil },
		SourceFileAt: func(string, *docmark.Doc, string) (string, string, bool) { return "", "", false },
		QuoteIn: func(_, content, quote string) (found, code bool, line int) {
			for i, l := range strings.Split(content, "\n") {
				if strings.Contains(l, quote) {
					return true, !strings.HasPrefix(strings.TrimSpace(l), "//"), i + 1
				}
			}
			return false, false, 0
		},
		InCodeAt: func(string, string, string) (bool, error) { return false, nil },
	}})
	os.Exit(m.Run())
}
