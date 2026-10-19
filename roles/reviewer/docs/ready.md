---
sources: [roles/reviewer/role.yaml, internal/builtin/reviewer/reviewer.go, internal/builtin/reviewer/record.go, ci/github/workline.yml, ci/gitlab/workline.gitlab-ci.yml]
checked: 4049d445
verified: agent:claude-code
---
# Reviewer — every lens once when it becomes ready

Part of [the reviewer](../README.md), on a merge request; the rest of a
run: [a run on code](run.md).

[#129](https://github.com/JN0V/workline/issues/129): the run on ready
reads every lens once. It is the run told `--input lenses=all`, or one
told the merge request is no draft (`--input draft=false`) whose record
does not say every lens read it
([#359](https://github.com/JN0V/workline/issues/359)). The templates pass
`lenses=all` when it is opened ready or marked ready, and `draft` on
every run, to both runs: the verdict's asks every lens that blocks, the
run apart every lens that warns, each its own record
([apart](apart.md)).

- **The whole change**, `base..head`: each commit was read by one lens
  only, and skipping them could leave nothing to read. The commits are
  recorded, one run counted, and, once every lens answered, the record
  says every lens read the change (`read=all`); the next push gets one
  lens, in turn.
- **A run on ready where a lens did not answer** (unreadable after its
  retry, its budget spent): `retried=once`, and the next push asks every
  lens again, once (`every-lens-again`). Failing again, `read=all` is
  written all the same, and the comment says the change was not
  reviewed whole by every lens (`every-lens-not-whole`).
- **A run on ready cancelled**: a push while it goes on cancels it, one
  run per merge request. It wrote nothing, so the next run reads every
  lens.
- **A draft again**: a run on a draft (`draft=true`) drops `read=all` and `retried=once`;
  ready again, every lens reads it once more.
- **A record that does not read** says nothing of it: one lens, every
  commit reviewed again. A record written before `read=all` existed: the
  next push to a merge request no draft reads every lens once.
- **GitHub**: the pull request `opened`, not a draft, or
  `ready_for_review`, which runs the line.
- **GitLab**: marking a draft ready starts no pipeline
  ([gitlab-org/gitlab#25426](https://gitlab.com/gitlab-org/gitlab/-/issues/25426)).
  The next pipeline, a push or **Run pipeline**, is the one when the
  merge request is no draft (`CI_MERGE_REQUEST_DRAFT`, else the API)
  and GitLab's note "marked this merge request as **ready**" is newer
  than the reviewer's comment, or there is no comment yet.
  - Read with `GITLAB_TOKEN` when the job has one, else the job's own
    token; unread, one lens, said in the log.
- **Cost**: a review of the whole change by every lens, as on a machine,
  within the same `ai-max-tokens`
  ([measured](../../../docs/research/code-review.md#the-reviewers-cost-measured)).
- **Held by a rule**: the run on ready asks no lens, and its record owes
  every lens (`owed=all`); the next run, the fix pushed, reads every lens
  on the whole change, whatever the template passed.
- **With no agent, or one out of reach**, nothing is written: the next
  run is the run on ready again.
- **GitLab's notes** are read page by page, the newest first, until the
  ready note or the reviewer's comment.
