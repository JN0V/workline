# A person on the loop — focused research (2026-11-02)

**Question.** A person states a need; the line should do the rest, by
its autonomy setting, never waiting on them for ever. How do others let
a question time out, let a fix through and keep features for a step a
person watches?

How projects keep a person out of the line's way and still in charge, in
their words: *lazy consensus*, *silence is consent*, *assumptions*,
*needs clarification*, *release train*, *release branch*, *release
candidate*, *feature gate*, *needs information*, *not reproducible*.
Read for [ADR-0060](../adr/0060-the-person-is-on-the-loop.md).

## Questions that time out: lazy consensus

| Where | Asking | Silence | Delay | Kept for a vote |
|---|---|---|---|---|
| [Apache, decision making](https://community.apache.org/committers/decisionMaking.html) | the plan stated on a public list | "silence indicates consent": the plan goes on | "usually 72 hours" | a release, a new committer: "official approval for legal or process reasons" |
| [Apache glossary](https://www.apache.org/foundation/glossary.html) | "assumes general consent if no responses are posted within a defined period" (also *lazy approval*) | consent | a period the proposer names; its example: three days | — |

- **An objection needs a reason**, and is open to other ways; then the
  plan goes to a discussion, not straight to a vote.
- **What it is not**: the reporter's silence taken as a yes to close an
  issue, which [product-owner.md](product-owner.md#closing-on-silence-announced-first-2026-10-05)
  rejects. Lazy consensus states the plan first, where people read, and
  goes on with it; nothing is closed or thrown away.
- **Kept for a person, even there**: a release. Apache votes on every
  release.

## An agent's assumptions, written down

[Spec Kit](https://github.com/github/spec-kit), GitHub's spec-driven
toolkit, read 2026-11-02:

- **Its `specify` command**: "Make informed guesses based on context and
  industry standards"; "Use reasonable defaults for unspecified details
  (document assumptions in Assumptions section)".
- **At most 3 `[NEEDS CLARIFICATION]` markers**, kept for a choice that
  changes the scope or the user's experience, has several readings, or no
  sensible default.
- **Its `clarify` command**: at most 5 questions a session; each answer
  logged under `## Clarifications`, dated, and applied to the section it
  changes; what is left is listed as *deferred* (with a reason) or
  *outstanding*.

What to take: an agent goes on with a default, and writes it where the
person reads it, apart from the person's text. Questions are few, and
kept for what changes the result.

## Fixes ride now, features ride a train

| Where | Features | Fixes | A person's step |
|---|---|---|---|
| [Rust's release train](https://doc.rust-lang.org/book/appendix-07-nightly-rust.html) | land on main "behind a feature flag", usable on nightly only; stabilised, they "ride the trains" | on main, then backported to beta | every six weeks `beta` is cut from main; six weeks later `stable` from `beta` |
| [Linux](https://docs.kernel.org/process/2.Process.html) | in the merge window, "approximately two weeks" | only fixes after `-rc1`, for six to ten weeks | an `-rc` "about once a week", usually rc6 to rc9 before the release; regressions are the gauge |
| [git-flow](https://nvie.com/posts/a-successful-git-branching-model/) | merged to `develop` before the release branch is cut; later ones wait | "bug fixes may be applied in this branch"; "adding large new features here is strictly prohibited" | the release branch merged and tagged |
| [Kubernetes feature gates](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/) | an alpha gate defaults to `false`; most beta gates to `true` | — | the gate removed at general availability |

- **A pre-release says so in its version**: [SemVer](https://semver.org/)
  rule 9, `1.0.0-rc.1`, "indicates that the version is unstable", and
  ranks below `1.0.0`.
- **Already in workline's research**: releaser-pleaser keeps `-rc.N` as a
  state of the release pull request; promoting is the person's merge of
  one without it ([release-manager.md](release-manager.md)).
- **Shared**: a feature reaches users only through a step a person
  watches (a beta, an `-rc`, a gate turned on); a fix goes in at once,
  and into the release branch too.

## A bug that cannot be reproduced

| Where | Mark | Who sets it | Without an answer |
|---|---|---|---|
| [Kubernetes labels](https://github.com/kubernetes/test-infra/blob/master/label_sync/labels.md) | `triage/needs-information`; `triage/not-reproducible`: "can not be reproduced as described" | people | the lifecycle robot, on silence ([product-owner.md](product-owner.md#closing-on-silence-announced-first-2026-10-05)) |
| [Go](https://go.dev/wiki/HandlingIssues) | `WaitingForInfo`, beside `NeedsInvestigation` | people | not said |
| [no-response](https://github.com/lee-dohm/no-response) | `responseRequiredLabel` | a person | closed after `daysUntilClose` |

- **Not reproduced is a state of its own**, never "fixed" nor "a
  feature". Nobody found turns it into another kind.
- **workline differs**: it never closes on silence
  ([ADR-0024](../adr/0024-obsolete-is-announced-then-closed-on-silence-and-a-second-judge.md));
  a not reproduced bug is Blocked, with its reason, and read again on
  news ([ADR-0059](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-01-blocked-is-read-again)).

## Pre-releases on a release branch: release-please

Read for [#861](https://github.com/JN0V/workline/issues/861) in
release-please's source (`src/versioning-strategies/prerelease.ts`,
`src/manifest.ts`) and release-please-action's inputs, 2026-11-02:

- **A release branch**: the action's `target-branch` runs the tool on
  any branch: its release pull request goes into that branch, its last
  release found by the manifest's version and that release's tag.
- **`versioning: prerelease`** counts `-rc.N` itself, but from a plain
  version it gives `1.1.0-rc`, no number, unless `prerelease-type`
  carries one; leaving a prerelease for the version needs `Release-As`
  anyway.
- **`Release-As: x.y.z`**, a commit's footer, sets the next version
  whatever the commits say: the tool's own way for a person, or a
  process, to name it.
- **`prerelease: true`** marks a GitHub release a pre-release when its
  version has a prerelease part, **or a major of 0**: on a 0.x project,
  every final version too. Marking the `-rc` release after is safer.
- **The job's own token** opens and merges the tool's pull request, but
  starts no workflow: the tool must run again in the same job to tag
  what it merged.
- **releaser-pleaser** keeps `-rc.N` as a state of its pull request
  ([release-manager.md](release-manager.md)); not wired.

What workline takes: it names each version by a `Release-As:` commit
on `release/vX.Y` — the milestone gives `X.Y` —, the tool tags and
publishes; nothing else of the tool's is replaced.

## What workline takes

- **Lazy consensus for questions**: ask, wait a delay the level sets,
  then go on with the assumption written on the issue. A person
  corrects it later; nothing is closed.
- **Assumptions beside the person's text**, never in it: Spec Kit's
  Assumptions section.
- **The release candidate as the person's step for features**: the
  train's beta and `-rc`, on a versioned branch, as git-flow's release
  branch; fixes need no step.
- **Not reproduced, a block with its reason**: never fixed blind, never
  another kind.
