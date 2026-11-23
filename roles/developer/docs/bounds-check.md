---
sources:
  - roles/developer/code/prbounds.go
  - defaults/ci/github/workline-bounds.yml
  - defaults/ci/gitlab/workline-bounds.gitlab-ci.yml
checked: 7c650fc50
verified: agent:claude-code
---
# The bounds on the pull request

Part of [the bounds](bounds.md).

## The check

The same rules hold on its pull request, as a check, on the agent's
commits. A guarded file (`steering-file`, `gate-file`) that only a
person's commits changed (none carries `Workline-Role`, and neither its author nor its committer is a bot) does not block:
the check says a person wrote it, and it merges on the person's
`workline:accepted` alone. The agent's is refused
([ADR-0058](../../../docs/adr/0058-the-forge-guards-the-project-not-the-agents-environment.md#4-the-developers-bounds-become-a-check-on-its-pull-request)).
The check of [the bounds](bounds.md#what-it-checks), before the push,
stays a fast first pass.

- **Run**: `workline develop --bounds --merge-request <n>`, or the
  developer listed on the `merge-request` event's line
  (`routing.events.merge-request`). No agent.
- **Which pull requests**: a `workline/developer/<n>` branch, held to
  issue `n`'s bounds. Another branch is a person's: it passes, said
  (`not-a-developers-branch`), and is reviewed as theirs.
- **What it reads**: each file the head changed since its merge base,
  its rules read at the base's tip, never at the head: a pull request
  cannot loosen its own check. The head and base: `--input
  range=<base>..<head>`, else the branches as fetched.
- **Frozen tests**: from the tester's record, with each test's blob. A
  test as the tester wrote it passes; changed or deleted, it blocks,
  deleted even when the range shows no trace of it. Frozen tests it
  cannot read (a record unread, a start commit not fetched) block,
  `frozen-tests-unknown`, never a pass.
  - A test file two Drafts froze, testers adding to a file the base
    holds, is read against the issue's own Draft.
- **A refused file blocks**, named with its rule; a link out of the
  tree is `symlink-out`. Its next round or a person reverts it.
- **A file beyond the Scope passes**, said (`beyond-scope`, info),
  whoever pushed it: the reviewer's `intent` lens judges it.
- **Kept for the auditor**: a comment on the pull request,
  `sticky=developer/bounds`, its hidden record `act: bounds` with each
  file refused and each beyond the Scope, kept even when it blocks.

## In CI

- **GitHub**: [workline-bounds.yml](../../../defaults/ci/github/workline-bounds.yml),
  the check `workline bounds`; require it on the default branch, its
  source GitHub Actions ([set up](../../../docs/ci-github.md#set-up)).
  - It runs on `pull_request_target`: the default branch's copy of
    the workflow, never the pull request's.
  - It never runs the pull request's code: the base checked out, the
    head fetched for git to diff, a read token, no agent.
  - A job apart, with the App's key, keeps its verdict on the pull
    request. Another branch's pull request skips the job, which the
    rule takes as passed.
  - A workflow on the branch naming a job `workline bounds` adds a
    second check of that name: both are required, so the red one
    still blocks.
- **GitLab**: the job `workline:bounds`
  ([the file](../../../defaults/ci/gitlab/workline-bounds.gitlab-ci.yml)). GitLab
  runs a merge request's pipeline from its own branch: a pipeline
  execution policy (Ultimate) or code owners (Premium) keep the job;
  on Free, the review. `workline doctor` reads the tier
  ([on GitLab](../../../docs/gitlab/bounds.md)).

