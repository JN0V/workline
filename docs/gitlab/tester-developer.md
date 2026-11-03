---
sources: [ci/gitlab/workline.gitlab-ci.yml, internal/forge/gitlab.go, cmd/workline, Dockerfile]
checked: 8538d558
verified: agent:documentalist
judged: e8a46f7
judged-in-parts: 9a88170
---
# The tester and the developer on GitLab

Part of [workline on GitLab CI](../ci-gitlab.md). GitLab has no CI rule
on a label or a comment: the project's own webhook, pointed at its
trigger URL, starts the pipeline. The engine reads an issue's event
itself; for a comment on a merge request, a tool passes its numbers
([with GitLab alone](../gitlab-trigger.md#a-label-or-a-comment-with-gitlab-alone)).

## The line of states

Routed `issue: [tester, developer]`
([the line of states on GitLab](../gitlab-trigger.md#the-line-of-states-on-it)):

- **The issue webhook**, its URL ending `variables[WORKLINE_TASK]=issue`:
  `workline:issue` reads the event, and a state's label put on an issue
  whose step is due starts that issue's own pipeline,
  `WORKLINE_TASK=develop` and `ISSUE_IID`.
- **The jobs `workline:develop:*`** run the role its state wakes: the
  tester on `workline:needs-tests`, the developer on `workline:ready`;
  one pipeline an issue at a time.
- **A pipeline schedule, hourly**, with `WORKLINE_TASK=reconcile`:
  `workline:reconcile` starts each step and review round an event
  missed. `workline setup --yes` creates the hook; the schedule is yours
  to create.

## The tester

- **Its scheduled run**: list `tester` in the `schedule` line; the
  gardening jobs run it. Install your test runner in their image.
- **An objection** on its Draft: a tool given the note webhook starts a
  pipeline on the default branch, with
  `WORKLINE_TASK=objection`, `MR_IID` and `MR_SOURCE_BRANCH`
  ([the variables](../gitlab-trigger.md#how-the-pipeline-knows-what-to-do));
  the jobs `workline:objection:*` answer it
  ([what each job holds](../objection-jobs.md)).

## The developer

- **Its scheduled run**: list `developer` in the `schedule` line; the
  gardening jobs run it. Install your test runner in their image.
- **Its environment**: redefine `.workline:setup` in your
  `.gitlab-ci.yml`, its script your test job's setup commands; its
  judges run it before the agent and the proof
  ([the project's environment](../../roles/developer/docs/triggers.md#the-projects-environment)).
- **An issue ready**: [the line of states](#the-line-of-states) wakes it
  ([the triggers](../../roles/developer/docs/triggers.md)).
- **A review round** on its merge request: a tool given the note
  webhook, `WORKLINE_TASK=review`, `MR_IID` and `MR_SOURCE_BRANCH`; the
  jobs `workline:review:*` run it; the hourly reconcile starts one the
  webhook missed. A gate or a pipeline red starts none on GitLab yet
  ([the review loop](../../roles/developer/docs/review-loop.md#what-wakes-it)).
- **Its runner**: Claude Code boxes the agent's commands with
  bubblewrap, which the image holds. gitlab.com's shared runners let it
  start. A self-managed Docker runner's default profile does not: give
  the developer's jobs a runner with `privileged = true` in its
  `[runners.docker]`. Refused, the developer says so and does not ask
  the agent ([its shell](../../roles/developer/docs/agent.md#claude)).
- **Its bounds** on its merge request: the job `workline:bounds`, and
  what keeps a merge request from editing it out, by tier
  ([the bounds on GitLab](bounds.md)).
