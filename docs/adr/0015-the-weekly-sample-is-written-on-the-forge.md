# ADR-0015: The weekly sample is written on the forge, not in commits

- **Status:** accepted; built (`workline sample`), not yet run for real;
  amended 2026-10-02 (which forge)
- **Date:** 2026-10-02
- **Amends:** ADR-0014 (step 4's weekly sample, "written to the measures")

## Context

ADR-0014 accepted steps 0 to 2 on their measures, and closed step 4 with a
standing check: every week, one in ten of the docs vouched for since is read
against the code by a model of another provider or a person, "written to the
measures"; one false `checked` sets the documentalist back to step 0. The
measures so far are files committed to this repository
(tests/evaluation/results.tsv, roles/documentalist/tried.md). A weekly result
committed that way would be a commit a week carrying only work state — what
docs/BACKLOG.md ("Work state lives in the forge's issues, not in commits")
asks to stop — and it would need a write to `main`, which takes pull
requests only (ADR-0011).

What exists already: the evaluation's judge, named by `WORKLINE_JUDGE`, at
the best independence available and saying which (ADR-0005,
internal/judge); the removal rule's quote check, which finds a source's
words outside its comments (ADR-0014, step 2); one tracking issue kept by
the forge, and a comment kept in place by a marker (internal/forge); and the
gardening's two jobs — the agent reads, a job without its key writes
(ADR-0011, ci/github/workline-gardening.yml).

## Decision

### Drawn without AI

The docs whose `checked` the documentalist moved — its commits carry
`Workline-Role: documentalist`, its own or squashed into another's message,
or set `verified: agent:documentalist` — in the commits that reached the
branch during the last whole ISO week, by the dates of its first-parent
history. One in ten, rounded up, ranked by a digest of the week, the doc and
the commit: a rerun of the week draws the same docs. `--since <rev>` takes
the commits after one instead. `sample.after` leaves out what a commit
reaches: on workline v0.2.1, the release whose engine earns `checked` —
the docs vouched for before it were undone already (0669684), and drawn
they would set the documentalist back for what step 0 fixed.

### Read whole, by a judge that is not the voucher

The doc as it was vouched for, and every source whole as it was at the
commit `checked` names. Sources past the project's `whole-chars` could not
have been read whole: that `checked` is reported unearned, with no call —
the same as a false one.

The judge is `--judge`, else `WORKLINE_JUDGE`, else the documentalist's
`sample.judge`. Engine commits now name the agents and models whose answers
they hold (`Workline-Model: claude:claude-sonnet-5`); a judge set to that
model is replaced by the other (ADR-0005), and one answering with it is not
counted. The result says the level reached; `sample.at-least` sets a floor.
Without a judge, the docs drawn are listed for a person.

A new role, `auditor`, reads: it lists each false passage as the
catalogue's `claim`, with a source's words. The engine finds every quote —
the passage in the doc, the words in a file under its sources, outside
comments, at that commit — or a name gone from the code then. A claim it
cannot find is no proof: the passage is listed for a person, nothing is set
back.

### Written on the forge

The job that holds the forge's token and no AI key (`workline sample
--apply`) keeps one tracking issue, "workline: the weekly sample of the docs
vouched for", with one comment a week, edited in place on a rerun: the docs
read, each verdict, the judge and the independence, and each proof.

A `checked` found false (or unearned) labels the issue
`documentalist-step-0`, which a person takes off once step 0 is passed
again, and a merge request on `workline/documentalist/sample-<week>` puts
that doc's `checked` back to what it was, `judged` at the commit vouched
for — as the undo of step 0 did (0669684) — for a person to review and
merge. A doc whose `checked` moved since is listed for a person instead.

Why a merge request and not only a report: putting a `checked` back is the
step ADR-0014 asks for, it reuses the gardening's path a person already
reviews, and it is the person, merging or closing it, who confirms the
verdict (ADR-0014's amendment: a false `checked` goes to the person).

## Consequences

- The weekly result is on the forge, where it is read and answered; the
  repository's history holds only what a person merges.
- A false `checked` costs a person one review, and the label stays until
  they take it off.
- Older commits name no model: the independence of their reads is said
  `unknown`. On this repository the gardening vouches with Sonnet and the
  sample is read by Opus (`sample.judge`), ADR-0005's level 2 — no other
  provider is set up here.
- The cost is one call a doc read: a doc and its sources, at most
  `whole-chars` (20,000 characters, about 5,000 tokens) and the doc, about
  8k to 12k tokens a doc, one doc on most weeks.
- Not built: a measure of the sample in tests/evaluation (results.tsv); a
  sample read by a person instead of a judge is said on the issue, not
  recorded as a read.

## Amendment (2026-10-02)

"The forge" is where the project lives (ADR-0016): `workline sample
--apply` writes the tracking issue, its comment, its label and the merge
request on GitHub, GitLab, another forge plugged by a command, or, with
`forge: local`, in the clone — the merge request a local branch, nothing
pushed. With no forge, it writes nothing and says so. The decision is
unchanged.
