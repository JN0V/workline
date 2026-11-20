---
sources:
  - defaults/ci/github/workline-apply.yml#jobs.apply.if
  - internal/report/gates.go#Gates
  - internal/report/gates.go#GatesShown.note
  - internal/report/gates.go#GatesShown.Title
  - internal/report/gates.go#GatesShown.Summary
  - internal/report/gates.go#GatesShown.Comment
  - internal/report/report.go#maxAnnotations
  - roles/reviewer/role.yaml#settings.findings-max
  - internal/engine/annotate.go#ShowGates
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:gates
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:base
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:main
  - tests/conformance/cases/gates/code-quality-fingerprint-kept-when-moved.yaml
checked: e2e165c2d
verified: agent:claude-code
---
# What the gates' findings become

Part of [gates in CI](gates-in-ci.md): what the change adds, shown on
the merge request.

## On GitHub

The job applying the run, in workline-apply.yml, shows what the gates
found the change adds:

- **Beside each line**: the annotations of the check `workline gates`,
  on the head judged. None is a conversation to resolve. Its title gives
  the count; past 500, the rest are counted, not written.
- **One summary comment**, edited on each push: a row per check, its new
  findings and those there before; a line per rule, its first places. It
  lists as many rules as the reviewer's `findings-max` (10).
- **Nothing new, nothing posted**: no check, no comment. A comment an
  earlier push left says "no new finding".
- **A formatter's check** (`fix:` set): its command is said once, rather
  than one annotation per line.
- **Not code scanning**: paid on private repositories, and a
  conversation per alert.
- **A fork's pull request**: no annotation, no summary comment; the
  gates' verdicts are in the report workline-fork.yml comments
  ([what a fork gets](gates-fork.md)).
- **The branch moved on** before the apply: nothing shown, and said; the
  next run shows its own.
- **On the developer's own pull request**, the gates' summary comment
  starts a review round, its fix pushed on its branch
  ([the review loop](../roles/developer/docs/review-loop.md#what-wakes-it)).

## On GitLab

- **The job's Code Quality report** holds the findings the change adds,
  each as severe as its tool says. The merge request's widget shows it on
  every tier; Ultimate puts each beside its line.
- **GitLab compares it** with the report of the target branch's pipeline
  at the merge base, which holds none of these: each shows as new. With
  no pipeline there, the widget shows nothing.
- **Each fingerprint holds no line number**: a finding that only moves
  keeps it.
- **The variables**: scoped to the environment `workline`, which the job
  does not declare, the agent's token and the project bot's never reach
  it; `workline:base` reads the merge base's findings with the bot's,
  and hands them over ([the environment](gitlab/environment.md)). The
  job still unsets them, for one left to every environment. Whoever may
  push a branch can still read them
  ([ADR-0044](adr/0044-the-apps-key-stays-with-the-default-branchs-runs.md)).
  A fork's merge request runs in the fork, which has neither.
- **Its summary** is `gates-summary.html`, linked from its page: the
  propose job writes its own.
- **One more job a pipeline**, and its minutes.
