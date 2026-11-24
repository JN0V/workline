# ADR-0071: workline's own line runs a pinned release, tried on a copy

- **Status:** proposed, until the maintainer merges it
- **Date:** 2026-11-17
- **Proposed by:** a three-agent panel after the day's five whys; the
  maintainer's choices of 2026-11-17 folded in
- **Builds on:** [ADR-0042], [ADR-0043], [ADR-0044], [ADR-0063],
  [ADR-0068] §5 to §7; principles 6, 7, 9, 10, 12
  ([PRINCIPLES.md](../PRINCIPLES.md))
- **Amends, once built:** [the engine action](../../.github/actions/engine/action.yml)
  (main's engine for the line's jobs); [ADR-0063]'s
  [working agreements](0063-every-open-issue-is-due-waiting-or-done.md#working-agreements)
  (a fix is tried on a clone with a fake forge: §2 says how they relate);
  [workline.yml](../../.github/workflows/workline.yml)'s header (judge
  "never runs the pin that commit ships", [ADR-0017]: §1 says why it
  now may); [AGENTS.md](../../AGENTS.md)'s "Try it for real"

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
  develop, answer, apply, bounds, merge, gardening and sample, on
  `workflow_run`, `pull_request_target`, `issues`, `schedule` and the
  like. A defect on `main` blocks the pull request that fixes it.
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
  | develop, answer, apply, bounds, merge, gardening, sample | yes | pinned release |
  | workline: judge | no write but `security-events` | pinned release |
  | workline: gates, on a pull request and a push to `main` | no | pinned release; Go set up for the gate's own tool |
  | warn | no | head: it only warns |
  | test | no | head: `go test`, conformance, simulator |

- **Run no engine**: workline-release, fork, objection, release-please,
  schedule-probe, release.yml (GoReleaser builds the binaries).
- **A pinned job downloads `WORKLINE_VERSION`**, checksum checked, as
  [the templates](../../defaults/ci/github/workline-merge.yml) do.
  No engine is built from `main` for a job with a write token.
- **Gates and judge give the line's verdict with the pinned release**: a
  head that breaks them is caught by the tests, not by the line refusing
  its own fix. Neither writes to the repository: judge holds only
  `security-events: write`, for code scanning ([ADR-0044]).
- **An engine-changing pull request** touches what the binary embeds or
  compiles: `cmd/`, `internal/`, `roles/`, `skills/`, `roles.go`,
  `defaults/routing.yaml`, `go.mod`, `go.sum`
  ([workline.go](../../workline.go), once `roles.go`, and `internal/` hold the `go:embed`).
- **It also got a head pass**, dropped on 2026-11-23
  ([amendment](#amendment-2026-11-23-the-head-pass-is-dropped)): judge and gates built from its head,
  non-blocking, their results told on the pull request. A new rule,
  lens, prompt, gate or `route main` change is exercised on its own
  pull request.
- **Crossed versions**: apply (pinned, from `main`) reads artifacts
  warn built from the head; judge and gates on `pull_request` read
  `WORKLINE_VERSION` from the head's workflow file, so the pin's pull
  request is judged by the new release, applied by the old. Both hold
  by the N-1 rule (Consequences). [ADR-0017]'s "never the pin that
  commit ships" guarded an untagged pin; this one is published, tried.
- **The pin moves only in its own pull request**, never a moved tag:
  accepted with the person's label, never by the merge policy alone
  ([ADR-0042]); its `## Tried` links the copy's merged upgrade pull
  request and the round's merged pull request
  ([amendment 2026-11-24](#amendment-2026-11-24-a-try-is-a-script-on-the-permanent-copy)); the gate on a `try.yml` run
  waits for 1.0.
- **release-please never moves it**: `.github/workflows/` carries no
  `x-release-please-version` marker, and
  [release-please-config.json](../../release-please-config.json)'s
  `extra-files` list only the templates under `defaults/` and
  [without-template.md](../gitlab/without-template.md). It stays so.

### 2. Two real tries, one command

Until 1.0, a try is [the script on the permanent copy](#amendment-2026-11-24-a-try-is-a-script-on-the-permanent-copy);
what follows comes after 1.0.

- **`scripts/try-on-copy.sh <version|ref> [repo]`**, and a manual
  `try.yml` running it: the line runs one issue end to end on a scratch
  repository made afresh from [JN0V/TraceGlass-trial] at a fixed tag.
  The copy itself is never reset: it stays as it is until the next
  round ([ADR-0063]'s working agreement).
- **One try at a time**: `try.yml`'s concurrency group queues them;
  the script refuses while a try runs.
- **It asserts forge facts**: a pull request opened, its checks green,
  the labels set; never the code's content.
- **Try of a ref**, a pull request head: the scratch repository gets
  the engine built from that commit and that commit's templates
  (`defaults/ci/github/`), not only the engine.
- **One rule**: every engine-changing pull request, and every fix to a
  line path (`.github/workflows/`, `.github/actions/`, the templates),
  links a green ref try under `## Tried` ([ADR-0043]); without it, it
  is not accepted, and its issue stays open.
- **A fix to `.github/workflows/*`** carries the same change to its
  template in the same pull request: the try runs the template. What
  only workline's own file holds (the pin step, the head pass) is
  proven by its run on that pull request, or said plainly untried
  under `## Tried`. A clone with a fake forge ([ADR-0063]) stays the
  fast local loop while building; it is no longer the proof.
- **Try of a release**: the version as published, downloaded as the
  pin would; the pin's pull request links it (§1).
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
- **A fix merged reaches the line after a release, its green try and
  the pin's pull request.** Everything else goes through the line.

### 4. Rollback

- **A broken pinned release**, in any path of §3's list: a one-line
  pull request reverts the pin, on the person's label.
- **The rollback itself goes through judge and merge**. When the
  pinned release breaks either, nothing lands: only
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
  release, try, pin. Fewer releases, each tried before the line takes it.
- **The N-1 rule, a format ships in a release before it is used**:
  role files ship inside the binary ([workline.go](../../workline.go), once `roles.go`); for
  every format the engine reads from the repository or a run
  (`.workline/config.yaml`, `.workline/roles/`, the docs' front
  matter, the artifacts warn and judge hand to apply), a release reads
  what itself and the previous one write. The config reader
  ([config.go](../../internal/role/config.go)) refuses an unknown key.
- **A check holds the rule**: a conformance case or gate runs the
  pinned release on the head's `.workline/` and docs, and on the
  head's artifacts, and fails when they do not parse.
- **A release's try is paid once**; a ref try per fix to a line path.

## Amendment 2026-11-23: the head pass is dropped

- **What it was**: the non-blocking jobs `head-gates` and `head-propose`
  of workline's own `workline.yml`, gates and propose on the engine built
  from an engine-changing head (§1). Users' templates never had them.
- **Why it goes**, by an independent analysis of the week to 2026-11-23
  ([#2096](https://github.com/JN0V/workline/issues/2096)):
  - about 71 runner-hours a week;
  - 5 timeouts that cancelled green runs, leaving 3 pull requests with
    no verdict;
  - 9 runs of 531 diverged from the pinned pass, none shown to be a real
    catch (the one still on an open pull request ended at the job's 45
    minute stop, not on a finding);
  - the try on a copy (§2) already runs a new engine on real
    repositories.
- **What stays**: the pinned release's verdict, `warn` and `test` on the
  head's engine, and the N-1 check that the pinned release reads the head.

## Amendment 2026-11-24: a try is a script on the permanent copy

The maintainer's decision of 2026-11-24, on an independent analysis.
It replaces §2 until 1.0, and §1's gate on a `try.yml` run.

- **A try of a release proves three facts**, on the permanent trial copy
  [JN0V/TraceGlass-trial], never a scratch repository:
  1. the release downloads, its checksum matches, and `workline doctor`
     is clean;
  2. the copy's upgrade pull request, the copy running the previous
     version, is green and mergeable under its existing ruleset;
  3. after its merge, one canned real issue goes through forge events to
     a pull request merged by policy, with the copy's real agent.
- **One local command**, under the person's own `gh` session:
  [`scripts/try-release.sh <version>`](../../scripts/try-release.sh).
  It opens or updates the copy's upgrade pull request, waits for its
  checks, and says the person labels it: it never labels. It then
  opens the canned issue and waits for the merge, and prints green or
  red with the links.
  - **One try at a time**: a lock tag on the copy; a second run
    refuses.
  - **`--dry-run`** prints the plan and calls no forge.
- **The copy moves with each try**: its upgrade pull request merges,
  and the canned issue's fix merges. It no longer stays as it was until
  the next round ([ADR-0063]'s working agreement holds for a fix's local
  run, not for a pin's try).
- **The pin's pull request** links two merged pull requests under
  `## Tried` ([ADR-0043]): the copy's upgrade pull request and the
  canned issue's pull request, the round's. It needs no `try.yml` run.
- **A fix to a role** still runs that role locally, once, on a clone
  of the copy, with a fake forge or local pushes ([ADR-0063]).
- **After 1.0**: scratch repositories, the manual `try.yml`, the try of
  a ref (a pull request head) on a forge, and §2's rule that a fix to a
  line path links a green ref try.
- **Not chosen**: a CI workflow holding a token that may create
  repositories, and scratch repositories to make and delete each try.

## Build order

Before Thursday 2026-11-19:

1. The pin: the pinned workflows above on `WORKLINE_VERSION`, and the
   config check.
2. Judge's and gates' non-blocking head pass on engine-changing pull
   requests; the gate on the pin's pull request.
3. The manual try, of a ref and of a release: the script and `try.yml`.
   Until 1.0: `scripts/try-release.sh` alone ([amendment 2026-11-24](#amendment-2026-11-24-a-try-is-a-script-on-the-permanent-copy)).
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
[ADR-0017]: 0017-the-release-manager.md
[ADR-0042]: 0042-the-engine-merges-by-the-projects-policy.md
[ADR-0043]: 0043-a-try-is-told-in-its-pull-request.md
[ADR-0044]: 0044-the-apps-key-stays-with-the-default-branchs-runs.md
[ADR-0063]: 0063-every-open-issue-is-due-waiting-or-done.md
[ADR-0068]: 0068-the-line-steps-back-on-its-own-drift.md
[ADR-0068 §5]: 0068-the-line-steps-back-on-its-own-drift.md#5-the-persons-assistant-goes-through-the-line
[§7]: 0068-the-line-steps-back-on-its-own-drift.md#7-a-persons-yes-is-assisted-never-delegated
[ADR-0068 §6]: 0068-the-line-steps-back-on-its-own-drift.md#6-nobody-bypasses-on-main
[JN0V/TraceGlass-trial]: https://github.com/JN0V/TraceGlass-trial
