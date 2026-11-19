package simulator

// The simulator runs the line in process on the real roles: they join the
// registry in their init() (internal/contract), as the binary links them
// (roles/builtin.go, docs/spec/layout.md, Tests).
import _ "github.com/JN0V/workline/roles"
