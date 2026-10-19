---
sources: [roles/reviewer/role.yaml, internal/builtin/reviewer/reviewer.go, internal/builtin/reviewer/record.go, internal/builtin/reviewer/once.go, internal/engine/engine.go, ci/github/workline.yml, ci/gitlab/workline.gitlab-ci.yml]
checked: b830e2b9
verified: agent:claude-code
---
# Reviewer — no lens on a draft, every lens once when it becomes ready, then what is new

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

## Before ready: a draft asks no agent

[ADR-0050](../../../docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md),
step 5: told `draft=true`, the engine gives every role of the line no
agent, the reviewer's two runs included.

- **What runs**: the rules, which still block; no lens, no judge.
- **What it says**: `ai-waits-for-ready`, a warning: every lens reads
  the whole change once it is marked ready. The line's summary says the
  AI review waits for ready.
- **What it writes**: no review; the last one, if any, stands. A draft
  again rewrites only its record ([below](#the-run-on-ready)).
- **The verdict**: a review to come (`waits`), never a pass: GitHub's
  `workline/verdict` pending, and the merge policy waits for the run on
  ready ([merge policy](../../../docs/merge-policy.md)).
- **Why nothing gets through**: a draft never merges
  ([ADR-0042](../../../docs/adr/0042-the-engine-merges-by-the-projects-policy.md)),
  and the run on ready reads the whole change, the draft's commits too.
- **With no agent set** (`ai: none`): nothing waits for ready; the
  review says, as on any merge request, that a person reviews it.
- **The templates install no agent on a draft**: they still name it, so
  the engine can say what waits.

## The run on ready

- **The whole change**, `base..head`: each commit was read by one lens
  only, and skipping them could leave nothing to read. The commits are
  recorded, one run counted, and, once every lens answered, the record
  says every lens read the change (`read=all`); the next push gets one
  lens, in turn, on what is new ([below](#after-ready-only-what-is-new)).
- **A run on ready where a lens did not answer** (unreadable after its
  retry, its budget spent): `retried=once`, and the next push asks every
  lens again, once (`every-lens-again`). Failing again, `read=all` is
  written all the same, and the comment says the change was not
  reviewed whole by every lens (`every-lens-not-whole`).
- **A run on ready cancelled**: a push while it goes on cancels it, one
  run per merge request. It wrote nothing, so the next run reads every
  lens.
- **A draft again**: its runs ask no lens, and the last review stands,
  its comment kept. Only its record drops `read=all` and
  `retried=once`; ready again, every lens reads it once more, even after
  a cancelled run.
- **A record that does not read** says nothing of it: one lens, every
  commit reviewed again. A record written before `read=all` existed: the
  next push to a merge request no draft reads every lens once.
- **Cost**: a review of the whole change by every lens, as on a machine,
  within the same `ai-max-tokens`
  ([measured](../../../docs/research/code-review.md#the-reviewers-cost-measured)).
- **Held by a rule**: the run on ready asks no lens, and its record owes
  every lens (`owed=all`); the next run, the fix pushed, reads every lens
  on the whole change, whatever the template passed.
- **With no agent, or one out of reach**, nothing is written: the next
  run is the run on ready again.

## After ready: only what is new

[ADR-0050](../../../docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md),
step 6: the run on ready is the one full review; after it, each push's
AI reads only what is new.

- **The commits no review read yet**: the lens in turn reads those the
  record does not hold, by commit id or patch id. A rebase, or a merge of
  the base that settled nothing, reads nothing again
  ([the record](run.md#3-the-record)).
- **The lenses that warn: five new commits a push** (`commits-per-push`),
  those changing code counted, the oldest first.
  - The five read are recorded; the rest are left, said
    (`commits-left`), with their ids.
  - They are read first at the next push, or when the run runs again on
    the same head (re-run its job).
  - `0`: every new commit, each push.
- **A lens that blocks reads every new commit**, and so does the run on
  ready: what can hold the verdict is never left unread.
- **A finding judged once keeps its verdict**, the same finding on the
  same patch: its judge is not asked again, said
  ([judged once](once.md)).
- **The head's verdict** still holds every block a review before found,
  while the head still has its cause
  ([a block stands](judge.md#a-block-stands-until-a-commit-answers-it)).

## What tells it ready, on each forge

- **GitHub**: the pull request `opened`, not a draft, or
  `ready_for_review`, which runs the line.
- **GitLab**: marking a draft ready starts no pipeline
  ([gitlab-org/gitlab#25426](https://gitlab.com/gitlab-org/gitlab/-/issues/25426)).
  The next pipeline, a push or **Run pipeline**, is the one when the
  merge request is no draft (`CI_MERGE_REQUEST_DRAFT`, else the API)
  and GitLab's note "marked this merge request as **ready**" is newer
  than the reviewer's comment, or there is no comment yet.
  - What the person sees: the draft's last summary says the AI review
    waits for ready, and asks, on GitLab, for a push or **Run
    pipeline** once it is.
  - A tool starting pipelines from GitLab's webhook does start one on
    ready ([`MR_READY`](../../../docs/gitlab-trigger.md#how-the-pipeline-knows-what-to-do)).
  - Read with `GITLAB_TOKEN` when the job has one, else the job's own
    token; unread, one lens, said in the log.
- **GitLab's notes** are read page by page, the newest first, until the
  ready note or the reviewer's comment.
