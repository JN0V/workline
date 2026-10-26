# ADR-0041: workline settles its own lines when a pull request's base moves

- **Status:** removed 2026-10-26 by
  [ADR-0051](0051-a-check-sits-where-its-catch-is-worth-its-cost.md#build-order)'s
  step 10, but for the recognition of a merge it made
  ([amendment](#amendment-2026-10-26-adr-0051-step-10-the-merges-removed));
  accepted before, "For the person" as proposed, built by
  [#319](https://github.com/JN0V/workline/issues/319)
- **Date:** 2026-10-12
- **Extends:** [ADR-0034](0034-the-release-fix-follows-its-base.md)
  (`follow` keeps workline's own branches on their base, a doc's header
  merged key by key) to people's pull requests, as a merge
- **Builds on:** [ADR-0011](0011-the-review-is-on-the-merge-request-not-the-push.md)
  (a person merges), [ADR-0013](0013-what-was-judged-is-not-asked-again.md)
  (the record in the doc's header), [ADR-0027](0027-derived-blocks-are-regenerated-on-the-default-branch.md)
  (derived blocks regenerated on the default branch); principles 1, 4, 6, 13, 14
- **Research:** [merge-conflicts.md](../research/merge-conflicts.md)

## Context

- **workline's own header lines are most of the conflicts.** From
  2026-09-28 to 2026-10-12, 24 of 147 people's pull requests met a conflict
  with `main`: 150 hunks, 89 of them `checked`, `judged` or `verified`
  lines, 7 derived blocks. 9 pull requests conflicted on nothing else.
- **Why**: docs name whole folders as their `sources`, so a feature pull
  request moves `checked` on 20 to 30 docs, and two of them open together
  change the same lines.
- **A person or an agent settles them by hand.**
  [#316](https://github.com/JN0V/workline/pull/316)'s rebase met 17 header
  hunks, in 16 docs and [README.md](../../README.md)'s header, beside 2
  code hunks; its builder checked the docs again.
- **workline's own branches no longer wait**: `follow` rebuilds them a
  median of 33 s after a push to `main`
  ([ADR-0034](0034-the-release-fix-follows-its-base.md)); a person's is
  left, named in the job's summary when it no longer merges.
- **The forge stops where this starts**: Update branch merges the base in
  and stops at a conflict; a merge queue removes a pull request in conflict,
  and is not offered on a user's repository; no forge runs a merge driver.

## Decision

**When the base moves, `workline follow` merges it into each open pull
request it left in conflict, if every conflict is in a line workline
owns.** It completes the forge's Update branch where the forge stops: the
same merge, the same message, workline's own lines settled. Which lines
conflict is told by git's own merge, in memory (`git merge-tree`), as in
[ADR-0034's amendment](0034-the-release-fix-follows-its-base.md#amendment-2026-10-12-gardenings-merge-requests-follow-too-or-are-closed);
no agent is called (principle 6).

### Which pull requests

- Open, into the branch that moved, from a branch of the same repository.
- **Opened by a person**: not `workline/…`, rebuilt by
  [ADR-0034](0034-the-release-fix-follows-its-base.md); not another bot's —
  release-please, Renovate and Dependabot rebuild their own branch, and
  Renovate stops rebuilding one somebody else committed to; not a
  long-lived branch, the base of another open one: named in the summary.
- **A fork's is left**: the job's token cannot write to it. Named in the
  summary when it no longer merges.
- **One merging clean is left** ("For the person", point 2).

### Which lines are workline's

- In a doc's header — a `---` front matter, or README's `<!-- workline …
  -->` comment — the keys `checked`, `judged`, `judged-in-parts` and
  `verified`, each set on its own line.
- A derived block (`<!-- workline:derive … -->…<!-- workline:end -->`).
- Everything else is the person's: a doc's text, the code, and every other
  header key, `sources` included.

### How they are settled

- **A header, key by key**, as `follow` merges its own branches: two keys
  changed on adjacent lines are no conflict, each side's change kept.
- **One of the four keys changed on both sides**: the base's value (below).
- **Any other key changed on both sides** (`sources`): a conflict, left to
  the person. Today's key-by-key merge takes the base's value for every
  key; on a person's branch, only for the four.
- **A derived block**: the base's, which gardening regenerates there
  ([ADR-0027](0027-derived-blocks-are-regenerated-on-the-default-branch.md)).
- **Any conflict left: nothing is written.** The job's summary names the
  pull request and its files, a warning, as `not-rebuilt` does. A merge
  commit cannot hold a conflict, so a pull request also in conflict on the
  person's lines is left whole.

### How it is written

- **A merge commit, never a rewrite**: the branch's tip as first parent,
  the base's tip as second, as `git merge` and the forge's Update branch
  make it. A person's commits keep their ids.
- **The forge's message**, "Merge branch 'main' into <branch>", a body
  naming the files settled, and the trailer `Workline-Role: documentalist`.
- **Pushed as a fast-forward, never forced**, a lease on the tip read: a
  push meanwhile stays, a branch deleted is not brought back; said.
- **It is workline's commit**: a line given a range ending on it judges
  the commits before it, through its first parent
  ([routing](../spec/routing.md)): a red verdict stays red.

### Which value stays when both sides moved a key: the base's

Both sides moved `checked` on a doc because both changed its sources.
Whichever value stays, the other side's commits are listed as changed since
it, and the doc is judged again, as
[#316](https://github.com/JN0V/workline/pull/316)'s builder did by hand.
The choice is what is judged, and when:

- **The base's (chosen)**, what `follow` keeps on its own branches: the
  pull request's own changes are listed. They are judged at its next push
  or, once merged, by gardening: after the squash, the doc reads as checked
  at the other pull request's squash. One more judgement of each such doc,
  never skipped.
- **The branch's**: `main`'s changes are listed, judged there already, and
  judged again at the next push for nothing new. Merged before it, the doc
  reads as checked at this pull request's squash, which holds both changes:
  nobody reads them together.
- **Neither, checked at the merge**: the documentalist would count the
  base's commits as checked when the base's own header was. No tokens; no
  reading of the two changes together, and its anchor rule changes. Not
  chosen; kept should the judgements cost too much.

### Who sees it

- **The person**: the commit on the pull request's timeline, by the
  token's bot, its message saying what was settled. No comment: the commit
  says it, and a comment at each push to the base would be noise.
- **The job's summary** lists each pull request merged into, and each left
  with its files ([ADR-0035](0035-the-engine-writes-the-jobs-summary.md)).
- **An agent working on the branch**: git refuses its next push ("fetch
  first"); it pulls, then pushes. A force-push over the merge only loses
  it, merged again at the next push to the base; a rebase on the base
  drops it and meets the header conflicts again. The rule is written where
  agents read: [AGENTS.md](../../AGENTS.md) here, the documentalist's page
  for a project.

### The setting and the token

- **Off by default**: a project opts in with a documentalist setting,
  `update-branches: conflicts` (default `off`). workline's own repository
  turns it on.
- **The `follow` job's token**: the project's GitHub App when it has one,
  so the checks run on the new head; otherwise the job's own, and the
  checks wait for a person
  ([ci-and-forge.md](../research/ci-and-forge.md#a-bots-commit-on-the-pull-request)),
  as for the release fix — a conflict waits on a person too. On GitLab,
  the project's bot ([ADR-0023](0023-the-project-bot-keeps-a-gitlab-backlog.md)).
- **No new permission on GitHub**: the job has `contents: write`. A base
  that changed a workflow file may need the App's `workflows` permission
  for the push: not tried live; a refused push is said (`merge-refused`).

### For the person (settled as proposed)

1. **The default**: off (proposed), so writing to a person's branch is a
   project's choice; or on, since the lines are workline's own.
2. **Branches behind but clean**: not updated (proposed). Updating them
   would end the 30 Update branch clicks of two weeks, but run every open
   pull request's checks at each push to `main`, and move branches under
   agents for nothing in conflict. Bringing a branch up to date just before
   it merges belongs to the decision below.
3. **The value kept**: the base's (proposed), against the branch's or none.

### Which role merges: decided apart

Which role, if any, merges pull requests is being researched. This
decision is its prerequisite, not its answer: whoever merges needs a branch
that merges. When to merge, and an update just before it, as a queue does,
belong to that decision.

### Not chosen

- **Rebasing the person's branch** (the forge's other Update branch): it
  rewrites their commits, and an agent working on it loses its base.
- **The forge's Update branch API**: it stops at the conflicts this is
  about, and GitLab's API offers a rebase only. git's merge runs the same
  on both (principle 10).
- **Settling a part**: a merge commit cannot hold a conflict.
- **A merge driver** (`.gitattributes`): no forge runs one
  ([research](../research/merge-conflicts.md#git)). A local driver, for an
  agent's or a person's own merges and rebases, stays open.
- **Taking `checked` out of the doc** (a trailer, a note, a ref): it removes
  the hotspot, but reverses
  [ADR-0013](0013-what-was-judged-is-not-asked-again.md)'s record seen in
  review, depends on the forge's squash message, and is weeks of work.
- **Narrowing `sources`**: fewer collisions, not none; open apart. **A
  merge queue**: not offered here, and it settles no conflict.

## Consequences

- **workline writes to branches it did not create**, where a project opts
  in: their author pulls before pushing again.
- **The checks run again** on each pull request merged into (with an App
  token): only those left in conflict.
- **Approvals**: a merge resets them where the forge dismisses them on a
  push (GitHub's "dismiss stale reviews", GitLab's "reset approvals"). Off here.
- **On the next push, the reviewer sees what the merge settled**
  ([ADR-0013's amendment](0013-what-was-judged-is-not-asked-again.md#amendment-2026-10-12)):
  header lines, no lens asked for them alone. A protected branch: `merge-refused`.
  A reviewer's block survives the merge once [#326](https://github.com/JN0V/workline/issues/326) is fixed; not exposed here, AI findings only warn.
- **9 of 24 pull requests in two weeks** would have met no conflict by
  hand; the 9 mixed with a real overlap,
  [#316](https://github.com/JN0V/workline/pull/316) among them, still meet
  their header lines when a person settles the rest.
- **`main` holds no merge commit**: this repository merges by squash.
  Conformance: documentalist/persons-branch* (18), forge/cmd-persons-branch-merged,
  routing/own-*, routing/follow-merge-keeps-a-red-verdict, reviewer/workline-merge-of-main-asks-no-lens.

## Amendment (2026-10-26, ADR-0051 step 10): the merges removed

- **Why**: since [ADR-0051](0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-3-no-docs-obligation-per-pr),
  no `checked` is pushed to a pull request and none needs 0 suspect: two
  pull requests no longer move the same header lines, the conflict this
  decision settled.
- **Removed**: `follow`'s merge of the base into a person's pull request,
  and the documentalist's `update-branches`. A project still setting it
  gets a warning, `setting-gone`; nothing is written to its branches.
- **Kept**: the recognition of a merge it made, still on an open branch
  ([routing](../spec/routing.md), the reviewer, the merge policy): judged
  through its first parent, as before. It goes once no open branch
  holds one.
- **A conflict on a header line** is its author's, as any other: rare now
  no `checked` is pushed to a pull request.
- Conformance: documentalist/persons-branch-left-update-branches-gone;
  the 18 `persons-branch*` cases, forge/cmd-persons-branch-merged and
  routing/follow-merge-keeps-a-red-verdict removed.
