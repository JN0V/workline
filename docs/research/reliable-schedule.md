# A periodic job on GitHub Actions that runs on time (2026-11-02)

**Question.** GitHub's scheduler runs workline's hourly reconciler 4 to 6
times a day. How do projects keep a periodic job on time on GitHub
Actions, with nothing outside the forge, no job sleeping on a runner,
and the job token's API budget (1,000 calls an hour)?

**Verdict.**

- **GitHub's `schedule` is best effort**, said by GitHub, seen on this
  repository: no setting makes it reliable.
- **Each run starts the next, by `workflow_dispatch`**: the job token may
  start it, and it is not queued behind the scheduler.
- **The wait between two runs is an environment's wait timer**, GitHub's
  own: the job waits in the forge, not on a runner.
- **The schedule and the line's events stay**, to start the chain again
  if it stops.
- **What was decided**: [ADR-0059, amended](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-02-the-reconciler-starts-its-next-run).

## What GitHub says of `schedule`

From [events that trigger workflows](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#schedule):

- "The `schedule` event can be delayed during periods of high loads of
  GitHub Actions workflow runs. High load times include the start of
  every hour."
- "If the load is sufficiently high enough, some queued jobs may be
  dropped."
- "In a public repository, scheduled workflows are automatically
  disabled when no repository activity has occurred in 60 days."
- Shortest interval: 5 minutes.

Community threads say the same, with no answer from GitHub staff:
runs hours late or missing, `workflow_dispatch` working meanwhile
([#201472](https://github.com/orgs/community/discussions/201472),
July 2026, a daily run 5 h 46 min late;
[#209514](https://github.com/orgs/community/discussions/209514),
October 2026, an hourly `51 * * * *` with gaps of several hours).

## What this repository saw

| Workflow | Cron | Runs seen | Gap between runs |
|---|---|---|---|
| `schedule-probe.yml` | `*/5 * * * *` | 167 from 2026-10-02 to 11-02, about 9,000 due | 3 to 7 h ([#845](https://github.com/JN0V/workline/issues/845)) |
| `workline-develop.yml`, reconcile | `41 * * * *` | 7 from 2026-10-31 17:21 to 11-01 23:02 UTC | 3 h 05 to 7 h 41, median 4 h 42 |
| `workline-gardening.yml` | `17 2 * * *` | 20 nights in October | started 4 h 37 to 7 h 05 late ([#607](https://github.com/JN0V/workline/issues/607)) |

- Off the hour (minute 41) did not help.
- An issue labelled starts a due reconcile too
  ([#789](https://github.com/JN0V/workline/issues/789)), but a quiet
  night has no event: none from 23:50 to 02:31 UTC on 11-01.

## What exists

| Way | Used by | Holds | Fit |
|---|---|---|---|
| An outside scheduler calling `workflow_dispatch` (cron-job.org, a cloud's cron, a server) | most answers in the threads above | a personal token outside GitHub | refused: nothing outside the forge |
| Several crons, the first that runs does the work | [#607](https://github.com/JN0V/workline/issues/607)'s proposal | nothing more | still the scheduler; 3 to 7 h gaps stay |
| A job that sleeps, then dispatches the next | a common answer | one runner all day; a private repository pays every minute | refused: a runner held, minutes billed |
| A watchdog workflow re-dispatching a missed run | some threads | nothing more | the watchdog is a schedule itself, dropped the same way |
| **Each run dispatches the next; the next waits an environment's wait timer** | GitHub's own pieces, put together here | no runner while waiting | **chosen** |

### The pieces, as GitHub documents them

- **`workflow_dispatch` by the job token**: "With the exception of
  `workflow_dispatch` and `repository_dispatch`, other
  `GITHUB_TOKEN`-triggered events do not create workflow runs at all."
  workline's line already starts each step this way.
- **A wait timer** on an environment: "between 1 and 43,200" minutes;
  "Wait time will not count towards your billable time." On GitHub Free,
  Pro or Team, wait timers exist for public repositories only
  ([deployments and environments](https://docs.github.com/en/actions/reference/workflows-and-actions/deployments-and-environments#wait-timer)).
- **No deployment record** a run: `environment: {name: …, deployment:
  false}`, "Wait timer protection rules still apply"
  ([control deployments](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/control-deployments)).
- **One heartbeat a branch**: a job's `concurrency` group with
  `cancel-in-progress: true`: a new one cancels the one waiting.
- **The environment limited to the default branch**: a branch's run
  cannot use it.

### When the last reconcile ran

- **Not the Actions cache**: a run started by an issue's label logs
  "cache write denied: token has no writable scopes" at the save. The
  line's own labels then each ran a reconcile, 3 in 7 minutes, each
  cancelling the heartbeat waiting
  ([#853](https://github.com/JN0V/workline/issues/853)).
- **The workflow's runs**, `GET /repos/{owner}/{repo}/actions/workflows/{id}/runs`:
  any run may read them, 100 a page, newest first: one call while fewer
  than 100 issue runs came since the last reconcile. The reconciler's
  runs are named (`run-name`), so their status says whether one waits,
  and the last that succeeded, its `updated_at`, when it ended.

## Piggy-backing on the line's runs, alone

The maintainer's first choice: each run of the line starts a reconcile
when the last is older than its period.

- **Built already** for an issue labelled, each hop of the line
  ([#789](https://github.com/JN0V/workline/issues/789)).
- **Not enough alone**: in a quiet night no run comes, and the issues
  the line left for the reconciler wait for one: a deferred API budget,
  a WIP wait, a Blocked revisit.
- **Not widened** to comments and pull requests: with the heartbeat, it
  would add short jobs and shorten no wait. It stays as a way to start
  the chain again.

## Tried on the sandbox (2026-11-02)

The template on [JN0V/workline-sandbox](https://github.com/JN0V/workline-sandbox),
with no issue, comment nor schedule trigger, so the heartbeat alone
started the reconciler; the environment's wait timer at 6 minutes.

- **The waiting job holds no runner**: GitHub listed it `waiting`, its
  `runner_name` empty, for the whole wait.
- Timings, in the pull request's
  [`## Tried`](https://github.com/JN0V/workline/pulls?q=is%3Apr+845).

## Sources

- [Events that trigger workflows, `schedule`](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#schedule)
- [Deployments and environments, wait timer](https://docs.github.com/en/actions/reference/workflows-and-actions/deployments-and-environments#wait-timer)
- [Control deployments, `deployment: false`](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/control-deployments)
- [Community #201472](https://github.com/orgs/community/discussions/201472), [#209514](https://github.com/orgs/community/discussions/209514)
