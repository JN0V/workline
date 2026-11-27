# Diagnosing a role — replaying its misses, measured (2026-11-26/27)

The benches of 2026-11-26/27 have ended. Each role's measures are on its
own page: [reviewer](diagnosing-the-reviewer.md),
[developer](diagnosing-the-developer.md),
[tester](diagnosing-the-tester.md),
[product owner](diagnosing-the-product-owner.md),
[documentalist](diagnosing-the-documentalist.md); the overview, the
pitfalls and every strategy's status on
[diagnosing-a-role-findings.md](diagnosing-a-role-findings.md).

**Question.** On 2026-11-25 an evaluation of recent versions scored the
roles 3 to 7 of 10, below the 8 a role needs to leave beta. Why does a
role miss, which change would raise it, and how do we know the change
works before we keep it? This page keeps the method and its lessons for
the future process engineer
([ADR-0019](../adr/0019-the-line-evaluates-itself.md); its issue
[#89](https://github.com/JN0V/workline/issues/89) was closed as not
planned), whose job is exactly this step back
([ADR-0068](../adr/0068-the-line-steps-back-on-its-own-drift.md),
[stepping-back.md](stepping-back.md),
[roles-panorama.md](roles-panorama.md#auditor-and-process-engineer-two-roles)).

**Verdict.**

- **A root cause is not found by one replay.** The first diagnosis
  (2026-11-25) named the agents' lack of repository access as the first
  cause of three roles, on 2 real calls. Measured on whole replay sets,
  reading moves the points whose evidence lies outside the input given.
- **Many misses are not the agent's**: a sentence the engine writes, an
  input it cuts, a filter, a timeout.
- **The model was a lever for two roles**, the reviewer and the
  developer; for the product owner and the tester, Opus gained nothing
  on the causes.
- **Every shipped change was measured on the cases it was written from**:
  its gain is an upper bound until held-out or shadow evidence.
- **The bench itself lied** before it was audited: a score by place, a
  non-production judge, arms on different commits.

## The method

A diagnosis is a measure, not a reading. Each step below exists because
skipping it misled us once (see [the pitfalls](diagnosing-a-role-findings.md#pitfalls-met)).

### 1. Replay every real miss

- Build one replay case per real miss in a window: the input as the role
  read it then (the issue's edit history, the repository at the base
  commit), a fake forge
  ([replay.md](../spec/conformance/replay.md),
  [tests/evaluation/replay/](../../tests/evaluation/replay/)).
- Add **kept catches** (what the role found and must keep finding) and
  **clean cases** (changes with no defect, to count false alarms).
- Every miss, not a chosen few: a hand-picked case confirms the
  hypothesis that picked it.

### 2. Classify each miss by cause

One independent reader per case, given the run's log, every agent call's
raw answer and the repository, never told a hypothesis. One primary
cause per missed point:

| Cause | Question | Lever it points to |
|---|---|---|
| Did not look | Was the code in front of it? | input, repository reading |
| Saw, did not say | It had the code and said nothing | model, effort, instruction |
| Severity rule | Said as a nit, a question, or as not blocking | the role's rules |
| Filter dropped | Said, then dropped by the judge or a quote check | engine filters |
| Format | Said, but the answer could not be read | answer format, the reader |
| Unreachable | Needs a maintainer's product choice, hardware, or a measure | none: report apart |
| Scoring artefact | Said, but the scorer did not credit it, or credited the wrong thing | the bench |

- Unreachable points count for no strategy, for or against.
- A miss the engine causes (a sentence it writes, a round it wakes) is
  not the agent's, whatever the role's score says.

### 3. Fix the criteria before the results

Written down before any strategy runs
([PR #2271](https://github.com/JN0V/workline/pull/2271) is the baseline
they compare with). On 2026-11-26 they were written after the first arms
had run ([#2273](https://github.com/JN0V/workline/pull/2273),
[#2279](https://github.com/JN0V/workline/pull/2279)): those arms are
read with that in mind.

- Same commit, same cases, same judge for both arms (an arm is one
  strategy run on the bench). Arms on different commits are not compared.
- A strategy is better only if misses go down beyond noise, kept catches
  do not drop, and clean cases do not get worse.
- Tokens reported fresh and with cache reads, per case run. A gain
  costing more than 2× fresh tokens needs the maintainer's yes.
- Say in advance how many strategies are tried: with 7 arms, one winner
  by chance is expected.
- Write each strategy's prediction before it runs.

### 4. Enough runs

Measured on the reviewer's set (15 miss cases, 34 points), with the
official score by place, by the bench audit:

| Runs an arm | Smallest gap that can be seen, of 102 points |
|---|---|
| 3 | 11 to 14 |
| 6 | 8 to 10 |
| 14 to 23 | 5 |

- One run moves by about ±2 points of 34 (standard deviation 1.6 to 2.0).
- A rule asking both 3-run groups to sit above both baseline groups
  passes 17 % of the time when nothing differs (simulated). Compare per-run totals with a
  two-sample test over 6 runs or more.
- Nothing inside ±10 points of 102 at 3 runs is a result.

### 5. Score by meaning, with a blind judge

- A planted point is found when a shown finding **states it**, wherever
  it is quoted: one yes-or-no question per point to a judge that is the
  same for all arms and blind to the arm
  ([PR #2282](https://github.com/JN0V/workline/pull/2282)).
- Check the judge on known answers first: the real sent-back head must
  fail each point, the accepted head pass it. The reviewer's scorer agreed
  on 30 of 30 hand-checked items. The developer's reached 26 of 26 heads
  only after its inputs and expected answers were corrected: the bench
  around a judge fails as often as the judge.
- Count apart: findings that state no point (volume), points said as a
  nit, points refused by the role's own judge.

### 6. Production-identical settings

- The bench runs the role as production runs it: model, effort, and the
  judge. In production the reviewer's judge is **the other model** than
  its lenses (`judgemark.Pick` in `internal/engine/engine.go`); a bench
  that forces both on Sonnet measures another reviewer.

### 7. A skeptic audits the bench

Before any decision, an independent agent audits the cases, the scorer
and the numbers, and re-checks the claims already made. On 2026-11-26 it
found: a case marked clean that holds a real defect, three points that
are one, points whose place is arbitrary, a check that fails a valid
fix, a judge word search that cannot match a prefix, and a tripled claim
in the first diagnosis.

### 8. Check in the real world

- The replay set tunes and judges at once: no case is held out, so any
  gain is an upper bound.
- The winner then runs in shadow on new real pull requests and issues,
  nothing posted, and is compared with what the maintainer sent back.

### 9. Never conclude from one oriented replay

A replay whose prompt was written after reading the expected answer
proves the prompt, not the cause.

## Lessons of 2026-11-26/27

### The bench

- **Score by meaning, never by place.** By place, 61 of 310 reviewer
  points were earned by a finding stating something else, and 79 true
  statements scored nothing. The documentalist's official checks passed a
  partly false rewording and called true edits false.
- **Preregister, then amend in writing.** Each role wrote its arms and
  predictions before any result (reviewer and documentalist 14:27Z,
  product owner 15:00Z, tester 15:46Z); every change came as a dated
  amendment before the runs it touched. Predictions that failed are kept:
  4 of the tester's 5 arms predicting a gain did not gain.
- **Cut clearly beaten arms early, and say what a cut leaves.** On the
  owner's token instruction (19:30Z to 19:55Z), arms below the baseline's
  range after one run, or unable to change a decision, stopped. An arm
  dropped untested is "not shown worse", never "worse".
- **Measure before lightening.** Reading lenses lost kept catches; the
  free reviewer without merge standards said 36 to 45 % of weak-test
  points, the lenses 64 to 85 %; with them, level.
- **GitHub's runners spend the same subscription.** The replay workflow
  ([#2293](https://github.com/JN0V/workline/pull/2293)) logs in with the
  line's `CLAUDE_CODE_OAUTH_TOKEN`: it frees the machine, not the tokens.
- **Check the judge against another model.** Opus judged Opus arms;
  re-judged by Sonnet, the reviewer's leaders agreed on 98 % of 156
  point-runs and the tester's arms on 98 % of 432 scenario-runs.
- **A triple shows only a large gap.** 3 runs against 3 cannot go
  below p = 0.10 by permutation; "every run above every run" is the most
  it says. Only the reviewer's leaders and the product owner's arms reached
  6 runs.

### The changes measured

- **In-sample prompts are upper bounds.** Each checklist bullet, merge
  standard and product-owner paragraph maps to a named failing case. The
  held-out evidence is thin: the product owner's one held-out bounds case
  scored 0 of 6 until v4; the documentalist's 4 held-out pages ran once
  a version.
- **One clause can move a score.** The product owner's v2 moved
  TraceGlass #8 from 6 of 6 to 0 of 6; v4 moved DomoticsCore #154 from 3
  of 6 to 0 of 6. Each wording is measured, never read.
- **A spec's scope error costs a round.** The issue for the
  documentalist's reading ([#2307](https://github.com/JN0V/workline/issues/2307))
  placed it in the task for a doc judged whole; the held-out pages are
  judged in parts, so the first head fixed 13 of 21 and was sent back
  (17 of 21 after). Name in the Scope the path the measured case takes.
