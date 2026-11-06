package codepick

import (
	"fmt"
	"strings"
)

// cutListed is how many files a cut names; the others counted.
const cutListed = 5

// Cut says, for the person, what the budget left out of the files the
// text names: how many of their lines were given, of how many, file by
// file (#1159); "" when nothing named was cut. The agent reads the same
// in its task (Write).
func (p Pick) Cut() string {
	if !p.Short {
		return ""
	}
	var files []string
	given, total, more := 0, 0, 0
	for _, f := range p.Files {
		if f.Test || f.Above || f.Whole() || !f.Named && len(f.Unread) == 0 && f.Cut == 0 {
			continue
		}
		n := f.GivenLines()
		given, total = given+n, total+len(f.Lines)
		if len(files) == cutListed {
			more++
			continue
		}
		how := ""
		if len(f.Decls) > 0 {
			how = ", its declarations"
		}
		files = append(files, fmt.Sprintf("`%s` %d of %d lines%s", f.Path, n, len(f.Lines), how))
	}
	if len(files) == 0 {
		return ""
	}
	if more > 0 {
		files = append(files, fmt.Sprintf("and %d more files", more))
	}
	return fmt.Sprintf("%d of the %d lines of the files named given (%s)", given, total, strings.Join(files, ", "))
}

// BytesName is the setting Ask.Bytes is, as a role names it.
const BytesName = "code-bytes-max"

// CutBy are the settings that cut the pick, as Cut says it (#1159): the
// lines' (Ask.BudgetName), the bytes' (BytesName), or both; none when
// nothing was cut.
func (p Pick) CutBy() []string {
	var out []string
	if p.byLines || p.Short && !p.byBytes {
		name := p.Ask.BudgetName
		if name == "" {
			name = "the code budget"
		}
		out = append(out, name)
	}
	if p.byBytes {
		out = append(out, BytesName)
	}
	return out
}
