# ADR-0071: workline's own line runs a pinned release, tried on a copy

- **Status:** proposed, until the maintainer merges it
- **Date:** 2026-11-17
- **Proposed by:** a three-agent panel's synthesis after the day's
  five whys; the maintainer's choices of 2026-11-17 on judge folded in
- **Builds on:** [ADR-0042], [ADR-0043], [ADR-0044], [ADR-0063],
  [ADR-0068] §5 to §7; principles 6, 7, 9, 10, 12
  ([PRINCIPLES.md](../PRINCIPLES.md))
- **Amends, once built:** [the engine action](../../.github/actions/engine/action.yml)
  (main's engine for the line's jobs); [ADR-0063]'s
  [working agreements](0063-every-open-issue-is-due-waiting-or-done.md#working-agreements)
  (a fix is tried on a clone with a fake forge: §2 says how they relate)

## Context

The five whys of 2026-11-17:

- **p1 opens as fast as it closes**: 21 opened, 7 closed in 7 days.
- **The line finds its defects while landing its fixes**: 4 of the
  last 6 p1 were line defects found while landing another p1:
  [#1678] → [#1704], [#1638] → [#1670] and [#1676], [#1686] →
  [#1687] and [#1701].
- **The line runs on itself, from main**: gates, judge and warn run
  the pull request head's engine; every job with a write token builds it from
  `main` ([engine action](../../.github/actions/engine/action.yml)):
  - develop, on `issues`, `issue_comment`, `workflow_dispatch`,
    `schedule` and `workflow_run`;
  - answer and apply on `workflow_run` (apply also on a push to `main`);
  - bounds and merge on `pull_request_target` (merge also on `issues`
    and `schedule`);
  - gardening and sample on `schedule` and `workflow_dispatch`.
  A defect on `main` blocks the pull request that fixes it.
- **Fixes are never tried on GitHub's CI**: `## Tried` says a local
  run, or boilerplate; V1 was accepted on a local forge.

## Decision

**workline's line runs a published release, pinned, as a user's does.
A release candidate moves the pin only after a real try on a GitHub
copy. A defect the pinned release carries in a line path is fixed by
the person's own pull request.**

### 1. The line runs a pinned release

- **Each workflow that runs the engine**, under `.github/workflows/`:

  | Workflow | Write rights | Engine |
  |---|---|---|
  | develop, answer, apply, bounds, merge | yes | pinned release |
  | gardening, sample | yes | pinned release |
  | workline: judge | no | pinned release; head pass apart (below) |
  | workline: gates, on a pull request and a push to `main` | no | pinned release; Go set up for the gate's own tool |
  | warn | no | head: it only warns |
  | test | no | head: `go test`, conformance, simulator |

- **Run no engine**: workline-release, fork, objection,
  release-please, schedule-probe. release.yml builds the release's
  binaries (GoReleaser) and runs none.
- **A pinned job downloads `WORKLINE_VERSION`**, checksum checked, as
  [the templates](../../defaults/ci/github/workline-merge.yml) do.
  No engine is built from `main` for a job with a write token.
- **Gates and judge give the line's verdict with the pinned release**: a
  head that breaks them is caught by the tests, not by the line refusing
  its own fix. Neither holds a write token ([ADR-0044]).
- **An engine-changing pull request** touches what the binary embeds or
  compiles: `cmd/`, `internal/`, `roles/`, `skills/`, `roles.go`,
  `defaults/routing.yaml`, `go.mod`, `go.sum`
  ([roles.go](../../roles.go) and `internal/` hold the `go:embed`).
- **It also gets a head pass**: judge built from its head,
  non-blocking, its result told on the pull request. A new rule, lens
  or prompt is exercised on its own pull request.
- **The pin moves only in its own pull request**, with a green try of
  the release itself (§2) linked in it. Never a moved tag.
- **release-please never moves it**: `.github/workflows/` carries no
  `x-release-please-version` marker, and
  [release-please-config.json](../../release-please-config.json)'s
  `extra-files` list only the templates under `defaults/`. It stays so.

### 2. Two real tries, one command

- **`scripts/try-on-copy.sh <version|ref> [repo]`**, and a manual
  `try.yml` running it: on [JN0V/TraceGlass-trial], reset to a fixed
  tag, the line runs one issue end to end.
- **It asserts forge facts**: a pull request opened, its checks green,
  the labels set; never the code's content.
- **Try of a ref**, a fix's pull request head: the engine built from
  that commit runs the copy's line.
  - **The proof a fix links** under `## Tried` ([ADR-0043]); a p1's
    fix without a green one is not accepted, so its issue stays open.
  - **With [ADR-0063]'s agreement**: a clone with a fake forge stays
    the fast local loop while building; it is no longer the proof. For
    an engine-changing fix, the ref's try on the copy is.
- **Try of a release**: the version as published, downloaded as the
  pin would. The pin's pull request requires it, green, linked.
- **After V1**: the try runs on each `-rc.N` by itself; a one-off
  fault-injection campaign (a conflict, several objections, a
  cancelled run, a brownout), each case ending as a conformance case
  (principle 7).

### 3. A defect the pinned release carries in a line path

- **The closed list of paths**: develop, answer, apply, bounds, merge,
  judge (with gates, workline.yml), gardening and sample. The last two
  are in: they write to the repository with the pinned release.
- **When the pinned release itself carries the defect there**, the
  fix is the person's own pull request: built by a contributor on the
  person's explicit instruction, accepted with their label
  ([ADR-0068 §5], [§7]).
- **A fix merged does not reach the line yet**: it does after a
  release, its green try and the pin's pull request.
- **Everything else goes through the line**, as before.

### 4. Rollback

- **A broken pinned release**, in any path of §3's list: a one-line
  pull request reverts the pin, on the person's label.
- **The rollback itself goes through judge and merge**. When the
  pinned release breaks either, neither the person's fix nor the
  rollback lands: only
  [the emergency](../merge-policy-no-bypass.md#the-emergency) remains,
  the person's own, by hand. No bypass is added ([ADR-0068 §6]).

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
  repository.
- **A format ships in a release before it is used**: the config reader
  ([config.go](../../internal/role/config.go)) refuses an unknown key,
  so a pull request adding a key to `.workline/config.yaml` would be
  refused by the pinned judge.
- **A check holds the rule**: a conformance case or gate runs the
  pinned release on the head's `.workline/config.yaml` and fails when
  it does not parse.
- **The try's latency and tokens are paid per release**, not per pull
  request.

## Build order

Before Thursday 2026-11-19:

1. The pin: the pinned workflows above on `WORKLINE_VERSION`, and the
   config check.
2. Judge's non-blocking head pass on engine-changing pull requests.
3. The manual try, of a ref and of a release: the script and `try.yml`.
4. This record.

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
