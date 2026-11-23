---
sources:
  - internal/engine/merge.go#mergePolicy
  - defaults/ci/github/workline-apply.yml#jobs.verdict
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:merge
checked: 3f432ab14
verified: agent:claude-code
---
# Where the head's verdict lives

Part of [the merge policy](merge-policy.md): auto-merge is turned on
only once the head's verdict passed. Where that verdict is kept, per
forge, and what it cannot guard.

## Kept per forge

- **The proposing job writes it**: `workline route … --verdict <file>`.
- **GitHub**: a job of its own keeps it, `workline follow --verdict <file>
  --verdict-only`: the status `workline/verdict` on the head, set by the
  App. Only the statuses `follow`'s own token set are read: one a person
  sets with their own token never counts.
- **GitLab**: in hand in the merge request's pipeline, where
  `workline:merge` reads it; nothing kept elsewhere.
- **The local forge**: in the clone, `.git/workline/verdicts/`, kept by
  `workline route merge-request` itself when `merge:` is set.
- **No verdict on the head yet**: auto-merge waits for it.

## A draft's verdict

- **A review to come** (`waits`): the rules and gates passed, the AI
  review waits for it to be marked ready. GitHub's `workline/verdict` is
  pending.
- **Marked ready**, auto-merge waits for the run on ready's verdict
  (`review-to-come`), never taking the draft's for a pass
  ([the reviewer on a draft](../roles/reviewer/docs/ready.md)).

## The App's key, out of a pull request's reach

- **GitHub runs a pull request's own copy of the workflow**, which its
  branch can change.
- **The key sits in an environment only the default branch may use**:
  the jobs holding it run the default branch's workflow and read the
  pull request's run as data
  ([ADR-0044](adr/0044-the-apps-key-stays-with-the-default-branchs-runs.md)).

## The limits

- **The verdict is the pull request's run's word**: one changing its
  workflow can pass its own head, as its own `propose`; the yes covers
  that head.
- **On GitLab**, whoever can run a pipeline can read the project bot's
  token, so a person who may push could act as it.

## Where it is read

- **The verdict on the head**: `merge.go` reads it before turning
  auto-merge on.
- **GitHub's job that keeps it**: the `verdict` job of
  `workline-apply.yml`.
- **GitLab's job that reads it**: `workline:merge` in
  `workline.gitlab-ci.yml`.
