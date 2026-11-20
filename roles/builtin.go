// Package roles lists the compiled-in roles: each joins the registry in
// its init() (internal/contract). The binary and the tests on the real
// roles import this one list, so a role dropped from it is dropped from
// both (docs/spec/layout.md). Each role's code is in its own folder,
// roles/<name>/code (ADR-0070, move e).
package roles

import (
	_ "github.com/JN0V/workline/roles/committer/code"
	_ "github.com/JN0V/workline/roles/developer/code"
	_ "github.com/JN0V/workline/roles/documentalist/code"
	_ "github.com/JN0V/workline/roles/inspector/code"
	_ "github.com/JN0V/workline/roles/product-owner/code"
	_ "github.com/JN0V/workline/roles/reviewer/code"
	_ "github.com/JN0V/workline/roles/tester/code"
)
