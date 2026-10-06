# ADR-0034: The release's own fix follows its base

- **Status:** accepted
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
