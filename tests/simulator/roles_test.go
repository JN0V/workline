package simulator

// The simulator runs the line in process on the real roles: they join the
// registry in their init() (internal/contract), as the binary's
// cmd/workline/builtins.go links them (docs/spec/layout.md, Tests).
import (
	_ "github.com/JN0V/workline/internal/builtin/committer"
	_ "github.com/JN0V/workline/internal/builtin/documentalist"
	_ "github.com/JN0V/workline/internal/builtin/inspector"
	_ "github.com/JN0V/workline/internal/builtin/productowner"
	_ "github.com/JN0V/workline/internal/builtin/reviewer"
	_ "github.com/JN0V/workline/internal/developer"
	_ "github.com/JN0V/workline/internal/tester"
)
