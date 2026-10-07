---
sources: [internal/builtin/documentalist/documentalist.go, roles/documentalist/role.yaml]
checked: 35ad2e5
verified: agent:claude-code
---
# Documentalist — cascades and the authority

How far a change of code reaches into the docs, what holds a release, and
which docs the code never rewrites. Part of
[the documentalist](../README.md); the findings named here are in
[the checks](checks.md).

## Cascades are cut

A change of code can make a technical doc suspect, which makes a product
doc suspect, and so on.

- Only the edges marked `now` are handled inside the change.
- Every other suspect goes on a pending list — one tracking issue, updated
  in place — with the moment it is due.
- On `release`, the documentalist runs first, and the release waits until
  the docs due then are up to date.
- Nothing is forgotten; nothing drags a small fix into a rewrite of the
  user guide.
- A doc following another doc along an edge due later stays `pending`
  until then, even once the doc it follows is fixed and merged.

## At the release

On `release` — or on a release tool's pull request, which is the release
([ADR-0017](../../../docs/adr/0017-the-release-manager.md)) — the
documentalist runs before the release tool tags.

- **A doc made suspect since the highest release tag** blocks until it is
  judged, by the agent or by a person moving `checked`.
- **One judged without being vouched for** (its `checked` not moved), or in
  parts, since its sources last changed, waits for a person and does not
  block.
- **A shallow clone**, which may lack the tag, blocks (`shallow-clone`).
- **Each doc due at the release** is `due`, and becomes the `propagate`
  task — brought up to date for its reader, growing when the reader gained
  something to know. A `due` doc left as it was blocks, so the line stops
  and nothing is released.

When gardening (the scheduled run), with a forge, two issues are kept in
place:

- "Docs due at the next release": the docs due;
- "Docs waiting for a person": the docs only a person can clear — too
  large for the agent, or judged without being vouched for — each with why
  and what to do.

## Code never rewrites the authority

When the code disagrees with a doc marked as the truth (`truth.doc`: a
spec, an ADR, the architecture), the doc is not updated to match:

- the documentalist opens an `issue` for the architect, titled "The code
  disagrees with <doc>", because the code may be the one that is wrong;
- the doc is only checked again, the issue tracking the disagreement;
- a patch changing what it says is refused (`truth-doc-changed`).

Every other doc follows the code.
