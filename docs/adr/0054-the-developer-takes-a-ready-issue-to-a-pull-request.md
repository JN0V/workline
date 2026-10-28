# ADR-0054: The developer takes a ready issue to a pull request

- **Status:** proposed
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
- **What others do** ([research](../research/issue-to-pull-request.md)):
  - an issue assigned to the agent starts it (Copilot, GitLab's Duo
    Developer, Claude's action); it never merges nor approves; the best
    designs give the agent no write right, a separate job writes;
  - one issue, one branch, one pull request; a file allowlist;
  - a cap on each session (time, turns, cost), none on review rounds;
  - rejected agent pull requests: 38 % never reviewed, 23 % duplicates,
    17 % red CI, 5 % a wrong change.

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
- **The pull request is the App's**, so the project's checks run on it
  (a job's own token starts none) and the reviewer tells it from a
  person's: judged at `judge-at-least: model`.
- **The agent is a choice**: none, `claude`, `cmd:<any command>`. With
  none, the role names the issue it would take and opens nothing:
  writing the code is judgement (principle 5).

### It takes work by the forge's assignment

- **Once the product owner has set an issue `ready`** ([ADR-0018]), the
  developer assigns itself the first ready, **unassigned** issue in the
  backlog's order
  ([`backlog.Order`](../spec/backlog-acts/next.md#the-order-and-what-waits)):
  waiting on no open issue, no parts, no open pull request closing it.
- **The assignee is the claim**, as on the forges studied: a person who
  assigns themselves keeps the issue from it; the developer never
  unassigns anyone.
- **The moment**: when an issue becomes `ready`, and its own scheduled
  run after the product owner's ([ADR-0050] §3). Nobody waits on it: no
  wait budget, a token and time cap instead.

### It proves locally first

- **Before the pull request opens**, in the agent's job on the patch:
  the project's gates ([gates.md](../spec/gates.md)), its tests, the
  tester's acceptance tests for the issue, and the commands the issue's
  Verification names.
- **Green**: the pull request opens ready, `Closes #<n>`, a
  `## Verification` section saying what ran and its result, a `## Tried`
  section ([ADR-0043]).
- **Red after its turns, or a cap reached**: the work is kept as a
  Draft saying what failed; a Draft wakes no AI ([ADR-0050] §4). The run
  after skips the issue while the Draft is open.

### What bounds it

- **Scope**: the patch may touch the paths the issue's Scope names, the
  product owner's `sources` for it, and the tests beside them
  ([ADR-0039]: Scope is read by meaning). Any other file is refused at
  `apply`, said in the pull request (principle 8).
- **The tester's acceptance tests are read-only to it**: it makes them
  pass, never edits them; a patch touching one is refused at `apply`.
- **Never, whatever the Scope**: the files that steer agents (AGENTS.md,
  CLAUDE.md, a page they link, a prompt — [ADR-0042]'s `normal` list),
  `.workline/`, the role folders, CI workflows, a gate's definition.
  Asked: a refusal, the issue told a person must do it.
- **Caps**: `ai-max-tokens` a run and a wall-clock cap (30 min, as
  gh-aw's agent step and under Copilot's 59). Reached: a Draft, the cap
  named.
- **The code and the tests apart**: separate commits, and the pull
  request lists the code it changed and the tests it added, so the
  tester's later job — adapting tests after a change — reads its part.

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
- **The reviewer decides a finding is fixed**, on its next read
  ([ADR-0013]); the developer never resolves a conversation.

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
- **Several developer pull requests may be open.** The limit, if any, is
  about conflicts: it does not take an issue whose Scope overlaps an
  open developer pull request's files. Never a second pull request for
  one issue.
- **The policy stays the project's**: at `cautious` every pull request
  still waits for a label; the developer's join `normal` under
  [ADR-0042]'s criteria, its governance line said where it is set.

### What it never does

- Merge, approve, turn auto-merge on, put `workline:accepted`, resolve
  a conversation.
- Take an issue not `ready`, or assigned to someone.
- Edit an acceptance test, or touch a file outside the bounds above.
- Close or move an issue: `Closes #<n>` does it at the merge.

## For the person

The maintainer's seven choices, 2026-10-28, each against what was first
proposed:

1. **CI**, the agent's job holding no token. *Not*: the person's machine
   first.
2. **The forge's assignment**: the first ready, unassigned issue; a
   person's assignment keeps it. *Not*: the schedule taking one
   unmarked, nor a person labelling each.
3. **Proven locally first**: gates, tests, Verification before the pull
   request opens.
4. **The same developer fixes the findings**: one identity, a written
   trace, a fresh run each verdict.
5. **On the loop**: several open, a yes by kind only. *Not*: one
   developer pull request open until a person has read it.
6. **Five rounds, the five whys at three**. *Not*: five rounds alone.
7. **The tester first**: its acceptance tests before the code, the
   developer never editing them.

## Build plan

Nothing starts before the tester's first step
([#264](https://github.com/JN0V/workline/issues/264)): the issue's
Validation (Given / When / Then) turned into acceptance tests before the
developer codes. Then each step one pull request, its cases first in
`tests/conformance/cases/`, `pending.txt` listing what does not pass.

1. **The role and its pick**, no agent: `roles/developer/`,
   `workline develop`, the assignment. First check the forge takes the
   App as an assignee (GitLab's service accounts can be; GitHub may
   refuse a bot), and write the fallback if not. Cases:
   `developer-picks-first-ready-unassigned`,
   `developer-skips-assigned-issue`, `developer-skips-issue-with-open-pr`,
   `developer-skips-overlapping-scope`, `developer-none-ready-says-so`,
   `developer-no-agent-opens-nothing`.
2. **The patch and its bounds** at `apply`. Cases:
   `developer-refuses-file-outside-scope`,
   `developer-refuses-acceptance-test-edit`,
   `developer-refuses-steering-file`, `developer-refuses-gate-file`,
   `developer-has-no-merge-intention`.
3. **The proof**: gates, tests, acceptance tests, the Verification's
   commands, the caps. Cases: `developer-green-opens-ready`,
   `developer-red-opens-draft`, `developer-cap-reached-pushes-draft`.
4. **The pull request**: the App's, `Closes #<n>`, its sections, code
   and tests apart, the hidden record; the agent's job holds no token.
   Cases: `developer-pr-closes-issue`, `developer-pr-splits-code-tests`,
   `developer-agent-env-has-no-token`, `developer-pr-judged-at-model`,
   and on GitLab and a `cmd:` forge.
5. **The triggers**: `ready` set and the schedule, GitHub and GitLab
   templates. Case: `routing-schedule-runs-developer`.
6. **The review loop**: a fresh run per verdict, from the trace; needs
   the five whys skill. Cases: `developer-fixes-blocking-finding`,
   `developer-skips-nits`, `developer-reads-trace-not-memory`,
   `developer-round-three-five-whys`,
   `developer-round-five-hands-to-person`,
   `developer-person-comment-is-a-round`.
7. **The merge**: its pull requests in [ADR-0042]'s `normal`, a yes by
   kind. Cases: `merge-normal-admits-developer-fix`,
   `merge-normal-holds-developer-feature`.
8. **Measured**: an evaluation on planted issues (`WORKLINE_EVAL`), then
   a try on a copy of a real repository, told in the pull request.

## Consequences

- The line runs from need to merge with no session opened; a person
  answers what the policy asks by kind, and reads the rest after.
- Its tokens run each time an issue becomes ready and on each reviewer
  verdict on its pull requests: one issue a run, capped.
- A project with no agent gets the pick and nothing else.
- The developer waits on the tester: nothing ships before its first step.
- The reviewer's Missing 4 is built by step 6; the role is beta until
  [ADR-0036]'s criteria hold.

## Not chosen

- **One developer pull request open at a time**: it waits on a person's
  reading, not on the work. Duplicates are kept off by one pull request
  an issue, conflicts by the Scope overlap; the rejection figures stay
  in the [research](../research/issue-to-pull-request.md).
- **A live developer waiting for the review**: a process held for hours
  costs and dies; the trace on the forge is what continues.
- **The agent pushing itself**: it would hold a write token (principle
  6); Claude's action and gh-aw both keep the push out of its hands.
- **No cap on rounds**, as the tools studied: each round costs tokens;
  past five, a person.
- **The developer writing its own acceptance tests**: the check and the
  checked would be one hand.

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
