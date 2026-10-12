# ADR-0042: The engine merges by the project's written policy, through the forge's auto-merge

- **Status:** accepted, "For the person" as proposed; `cautious` built by
  [#321] ([page](../merge-policy.md), [below](#built-2026-10-12-321)),
  `normal` parked in [#322]
- **Date:** 2026-10-12
- **Decided by:** the maintainer, its direction (2026-10-12), after four
  agents argued role or not
  ([research](../research/merging.md#role-or-not-four-agents-then-the-person))
- **Completes:** [ADR-0041], which left updating a branch behind but
  clean, just before it merges, to the decision on who merges: this one
- **Builds on:** [ADR-0017] (merging a release pull request is the
  decision to release), [ADR-0026] (a level; an act undone demotes),
  [ADR-0034] (`follow` holds the write token, calls no agent), [ADR-0036]
  (a status earned by written criteria), [ADR-0038] (a person's yes is a
  forge label; silence is never a yes); principles 1, 4, 6, 12, 13
- **Research:** [merging.md](../research/merging.md)

## Context

- **The person's merge is mostly a click.** In 19 days, 142 of 191 merges
  (74%) needed nothing else; since the ruleset settled on 10-04, 93 of 120
  (78%), and 27 more only an Update branch. No comment, approval or merge
  message edited.
- **What the person adds is order and timing**: 38 Update branch clicks,
  and the release order kept by hand in 9 of 11 releases — the
  documentalist's release docs first, then the release pull request
  updated, then merged.
- **What the click did not catch**: two defects no check saw; an
  Important finding merged unanswered ([#260]). The four wrong merges
  were order and timing, all before 10-04: what a machine keeps better.
- **The forge already merges.** `main` requires `test` and `judge` green
  and up to date, squash only; auto-merge is allowed. A user's repository
  has no merge queue: a pull request behind `main` waits for a click.
- **workline's texts say it never merges**: principle 1,
  [concepts.md](../concepts.md#people), [ADR-0011], [ADR-0020].

## Decision

**No new role. `workline follow` turns the forge's own auto-merge on for
the pull requests a written project policy admits, and keeps them up to
date with their base, one at a time.** The forge merges, checking its own
rules; the engine decides which, from the verdicts, with no agent
(principle 6). An agent never merges, never turns auto-merge on, and never
puts the label.

### The policy: a project key

- **`merge: {autonomy: cautious}`** in `.workline/config.yaml`, read by
  the engine as `release:` is. No `merge:`, nothing is touched.
- **The levels are the engine's**, named as the product owner's:
  `cautious`, later `normal`. Not a role's `levels`: no role runs here.
- **On GitHub, the App's token**: a merge made with `GITHUB_TOKEN` starts
  no workflow on `main` — no `follow`, no release. Without the App, said
  in the job's summary, and nothing is turned on (principle 12).

### `cautious`: a person's yes, the forge's merge

- **Admitted**: a pull request labelled `workline:accepted` by someone who
  can write ([ADR-0038]'s "who counts"), its head's verdicts passed:
  - the `judge` check green on the head;
  - an agent really answered where the project sets one: no pass made
    with no agent, no `human`, no `blocked-external`;
  - no lens "not reviewed whole".
- **A label, not auto-merge by hand**: a person puts it in bulk, from a
  saved filter ([ADR-0038]); the engine checks the verdicts first. A pull
  request whose auto-merge a person turned on joins the queue too.
- **When**: the label put, the head's checks finished, a push to the
  base. Each act is said in the job's summary ([ADR-0035]).
- **Up to date, one at a time.** On each push to the base, `follow`
  updates the first admitted pull request that is behind: the base merged
  in, as Update branch does, workline's lines settled as in [ADR-0041]
  (its own branches rebuilt, [ADR-0034]). Its checks run, it merges, the
  base moves, and the next one is updated. Not all at once: each update
  runs a pull request's checks, and only one merges before the rest are
  behind again.
- **The order**: the documentalist's release fix ([ADR-0034]) before the
  release pull request; then the order the labels were put.
- **Out of the queue**: a red check, a conflict on a person's line, a
  push refused — said in the job's summary, and the next one is updated.
- **The veto is the forge's**: a Draft never merges, and `follow` turns
  nothing on for it; auto-merge turned off by a person stays off until
  the label is put again; the label taken off turns auto-merge off.

### Never without a person's yes, at any level

- **A release pull request**: merging it is the decision to release
  ([ADR-0017]), and a published tag cannot be taken back. Labelled, it
  merges once its docs are in: the person chose when.
- **A fork's**: admitted only labelled, and never updated: the token
  cannot write to it.
- **A person's pull request**, or an agent's, with no label.

### `normal`, later ([#322])

- **Also with no label**: workline's own routine pull requests — records
  only, derived blocks, `follow`'s rebuilds; then an agent's, told from
  a person's ([#117]), green, no Important finding open.
- **Before it is offered**: a record of each merge made alone; a revert
  of one sends that kind back to a person's yes ([ADR-0026]); written
  criteria for offering it, as [ADR-0036] wrote a role's status.
- **The governance line**: from `normal`, a change an AI wrote and an AI
  reviewed merges with no person's approval. That crosses the two-person
  rule of change control: SLSA's source level 4 asks two trusted
  persons, each a human, on the final revision; SOC 2's change management
  asks authorised people. A level a project chooses, said where it is
  set; never a default.

### Why not a role

- **Nothing to judge**: whether a pull request may merge is rules over
  verdicts given already. A role brings judgement, not screens the forge
  has (principle 13); mechanical work goes to scripts (principle 4); a
  role must beat the tool it replaces ([ADR-0017]); every shipped role has
  an agent. Acts with no agent are engine commands: `follow`, `sample`.
- **Not the product owner**: it works on issues, at night; acceptance is
  the person's ([ADR-0018]); comparing a change with its issue is the
  reviewer's `intent` lens.
- **Not the committer**: judging a message is not accepting a change
  (principle 2). **Not the documentalist**: it would approve its own pull
  requests ([ADR-0005]). **Not the reviewer**: it never approves
  ([ADR-0020]).

### For the person (settled as proposed, 2026-10-12)

1. **A push after the yes.** The label covers the head it was put on, as
   approvals reset on a push elsewhere. Proposed: any commit but
   `follow`'s own update turns auto-merge off and takes the label off,
   said in one comment ([#73] merged while an agent still pushed). Or
   kept, as GitHub keeps auto-merge after a writer's push.
2. **A release pull request labelled**: merged once its docs are in
   (proposed); or never auto-merged, the click kept.
3. **The key**: `merge: {autonomy: …}`, a project key (proposed); or a
   role's `levels` opened to project-wide settings, for one key.

### Not chosen

- **A merge role**, or the product owner merging: above.
- **The engine calling the merge itself**: the forge's auto-merge goes on
  checking the forge's rules, and its veto stays the forge's.
- **A hold label**: Draft and turning auto-merge off exist on both forges.
- **Updating every admitted pull request at each push**: each one's
  checks run again, for one merge.
- **Mergify, Kodiak**: one more app holding the write token, beside
  `follow`, which holds it already. **An organisation, for GitHub's merge
  queue**: the module path moves
  ([merge-conflicts.md](../research/merge-conflicts.md#options)).

## Consequences

- **The person** puts a label, in bulk; no Update branch, no release
  order kept by hand. The release's timing stays theirs.
- **To build** ([#321]): forge methods to turn auto-merge on and off,
  read its state, whether a pull request is behind, its checks; each
  head's verdicts where `follow` reads them, today only a job's artifact.
- **Here, the label is the person's by rule only**: agents act with the
  person's account until they have their own ([#117]).
- **Lost**: the person's override of a false red, 6 early merges over a
  red `judge`. The queue stops instead: a safe failure.
- **No agent call**; on GitHub, one GraphQL call for each pull request
  turned on.
- **Texts it changes, once built**, left as they are in this proposal:
  they say what workline does today, true until the build ships; a
  "proposed" pointer in a principle would tell readers of what is not
  decided. The build's pull request changes them, as docs follow code:
  - principle 1 holds at `cautious`, the label being the acceptance; from
    `normal`: people accept the result, or write in advance the narrow
    class the line accepts for them;
  - [concepts.md](../concepts.md#people): "workline never merges, never
    approves" becomes "never approves; turns the forge's auto-merge on
    by the project's `merge:` policy";
  - [ADR-0011]: an agent never merges, never turns auto-merge on — kept;
    the engine does, by the project's policy;
  - [ADR-0020]: "the person merges" becomes "the person accepts, the
    forge merges";
  - [AGENTS.md](../../AGENTS.md): the "never merge" lines kept; an agent
    never puts `workline:accepted`.

## Built (2026-10-12, #321)

`cautious`, as decided; what the build settled
([page](../merge-policy.md)):

- **Each head's verdict, where `follow` reads it**: the judging job
  writes it (`route --verdict`); the job holding the write token keeps it
  as a git note on the head (`follow --verdict`,
  `refs/notes/workline/verdicts`), as Gerrit keeps its reviews. A later
  run — a label, a push to the base — reads it with the token it holds:
  no artifact, no further permission for the App.
- **A push after the yes** is told by the forge's own push times —
  GitHub's repository activity, GitLab's versions —, else by the commits'
  dates; a head opened with the label on is the one accepted. `follow`'s
  own update is a merge carrying workline's trailer, or a role's branch
  rebuilt, its commit's message, author and date kept.
- **GitHub turns no auto-merge on for a pull request it could merge now**
  ("clean status"): it is merged at once, at the head checked, as `gh pr
  merge --auto` does — the forge's rules met already.
- **The workflows**: `follow` runs after `judge`, and on a label put or
  taken off in a workflow of its own (`pull_request_target`): a run with
  `judge` skipped would show the check passed. GitLab starts no pipeline
  for a label: it is acted on at the next one.
- **Draft** turns auto-merge off; **another bot's own auto-merge** (a
  dependency bot's) is left to it; a fork's run is the fork's, so the
  templates keep no verdict of it.
- **The texts**: principle 1 holds as written, the label being the
  acceptance; concepts.md, ADR-0011, ADR-0020 and AGENTS.md amended.
- **Tried live** on both forges, real auto-merges and merges, no agent:
  [tried.md](../../roles/documentalist/docs/tried.md#2026-10-12--the-merge-policy-cautious-live).
- Conformance: merge/* (29 cases), the simulated forge, GitLab's mock, a
  `cmd:` forge; TestMergePolicyRunsAfterTheJudgeAndOnLabels.

[ADR-0005]: 0005-independence-takes-the-best-level-available.md
[ADR-0011]: 0011-the-review-is-on-the-merge-request-not-the-push.md
[ADR-0017]: 0017-the-release-manager.md
[ADR-0018]: 0018-the-product-owner.md
[ADR-0020]: 0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md
[ADR-0026]: 0026-the-product-owners-autonomy-is-a-level.md
[ADR-0034]: 0034-the-release-fix-follows-its-base.md
[ADR-0035]: 0035-the-engine-writes-the-jobs-summary.md
[ADR-0036]: 0036-a-roles-status-is-earned-by-written-criteria.md
[ADR-0038]: 0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md
[ADR-0041]: 0041-workline-settles-its-own-lines-when-the-base-moves.md
[#73]: https://github.com/JN0V/workline/pull/73
[#117]: https://github.com/JN0V/workline/issues/117
[#260]: https://github.com/JN0V/workline/pull/260
[#321]: https://github.com/JN0V/workline/issues/321
[#322]: https://github.com/JN0V/workline/issues/322
