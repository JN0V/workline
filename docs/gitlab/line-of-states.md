---
sources:
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:issue
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:start
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:develop
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:reconcile
  - internal/line/room.go#RoomInput
  - internal/line/payload.go#payload
  - internal/line/payload.go#readPayload
  - internal/line/payload.go#readNote
  - internal/line/payload.go#onLine
  - internal/doctor/hook.go#issueHook
checked: e2e165c2d
verified: agent:claude-code
---
# The line of states on GitLab

Part of [GitLab: pipelines a tool starts](../gitlab-trigger.md): what the
[issue webhook](../gitlab-trigger.md#a-label-or-a-comment-with-gitlab-alone)
starts.

[ADR-0059](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md),
the template's jobs ([workline.gitlab-ci.yml](../../defaults/ci/gitlab/workline.gitlab-ci.yml)):

- **The hook's URL** ends `&variables[WORKLINE_TASK]=issue`: the job
  `workline:issue` runs `workline route issue --input reconcile=list
  --input room=true`,
  the engine reading `$TRIGGER_PAYLOAD`, no agent.
- **A state's label put on an issue** whose step is due starts that
  issue's own pipeline, with the job's token, `WORKLINE_TASK=develop`
  and `ISSUE_IID`: the `workline:develop:*` jobs run the role its state
  wakes, one pipeline an issue at a time (`resource_group:
  workline-issue-<n>`).
- **The engine's own state labels** fire the hook too: that starts the
  next step, with no explicit start.
- **Room freed** (`--input room=true`): with the issue's own step, each
  issue whose step waited for room the move freed starts its pipeline,
  as Ready freed when the developer takes an issue
  ([#960](https://github.com/JN0V/workline/issues/960)).
- **The reviewer's comment on a spec**, on an issue in refine, starts
  the product owner's step again, to read it
  ([#846](https://github.com/JN0V/workline/issues/846)). The engine's
  other comments start nothing.
- **A person's comment or body edit** on an issue bearing a state's
  label names it: the answer to a block or a draft is read at once, not
  at the hourly reconcile, when the step is due; else nothing runs
  ([#1414](https://github.com/JN0V/workline/issues/1414)).
- **Hourly**, a pipeline schedule with `WORKLINE_TASK=reconcile` runs
  `workline:reconcile`: each due issue an event missed, its own
  pipeline; each developer's merge request a review round is due on,
  `WORKLINE_TASK=review`. Without the hook, it alone moves the line.
- **A config error answers 4xx**: GitLab disables a hook that keeps
  failing; the reconciler of [ADR-0059](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md)
  picks up what it missed.

Tried on gitlab.com Free, 2026-10-31: a label added, then removed, each
started a pipeline in about 1 to 2 s; the job printed the label change
([the try](../research/line-of-states.md#tried-after-the-review-gitlab-alone-starts-a-pipeline-on-a-label)).
