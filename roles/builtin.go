// Package roles lists the compiled-in roles: each joins the registry in
// its init() (internal/contract). The binary and the tests on the real
// roles import this one list, so a role dropped from it is dropped from
// both (docs/spec/layout.md). The roles whose code is not under roles/
// yet are listed from internal/ until they move (ADR-0070, move e).
package roles

import (
	_ "github.com/JN0V/workline/internal/builtin/documentalist"
	_ "github.com/JN0V/workline/internal/builtin/inspector"
	_ "github.com/JN0V/workline/internal/builtin/reviewer"
	_ "github.com/JN0V/workline/internal/tester"
	_ "github.com/JN0V/workline/roles/committer/code"
	_ "github.com/JN0V/workline/roles/developer/code"
	_ "github.com/JN0V/workline/roles/product-owner/code"
)
