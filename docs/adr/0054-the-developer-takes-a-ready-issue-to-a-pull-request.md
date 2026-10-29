# ADR-0054: The developer takes a ready issue to a pull request

- **Status:** proposed; steps 1 to 4 built, the claim a label
  ([settled while building](#settled-while-building))
- **Date:** 2026-10-27; rewritten 2026-10-28 to the maintainer's choices
- **Decided by:** the maintainer: build the developer
  ([#117](https://github.com/JN0V/workline/issues/117), 2026-10-27); its
  seven choices (2026-10-28), [below](#for-the-person)
- **Depends on:** the tester's first step
  ([#264](https://github.com/JN0V/workline/issues/264)), built first
- **Builds on:** [ADR-0013], [ADR-0018], [ADR-0020], [ADR-0038],
  [ADR-0039], [ADR-0042], [ADR-0044], [ADR-0050], [ADR-0051]; principles
  1, 5, 6, 8, 12
- **Research:** [issue-to-pull-request.md](../research/issue-to-pull-request.md)

## Context

- **A `workline:ready` issue waits for a person**: no role takes it. The
  line of principle 1 — need, ready, pull request, acceptance — has a hole
  in the middle ([routing.md](../spec/routing.md#work-starts-from-a-clear-need)).
- **The reviewer finds, nobody fixes**: its loop waits for this role
  ([reviewer status](../../roles/reviewer/docs/status.md), Missing 4).
- **Elsewhere**, an issue assigned to the agent starts it; a separate
  job writes; sessions are capped, rounds not.

## Decision

### It runs in CI

- **CI by default, the same command on a machine** (principle 10):
  `workline develop --issue <n>`, or a manual dispatch for one issue.
- **Two jobs, as for every role** (principle 6, [ADR-0044]):
  - the agent's: the default branch's checkout, a branch
    `workline/developer/<issue>`, the agent, the project's gates and
    tests; no forge token, no App key;
  - `apply`'s: the App's key, no AI key; validates the patch against the
    catalogue and the bounds, pushes, assigns, opens the pull request.
- **The pull request is the App's**: the project's checks run on it, and
  the reviewer judges it at `judge-at-least: model`.
- **The agent is a choice**: none, `claude`, `cmd:<any command>`; with
  none, it names the issue it would take, opens nothing (principle 5).

### It takes work by the forge's assignment

- **Once the product owner has set an issue `ready`** ([ADR-0018]), the
  developer assigns itself the first ready, **unassigned** issue in the
  backlog's order
  ([`backlog.Order`](../spec/backlog-acts/next.md#the-order-and-what-waits)):
  waiting on no open issue, no parts, no open pull request closing it.
- **The assignee is the claim**: a person assigning themselves keeps the
  issue from it; it never unassigns anyone.
- **It starts from the tester's Draft** (amended 2026-10-28,
  [ADR-0055] step 4): its branch `workline/tester/issue-<n>`, pushed on
  top; the Draft says `Refs`, so it is no pull request closing the
  issue. Where the tester is routed, a ready issue with no Draft waits
  for it ([the hand-over](../../roles/tester/docs/hand-over.md)).
- **The moment**: an issue set `ready`, and a scheduled run after the
  product owner's ([ADR-0050] §3); no wait budget, a cap instead.

### It proves locally first

- **Before the pull request opens**, in the agent's job on the patch:
  the project's gates ([gates.md](../spec/gates.md)), its tests, the
  tester's acceptance tests for the issue, and the commands the issue's
  Verification names.
- **Green**: the pull request opens ready, `Closes #<n>`, a
  `## Verification` section saying what ran and its result, a `## Tried`
  section ([ADR-0043]).
- **Red, or a cap reached**: a Draft saying what failed, which wakes no
  AI ([ADR-0050] §4); the issue is skipped while it is open.

### What bounds it

- **Scope**: the patch may touch the paths the issue's Scope names, the
  product owner's `sources` for it, and the tests beside them
  ([ADR-0039]: Scope is read by meaning). Any other file is refused at
  `apply`, said in the pull request (principle 8).
- **The tester's acceptance tests are read-only to it**: a patch
  touching one is refused at `apply`. One that does not fit, it objects
  to on the test's line (`workline:objection`); the tester keeps or
  rewrites it, three exchanges, then a person
  ([ADR-0055](https://github.com/JN0V/workline/pull/648)).
- **Never, whatever the Scope**: the files that steer agents (AGENTS.md,
  CLAUDE.md, a page they link, a prompt — [ADR-0042]'s `normal` list),
  `.workline/`, the role folders, CI workflows, a gate's definition.
  Asked: a refusal, the issue told a person must do it.
- **Caps**: `ai-max-tokens` and 30 min a run. **Code and tests apart**:
  separate commits, both listed, for the tester's adapting job.

### The same developer fixes the review's findings

- **"Same" is an identity and a written trace, not a live process**: the
  App, the branch, the pull request, the issue, the reviewer's findings,
  and a hidden record in its comment on the pull request, as the product
  owner's ([ADR-0038], [#637](https://github.com/JN0V/workline/pull/637)).
- **Each reviewer verdict on its pull request starts a fresh developer
  run** from that trace, off the verdict's path ([ADR-0050] §2).
  "Waiting" is a state of the pull request, never of the developer.
- **Its input**: the findings that hold the merge ([ADR-0051]:
  correctness) and the Important ones given a round; nits shown, not
  asked. A person's review comment is read the same way.
- **The reviewer decides a finding is fixed** ([ADR-0013]).

### Review rounds: five, the five whys at three

- **Up to five rounds**, the pull request saying which.
- **A finding still there at round three**: the developer runs the five
  whys — the shared skill any role loads
  ([role-adapting.md](../spec/role-adapting.md#defects-become-guards),
  [backlog-parked.md](../backlog-parked.md)) — writes them in the pull
  request, and fixes the cause, not the symptom.
- **Past five**: the pull request goes to a person (`human`).

### A person on the loop, not in it

- **No limit that waits on a person.** A pull request that passes every
  gate — checks green, the reviewer's verdict passed, no finding open —
  merges by the project's written merge policy, through the forge's
  auto-merge ([ADR-0042]).
- **A person's yes is asked by kind only** ([ADR-0039]: the kind is the
  forge's): a new user-facing feature (UI, UX) by default. A task or a
  fix never waits for one.
- **Several developer pull requests may be open**, limited by conflicts
  only: no issue whose Scope overlaps an open one's files, never two
  pull requests for one issue.
- **The policy stays the project's**: at `cautious` all wait for a
  label; `normal` takes the developer's under [ADR-0042]'s criteria.

### What it never does

- Merge, approve, turn auto-merge on, put `workline:accepted`, resolve a
  conversation, close or move an issue (`Closes #<n>` does it).
- Take an issue not `ready` or assigned; edit an acceptance test.

## For the person

The maintainer chose the headings above on 2026-10-28, against the first
draft (an unmarked pick, one pull request open until read, no five whys,
no tester first).

## Build plan

Nothing starts before the tester's first step
([#264](https://github.com/JN0V/workline/issues/264)): the issue's
Validation (Given / When / Then) turned into acceptance tests before the
developer codes. Then one pull request a step, its cases first in
`tests/conformance/cases/` (names prefixed `developer-`), `pending.txt`
listing what does not pass.

1. **The pick**, no agent: `roles/developer/`, `workline develop`, the
   assignment; first check the forge takes the App as an assignee
   (GitLab's service accounts can be, GitHub may refuse a bot), else a
   fallback. Cases: `picks-first-ready-unassigned`,
   `skips-assigned-issue`, `skips-issue-with-open-pr`,
   `skips-overlapping-scope`, `none-ready-says-so`,
   `no-agent-opens-nothing`.
2. **The bounds** at `apply`: `refuses-file-outside-scope`,
   `refuses-acceptance-test-edit`, `refuses-steering-file`,
   `refuses-gate-file`, `has-no-merge-intention`, `objects-to-a-test`.
3. **The proof** and the caps: `green-opens-ready`, `red-opens-draft`,
   `cap-reached-pushes-draft`.
4. **The pull request**: the App's, its sections, code and tests apart,
   the hidden record, no token in the agent's job: `pr-closes-issue`,
   `pr-splits-code-tests`, `agent-env-has-no-token`,
   `pr-judged-at-model`; on GitLab and a `cmd:` forge.
5. **The triggers**, `ready` and the schedule, both forges' templates:
   `routing-schedule-runs-developer`.
6. **The review loop**, a fresh run a verdict, with the five whys skill:
   `fixes-blocking-finding`, `skips-nits`, `reads-trace-not-memory`,
   `round-three-five-whys`, `round-five-hands-to-person`,
   `person-comment-is-a-round`.
7. **The merge** in [ADR-0042]'s `normal`, a yes by kind:
   `merge-normal-admits-developer-fix`,
   `merge-normal-holds-developer-feature`.
8. **Measured**: an evaluation on planted issues (`WORKLINE_EVAL`), then
   a try on a copy of a real repository, told in its pull request.

## Settled while building

Step 1, the pick (2026-10-29,
[the pick](../../roles/developer/docs/pick.md)):

- **GitHub does not assign an issue to an App**: tried with workline's
  App on its own repository, the REST API answered 403 for the
  `[bot]` login, and GraphQL said "Assigning agents is not supported
  with GitHub App installation tokens". **The claim is a label**,
  `claim-label` (`workline:developing`), on every forge, with a comment
  holding the pick's record. A person's assignment still keeps an issue
  from it. GitLab's assignment of a service account: later, in Missing.
- **"Overlaps an open one's files"**, before any code: the paths in
  backticks of the candidate's Scope, under or over those of the issues
  the developer holds (its label on them).
- **With no agent**, nothing is claimed: a claim nobody codes would keep
  the issue from people.
- **The cases** sit in `tests/conformance/cases/developer/`, as the
  tester's in their folder.

Step 2, the bounds (2026-10-29,
[the bounds](../../roles/developer/docs/bounds.md)):

- **Refused in post, with no agent**: a refused patch is left out, the
  rest applied; when the bounds cannot be read, every patch.
- **The steering test** is the routine merge's ([ADR-0042]), shared;
  the frozen tests, the tester's record. The rules and why: the page.

Step 3, the proof and the caps (2026-10-29,
[the proof](../../roles/developer/docs/proof.md)):

- **The tests are the tester's `test-command`**, run once on the code
  laid on the commit read, in post, no agent: one setting for both
  roles. The gates and the Verification's commands: later, in Missing.
- **The pull request is the engine's**, as the tester's Draft: a
  branch `workline/developer/<n>`, ready or a Draft, `Closes #<n>` and
  a `## Verification`; its other sections are step 4's.
- **Two time caps**: `ai-max-seconds` (1200) on the agent's wait,
  `proof-seconds` (600) on the tests, 30 min in all; past either, nothing waited for
  or a Draft. A cap reached makes a Draft even when green.

Step 4, the pull request (2026-10-29,
[the pull request](../../roles/developer/docs/pull-request.md)):

- **On the tester's Draft, to the default branch**: the engine builds
  the branch from the Draft's commit (`from` in the role's
  out/merge-request.yaml); its pull request's base is the branch the
  run is on. The agent is shown the frozen tests in its question.
- **Code and tests apart** by name: a file named `test` or `spec`, or in
  a test folder, is a test (`commits` in out/merge-request.yaml).
- **No token in the agent's job**: the engine takes the forge's tokens,
  the App's key and the signing key out of every agent's environment,
  whatever the role; the job split is step 5's.
- **Judged at model** by the reviewer's `judge-at-least-by-role`, read
  from the commits' `Workline-Role` trailer, not the branch's name.
- **The auditor's draw** of its code waits: its record is on the pull
  request.

## Consequences

- Need to merge with no session opened; a person answers by kind only.
- Tokens on each `ready` issue and each verdict on its pull requests,
  one issue a run, capped. With no agent, the pick alone.
- Nothing before the tester's first step; the reviewer's Missing 4 built
  by step 6; beta until [ADR-0036]'s criteria hold.

## Not chosen

- **One developer pull request open at a time**: it waits on a person's
  reading, not the work; duplicates and conflicts have their own bounds.
- **A live developer waiting for the review**: a process held for hours
  costs and dies; the trace on the forge continues.
- **The agent pushing itself**: a write token in its hands (principle 6).
- **No cap on rounds**, as the tools studied; **its own acceptance
  tests**, or editing the tester's: the check and the checked one hand.

[ADR-0013]: 0013-what-was-judged-is-not-asked-again.md
[ADR-0018]: 0018-the-product-owner.md
[ADR-0020]: 0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md
[ADR-0036]: 0036-a-roles-status-is-earned-by-written-criteria.md
[ADR-0038]: 0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md
[ADR-0039]: 0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md
[ADR-0042]: 0042-the-engine-merges-by-the-projects-policy.md
[ADR-0043]: 0043-a-try-is-told-in-its-pull-request.md
[ADR-0044]: 0044-the-apps-key-stays-with-the-default-branchs-runs.md
[ADR-0050]: 0050-each-act-has-its-moment-and-a-wait-budget.md
[ADR-0051]: 0051-a-check-sits-where-its-catch-is-worth-its-cost.md
[ADR-0055]: 0055-the-tester-writes-the-acceptance-tests-before-the-code.md
