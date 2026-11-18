---
sources:
  - defaults/ci/github/workline-release.yml
  - defaults/ci/github/workline.yml#jobs.judge.permissions
  - defaults/ci/github/workline.yml#jobs.judge.steps
  - defaults/ci/github/workline-gardening.yml#jobs.dry-run.permissions
checked: e2e165c2d
verified: agent:claude-code
---
# What holds a release on GitHub

Part of [what holds a release](../release-holds.md#read-again).

- **The wait**: `judge` passes `--release-wait 30m` on the release
  pull request: main's checks not done are waited for, 30 minutes at
  most, the check pending meanwhile.
- **Read again**: [workline-release.yml](../../defaults/ci/github/workline-release.yml)
  runs the release pull request's last workline run again:
  - when the label is put on an issue or taken off, or such an issue is
    closed, reopened or deleted;
  - when main's workflows end after a push, none still running: list
    each one in its `workflow_run`, the tests' too. A run that started
    after main's push, waited for its checks and ended green is left as
    it is.
- **Its token**: `judge` reads with its own token: `issues`, `checks`,
  `statuses` and `actions` read, as
  [workline.yml](../../defaults/ci/github/workline.yml) gives; the
  gardening's `dry-run` gets the same.
