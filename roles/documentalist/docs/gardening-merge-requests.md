---
sources:
  - roles/documentalist/code/documentalist.go#preRun.writeRecords
  - roles/documentalist/code/kept.go#recordsMerged
  - roles/documentalist/code/kept.go#asJudgedAt
  - internal/forge/forge.go#MergeRequest
  - roles/documentalist/code/documentalist.go#preRun.pauseIfBackpressure
  - roles/documentalist/code/waiting.go#readWaiting
  - roles/documentalist/code/waiting.go#holdWaitingTasks
  - roles/documentalist/code/waiting.go#waiting.holds
  - roles/documentalist/code/waiting.go#waiting.changing
  - roles/documentalist/code/waiting.go#partsKey
  - roles/documentalist/code/documentalist.go#preRun.rewriteWaits
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:garden:propose
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:garden:apply
checked: e81df7b34
verified: agent:claude-code
---
# Documentalist — gardening's merge requests

Part of [the gardening tasks](gardening.md).

## Records kept on merged merge requests

First, in a round of its own: the docs judged on a person's merge request
with nothing to fix, whose record was kept in the documentalist's comment
there, not committed
([on a merge request](push-records.md)).

- **Which merge request**: for each suspect doc, the last commit changing
  it or a source, and the merge request the forge says became it, or the
  closest after: the commit it became on its base — the squash, the merge
  commit, or the last one rebased.
- **The record holds** when the doc and each source are, in that commit and
  now, as judged: the doc is not put before an agent, and `judged: <that
  commit>` (or `judged-in-parts`, as the record says) goes in
  `workline/documentalist/records`, which only records
  when docs were read: it merges alone at `normal`
  ([the merge policy](../../../docs/merge-policy-normal.md)).
- **Otherwise the doc is judged again**, its finding naming the file no
  longer as judged.
- **Not for a doc read in parts that night**: its sources too large to go
  whole, its record says only what changed was read; it is
  [read in parts](parts.md) instead
  ([#462](https://github.com/JN0V/workline/issues/462)).
- **The cost**: one listing of the merge requests merged since the oldest
  such commit, and one read of the comments of each one a doc points to
  ([ADR-0010, amended](../../../docs/adr/0010-docs-judged-on-the-merge-request-and-by-gardening.md#amendment-2026-10-17-the-commit-a-merge-request-became-stands-for-its-record)).
- **Never a pass by guess**: a listing or a comment unread is said
  (`record-unread`), and the docs are judged again; so is a comment
  carrying its marker that workline's own account did not write
  ([why](push-records.md)).

## One merge request per task

Run with `--open-merge-request` (the CI templates' scheduled jobs):

- a gardening task's patch goes on a branch of its own,
  `workline/documentalist/<task>`, with a merge request titled after the
  task, short enough for a squash's ` (#N)`
  ([the squash title](../../committer/docs/squash-title.md)); running the
  task again updates it
  ([ADR-0006](../../../docs/adr/0006-gardening-opens-one-merge-request-per-task.md));
- while `max-open-merge-requests` of them wait for review, gardening
  proposes nothing (`gardening-paused`), and what it finds is still
  reported;
- a doc another open merge request changes is left to it until it is
  merged or closed;
- while a task's own merge request waits, the docs it would judge wait too
  ([ADR-0013](../../../docs/adr/0013-what-was-judged-is-not-asked-again.md)):
  the docs judged whole, while `suspect` or `stale` waits;
- a doc judged in parts waits while any of these merge requests changes
  it — its fix, or only the record of when it was judged —; the next doc
  goes, each run's on a merge request of its own, `fix-<its first doc>`
  (a short hash added when another doc's has that name)
  ([ADR-0013, amended 2026-10-14](../../../docs/adr/0013-what-was-judged-is-not-asked-again.md#amendment-2026-10-14-a-doc-waits-while-a-merge-request-holds-it));
- a doc to split, a card to merge, a repeated passage wait
  while their own merge request does; the next one goes;
- `records` is proposed again only for a doc it does not hold yet: then
  with every record that still holds;
- a task proposed by an earlier round of the same run counts as waiting: a
  CI night judged with `--no-apply` that proposes the docs judged whole
  goes round again, and the docs put off to parts are judged the same
  night
  ([ADR-0013, amended](../../../docs/adr/0013-what-was-judged-is-not-asked-again.md#amendment-2026-10-02)).
  Workline's first nightly had put thirteen off "to a later round" a CI
  run never had;
- the merge request lists each doc it judged with what became of it: fixed
  and in how many places, still true, or nothing found wrong; `checked`
  moved or `judged` recorded; the code examples a fix wrote, unverified
  or checked ([code examples](examples.md)); for a doc whose sources did
  not all go whole, what was read and what was not
  ([the check](judge-findings.md#what-the-finding-says)).

## When the base moves

Each night, after applying what it proposed, gardening tends the merge
requests it opened before (`workline follow --gardening`,
[ADR-0034, amended](../../../docs/adr/0034-the-release-fix-follows-its-base.md#amendment-2026-10-12-gardenings-merge-requests-follow-too-or-are-closed);
[the command](../../../docs/commands-follow.md)).
One the base moved under is:

- **rebuilt** on the base's tip, its change redone, the merge request kept:
  a `checked` the base moved beside the `judged` it wrote is no conflict;
- **closed**, one comment saying why, when nothing is left to merge:
  - the base holds its change already;
  - the base changed the same lines: the next gardening judges the docs
    again and proposes what is still needed;
  - it only records when docs were read (`judged`), and the base changed
    those docs since; a record whose docs the base left as they were is
    rebuilt, so they are not judged again for the same record;
- **left to a person** who committed to it: neither rebuilt nor closed,
  and named in the job's summary when it no longer merges, or when git
  cannot tell (`conflict-unread`). Running the task again never pushes
  over that commit either (`not-proposed`,
  [more](../../../docs/usage.md#writing-to-a-merge-request)).

A push to the base leaves them alone, and people's pull requests too.
