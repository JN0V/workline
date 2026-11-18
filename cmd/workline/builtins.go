package main

// The compiled-in roles: each joins the registry in its init()
// (internal/contract), and `workline builtin` finds its steps there. The
// list moves to roles/builtin.go with the roles' code (docs/spec/layout.md).
import (
	_ "github.com/JN0V/workline/internal/builtin/committer"
	_ "github.com/JN0V/workline/internal/builtin/documentalist"
	_ "github.com/JN0V/workline/internal/builtin/inspector"
	_ "github.com/JN0V/workline/internal/builtin/productowner"
	_ "github.com/JN0V/workline/internal/builtin/reviewer"
	_ "github.com/JN0V/workline/internal/developer"
	_ "github.com/JN0V/workline/internal/tester"
)
