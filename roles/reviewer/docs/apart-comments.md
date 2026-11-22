---
sources:
  - roles/reviewer/code/reviewer.go#summaryComment
  - roles/reviewer/code/lines.go#reviewOf
  - roles/reviewer/code/record.go#keyOf
  - roles/reviewer/code/record.go#loadRecord
  - internal/sample/draw.go#reviewerActs
  - defaults/ci/github/workline-warn.yml#concurrency
  - defaults/ci/github/workline-apply.yml#jobs.apply.concurrency
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:warn:apply
checked: dde7525a
verified: agent:claude-code
---
# Reviewer — the two comments and their records

Part of [the lenses that warn, apart from the verdict](apart.md).

## Two comments, two records

- **The summary comment** is the verdict's run's: what blocks, its
  record, the blocks standing
  ([more](judge.md#a-block-stands-until-a-commit-answers-it)). It says the
  lenses that warn read apart.
- **The comment of the lenses that warn**: their findings, questions and
  record, edited each run; their comments on lines in a review of their
  own.
- **Two comments, not one**: each is written by its own applying job, one
  at a time per merge request. One comment written by both could lose a
  run's write when both apply at once.
- **Each run its own record**: neither waits for the other, nor loses
  what the other did not read. A run apart cancelled by a newer push
  leaves its record as it was: the next one reads those commits too.
- **The first run apart** on a merge request reviewed before takes up the
  summary comment's commits and turn.
- **The weekly sample** draws the findings judged from both comments
  ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md)).
