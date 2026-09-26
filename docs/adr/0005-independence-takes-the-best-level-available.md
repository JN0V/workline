# ADR-0005: Independent judgement takes the best level available, and says which

- **Status:** accepted, not built
- **Date:** 2026-09-26

## Context

A reviewer, or the evaluation's judge, that shares the author's model shares
its blind spots. The model grid asked for another provider, and when none was
available, ran the role without AI (docs/spec/model-grid.md). With one
provider — the common case, on one subscription — no role judging another's
work could run with AI at all: "better" had become "nothing".

Independence has levels, not one:

1. **Another provider**: other training, other blind spots.
2. **Another model of the same provider**: code written by Opus, reviewed by
   Sonnet, or the other way round; blind spots partly shared.
3. **The same model, in a context of its own**: the reviewer sees the change,
   never the author's reasoning, with a persona of its own. Much of a review's
   worth comes from there: the author rereads with its own reasons in mind
   (docs/research/gates.md: "the reviewer sees only the pushed branch, never
   the implementer's reasoning").
4. **No AI**: the mechanical checks alone.

Which of levels 2 and 3 catches more — Opus reviewed by Sonnet, or by another
Opus with a fresh context — is not known, and was not measured.

## Decision

- **Best available.** `independent-of` takes the highest level this machine
  can reach, instead of provider or nothing.
- **Never silent.** The verdict says the level reached and the models on both
  sides, e.g. `independence: model (claude-opus-5-5 → claude-sonnet-5)`; the
  evaluation records it next to each judged score.
- **A floor when it matters.** A role or a project may require a minimum
  (`at-least: provider`); below it, the role runs without AI, as before.
- **Measure it.** Evaluation cases with defects planted on purpose, reviewed
  by each model and level, say what each one catches (docs/BACKLOG.md).

## Consequences

- The reviewer no longer waits for a second provider; the evaluation's judge
  can run on another Claude model now, marked as such.
- A score or a review is read with its level: one reached at level 3 is worth
  less than one at level 1, and says so.
- Until the measurement exists, the order between levels 2 and 3 is a guess.
