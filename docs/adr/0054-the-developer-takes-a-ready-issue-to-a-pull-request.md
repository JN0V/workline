# ADR-0054: The developer takes a ready issue to a pull request

- **Status:** proposed
- **Date:** 2026-10-27
- **Decided by:** the maintainer, on 2026-10-27: build the developer now
  ([#117](https://github.com/JN0V/workline/issues/117)); the five
  choices [below](#for-the-person) wait for the maintainer
- **Builds on:** [ADR-0013], [ADR-0018], [ADR-0020], [ADR-0039],
  [ADR-0042], [ADR-0044], [ADR-0050], [ADR-0051]; principles 1, 5, 6, 8,
  12
- **Research:** [issue-to-pull-request.md](../research/issue-to-pull-request.md)

## Context

- **A `workline:ready` issue waits for a person**: no role takes it. The
  line of principle 1 — need, ready, pull request, acceptance — has a hole
  in the middle ([routing.md](../spec/routing.md#work-starts-from-a-clear-need)).
- **The reviewer finds, nobody fixes**: its loop waits for this role
  ([reviewer status](../../roles/reviewer/docs/status.md), Missing 4).
- **What others do** ([research](../research/issue-to-pull-request.md)):
  - a person starts each agent; it never merges nor approves; the best
    designs give the agent no write right, a separate job writes;
  - one issue, one branch, one pull request; a file allowlist;
  - a cap on each session (time, turns, cost), none on review rounds;
  - most rejected agent pull requests were never read (38 %) or
    duplicates (23 %); red CI next (17 %); a wrong change 5 %.

## Decision

### What it takes, and when

- **The first `ready` issue in the backlog's order**
  ([`backlog.Order`](../spec/backlog-acts/next.md#the-order-and-what-waits),
  the product owner's "next"): ready, waiting on no open issue, no
  parts, and no open pull request closing it.
- **One issue a run, one open developer pull request at a time** by
  default (`open-max: 1`): the next waits until a person has accepted,
  sent back or closed the last. Against the two main rejections, unread
  and duplicate.
- **The moment**: its own scheduled run, after the product owner's and
  the spec review's ([ADR-0050] §3, the night column); `workline develop
  --issue <n>` on a person's machine or a manual dispatch for one issue.
  Nobody waits on it: no wait budget, a token and time cap instead.
- **`workline:ready` is the trigger**, the person's act others make by
  assigning or labelling: a person's, or the product owner's under the
  rules the person wrote ([ADR-0018]).

### Where it runs, with which rights

- **CI by default, the same command on a machine** (principle 10).
- **Two jobs, as for every role** (principle 6, [ADR-0044]):
  - the agent's: the default branch's checkout, a branch
    `workline/developer/<issue>`, the agent, the project's gates; no
    forge token, no App key;
  - `apply`'s: the App's key, no AI key; validates the patch against the
    catalogue and the scope, pushes, opens the pull request.
- **The pull request is the App's**, so the project's checks run on it
  (a job's own token starts none) and the reviewer tells it from a
  person's: judged at `judge-at-least: model`.
- **The agent is a choice**: none, `claude`, `cmd:<any command>`. With
  none, the role picks the issue and says so, opens nothing: writing the
  code is judgement, which goes to a person (principle 5).

### What bounds it

- **Scope**: the patch may touch the paths the issue's Scope names, the
  product owner's `sources` for it, and the tests beside them
  ([ADR-0039]: Scope is read by meaning). Any other file is refused at
  `apply`, said in the pull request (principle 8).
- **Never, whatever the Scope**: the files that steer agents (AGENTS.md,
  CLAUDE.md, a page they link, a prompt — the list of [ADR-0042]'s
  `normal`), `.workline/`, the role folders, CI workflows. Asked: a
  refusal, the issue told a person must do it.
- **Caps**: `ai-max-tokens` a run and a wall-clock cap (30 min, as
  gh-aw's agent step and under Copilot's 59). Reached: what was done
  pushed as a Draft, the cap named.
- **No merge intention** in its catalogue: merging stays `follow`'s, by
  the project's policy ([ADR-0042]); its pull requests merge at
  `cautious` only, the `code` class of `normal` decided on [#117](https://github.com/JN0V/workline/issues/117) waiting
  for the criteria ADR-0042 writes.

### The proof, before the pull request leaves Draft

- **The project's gates, as on a merge request** ([gates.md](../spec/gates.md)),
  and the commands the issue's Verification names, run in the agent's
  job on the patch.
- **Green**: the pull request opens ready, `Closes #<n>`, a
  `## Verification` section saying what ran and its result, a `## Tried`
  section ([ADR-0043]).
- **Red after its turns, or a cap reached**: a Draft saying what
  failed; a draft wakes no AI ([ADR-0050] §4). The issue stays `ready`;
  the run after skips it while the Draft is open.

### The review loop

- **The reviewer's findings are its input**: on its own pull request, the
  findings that hold the merge ([ADR-0051]: correctness) and the
  Important ones given a round (edge cases). Nits are shown, not asked.
- **The reviewer decides a finding is fixed**, on its next read of the new
  commits ([ADR-0013]); the developer never resolves a conversation nor
  says "done" in its place.
- **Five rounds, then a person** (the reviewer's own count, its status
  Missing 4): the pull request says the round; past five, `human`.
- **A person's review comment** is a round too, read the same way.
- **The moment**: a job of its own after the reviewer posts, off the
  verdict's path ([ADR-0050] §2).

### What it never does

- Merge, approve, mark its Draft ready by hand, resolve a conversation.
- Take an issue not `ready`, or two at a time over `open-max`.
- Touch a file outside the bounds above, or a gate's definition.
- Close or move an issue: `Closes #<n>` does it at the person's merge.

## For the person

Five choices, each with the recommendation and its alternative:

1. **Trigger**: the schedule takes the first ready issue, one open at a
   time. *Alternative*: a person labels each issue to develop, as Copilot
   and OpenHands do — one more step for the person.
2. **Where**: CI, the agent's job holding no token. *Alternative*: the
   person's machine first, CI later — faster to try, but not the Need
   ("without opening a session").
3. **Scope**: a hard bound at `apply`, from the Scope's paths.
   *Alternative*: free writes, the reviewer's out-of-scope lens judging —
   nothing mechanical (principle 7).
4. **Proof failed**: a Draft saying what failed. *Alternative*: no pull
   request, a comment on the issue — the work lost.
5. **Review loop**: blocking and Important findings, five rounds, the
   reviewer deciding. *Alternative*: every finding, nits included, or two
   rounds — more tokens, or a person sooner.

## Build plan

Each step one pull request, its cases first in
`tests/conformance/cases/`, `pending.txt` listing what does not pass.

1. **The role and its pick**, no agent: `roles/developer/`,
   `workline develop`. Cases: `developer-picks-first-ready`,
   `developer-skips-issue-with-open-pr`, `developer-open-max-waits`,
   `developer-none-ready-says-so`, `developer-no-agent-opens-nothing`.
2. **The patch and its bounds** at `apply`. Cases:
   `developer-refuses-file-outside-scope`,
   `developer-refuses-steering-file`, `developer-refuses-gate-file`,
   `developer-has-no-merge-intention`.
3. **The proof**: gates and the Verification's commands run, the caps.
   Cases: `developer-green-opens-ready`, `developer-red-opens-draft`,
   `developer-cap-reached-pushes-draft`.
4. **The pull request**: the App's, `Closes #<n>`, its sections; the
   agent's job holds no token. Cases: `developer-pr-closes-issue`,
   `developer-agent-env-has-no-token`, `developer-pr-judged-at-model`,
   and on GitLab and a `cmd:` forge.
5. **The schedule**: the templates, GitHub and GitLab. Case:
   `routing-schedule-runs-developer`.
6. **The review loop**. Cases: `developer-fixes-blocking-finding`,
   `developer-skips-nits`, `developer-round-five-hands-to-person`,
   `developer-person-comment-is-a-round`.
7. **Measured**: an evaluation on planted issues (`WORKLINE_EVAL`), then a
   try on a copy of a real repository, told in the pull request.

## Consequences

- The line runs need to pull request with no session opened; a person
  still reads every pull request and accepts it.
- Its tokens run every scheduled run that finds a ready issue: one issue,
  capped.
- A project with no agent gets the pick and nothing else.
- The reviewer's Missing 4 is built by step 6; the role is beta until
  [ADR-0036]'s criteria hold.

## Not chosen

- **The agent pushing itself**: it would hold a write token (principle
  6); Claude's action and gh-aw both keep the push out of its hands.
- **Several pull requests a run**: duplicates and unread pull requests
  are most rejections measured.
- **No cap on rounds**, as the tools studied: each round costs tokens
  and a person's attention; the reviewer's five is already written.

[ADR-0013]: 0013-what-was-judged-is-not-asked-again.md
[ADR-0018]: 0018-the-product-owner.md
[ADR-0020]: 0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md
[ADR-0036]: 0036-a-roles-status-is-earned-by-written-criteria.md
[ADR-0039]: 0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md
[ADR-0042]: 0042-the-engine-merges-by-the-projects-policy.md
[ADR-0043]: 0043-a-try-is-told-in-its-pull-request.md
[ADR-0044]: 0044-the-apps-key-stays-with-the-default-branchs-runs.md
[ADR-0050]: 0050-each-act-has-its-moment-and-a-wait-budget.md
[ADR-0051]: 0051-a-check-sits-where-its-catch-is-worth-its-cost.md
