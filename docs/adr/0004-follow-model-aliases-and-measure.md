# ADR-0004: Follow the agents' model aliases, notice when they move, measure

- **Status:** accepted
- **Date:** 2026-09-25

## Context

A role asks for a tier and an effort, never a model (docs/spec/model-grid.md).
Until the grid exists, the engine gives Claude Code its aliases: `haiku`,
`sonnet`, `opus`. An alias floats: when the provider ships a new generation,
the same alias names it the next day, and a role's behaviour can change with
no change in workline.

The usual advice is to pin exact model ids and re-run a regression suite
before each upgrade ([Lyceum](https://lyceum.technology/magazine/model-deprecation-risk-version-pinning-notice-periods/)).
Regression suites exist to catch the "silent model swap"
([FutureAGI](https://futureagi.com/blog/llm-regression-testing-model-swap/)).
Evaluation platforms (Langfuse, Promptfoo) trigger their runs on a change of
prompt or code; none was found that triggers on the model behind an alias
changing.

Pinning costs a person an update each time a model ships, for every agent:
at today's pace, nearly every day. Floating unwatched loses the evaluation's
worth: a score says nothing once another model answers.

Each agent call now records the exact model that answered, from the agent's own
report (`claude-haiku-4-5-20251001`).

## Decision

- **Follow.** Tiers keep resolving to the agents' aliases. A new generation
  comes in without anyone updating workline.
- **Notice.** The engine keeps, per machine, the last exact model seen for each
  agent and tier. When a call is answered by another one, the run reports it
  once (`model-changed`, never blocking): which tier, which model before, which
  now, and that the evaluation's scores were earned by the old one.
- **Measure.** The scheduled evaluation measures the new model. Its summary
  marks the scores earned by a model that no longer answers.
- **Pin by exception.** A role, or a project, pins an exact model only when a
  measurement shows the new one doing worse — and says so where it pins.

## Consequences

- No list of model ids to keep up to date; the record, the notice and the
  evaluation do the watching.
- Between a swap and the next evaluation, a role may run on an unmeasured
  model. The notice makes that visible; it does not prevent it.
- The machine's record lives outside the repository: two machines notice the
  same change each on their own.
- A pinned model the provider retires fails the call, as an unavailable agent
  does; the pin's note says why it was pinned, so a person can drop it.
