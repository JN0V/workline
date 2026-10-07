# ADR-0037: The schedule runs every step

- **Status:** accepted
- **Date:** 2026-10-07
- **Builds on:** ADR-0006 (gardening opens one merge request per task),
  ADR-0018 (the product owner keeps the backlog); principles 2, 12
- **Amends:** the routing spec's "the first step that does not pass stops
  the sequence" ([routing.md](../spec/routing.md#the-line))

## Context

- The line stopped at the first step that did not pass: right for a
  gate (`pre-push`, `merge-request`, `release`), where each step guards
  the same change and a block means "do not go on".
- On `schedule` the roles are upkeep, each on its own: the documentalist
  keeps the docs, the product owner the backlog. A gardening run blocked
  by the documentalist (one task over its context budget) left the
  backlog untouched that night: the product owner never ran.
- How CI says it:
  - GitHub Actions' matrix `fail-fast: false` runs every job and fails
    the workflow if one fails; `continue-on-error` and GitLab's
    `allow_failure` let a failed job pass the pipeline.
  - workline takes the first: a step that does not pass is never read as
    a pass (principle 12), only kept from stopping the others.

## Decision

- **`fail-fast`, per event, in the routing**: `true` stops the line at the
  first step that does not pass (the default for every event); `false`
  runs every step, and the line takes the worst verdict.
- **The shipped line sets `fail-fast: {schedule: false}`**; a project
  changes it per event in `.workline/config.yaml`, as it changes `events`.
- **Worst verdict**: `block`, then `human`, then `blocked-external`, then
  `pass`. The summary names each step that did not pass and its verdict.
- **A role that does not pass runs no handoff**, whatever `fail-fast`
  says; a chain past `max-handoffs` is a `block` of that step.
- Nothing changes for apply: `workline apply --line` applies the pending
  runs in order and stops at the first that does not pass, since a run
  applied out of order could see a tree it was not judged on.

## Consequences

- A gardening night keeps the backlog even when the docs block, and the
  job still fails, so a person sees why.
- A project wanting the old schedule sets `fail-fast: {schedule: true}`.
- Cases: `routing/schedule-runs-every-step`,
  `routing/fail-fast-set-per-event`,
  `routing/keep-going-on-merge-request`.
