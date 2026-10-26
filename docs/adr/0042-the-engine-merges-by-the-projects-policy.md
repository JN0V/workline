# ADR-0042: The engine merges by the project's written policy, through the forge's auto-merge

- **Status:** accepted, "For the person" as proposed; `cautious` built by
  [#321] ([page](../merge-policy.md), [below](#built-2026-10-12-321));
  `normal` built for workline's own routine pull requests by [#322]
  ([amendment](#amendment-2026-10-13-322-normal-routine-pull-requests-first)),
  agents' pull requests next; workline's own record commits keep the yes
  ([#380], [amendment](#amendment-2026-10-14-380-worklines-own-record-commits-keep-the-yes));
  an agent's rewording for a reader waits for a yes unless the project
  lets it ([#438], [amendment](#amendment-2026-10-17-438-an-agents-rewording-waits-for-a-yes));
  up to date and one at a time dropped by [ADR-0051]
  ([amendment](#amendment-2026-10-21-adr-0051-no-up-to-date-no-one-at-a-time))
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

Built for workline's own routine pull requests: the
[amendment](#amendment-2026-10-13-322-normal-routine-pull-requests-first).

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
([page](../merge-policy.md)), an independent review's findings answered:

- **The yes holds on its head through a required check, not a race**:
  GitHub keeps auto-merge on after a writer's push, and the checks of the
  new head could pass before `follow` runs again. `follow` sets the status
  `workline/accepted` on the head a yes covers, by the App; the base
  requires it, from the App, so a new head cannot merge until a yes covers
  it. A base that does not require it gets nothing turned on, said.
  Making `judge` fail instead was not chosen: a person putting the label
  again would wait for the job to run again. On GitLab the merge waits
  for the new head's pipeline, which holds `workline:merge`.
- **Each head's verdict, where only the App writes it**: the judging job
  writes it (`route --verdict`); a job of its own keeps it on GitHub as the
  status `workline/verdict`, set by the App, bound by the forge to the
  commit — in no concurrency group, so no newer run cancels it, and only
  for the head the event names. Only what `follow`'s own token set is
  read: a status a person sets with their own token never counts, as a git
  note anyone could push would. `workline/accepted` must be required from
  the App itself: unpinned, nothing is turned on. On GitLab, the verdict
  is in hand in the pipeline that judged the head; a status set there from
  elsewhere could become the merge request's pipeline.
- **The limits, where a token is in reach**: GitHub runs a pull request's
  own copy of the workflow with the App's key, so whoever can push a
  branch could set both statuses as the App — since
  [ADR-0044](0044-the-apps-key-stays-with-the-default-branchs-runs.md)
  ([#331](https://github.com/JN0V/workline/issues/331)), the key lives in
  an environment only the default branch may use, and the jobs holding it
  run the default branch's workflow; a GitLab project bot's token can be
  read by whoever runs a pipeline.
- **Who says yes** may push and merge: GitHub's write access, GitLab's
  Developer — not the product owner's wider "person of the project".
- **A push after the yes** is told by the forge's own push times —
  GitHub's repository activity, GitLab's versions —, else by the commits'
  dates; a head opened with the label on is the one accepted. `follow`'s
  own update is checked by its tree — the base merged in, never another
  branch, workline's lines settled, a role's branch rebuilt on the base —,
  never by its trailer; the line, too, judges a commit whose trailer its
  tree does not bear out, leaves out only doc pages (never a role's prompt
  nor a file steering agents), and has the committer check every commit.
- **GitHub turns no auto-merge on for a pull request it could merge now**
  ("clean status", or "unstable" while a check it does not require still
  runs, as the merge job's own on workline#341): it is merged at once, at
  the head checked, as `gh pr merge --auto` does — the forge's rules met
  already.
- **The workflows**: `follow` runs after `judge` (in workline-apply.yml since
  [ADR-0044](0044-the-apps-key-stays-with-the-default-branchs-runs.md)), and on a label put or
  taken off in a workflow of its own (`pull_request_target`): a run with
  `judge` skipped would show the check passed. GitLab starts no pipeline
  for a label: it waits for that merge request's own next pipeline.
- **The line never stalls**: one not judged yet waits, named; one whose
  check failed is out; only one ready and up to date ends a pass.
- **Draft** turns auto-merge off; **another bot's own auto-merge** (a
  dependency bot's) is left to it; a fork's run is the fork's, so the
  templates keep no verdict of it.
- **The texts**: principle 1 holds as written, the label being the
  acceptance; concepts.md, ADR-0011, ADR-0020 and AGENTS.md amended.
- **Tried live** on both forges, real auto-merges and merges, no agent;
  on GitHub, a head pushed after the yes, its other checks green, held
  five minutes by the required status until `follow` took the yes back:
  [tried.md](../../roles/documentalist/docs/tried.md#2026-10-12--the-merge-policy-cautious-live).
- Conformance: merge/* (41 cases), the simulated forge, GitLab's mock, a
  `cmd:` forge; routing/forged-merge-is-judged, merge-of-another-branch-is-judged,
  committer-checks-workline-s-own-commits;
  TestMergePolicyRunsAfterTheJudgeAndOnLabels, TestGitHubGuarded,
  TestSumChecks.

## Amendment (2026-10-13, #322): `normal`, routine pull requests first

The maintainer, 2026-10-13: build `normal` now, workline's own routine
pull requests first; the agents' pull requests in a second part. What the
build settled ([page](../merge-policy.md#normal-routine-work-with-no-yes)):

### What merges with no person's yes

- **Workline's own pull request, as the forge says it** — never as a
  commit says it of itself, which whoever can push writes:
  - from a role's branch, `workline/<role>/<task>`, of this repository,
    not a release tool's;
  - opened by workline's App — on GitHub the App the workflow names
    (`WORKLINE_APP`), on GitLab the project's bot —; never another bot
    (an agent's app), never a person's token, which agents use too: with
    one, nothing merges alone, said;
  - every push to its branch since it was created the App's, as the
    forge names the pusher: GitHub's repository activity, GitLab's push
    events; its creation not among them, or no push named, it waits;
  - every commit committed under the name the App's jobs commit with, and
    workline's own by its tree as `follow` tells it ([#329]);
  - every page it changes a doc of the documentalist's (its `docs`
    setting) and a doc page, never one agents are told to read: an agent
    file (AGENTS.md, CLAUDE.md…), a page one links to or imports — a
    folder linked, all of it —, a file named a prompt, an instruction or
    a template;
  - not a Draft, its auto-merge not turned off since it was turned on.
- **Its head's verdict passed**, as for a yes: `judge` green, an agent
  really answered where one is set, no lens short.
- **Three kinds, told by what the change is**, never by the branch's name:
  - `records`: only the lines recording when docs were read (`judged`,
    `judged-in-parts`, `verified`);
  - `derived`: derived blocks, read as the documentalist reads them (a
    marker with no end is no block), each as its command gives it on the
    head — run in a worktree of it —; else `docs`;
  - `docs`: any other doc text — the release's docs fix, gardening's
    fixes, a page condensed.
- **The same path as a yes**: `follow` sets `workline/accepted` on the
  head itself, by the App, then turns auto-merge on or merges at once; in
  line, one at a time, the release's docs first.

### Each merge recorded, a revert demotes its kind

- **On the pull request**: one comment as auto-merge is turned on — no
  person's yes, its kind, why, how to hold it —, its marker naming the
  kind, read back only from the App. **In the job's summary**: the pull
  request, "with no person's yes", its kind and why; its first line says
  which kinds merge alone and which wait again.
- **Every revert on the base is read, once, from the App's mark**: a
  status, `workline/merge-read`, the App sets on the base commit it read,
  with the demotion issues then held — read back from the App only, as
  `workline/verdict`: whoever can push sets a status of their own, never
  the App's. Each run reads every commit past the newest mark on the
  base's last 100 first-parent commits, every parent. A ref was tried
  first: whoever can push rewrites one, and nothing records who did.
- **No mark found — the first run at `normal`, a mark lost, `follow` not
  run for long —: every kind waits for a person's yes**, in an issue the
  App opens (`merge-record-lost`), until a person who may push and merge
  closes it; the last 100 commits are read meanwhile.
- **A revert names what it reverts** in git's words (`This reverts commit
  …`, the commit reverted naming `#N`), GitLab's (`This reverts merge
  request !N`) or GitHub's (`Reverts owner/repo#N`). GitHub's own Revert
  says only `Revert "<title>"`, tried live: the merge of that title it
  undoes exactly — the same change, inverted (`git patch-id`) —, else the
  newest.
- **Its kind waits for a person's yes again** ([ADR-0026]) when the pull
  request reverted holds the App's record. **A revert tied to none** — a
  rebase merge, an edited squash message, an undo by hand — **sends every
  kind back**, said (`revert-unread`): failing closed.
### What holds a demotion, what lifts it

- **Held by an issue the App opened**, one per revert — its title names
  the revert, so nobody opens it before. The forge records who opened an
  issue, and that cannot be faked: each open demotion issue of the App's
  holds its kind, whatever any mark says. One of that title by anyone
  else is never taken for it: the revert is read again next run, held
  meanwhile.
- **Lifted only by a person**: the issue closed by someone who may push
  and merge, not a bot, as the forge says who closed it. Closed by anyone
  else, the demotion holds while the mark lists it, said
  (`restore-ignored`).
- **Why an issue**: written where people look, with what to do
  (principle 14), one notification, one click to restore. Not a label on
  the reverted pull request: closed, nobody looks there. Not the setting:
  changing the level never restores trust lost ([ADR-0026]).

### The veto, what never merges alone, the criteria

- **The veto stays the forge's**: Draft; auto-merge turned off after it
  was turned on stays off until a person labels it — by whoever turned it
  off, the forge too, on the safe side. No hold label: "Not chosen" above
  stands.
- **Never alone, at any level**: the release tool's pull request (said
  in the summary at `normal`: merging it is the decision to release), a
  fork's, a person's, another bot's, one with a push or a commit not the
  App's, one changing what steers agents, a Draft.
- **[#331] first, before `normal` is turned on anywhere, workline
  included**: a pull request's own workflow run can reach the App's key
  today, and with it open and push a `workline/` pull request as the App —
  merged alone, bounded to the documentalist's docs agents are not told
  to read. `normal` waits for the key to leave the pull request's runs —
  done by [ADR-0044](0044-the-apps-key-stays-with-the-default-branchs-runs.md).
- **Criteria for offering `normal`** to other projects, as [ADR-0036]
  wrote a role's status — until they hold, the page says it is tried on
  workline's own repository first:
  1. built: the record, the demotion, their cases (this build);
  2. run on workline's own repository for two weeks and at least 20
     merges with no yes, each recorded, a revert demoting its kind as
     written;
  3. the maintainer validates [#322]'s Validation on real use.
- **Limits**: whoever holds the App's key acts as it ([#331], above);
  since [ADR-0044](0044-the-apps-key-stays-with-the-default-branchs-runs.md),
  only the default branch's runs hold it.
  GitLab: the push events read from a week before the merge request
  opened, the branch's creation among them or it waits; the mark is a
  commit status of the project bot's — on a commit with a pipeline, an
  external job of it —, not tried live. A forge that does not name who
  pushed, who closed, or keeps no mark only the App writes (a `cmd:`
  forge without `pushers`, `closing`, `mark`): nothing merges alone, no
  demotion is lifted.
- **Texts changed**: principle 1 — people accept the result, or write in
  advance the narrow class the line accepts for them;
  [concepts.md](../concepts.md#people); [config.md](../config.md), the
  governance line beside the key.

Conformance: merge/normal-* (14), gitlab-normal-routine-merges-alone,
cmd-normal-routine-merges-alone, cautious-routine-waits-for-a-yes;
TestRoutineKind, TestSteering, TestRevertOf, TestOwnNote,
TestMarkAndIssuesHoldDemotions.

## Amendment (2026-10-14, #380): workline's own record commits keep the yes

Seen 2026-10-14 ([#380]): `main` moved; workline's bot merged it into
accepted pull requests, then pushed commits changing only record lines
(`judged` and `verified`; `judged-in-parts`). Each took the yes back; the
batch of 11 was merged by hand. "For the person", point 1, now reads: any
commit but workline's own.

- **A record commit is no push**, as `follow`'s own update: one parent,
  its tree changing against it only the lines [ADR-0041] says workline
  writes in the docs — a header's `checked`, `judged`, `judged-in-parts`
  or `verified`; a derived block only as its command gives it at that
  commit, as `normal`'s `derived` kind. The yes and auto-merge stay.
- **The tree tells it; the trailer is needed, never enough**: a commit
  with no workline trailer is a person's, whatever it changes.
- **Still a push**: a doc's text or another header key changed, a file
  added, removed, no doc of the documentalist's or its mode changed, an
  empty commit.
- **The limit**: whoever may push can write the trailer on such a commit.
  It changes no line a person accepted; a `checked` moved only stops a
  doc being suspect, as a merge of the base settling it does.

Conformance: merge/record-commit-keeps-the-yes,
docs-fix-after-the-yes-takes-it-back, persons-record-commit-takes-it-back;
TestOwnRecord.

## Amendment (2026-10-17, #438): an agent's rewording waits for a yes

Seen on 2026-10-16 and 10-17: the documentalist's reader task
([ADR-0048](0048-the-agent-reads-the-pages-the-reader-checks-flag.md))
opened two pull requests rewording pages for their reader, [#421] and
[#423]. Each counted as `docs` and merged alone, within four minutes,
no label. The maintainer decided on 2026-10-17 to read them first, for
the first weeks: [#256]'s Validation is the maintainer agreeing with
what it proposes.

- **A fourth kind, `reader`, the highest**: a change writing a page's
  `reader:` line, the record the reader task leaves in each page's
  header. Told by what the change is, as the other kinds, never by the
  branch's name; a `reader:` line elsewhere than a header is text.
- **It waits for a person's yes at `normal`**, through the same path
  as a kind demoted: auto-merge left off, or taken back, and said
  (`waits-for-a-yes`). Said once on the pull request too: the label
  that accepts it, and the setting that lets it merge alone.
- **A project lets it merge alone**: `merge: {autonomy: normal, alone:
  [reader]}`. Written in advance, like the level: the narrow class the
  line accepts for a person (principle 1). Then it is recorded as
  `reader`, and a revert demotes it as any kind.
- **`alone` is checked**: a kind not waiting by default (`docs`), or
  `alone` below `normal`, stops `follow` before anything is touched.
- **Not chosen**:
  - an issue the App opens, closed to let the kind merge alone: a
    demotion is trust lost; this is a choice made in advance, and
    belongs where the level is set;
  - the branch's name, `workline/documentalist/reader-…`: the kinds are
    told by the change;
  - holding every `docs` change: gardening's fixes merged alone as
    designed.

Conformance: merge/normal-reader-waits-for-a-yes,
normal-reader-alone-when-let, alone-needs-normal,
alone-unknown-kind-refused; TestRoutineKind, TestDemotedKind.

## Amendment (2026-10-21, ADR-0051): no up to date, no one at a time

[ADR-0051]'s cut 2, measured: strict up to date cost 103 rounds in four
weeks, and no test went red after a merge of main. "Up to date, one at a
time" and "The order" above are replaced:

- **Each accepted pull request merges once its own head is green**:
  auto-merge on for every one admitted, in the same run; none waits for
  another, none is brought up to date to merge.
- **The base requires no branch up to date**: on GitHub, "Require
  branches to be up to date" off; the required checks and
  `workline/accepted` stay.
- **Kept**: a release held for its docs, merged since, is still brought
  up to date, so its line runs again, until [ADR-0051]'s step 3 removes
  the release's docs. [ADR-0041]'s merges of the base, for conflicts on
  workline's own lines, stay, until [ADR-0051]'s step 10 removes them.
- **The net is main's run**: red, it says so in its one issue and once
  on each open pull request whose base holds the red commit ("base red,
  not yours"). The issue counts red mains a week; above one, the forge's
  queue goes on: GitHub's merge queue in an organisation, GitLab's merge
  train. GitLab projects keep merged-results pipelines.
- **Gone**: the line's order, `fork-behind`, the first in line's update.

Conformance: merge/behind-main-merges-as-it-is,
accepted-all-turned-on-at-once, gitlab-accepted-turned-on;
routing/main-red-says-base-red-on-open-merge-requests;
TestMainIssueCountsRedMainsAWeek.

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
[ADR-0051]: 0051-a-check-sits-where-its-catch-is-worth-its-cost.md
[#73]: https://github.com/JN0V/workline/pull/73
[#117]: https://github.com/JN0V/workline/issues/117
[#256]: https://github.com/JN0V/workline/issues/256
[#260]: https://github.com/JN0V/workline/pull/260
[#321]: https://github.com/JN0V/workline/issues/321
[#322]: https://github.com/JN0V/workline/issues/322
[#329]: https://github.com/JN0V/workline/pull/329
[#331]: https://github.com/JN0V/workline/issues/331
[#380]: https://github.com/JN0V/workline/issues/380
[#421]: https://github.com/JN0V/workline/pull/421
[#423]: https://github.com/JN0V/workline/pull/423
[#438]: https://github.com/JN0V/workline/issues/438
