# Diagnosing the product owner — measured (2026-11-26/27)

**Question.** The evaluation of 2026-11-25 gave the product owner 5 of
10. Which of its misses does v0.91.0 still make, why, and which change
removes them? The method is on [diagnosing-a-role.md](diagnosing-a-role.md).

**Verdict.**

- **Most misses were already gone on v0.91.0**: on the 16 real misses
  replayed, the Validation rules pass 22 of 30, format-only questions 15
  of 15, the feature question 5 of 5; 37 of 40 on issues they were not
  written on.
- **Three losses remained, none of model or effort**: it readies work
  only a person can do (a CI workflow, 0 of 24); it drafts from code no
  longer in the file (4 of 15); it assumes its own product questions
  after 24 hours (1 of 15 waits).
- **25 lines of instruction (S4) move the first two beyond the noise**,
  at the baseline's cost: 0 → 19 of 30 and 4 → 14 of 18. Shipped as v4
  of its text ([#2297](https://github.com/JN0V/workline/pull/2297),
  v0.92.0).
- **One clause moved a case from 6 of 6 to 0 of 6**: every wording needs
  its own measure.
- **Unproven**: the held-out bounds evidence is thin (1 case at 0 of 6,
  2 fresh ones at 9 to 10 of 12).

## The set and the judge

- **28 replay cases** ([#2292](https://github.com/JN0V/workline/pull/2292)):
  16 real misses, 8 held out, 4 to keep; each a real issue as it read
  before the role acted.
- **Scored by meaning**: one yes-or-no question a point to an Opus judge
  that reads the issues as the run left them, never the arm.
- **Checked on known answers**: 18 real issues as sent back and as
  corrected, twice each: 22 of 22 sent-back states fail, 14 of 14
  corrected ones pass; 14 sampled verdicts read by hand, 14 right.
- **Same commit** for every arm: v0.91.0 plus the cases plus its own
  change. About 1 000 case runs, one at a time under a 2G cap, outside
  the bench slots.
- **Thrown away**: a first rep, 53 runs, where the judge was given a diff
  and not the issues; fixed with a test before any counted run.

## Causes of the 21 misses

| Cause | Issues | Still lost on v0.91.0 |
|---|---|---|
| Instructions | 8 | 3: the line's bounds were never told |
| Missing input: the code picker cut the file (7 to 35 lines of 405) | 4 | 4 |
| Engine mechanics: timeouts, forced questions | 6 | 1 |
| Needs the owner's choice (unreachable, apart) | 3 | — |
| Model capacity | 0 | — |

## Strategies, preregistered

Written at 15:00Z, before any arm ran: five strategies, 6 runs each, a
class difference called at p < 0.01 (Fisher, two-sided) with both
triples in the same direction. Sonnet medium unless said.

| # | Strategy | Predicted | Found |
|---|---|---|---|
| S1 | Opus medium | bounds stay 0, small gain on absent code | bounds 0 of 20; absent 1 of 12, no gain; a format-only question asked 4 runs of 4 |
| S2 | Sonnet high | nothing | nothing, 3 runs |
| S3 | reads the repository ([#2277](https://github.com/JN0V/workline/pull/2277)) | absent up, bounds 0 | absent 9 of 11, bounds 0 of 20 |
| S4 | 25 lines: the line's bounds, what an issue quotes, own questions | bounds to most, absent partly | bounds 19 of 30, absent 14 of 18 |
| S5 | S3 and S4 | S4's gains plus absent | bounds 21 of 30, absent 18 of 18 |
| S6 | S5 plus a last check of the answer (added on the owner's request, 3 runs) | API up, kept 6 of 6 | nothing differs from S5 (every p > 0.18) |

Points earned of points × runs:

| Class (points a run) | Base | S2 | S1 | S3 | **S4** | **S5** |
|---|---|---|---|---|---|---|
| A CI file is a person's (5) | 0/24 | 0/15 | 0/20 | 0/20 | **19/30** | **21/30** |
| Quoted code is gone (3) | 4/15 | 1/9 | 1/12 | 9/11 | **14/18** | **18/18** |
| Held out (9) | 38/45 | 24/27 | 30/35 | 28/36 | 47/54 | 50/54 |
| Owner's question waits (2) | 0/10 | 0/6 | 0/8 | 0/8 | 1/12 | 5/12 |
| Kept (2) | 9/10 | 4/6 | 8/8 | 7/8 | 11/12 | 9/12 |
| Clean (11) | 54/55 | 33/33 | 44/44 | 44/44 | 64/66 | 64/66 |
| Runs | 5 | 3 | 4 | 4 | 6 | 6 |

- **S4 and S5** beat the baseline on bounds (p < 0.0001) and absent code
  (p = 0.005 and p < 0.0001), both triples. S5 against S4: no class
  differs at p < 0.01.
- **Tokens a case run, medians**: baseline 21.0k in and out (1.0k
  fresh), 23 s; S4 21.7k (1.3k fresh); S5 80k (6.1k fresh), 19 s: 6 times
  fresh, which needs the owner's yes.
- **Whole bench**: 889 case runs, 43.6M tokens in and out, 4.1M fresh;
  the judge about 2.5M.

## Cuts, and why

- **19:45Z, the owner: stop only what is surely worst or cannot change a
  decision.** S2 stopped at 3 runs (no class above the baseline in its
  first triple), S1 at 4 (no gain and a format point lost 4 of 4), S3 at
  4 (S5 holds every gain it has).
- **The baseline stopped at 5 runs** by the owner's decision, kept as
  the reference.
- **S6 added** at 20:20Z, declared before it ran, on what S5 still lost.

## The text of #2297, measured clause by clause

The line's reviewer blocked v1 on one point: an issue said obsolete only
because a quote was not at its line. Each revision ran on the same 12
cases, 6 runs, Sonnet medium, Opus judge.

| Class | v1 | v2 | v3 | v4 (shipped) | v5 |
|---|---|---|---|---|---|
| Bounds, 5 first cases | 19/30 | 9/30 | 16/30 | 16/30 | 14/29 |
| — TraceGlass #8 | 6/6 | **0/6** | 6/6 | 6/6 | 5/6 |
| — DomoticsCore #154 | 5/6 | 3/6 | 3/6 | **0/6** | 4/6 |
| — workline #1599 (`split` off) | 0/6 | 0/6 | 0/6 | **4/6** | 0/6 |
| Bounds, 2 fresh cases | not run | 9/12 | 9/12 | 10/12 | 10/12 |
| Absent code | 14/18 | 17/18 | 16/18 | 16/18 | 16/18 |
| Clean | 36/36 | 36/36 | 36/36 | 36/36 | 36/36 |

- **v2** added a fallback clause (ask where `split` is off): the role
  named its sources and stopped. Bounds fell (p = 0.019), TraceGlass #8 6
  of 6 → 0 of 6. Dropped.
- **v3** kept only the quotes change: held against v1. **v4**, an
  independent review's five points: held against v3, said workline #1599's
  person's part for the first time, and lost DomoticsCore #154 (the new
  `sources` clause read as a reason to wait). **v5** was not pushed:
  #154's limit question stayed at 1 of 6, under the owner's bar of 3.
- **The robust fix is the engine's**: a person's part split at refine
  ([#2186](https://github.com/JN0V/workline/issues/2186)).

## Held out, and unproven

- **In-sample**: S4 and S5 were written after reading the bounds and
  absent cases. The one held-out bounds case, workline #1599, scored 0 of
  6 under every text until v4. The unit is the case: 4 in-sample bounds
  cases, so the point-run p-values overstate.
- **Kinds read in full** were not run here; another bench gave Sonnet 4
  of 9 and Opus 9 of 9, 3 runs.
- **No shadow run on new issues.** What would refute v4: an issue said
  obsolete or asked about wrongly; a CI-workflow issue readied for the
  developer again.
- **The 24-hour timeout** that readies a bug's drafts is the owner's
  rule; it carried four of these issues to the developer.
