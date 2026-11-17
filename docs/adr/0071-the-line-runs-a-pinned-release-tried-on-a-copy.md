# ADR-0071: workline's own line runs a pinned release, tried on a copy

- **Status:** proposed, until the maintainer merges it
- **Date:** 2026-11-17
- **Decided by:** the maintainer, on 2026-11-17, from a three-agent
  panel's synthesis after the day's five whys
- **Builds on:** [ADR-0042], [ADR-0043], [ADR-0044], [ADR-0063],
  [ADR-0068] §5 to §7; principles 6, 7, 9, 10, 12
  ([PRINCIPLES.md](../PRINCIPLES.md))
- **Amends, once built:** [the engine action](../../.github/actions/engine/action.yml)
  (main's engine for the line's jobs); [ADR-0063]'s working agreements
  (a fix is tried on a clone with a fake forge)

## Context

The five whys of 2026-11-17:

- **p1 opens as fast as it closes**: 21 opened, 7 closed in 7 days.
- **The line finds its defects while landing its fixes**: 4 of the
  last 6 p1 were line defects found while landing another p1:
  [#1678] → [#1704], [#1638] → [#1670] and [#1676], [#1686] →
  [#1687] and [#1701].
- **The line runs on itself, from main**: judge runs the pull
  request head's engine; the `workflow_run` jobs (develop, answer,
  apply) and the `pull_request_target` jobs (bounds, merge) build the
  engine from `main` ([engine action](../../.github/actions/engine/action.yml)).
  A defect on `main` blocks the pull request that fixes it.
- **Fixes are never tried on GitHub's CI**: `## Tried` says a local
  run, or boilerplate; V1 was accepted on a local forge.

## Decision

**workline's line runs a published release, pinned, as a user's does.
A release candidate moves the pin only after a real try on a GitHub
copy. A defect the pinned release carries in a line path is fixed by
the person's own pull request.**

### 1. The line runs a pinned release

- **The line's jobs take a release**: develop, answer, apply, bounds,
  merge download `WORKLINE_VERSION`, checksum checked, as
  [the templates](../../defaults/ci/github/workline-merge.yml) do.
  No engine is built from `main` for them.
- **The pull request's code stays tested from its head**: `go test`,
  conformance, the simulator, the gates' tools.
- **Judge runs the pinned release too**: its verdict is the line's
  verdict, so the line's engine gives it; a head that breaks judge is
  caught by the tests, not by the line refusing its own fix. Judge
  still holds no write token ([ADR-0044]).
- **The pin moves in its own pull request**, after a release candidate
  passed a real try (§2), linked in it. Never a moved tag.

### 2. A real try, in one command

- **`scripts/try-on-copy.sh <version> [repo]`**, and a manual
  `try.yml` running it: on [JN0V/TraceGlass-trial], reset to a fixed
  tag, the version's line runs one issue end to end.
- **It asserts forge facts**: a pull request opened, its checks green,
  the labels set; never the code's content.
- **A p1 closes only with the link of a green try.** `## Tried`
  carries it ([ADR-0043]).
- **After V1**: the try runs on each `-rc.N` by itself; a one-off
  fault-injection campaign (a conflict, several objections, a
  cancelled run, a brownout), each case ending as a conformance case
  (principle 7).

### 3. A defect the pinned release carries in a line path

- **The closed list of paths**: develop, answer, apply, merge, bounds.
- **When the pinned release itself carries the defect there**, the
  fix is the person's own pull request: built by a contributor on the
  person's explicit instruction, accepted with their label
  ([ADR-0068 §5], [§7]).
- **Everything else goes through the line**, as before.

### 4. Rollback

- **A broken pinned release**: a one-line pull request reverts the
  pin, on the person's label.
- **Else** [the emergency](../merge-policy-no-bypass.md#the-emergency),
  the person's own. No bypass is added ([ADR-0068 §6]).

## Not chosen

- **The pin alone**: the buggy pinned engine still drives its own fix.
- **The person's pull request alone**: merge and apply still run
  `main`; the person a bottleneck; nothing stops a cascade.
- **Everything from the head**: `pull_request_target` with a write
  token would run the pull request's code ([ADR-0044], "pwn request").
- **A last-known-good `main`, moved by a canary**: kept as a possible
  evolution after V1.

## Consequences

- **A fix reaches the line 1 to 2 h after its merge**, as for a user:
  release, try, pin.
- **Fewer releases**: each one tried before the line takes it.
- **The previous release reads the repository**: the role files
  (`role.yaml`, prompts, facets) ship inside the binary
  ([roles.go](../../roles.go), `go:embed`); the pinned release reads
  only `.workline/config.yaml` and `.workline/roles/` from the
  repository. Those must stay readable by the previous release; a
  change to their format lands in a release before it is used.
- **The try's latency and tokens are paid per release**, not per pull
  request.

## Build order

Before Thursday 2026-11-19:

1. The pin: the line's workflows on `WORKLINE_VERSION`.
2. The manual try: the script and `try.yml`.
3. This record.

After V1: the try on each `-rc.N`; the fault-injection campaign.

[#1638]: https://github.com/JN0V/workline/pull/1638
[#1670]: https://github.com/JN0V/workline/issues/1670
[#1676]: https://github.com/JN0V/workline/issues/1676
[#1678]: https://github.com/JN0V/workline/pull/1678
[#1686]: https://github.com/JN0V/workline/pull/1686
[#1687]: https://github.com/JN0V/workline/issues/1687
[#1701]: https://github.com/JN0V/workline/issues/1701
[#1704]: https://github.com/JN0V/workline/issues/1704
[ADR-0042]: 0042-the-engine-merges-by-the-projects-policy.md
[ADR-0043]: 0043-a-try-is-told-in-its-pull-request.md
[ADR-0044]: 0044-the-apps-key-stays-with-the-default-branchs-runs.md
[ADR-0063]: 0063-every-open-issue-is-due-waiting-or-done.md
[ADR-0068]: 0068-the-line-steps-back-on-its-own-drift.md
[ADR-0068 §5]: 0068-the-line-steps-back-on-its-own-drift.md#5-the-persons-assistant-goes-through-the-line
[§7]: 0068-the-line-steps-back-on-its-own-drift.md#7-a-persons-yes-is-assisted-never-delegated
[ADR-0068 §6]: 0068-the-line-steps-back-on-its-own-drift.md#6-nobody-bypasses-on-main
[JN0V/TraceGlass-trial]: https://github.com/JN0V/TraceGlass-trial
