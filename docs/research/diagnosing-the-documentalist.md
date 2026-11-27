# Diagnosing the documentalist — measured (2026-11-26/27)

**Question.** The evaluation of 2026-11-25 gave the documentalist 4 of
10: docs stamped while false claims stayed. Why does it not repair, and
which change makes it fix a false claim itself without a false edit?
The method is on [diagnosing-a-role.md](diagnosing-a-role.md).

**Verdict.**

- **Reading the repository alone is not better**: 2, 2 of 9 against
  `main`'s 3, 2. It trades claims: it finds an event nothing publishes,
  and misses what the parts find.
- **Half of what reading does not repair was seen and left in a note**,
  held by the citation rule ("no code to quote for an absence"). A prompt
  on what to cite for an absence (arm C) cut that from 5 of 14 to 1 of
  21 and scored 3, 4, 4 of 9.
- **Held out, one run each**: C fixed 18 of 21 known false claims on 4
  DomoticsCore pages, production 9; no false edit in either.
- **Shipped**: a doc too long for a task read in place
  ([#2289](https://github.com/JN0V/workline/pull/2289), v0.92.0); the
  agent reads the repository and cites
  ([#2313](https://github.com/JN0V/workline/pull/2313), v0.93.0), 17 of
  21 held out, one short of the issue's threshold.
- **Unproven**: one run a version on the held-out pages; the tuning set
  is 3 cases of one repository.

## The set and the reader

- **Cases**: DomoticsCore #295 with its 7 wrong claims (the owner's), a
  true page to vouch for (dc-293), a spec page "left to a person" on 19
  workline pull requests (wl-2224); the 4 new cases are in
  [#2286](https://github.com/JN0V/workline/pull/2286).
- **Primary score: 9 targets a run.** Three targets (two README bumps,
  one table sign) were moved out at 15:01Z as ambiguous, on the first
  run of arm A alone.
- **A blind reader** (an agent given the diffs under shuffled ids, the
  facts and the repository, not the arm) decides fixed, false edit, or
  not. Fable read arms A and B; Opus read C and the held-out runs.
- **The official checks mislead twice**: `file-lacks` passes a rewording
  still partly false, and the `judge` check, with the case's facts and no
  repository, called true edits false.
- **Held-out set**, made at 15:20Z before arms C to F ran: 4 DomoticsCore
  pages no arm was tuned on, read against the code by independent
  readers: 21 false claims they were sure of.

## Causes, before any strategy

Counted on 49 claim-runs of dc-295 and 20 runs of the other cases:

| Cause | Where | Count |
|---|---|---|
| Input: no part of the sources can show an absence or a missing row | `main`, parts | 15 of 23 misses |
| Seen and left in a note: the citation rule, "for a person" | reading | 5 of 10 misses |
| Read and not noticed: 3 docs and 29 files in one call | reading | 5 of 10 misses |
| `checked` not earned: sources do not cover the page, or Grep reads not counted | both | dc-293 every run |
| No agent asked: the page past the task's size, an old stamp holding a doc | both | wl-2224 every run |
| The engine's filters: a quote check, a withheld fix | `main` | 2 |
| After the role: a fix proposed in a comment, applied by nobody | both | the README bump |

## Arms

Said in advance: 6 comparisons at most, 6 runs an arm, Sonnet medium.

| Arm | What it changed | Predicted, of 9 |
|---|---|---|
| A | `main` | 1–2 |
| B | reads the repository ([#2276](https://github.com/JN0V/workline/pull/2276)) | 2–4 |
| C | B, plus what to cite when nothing does what the doc says; fix, not a note | 4–6 |
| D | B, the citation rule off for removals | as C or +1 |
| E | B at effort high | B + 0–1 |
| F | the best with effort high or Opus | best + 0–1 |

| | A, `main` | B, reading | C, reading and the prompt |
|---|---|---|---|
| Primary of 9, per run | 3, 2 | 2, 2 | 3, 4, 4 |
| dc-295 claim 2 (an event nothing publishes) | 0 of 5 | 0 of 2 | 3 of 3 |
| False edits (reader) | 0, 0 | 0, 0 | 0, 0, 0 |
| Tokens a run, cache included / fresh | 513k / 247k (6 cases) | 905k / 143k (6 cases) | 662k / 105k (3 cases) |

- C against B: every C run above B's best; p = 0.10 with 2 B runs, not a
  decision. wl-2224 got no agent call in any arm.
- Opus on `main` (another bench, 2 runs) missed the absent claims too.

## Cuts, and why

- **16:20Z, the coordinator**: the documentalist's runs stopped until the
  other roles' queues drained. A and B had 2 runs each.
- **The owner's token budget**: A stopped for good (0 of 7 runs reach the
  absent claims: an input cause, not chance); the ambiguous cases and the
  `judge` check dropped in every arm; E stopped, **not shown worse**.
  C ran first with a stop rule; D and F only if C failed or won.
- **C's stop rule did not fire**, and its 6-run test was left to the
  owner. The held-out check ran instead, on the skeptic's request.

## Held out: one run each, 21 claims

| Version | Fixed / noted / missed | False edits | Calls | Fresh tokens |
|---|---|---|---|---|
| Production (A) | 9 / 7 / 5 | 0 | 20 | 341k |
| C (#2276 and the prompt) | 18 / 2 / 1 | 0 | 1 | 120k |
| #2313, first head | 13 / 4 / 4 | 0 | 24 | 353k |
| #2313, after its send-back | 17 / 0 / 4 | 0 | 20 | 350k |

- **#2313's first head did not reach these pages**: the issue's Scope
  ([#2307](https://github.com/JN0V/workline/issues/2307)) placed the
  reading in the task for a doc judged whole, and all 4 pages are judged
  in parts. Sent back, its second head gives the reading line and a
  read tree to every part and fix task.
- **The 4 left** need a search of the project: a feature the code does
  not have, two test counts and a total in files outside the sources. 18
  of 20 calls answered from the excerpts in one turn. Follow-up:
  [#2317](https://github.com/JN0V/workline/issues/2317).
- **No regression with no agent**: the zero-token replay checks are
  identical on the head and on `main`.

## Unproven

- One run a version on the held-out pages; 17 against C's 18 is within
  one run's spread.
- B and C were never run 6 times; D and F never ran.
- C's prompt was written after B's notes on the claims it then fixed.
- Mixed readers: Fable read A and B, Opus read C and the held-out runs.
- [#2276](https://github.com/JN0V/workline/pull/2276) stays on hold;
  [#2313](https://github.com/JN0V/workline/pull/2313) built the reading
  on `main` instead.
