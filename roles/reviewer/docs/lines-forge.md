---
sources:
  - internal/forge/review.go#LineReviewer
  - internal/forge/review.go#errRefused
  - internal/forge/review.go#review
  - internal/forge/review.go#Unplaced
  - internal/forge/review.go#Reviewed.Section
  - internal/forge/github_review.go#github.Review
  - internal/forge/github_review.go#github.shownLines
  - internal/forge/github_review.go#github.postLines
  - internal/forge/gitlab_review.go#gitlab.Review
  - internal/forge/gitlab_review.go#gitlab.shownLines
  - internal/forge/gitlab_review.go#gitlab.postLines
  - internal/engine/review.go#applier.lineReview
  - roles/reviewer/code/lines.go#reviewOf
  - roles/reviewer/code/lines.go#shortMessage
  - internal/report/report.go#ForCodeScanning
  - defaults/ci/github/workline-fork.yml#jobs.comment
checked: c46a93d6c
verified: agent:claude-code
---
# Reviewer — the forge's own review

Part of [each finding on its line](lines.md).

## The forge's own review

| Forge | What it posts | A line its diff does not show |
|---|---|---|
| GitHub | one pull request review, event `COMMENT`, each comment on its line at the head reviewed | refuses the whole review (422) |
| GitLab | a discussion on each line; a kept line given with its line before the change too | taken as a discussion on no line, silently |

- **Never an approval**, nor a request for changes
  ([ADR-0020](../../../docs/adr/0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md)).
- **Once, not twice**: on a merge request, its findings stay out of the
  SARIF for code scanning, which would open a second conversation each
  ([#412](https://github.com/JN0V/workline/issues/412)).
- **The diff read first**: the pull request's files, the merge request's
  diffs; a line the forge does not show is known before anything is posted.
- **The summary comment** keeps a row for each finding on a line, its
  title only, and links to the review: the one posted, else the merge request's
  changes, where the comments show. The lenses that warn, read apart, post
  a review and rows of their own, in their own comment ([apart](apart.md)).
- **Other forges** (`local`, `cmd:`) take no comment on a line: each
  finding, in full, in the summary comment.

## When it cannot be on its line

Never a failed run for it: each such finding goes, in full, into the
summary comment under **Here only**, with why.

- **A line the merge request's diff does not show**: a range reviewed
  other than the merge request's, a file the forge gives no patch of
  (binary, too large).
- **A forge that refuses** (403, 422, 400; on GitLab 405, 406, 409 too):
  a token that may not review, GitHub's without `pull-requests: write`.
  Said in the run's findings too, `line-comment-refused`, a warning, and
  in the job's summary.
- **A fork's merge request**: nothing is applied there. On GitHub the
  fork workflow comments the job's summary, each finding in full; on
  GitLab nothing is commented
  ([ci-gitlab](../../../docs/ci-gitlab.md#what-the-template-does)).
- **A forge out of reach**: the run stops, resumable
  (`workline apply <run>`); the comments already posted are found open,
  not posted twice.
