// Package all lists the compiled-in roles: each joins the registry in its
// init() (internal/contract). The binary and the tests import this one list,
// so a role dropped from it is dropped from both.
package all

import (
	_ "github.com/JN0V/workline/internal/builtin/committer"
	_ "github.com/JN0V/workline/internal/builtin/documentalist"
	_ "github.com/JN0V/workline/internal/builtin/inspector"
	_ "github.com/JN0V/workline/internal/builtin/productowner"
	_ "github.com/JN0V/workline/internal/builtin/reviewer"
	_ "github.com/JN0V/workline/internal/developer"
	_ "github.com/JN0V/workline/internal/tester"
)
