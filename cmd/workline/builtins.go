package main

// The compiled-in roles: each joins the registry in its init()
// (internal/contract), and `workline builtin` finds its steps there. The
// list is shared with the tests; it moves to roles/builtin.go with the
// roles' code (docs/spec/layout.md).
import _ "github.com/JN0V/workline/internal/builtin/all"
