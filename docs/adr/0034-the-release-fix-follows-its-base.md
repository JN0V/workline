# ADR-0034: The release's own fix follows its base

- **Status:** accepted; amended 2026-10-12 (gardening's merge requests
  follow too, or are closed), 2026-10-19 (the fix's record stands for the
  heads the tool rebuilds), 2026-10-20 (one fix for every doc a run
  reaches)
- **Date:** 2026-10-06
- **Builds on:** ADR-0017 (a release tool's pull request is the release;
  its fix goes to a merge request of its own), ADR-0027 (derived blocks
  regenerated on the default branch); principles 4, 6, 12, 14
- **Settles:** #184 (the documentalist's release branch follows main)

## Context

On a release tool's pull request, the documentalist's fix goes to a merge
request of its own, `workline/documentalist/release`, into the release's
base (ADR-0017), and the release waits until it is merged. The branch was
written once and then left alone. Twice (#182, #189) a feature pull request
merged meanwhile moved `checked:` on docs/gitlab-trigger.md, the line
beside the `judged:` the fix wrote; git saw two changes to adjacent lines,
the release merge request conflicted, and the release stayed held until
someone rebased it by hand — keeping main's `checked`, the branch's
`judged` and version bump.

release-please does not have the problem: on each push to its base it
rebuilds its own branch from the base's tip and force-pushes it; the pull
request stays, its commit is new. ADR-0027 met the same kind of conflict
for README's case count and moved the regeneration to the default branch,
rejecting a job on each merge there because gardening refreshes that block
anyway. Nothing refreshes the release fix: gardening does not take it up
(it is not its task), and the release pull request's own run judges only
docs still suspect — a doc whose `checked` main moved is not, so that run
writes nothing and leaves the conflicting branch as it was.

## Decision

**When the base moves, the release fix is rebuilt on it.** `workline
follow`, run on a push to the default branch (a `follow` job in the GitHub
and GitLab templates), lists the open merge requests from
`workline/<role>/release` and, for each whose branch the base moved under,
redoes the branch's change on the base's new tip and force-pushes it, with
a lease on the tip it read. The merge request is the same; its commit is
new, with the same message, author and `Workline-Role` trailer. It calls no
agent and holds the write token, as `apply` does (principle 6).

- **Rebuilt, not rebased.** The change is redone file by file from three
  versions — where the branch left the base, the base's tip, the branch's
  tip. A doc's front matter, when every line sets one key, is merged key
  by key: the branch's value where only the branch changed a key, the
  base's where both did (the later judgement, as the person resolved it
  by hand), a key the branch added kept after the key it followed. The
  body, and any other file, is merged as `git merge-file` does. A cherry-pick
  or a rebase would meet exactly the conflict this is about: two changes
  to adjacent header lines.
- **Already on the tip, left alone**: no push, no new checks.
- **A person's commit is never overwritten.** Every commit of the branch
  past the base must carry `Workline-Role: <role>` and be no merge;
  otherwise the branch is left as it is and the run says so
  (`not-rebuilt`, a warning). The same check now guards the release run
  itself: a new fix judged on the release pull request is not force-pushed
  over a person's commit, it goes in a comment there (`fix-in-comment`).
  Gardening's merge requests (ADR-0006) still say commits added by hand
  are overwritten: they are not touched here.
- **Only onto its own base.** The forge lists the open branches, not where
  their merge requests go: a branch whose first commit by the role did not
  leave from the base's history goes elsewhere — a maintenance branch's
  release — and is left alone (`not-rebuilt`).
- **Only when its own change no longer applies is a person asked**:
  `no-longer-applies`, status `human` (exit 2), naming the files; the
  branch is left as it is, to rebase or close — closed, the next release
  run proposes the fix again.

### Not chosen

- **Riding the release pull request's run** — it has no write token in
  its judging job, its `apply` job runs only when something was proposed,
  and it runs only when the release tool pushes its branch, which a push
  to main that changes nothing in its pull request may not do.
- **Rebasing** (or cherry-picking) the branch — the conflict this is about
  is git's own; a rebase meets it again.
- **Rebuilding only when the branch no longer merges** — fewer pushes, but
  a branch behind its base is still checked against a base that is gone,
  and a project requiring branches up to date could not merge it. Each
  rebuild runs the pull request's checks again; the release pull request
  is open for hours, and small.
- **Waiting for gardening**, as derived blocks do (ADR-0027) — the release
  is held meanwhile, a week here.

## Consequences

- One more job holding a write token, on each push to the default branch,
  which ADR-0027 avoided for derived blocks: here nothing else refreshes the
  branch, and the release waits on it. With no release merge request open,
  it lists the open merge requests and stops.
- With the job's own token (no GitHub App), a force-push runs no workflow,
  so the rebuilt pull request's checks wait for a person, as `apply`'s
  commits do.
- The template's job needs the first release with `workline follow`;
  workline's own workflow builds the engine from `main`'s commit, merged
  and reviewed.
- Conformance: documentalist/release-branch-follows-main,
  release-branch-follows-main-on-gitlab,
  release-branch-rebuilt-keeps-its-fix,
  release-branch-up-to-date-left-alone,
  release-branch-a-person-committed-to-is-kept,
  release-branch-conflict-asks-a-person,
  release-branch-on-another-base-left-alone,
  release-fix-keeps-a-person-commit.

## Amendment (2026-10-12): gardening's merge requests follow too, or are closed

On 2026-10-11, two of the documentalist's gardening pull requests had
waited four days, both in conflict with `main`
([#303](https://github.com/JN0V/workline/issues/303)):
[#265](https://github.com/JN0V/workline/pull/265) recorded `judged` on three
docs and README's derived count, which `main` regenerates
([ADR-0027](0027-derived-blocks-are-regenerated-on-the-default-branch.md));
[#269](https://github.com/JN0V/workline/pull/269) condensed
docs/spec/conformance.md, which `main` changed since. Nothing rebuilt or
closed them, and a task waits while its merge request is open
([ADR-0013](0013-what-was-judged-is-not-asked-again.md)): its docs were
judged no more. A person closed both by hand.

- **When gardening runs, its merge requests are tended**: `workline follow
  --gardening`, in the job that applies, after it applied the night's
  proposals. Each open `workline/<role>/<task>` the base moved under is
  rebuilt on its tip as the release's is, its merge request kept.
- **Closed, one comment saying why, when nothing is left to merge**
  (`closed`), with the forge's own close; the branch stays:
  - the base holds its change already;
  - the base changed the lines it changed: it no longer applies, and the
    next gardening judges the docs again on the base, which a task closed
    no longer holds back;
  - it only records when docs were read (`judged`, `judged-in-parts`,
    `verified`), and the base changed those docs since: read again or
    vouched for there, the record says nothing more. A record whose docs
    the base did not change is rebuilt: closed, its docs would go before
    an agent again the next night, to the same record, each night the
    base moved.
- **A person's commit**: the branch is theirs. Neither rebuilt nor closed;
  named in the job's summary when git's own merge, in memory (`git
  merge-tree`), meets a conflict, as the forge shows it (`not-rebuilt`), or
  cannot tell (`conflict-unread`), both warnings. Proposing the task again
  never force-pushes over it (`not-proposed`): ADR-0006's "a person's
  commits are overwritten" no longer holds.
- **Not on each push to `main`**: `workline follow` alone tends the release
  fix only. A gardening merge request waits days for review; rebuilt at
  every push, each would run its checks again for nothing a night's
  rebuild does not give.

### Not chosen

- **The forge's mergeable state** to tell a person's merge request in
  conflict: GitHub computes it in the background and answers `null` until
  then; the local forge and a `cmd:` forge have none. `git merge-tree` is
  the merge both forges run, the same everywhere.
- **Tending before judging**, a job of its own: a merge request closed
  would be proposed again the same night, but a third job would hold the
  write token. After applying, in the job that holds it already, a closed
  task is taken up the next night.
- **Closing every record on a moved base**, as #303 first asked: on a
  base that moves daily, its docs judged again every other night, tokens
  spent for the same record.

Conformance: documentalist/gardening-branch-follows-main,
gardening-branch-left-on-a-push-to-main,
gardening-branch-closed-when-main-fixed-it,
gardening-branch-closed-when-main-changed-its-lines-on-gitlab,
gardening-branch-only-judged-closed, gardening-branch-only-judged-rebuilt,
gardening-branch-a-person-committed-to-is-left,
gardening-branch-close-refused-said, gardening-keeps-a-person-commit.

## Amendment (2026-10-19): the fix's record stands for the rebuilt heads

On the 0.36.0 release pull request ([#499](https://github.com/JN0V/workline/pull/499)),
the fix only set `judged` on docs/gates-in-ci.md, to the release's head.
Merged, release-please rebuilt its branch: a new head, the same version
bump in three of the doc's sources. Measured from the commit the fix
became, the bump read as a change, kept as the doc quotes the version it
replaces: the doc was judged again, a new fix moved `judged` to the new
head, eight in an hour ([#510](https://github.com/JN0V/workline/issues/510)).

- **The documentalist's comment on the release merge request keeps** the
  head each fix recorded a doc at (`checked` or `judged`) and the git ids
  of the doc's sources there, as on a person's merge request
  ([ADR-0013, amended](0013-what-was-judged-is-not-asked-again.md#amendment-2026-10-17-on-a-persons-merge-request-the-record-waits-in-a-comment)).
  Only a comment a bot or a person of the project wrote is read.
- **A record naming such a head stands at the new head** when the doc's
  sources have the same ids there: nothing changed, the bump included.
  The doc is not judged again; one not vouched for waits for a person.
- **A source changed since**, the bump to another version included: the
  doc is judged again, and the release waits for it (ADR-0017), as it
  waits for the first fix.

Not chosen: leaving the bump out after a record a release fix brought,
which needs the forge's merged merge requests and takes a person's commit
on that branch for the role's; a doc found with nothing to fix kept out
of any merge request, its record in the comment alone: a doc vouched for
would still be judged again.

Conformance: the five documentalist/release-record-* cases.

## Amendment (2026-10-20): one fix for every doc a run reaches

On a release tool's pull request, one run judged a single batch of the
docs due: the docs one call can take (`ai-max-calls`, the task's size).
Its fix went to `workline/documentalist/release`, and no next round
followed, as none could see that fix
([#572](https://github.com/JN0V/workline/issues/572)). The next batch
waited until a person merged it, then for the next night. Replayed on
the 0.38.0 release with a scripted agent
([#567](https://github.com/JN0V/workline/pull/567)): one doc a round,
each task filled by one large doc.

- **A round that left docs for the next keeps its fix in the clone**,
  not applied, and the run goes round, at night (`--no-apply`) as by
  hand. The next round sees those docs fixed, does not ask again those
  an earlier round asked, and proposes the earlier rounds' patches, as
  judged, ahead of its own. The comment keeping the records and the merge
  request's body carry every round's docs.
- **One fix**: the last round's proposal, applied once or listed once
  for the job that applies, one commit on `workline/documentalist/release`,
  one merge request. A last round that proposed nothing leaves the round
  before it to apply.
- **The bound**: each round asks at most `ai-max-calls` docs, and only
  docs no round of the run asked, so a run takes no more rounds than
  there are docs due; it stops sooner at the run's `ai-max-tokens` or
  `ai-max-seconds`, and at 20 rounds (`rounds-max`), five for any other
  run. What is left still holds the release, said, for the next night's
  run or a person's.
- **The clone is put back as it was** when the run ends.

Not chosen:

- **A fix a round, stacked on the branch**: the apply job would push as
  many times, each push the round's patches on the commit before, and
  the merge request's body and the records comment would say the last
  round only.
- **Several calls in one round**, as parts are: one answer judged for
  every batch, a refusal asking every batch again.

Conformance: documentalist/release-pull-request-fixes-every-due-doc-in-one-night,
release-pull-request-by-hand-fixes-every-due-doc,
release-pull-request-night-stops-at-its-tokens.
