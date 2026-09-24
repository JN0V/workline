// Package workline ships the built-in roles and the default routing inside the
// binary, so the engine works from any repository without a checkout of this one.
package workline

import "embed"

// Roles holds the roles/ folder as shipped.
//
//go:embed all:roles
var Roles embed.FS

// DefaultRouting is the line as shipped (routing.default.yaml).
//
//go:embed routing.default.yaml
var DefaultRouting []byte
