# Diagnosing a role — replaying its misses, and the pitfalls met (2026-11-26)

> **INTERIM — to be completed when the benches end.** The 6-run benches
> of 2026-11-26 are still running. Every number marked **INTERIM** below
> comes from 3 runs or fewer and is not a result yet (see
> [noise](#4-enough-runs)). The findings per role and the strategies
> are on [diagnosing-a-role-findings.md](diagnosing-a-role-findings.md),
> to be updated with the final tables.

**Question.** On 2026-11-25 an evaluation of recent versions scored the
roles 3 to 7 of 10, below the 8 a role needs to leave beta. Why does a
role miss, which change would raise it, and how do we know the change
works before we keep it? This page keeps the method and its pitfalls for
the future process engineer
([#89](https://github.com/JN0V/workline/issues/89),
[ADR-0019](../adr/0019-the-line-evaluates-itself.md)), whose job is
exactly this step back
([ADR-0068](../adr/0068-the-line-steps-back-on-its-own-drift.md),
[stepping-back.md](stepping-back.md),
[roles-panorama.md](roles-panorama.md#auditor-and-process-engineer-two-roles)).

**Verdict (so far).**

- **A root cause is not found by one replay.** The first diagnosis
  (2026-11-25) named "the agents cannot read the repository" as the first
  cause of three roles. It rested on 2 real calls on one pull request,
  one with a leading prompt. Measured on the whole replay set, repository
  reading moves only the points whose evidence lies outside the diff.
- **For the reviewer, the model is the lever that repeats**: Opus states
  about 2 to 4 more planted points a run than Sonnet, whatever the
  structure. Structure (lenses, lenses with reads, one free agent) is
  level on the total at the same model.
- **Many misses are not the agent's.** For the developer, 6 of 21
  send-backs are the engine writing "Closes #n" over the agent's own "not
  done". For the tester, the false "nothing guards it" is a sentence the
  engine writes after a name grep.
- **The bench itself lied** before it was audited: a score by place, a
  non-production judge, arms on different commits.

## The method

A diagnosis is a measure, not a reading. Each step below exists because
skipping it misled us once (see [Pitfalls](#pitfalls-met-on-2026-11-25-and-26)).

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
| Severity rule | Said as a nit, a question, "does not block" | the role's rules |
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
they compare with):

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

Measured on the reviewer's set (15 miss cases, 34 points, k runs an arm):

| Runs an arm | Smallest gap seen, of 102 points |
|---|---|
| 3 | 11 to 14 |
| 6 | 8 to 10 |
| 14 to 23 | 5 |

- One run moves by about ±2 points of 34 (standard deviation 1.6 to 2.0).
- "Both 3-run groups above both baseline groups" passes 17 % of the time
  when nothing differs (simulated). Compare per-run totals with a
  two-sample test over 6 runs or more.
- Nothing inside ±10 points of 102 at 3 runs is a result.

### 5. Score by meaning, with a blind judge

- A planted point is found when a shown finding **states it**, wherever
  it is quoted: one yes-or-no question per point to a judge that is the
  same for all arms and blind to the arm
  ([PR #2282](https://github.com/JN0V/workline/pull/2282)).
- Check the judge on known answers first: the real sent-back head must
  fail each point, the accepted head pass it. The reviewer's scorer agreed
  on 30 of 30 hand-checked items; the developer's on 26 of 26 heads.
- Count apart: findings that state no point (volume), points said as a
  nit, points refused by the role's own judge.

### 6. Production-identical settings

- The bench runs the role as production runs it: model, effort, and the
  judge. In production the reviewer's judge is **the other model** than
  its lenses (`judgemark.Pick` in `internal/engine/engine.go`); a bench
  that forces both on Sonnet measures another reviewer.
- Narrow what the bench's commit covers: a change to the summary tool
  should not make two arms "different commits"
  ([PR #2280](https://github.com/JN0V/workline/pull/2280)).

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
("read the callees and their defaults, the tests and what runs them",
for a case whose point is a callee's default) proves the prompt, not
the cause.

## Pitfalls met on 2026-11-25 and 26

| Pitfall | Measure |
|---|---|
| An oriented 2-call replay taken for a root cause | "Read tools found every point on 3 pull requests": 2 calls with tools ran, both on one pull request, one with a leading prompt. On the full set, repository reading recovers only the out-of-diff points: 6 of 34 |
| Code removal pursued by principle | Agents reading the repository, the input pickers deleted: equal results within noise at 1.6 to 2.8× the tokens (reviewer [#2273](https://github.com/JN0V/workline/pull/2273) +55 %, documentalist [#2276](https://github.com/JN0V/workline/pull/2276) ×2.2, reviewer on Opus with reads 177k against 64k a case run). Removing dead code gave 122 lines, not the 4.7k first counted. The goal is a better role, not fewer lines |
| Place-based scoring rewards volume | Any finding on a planted line earned the point: 61 of 310 points were by place only, 79 true statements scored nothing. Lens arms with 30 to 50 findings a run collected accidental credits; the free agent with 18 to 21 lost about 15 a set. By meaning, free and lenses are level at the same model |
| A bench forcing a non-production judge | Every bench forced a Sonnet judge; production judges Sonnet lenses with Opus. The production baseline had to be added as its own arm |
| Arms on different commits | The Sonnet baseline ran at another engine commit than every candidate; only one pair (Opus before and after [#2273](https://github.com/JN0V/workline/pull/2273)) shared a merge base. Opus lenses alone gave 19, 19 at one commit and 22, 21, 22 at another |
| Bench slots starved | Two bench slots on an 18 GB machine, `/tmp` in RAM; the slot script had no order, so a later run could pass an earlier one, and 6 runs × 6 arms of a role did not fit. A first-in first-out queue is needed, and leftover build folders cleaned |

Also met, smaller:

- **The case at fault**: `clean-tg-11` has an untested third scenario,
  so its 21 "false alarms" were mostly true. A developer check
  (`dc-240`) failed runs that kept a real measure, which the issue
  allowed.
- **The summary counted a case twice** when some runs had no judge
  ([PR #2280](https://github.com/JN0V/workline/pull/2280)).
- **Instructions edited on the very cases** then used to judge them:
  the product owner's kind rule was written after the maintainer
  corrected these issues. Only held-out cases say it generalises.

## Findings and strategies

Per role, and the strategies tried or queued with their status:
[diagnosing-a-role-findings.md](diagnosing-a-role-findings.md)
(**INTERIM**).
