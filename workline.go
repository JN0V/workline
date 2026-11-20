// Package workline ships the built-in roles and the default routing inside the
// binary, so the engine works from any repository without a checkout of this one.
package workline

import "embed"

// Roles holds what each role ships, named one by one (docs/spec/layout.md,
// "A role folder"): its role.yaml, prompts/ and kinds/, and its pre and
// post. Never its code/, docs/, README.md nor cases: the binary carries no
// file it does not read (ADR-0070, move c).
//
//go:embed roles/*/role.yaml all:roles/*/prompts roles/*/kinds roles/*/pre roles/*/post
var Roles embed.FS

// Skills holds the shared skills, skills/<name>/SKILL.md in the Agent
// Skills format, that a role's run loads when its pre names them
// (docs/spec/role-contract.md, "Shared skills").
//
//go:embed skills
var Skills embed.FS

// DefaultRouting is the line as shipped (defaults/routing.yaml).
//
//go:embed defaults/routing.yaml
var DefaultRouting []byte
