---
sources:
  - defaults/ci/github/workline-develop.yml
  - roles/developer/code/developer.go
  - internal/line/states.go
checked: 56278a72f
verified: agent:claude-code
---
# The triggers: the issue line on GitHub

Part of [when an issue becomes ready](triggers.md#when-an-issue-becomes-ready),
with [workline-develop.yml](../../../defaults/ci/github/workline-develop.yml).

## What starts a run

- **A person's answer**: a comment or a body edit on an issue bearing
  a `workline:` label starts its run, which asks the line, no agent,
  whether its step is due: a blocked issue's answer is read at once
  ([#1414](https://github.com/JN0V/workline/issues/1414)). A comment
  counts from the project's people or the issue's author.
- **The next step**: once `apply` applied, it asks the line, no
  agent, whether the issue's new state is due, and starts that run
  (`workflow_dispatch`, which the job's own token may start). Not
  due, it asks again after 10 and 20 s: GitHub lists a comment the
  apply just wrote some seconds later.
- **Back from a Scope ask**: a ready issue whose own pull request is
  open is no new claim. Its step is the review round on that pull
  request: the line lists it, and `apply` starts the round
  ([#1013](https://github.com/JN0V/workline/issues/1013)).
- **Room freed**: then it asks the line which issues waited for room
  its move freed, as the developer taking a ready issue frees Ready,
  and starts each one's run
  ([#960](https://github.com/JN0V/workline/issues/960)).
## The hourly reconcile, one run at a time

- **Hourly**, the `reconcile` job starts what an event missed, at most
  `reconcile-max` issues a run, the one waiting longest first, each
  its own run. Each reconcile
  starts the next, 55 minutes on by default: [the heartbeat](../../../docs/github/heartbeat.md),
  an environment to create once, its period yours to set. A reconcile
  that found nothing due and nothing waiting on a time starts none.
- **Not on the heartbeat alone**: the schedule, or a `workline:` label
  on an issue, each hop of the line, starts it when no heartbeat waits
  and the gap (55 minutes, or the variable
  `WORKLINE_RECONCILE_GAP_MINUTES`) passed since the last that succeeded, read from the
  workflow's runs. One at a time
  ([#789](https://github.com/JN0V/workline/issues/789),
  [#853](https://github.com/JN0V/workline/issues/853)). Its cost: a
  job of a few seconds and 1 API call each such event; comments start
  none.
- **One run an issue at a time**
  ([#806](https://github.com/JN0V/workline/issues/806)): a run of an
  issue's step is named `workline develop #<n>`. The reconciler and
  the night skip an issue with such a run queued or running; the next
  step starts its run even so, which ends at once if not due.
- **Not due, ended at once**: `propose` asks the line first, no agent;
  a step acted on already ends there, before your setup steps and the
  agent are installed.
- The `propose` job holds the agent and a read token, your setup steps
  run there ([the project's environment](triggers.md#the-projects-environment)); `apply` holds the
  App's key, no AI key.
