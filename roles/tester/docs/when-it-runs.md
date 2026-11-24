---
sources:
  - roles/tester/role.yaml
  - roles/tester/code
  - internal/kit/testmark
  - defaults/ci/github/workline-objection.yml
  - defaults/ci/github/workline-answer.yml
  - defaults/ci/github/workline-gardening.yml
checked: e81df7b34
verified: agent:claude-code
---
# When it runs

Part of [the tester](../README.md). Which runs wake it, and in CI.

## At once and on the schedule

- **At once**, routed on `issue` too: an issue labelled
  `workline:needs-tests` ([the hand-over](written.md)).
- **On the schedule**, weekly in the template, list it in the `schedule` line, after the product owner
  and the reviewer
  ([routing](../../../docs/spec/routing.md#the-line)):
  `schedule: [documentalist, product-owner, reviewer, tester]`.
  - It takes **one issue a run**: the first waiting for its tests, in the backlog's
    order, whose Validation names scenarios and whose Draft is not open.
  - Then as `test write`. None waiting: it says so and passes.
  - It needs `--open-merge-request`, as the gardening templates give
    it; without, `no-merge-request` blocks.
  - Install your test runner in the gardening's propose job: the tests are
    proven red there.

## On an objection, in CI

- **On an objection**, in CI:
  - GitHub: [workline-objection.yml](../../../defaults/ci/github/workline-objection.yml)
    notices the comment, with no token;
    [workline-answer.yml](../../../defaults/ci/github/workline-answer.yml) answers,
    from the default branch, one objection a run, then the next; a
    reply in its thread wakes it again, and the reconciler resumes an
    objection left waiting. Copy both.
  - GitLab: the project's note webhook, on its own trigger URL,
    starts the template's `workline:objection:*` jobs
    ([how](../../../docs/gitlab-trigger.md#a-label-or-a-comment-with-gitlab-alone)).
  - Woken by a person of the project or an App installed on it only.
- **Both** propose with no write token (`--no-apply`); a job apart
  applies, holding it and no agent's key (`workline apply <run>`; an
  objection's from the default branch, its run as data:
  [what each job holds](../../../docs/objection-jobs.md)).
