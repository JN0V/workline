# What a run costs, in tokens, per role

Token figures measured on real repositories, kept here so the user docs
say what a setting does and link to what it cost. The numbers are those
the user docs held before they moved; each row names the run it came
from and its date.

**Question.** What does each role's call spend, so a person can size a
cap?

## Measures

### Documentalist

- **A doc judged whole**: 13k to 26k tokens on a real repository
  (commit `3717b4e0a`, 2026-10-01). One too large, judged in parts,
  130k to 250k
  ([measured](https://github.com/JN0V/workline/issues/82#issuecomment-6163487736),
  [ADR-0009's amendment](../adr/0009-docs-far-behind-are-judged-in-parts.md#amendment-2026-10-14)).
- **A README judged in five parts, per run**: about 200k tokens a part,
  192.8k and 206.0k (commit `497155fcb`, 2026-10-01).
- **A doc judged in parts, on each merge request**: 6 calls at 102,324
  to 103,611 tokens in, in 5 runs of 5; 7 calls at 128,967 and 129,849 on
  pull requests [#436](https://github.com/JN0V/workline/pull/436) and
  [#444](https://github.com/JN0V/workline/pull/444) (issue
  [#458](https://github.com/JN0V/workline/issues/458), 2026-10-17). That
  is why gardening judges it once, fixed by pull request
  [#460](https://github.com/JN0V/workline/pull/460) (2026-10-17).
- **Reading pages for their reader**: about 36k tokens a run on workline,
  3 pages. Measured on workline at v0.30.0 with a recording agent (no
  tokens spent): a prompt of 43,187 characters, about 36k tokens
  estimated (711 + 0.82 a character); each page about 9k to 10k of it
  (pull request [#417](https://github.com/JN0V/workline/pull/417),
  2026-10-15).
- **The weekly sample's read of a doc**: about 8k to 12k tokens a doc
  ([ADR-0015](../adr/0015-the-weekly-sample-is-written-on-the-forge.md),
  2026-10-02).
- **A refusal that sent the whole task again**: two docs, 35k tokens,
  for one place the engine had already named (commit `d890be583`,
  2026-10-02).

### Product owner

- **One call a run**: 45k to 62k tokens in for four issues on a real
  backlog, ten runs (commit `5bc542fbf`, 2026-10-03). Up to 97.6k with
  their code, in standard tier with a context budget of 120k tokens,
  estimated (pull request
  [#238](https://github.com/JN0V/workline/pull/238), 2026-10-06).
- **Naming an issue's kind**: 400 to 600 tokens an issue beside a full
  read; alone, about 6.4k tokens for 10 issues, against 24.5k with the
  full instruction, by the engine's estimate on a real backlog
  ([#885](https://github.com/JN0V/workline/pull/885#tokens-per-issue),
  2026-11-02).

### Reviewer

- **A change in parts**: two parts spend at most two budgets, about 200k
  tokens, leaving the judges a third of `ai-max-tokens`. This is a bound
  from the settings, not a measure: 2 times the 100,000 context budget
  against an `ai-max-tokens` of 300000 (pull request
  [#450](https://github.com/JN0V/workline/pull/450), 2026-10-17).
- **A spec on the forge**: 7k to 37k tokens in, judges included; five a
  night stay within `ai-max-tokens`
  ([measured](code-review.md#the-reviewers-cost-measured), pull request
  [#434](https://github.com/JN0V/workline/pull/434), 2026-10-17).

### Tester

- **The judge's call on a missing name**: 2k to 7.5k tokens in a call, on
  a real build
  ([#1252](https://github.com/JN0V/workline/pull/1252#tried),
  2026-11-09).
- **A curator's removal of redundant tests**: nothing redundant in three
  folders (`internal/engine`, `internal/scale`, `internal/agent`), 14 to
  25 thousand tokens in, under 900 out, 7 to 10 s a run; a planted
  duplicate in `internal/scale`, found and removed, 15 thousand tokens
  in, 1,100 out (pull request
  [#668](https://github.com/JN0V/workline/pull/668), 2026-10-28).

### Auditor

- **One call an act**: about 4k to 12k tokens; about 380k on a week as
  busy as workline's busiest, estimated
  ([ADR-0046](../adr/0046-the-auditor-re-checks-every-roles-acts.md#cost-a-week),
  2026-10-14).
