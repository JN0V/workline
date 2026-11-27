---
sources: [internal/engine/engine.go#run, internal/engine/engine.go#askAgain, internal/engine/engine.go#stepUp]
checked: e2e165c2d
verified: agent:claude-code
---
# Stepping up after failures

Part of [the model grid](model-grid.md).

A light model is the right default, not a ceiling. When `post` blocks on
what the agent produced — not a `final` block, which the answer did not
cause ([the role's outcome](role-outcome.md)) — the engine may retry one
tier up, as the role's `promote-after` says
([what a role declares](model-grid.md#what-a-role-declares)).

The ladder stops at `frontier`. The number of attempts is capped at
`promote-after` + 1: the last one is the only one a tier up. A role's
`ai-max-seconds` caps them all together, in time
([the role contract](role-contract.md)).
An answer that could not be read at all is asked once more on the same tier,
besides, even when `promote-after` is 0 — a part's too, a reviewer's lens,
when `promote-after` is above 0 ([one run](role-contract.md#one-run)). So
is a block `post` marks `again`, one the answer can mend, when
`promote-after` is 0: one more call in all, three for the tester, each
with its reason.
