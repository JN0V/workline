---
sources: [internal/builtin/documentalist/documentalist.go, roles/documentalist/role.yaml]
checked: c2e40104
verified: agent:claude-code
judged-in-parts: c7e1e98
---
# Documentalist — cascades and the authority

How far a change of code reaches into the docs, and which docs the code
never rewrites. Part of
[the documentalist](../README.md); the findings named here are in
[the checks](checks.md).

## Cascades are cut

A change of code can make a technical doc suspect, which makes a product
doc suspect, and so on.

- Only the edges marked `now` are handled inside the change.
- Every other suspect goes on a pending list — one tracking issue, updated
  in place — with the moment it is due: gardening (`schedule`) by default.
- At that moment, each doc due becomes the `propagate` task: brought up
  to date for its reader, growing when the reader gained something to
  know. A doc due never blocks; one left as it was waits for the next
  run.
- Nothing is forgotten; nothing drags a small fix into a rewrite of the
  user guide.
- A doc following another doc along an edge due later stays `pending`
  until then, even once the doc it follows is fixed and merged.

## When gardening

With a forge, two issues are kept in place:

- "Docs due later": the docs due at another moment their edge names;
- "Docs waiting for a person": the docs only a person can clear — too
  large for the agent, naming a source that cannot be read, or judged
  without being vouched for — each with why and what to do.

The release never waits for them
([ADR-0051](../../../docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-1-the-release-weekly)).

## Code never rewrites the authority

When the code disagrees with a doc marked as the truth (`truth.doc`: a
spec, an ADR, the architecture), the doc is not updated to match:

- the documentalist opens an `issue` for the architect, titled "The code
  disagrees with <doc>", because the code may be the one that is wrong;
- the doc is only checked again (or, if it was not given whole, only
  `judged`), the issue tracking the disagreement;
- a patch changing what it says is refused (`truth-doc-changed`).

Every other doc follows the code.
