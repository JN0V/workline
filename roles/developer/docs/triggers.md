---
sources:
  - roles/developer/role.yaml
  - internal/developer/developer.go
  - ci/github/workline-develop.yml
  - ci/gitlab/workline.gitlab-ci.yml
checked: eebe4270
verified: agent:claude-code
---
# The triggers

```mermaid
flowchart LR
  ready["<b>An issue</b><br/>labelled ready"]
  night["<b>The schedule</b>"]
  line(["The project's line"])
  dev["<b>Developer</b><br/>one issue a run"]
  ready -- "issue line" --> line
  night -- "schedule line" --> line
  line --> dev
```

The developer runs where the project routes it
([ADR-0054](../../../docs/adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md),
step 5). Nothing routes it by default: the project opts in.

## When an issue becomes ready

```yaml
routing:
  events: {issue: [developer]}
```

- **CI runs** `workline route issue --target issue:<n>
  --open-merge-request`: that issue alone, if it is free to take
  ([the pick](pick.md)).
- **GitHub**: copy
  [workline-develop.yml](../../../ci/github/workline-develop.yml). The
  label `workline:ready` wakes it.
  - A label the job's own token puts starts no workflow on GitHub; the
    App's does ([the App](../../../docs/ci-github-app.md)). Without the
    App, the schedule takes the issue.
- **GitLab**: no pipeline starts on a label. A tool given the issue
  webhook starts one on the default branch, with `WORKLINE_TASK=develop`
  and `ISSUE_IID`
  ([the variables](../../../docs/gitlab-trigger.md#how-the-pipeline-knows-what-to-do));
  the jobs `workline:develop:*` run it.
- **Run by hand** with no issue: the first ready issue nobody took.

## On a schedule

```yaml
routing:
  events: {schedule: [documentalist, product-owner, reviewer, tester, developer]}
```

- **The gardening jobs run it**, on both forges: the first ready issue
  nobody took, one a run.
- **After the tester** in the line: the tester's Draft is opened only
  when the run is applied, after the developer read the issues. Both
  may take the same issue in one run, the code then written before its
  tests exist ([Missing](status.md#missing)).

## What both need

- **`--open-merge-request`**: its code goes to a pull request of its
  own. A line without it blocks (`no-merge-request`), the agent not
  asked, nothing claimed. `workline develop` always opens one.
- **Your test runner** in the judging job: the code is proven there
  ([the proof](proof.md)).
- **Two jobs**: the judge holds the agent and a read token for the
  pick; the engine gives the agent no forge token. The applying job
  holds the write token, no AI key.
- **One run at a time**: the GitHub workflow's concurrency group, the
  GitLab jobs' `resource_group`. A second run reads the first's claim.

## What a run leaves

- **An issue taken**: the claim's label and its record on the issue
  ([the pick](pick.md#the-claim)).
- **None to take**: the line's summary says "no ready issue waits for
  the developer", with each issue passed and why; nothing on the forge.

Cases: `developer/routing-schedule-runs-developer`,
`routing-ready-runs-developer`, `routing-schedule-none-ready-said`,
`routing-needs-a-merge-request`.
