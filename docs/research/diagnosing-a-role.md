# Diagnosing a role — replaying its misses, and the pitfalls met (2026-11-26)

> **INTERIM — to be completed when the benches end.** The 6-run benches
> of 2026-11-26 are still running. Every number marked **INTERIM** below
> comes from 3 runs or fewer and is not a result yet (see
> [noise](#4-enough-runs)). Sections [Findings](#findings-per-role) and
> [Strategies](#strategies-tried-or-queued) will be updated with the
> final tables.

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

## Findings per role

### Reviewer

Levers ranked on the 15 miss cases (34 points, 102 point-runs a 3-run
set), prior arms, by meaning:

| Lever | Size | Evidence |
|---|---|---|
| Model (Sonnet to Opus) | about +10 of 102, in every structure | "saw, did not say" falls from 22 to 8 (lenses), 33 to 24 (free agent) |
| Severity and decision rules | 4 to 13 | a true defect filed as a decision becomes a question for a person; weak tests "true, but I would not block" |
| Repository reading | 8 to 11, only on 4 out-of-diff points | out-of-diff points: 0 % without reads, 28 to 83 % with |
| The reviewer's own judge | 1 to 7 | it is not shown the author's "Left out" nor the title; its word search misses a prefix |
| Answer format | 2 to 4, read arms only | prose before the YAML; the re-ask is a new draw, not a repair |

- **False alarms on clean changes**: 23 of 36 come from one intent-lens
  rule, "a behaviour kept as before has no test", which the tests lens
  exempts. The free agent shows none in 6 runs.
- **Profiles differ** under level totals: the free agent finds
  out-of-diff points (67 to 83 %) and misses weak tests (36 to 45 %); the
  lenses do the opposite (0 to 17 %, 64 to 85 %).
- **Kept catches fall with reads**, by meaning: 5 to 7 of 15 against 8 to
  12 without.
- **INTERIM**: the 6-run arms (production baseline with its Opus judge,
  lenses on Opus, a verifier agent after the lenses on Opus or Sonnet,
  the free agent with merge standards, a prompt variant, effort high)
  and the judge question below are running.
- **INTERIM, the judge question**: must the judge be another model than
  the lenses? Four arms, 6 runs each, Sonnet or Opus lenses × Sonnet or
  Opus judge, plus the same calls re-scored with no judge. Self-preference
  would show as a same-model judge refusing fewer of its own lens's empty
  findings.

### Developer

From the records of 78 pull requests, 2026-11-23 to 26, and 133 review
rounds:

- **19 of 43 rounds with no fix** were woken by a label-trace comment;
  v0.91.0 stops this: a comment within 2 minutes of the same account's
  accepted label asks no round
  ([#2220](https://github.com/JN0V/workline/pull/2220)). Not yet seen in
  production on every repository.
- **A person's send-back is answered**: 15 of 18 fixed green in one
  round. The 3 others needed a workflow file, a board measure, or hit a
  merge skew.
- **Of 21 send-backs**: 8 the agent's craft (did not do what the issue
  says 4, regression or side effect 2, code it calls not opened 2), 6
  partial work presented as done, 3 round mechanics, 4 the issue itself.
- **"Closes #n … proven" is the engine's**: in 6 of 6 partial-work
  send-backs, the agent's own text said what was missing.
- **Reviewer Importants**: 0 of 14 declined findings were true and
  fixable but argued away; 1 got an empty round.
- **INTERIM**: 4 arms queued at the same commit (a checklist before the
  last message, effort high, a second pass by a fresh call, Opus),
  scored by meaning per send-back point.

### Tester

From 24 replay cases (69 scenarios), one per issue whose test plan held
a bad scenario, 2026-11-23 to 25:

- **9 weak tests**: 5 find words in a source, 3 take another or easier
  path, 1 greps where the scenario asks about text.
- **24 "kept as today, nothing guards it"**: all 24 are true of the code;
  in 10 (and partly 3 more) an existing test already guarded it. The
  engine wrote "nothing guards it" after a grep for a test name; the
  agent was not asked to look.
- One "false kept claim" of the 11-25 evaluation was a degenerate
  Validation (its numbers already hold), a product owner point.
- **INTERIM**: causes at v0.91.0 and strategies not yet measured.

### Product owner

- **Full read of five issues the maintainer re-kinded**: Opus 15 of 18,
  Sonnet 10 of 18 (3 runs each, on `main`). With repository reading
  ([#2277](https://github.com/JN0V/workline/pull/2277)): Sonnet 13, Opus
  12. **INTERIM**: 3 runs, one case.
- **Kinds alone**: 14 to 16 of 18 in every arm, level.
- One issue cannot be kinded from its body nor its repository: the bug
  needs a platform's status values not in the code. Unreachable.
- **INTERIM**: 5 strategies, 6 runs each, queued: Opus, effort high,
  repository reading, instructions on the line's bounds and own product
  questions, and reading plus instructions. Three held-out cases say
  whether the instruction generalises.

### Documentalist

- **It refuses removals it cannot cite.** In every run it found that
  nothing publishes two documented events, then left them "for a person"
  once saying "I had no source line to cite for removing the claim". The
  citation rule (every word removed is cited from a source) holds it.
- With repository reading ([#2276](https://github.com/JN0V/workline/pull/2276)):
  the same claims fixed (2 of 21), 0 runs with a false edit against 2 of
  3, at 2.2× the tokens. Opus on `main`: 6 of 14 claims, a false edit in
  2 of 2 runs.
- **INTERIM**: 6 arms queued on 6 cases (9 targets a run): `main`,
  reading, a prompt on what to cite when nothing does what the doc says,
  the citation rule off for removals, effort high, then the best with
  Opus.

## Strategies tried or queued

| Strategy | Role | Status |
|---|---|---|
| Agents read the repository, pickers deleted ([ADR-0074](../adr/0074-agents-read-the-repository-the-engine-stops-preparing-their-input.md) step 2) | reviewer | closed ([#2273](https://github.com/JN0V/workline/pull/2273)): equal within noise, +55 % tokens |
| Same | documentalist | held ([#2276](https://github.com/JN0V/workline/pull/2276)): 0 false edits, same claims, ×2.2 tokens |
| Same | product owner | held ([#2277](https://github.com/JN0V/workline/pull/2277)): equal |
| ADR-0074's bar amended to "not worse" | all | open ([#2274](https://github.com/JN0V/workline/pull/2274)) |
| One free agent with read tools, no lens, no judge | reviewer | measured ([#2279](https://github.com/JN0V/workline/pull/2279)): level by meaning at the same model; no false alarm on clean; weak on weak tests |
| Lenses on Opus | reviewer | strongest signal; a second 6-run set to confirm, then the maintainer's label |
| Verifier agent after the lenses (Opus or Sonnet) | reviewer | **INTERIM**, running |
| Prompt: decision rule, unannounced break, a test nothing runs, no test owed for kept behaviour | reviewer | **INTERIM**, running |
| Judge model: same as lenses or the other | reviewer | **INTERIM**, running |
| Effort high | reviewer, product owner, documentalist, developer | **INTERIM**, queued |
| Only an explicit request wakes a round | developer | merged in v0.91.0 ([#2220](https://github.com/JN0V/workline/pull/2220)) |
| Checklist, second pass, Opus | developer | **INTERIM**, queued |
| Instructions on the line's bounds and product questions | product owner | **INTERIM**, queued |
| Citation rule off for removals; prompt on what to cite | documentalist | **INTERIM**, queued |
| Score by meaning | bench | open ([#2282](https://github.com/JN0V/workline/pull/2282)) |
| Count a case once in the summary | bench | open ([#2280](https://github.com/JN0V/workline/pull/2280)) |
| Baseline rows of the replay set | bench | open ([#2271](https://github.com/JN0V/workline/pull/2271)) |
| Dead code removal | engine | done: 122 lines |

## To be completed when the benches end

- The 6-run tables per role, with each arm's commit, judge and runs.
- Which predictions held.
- The shadow check of each winner on new real changes.
- The tester's causes at v0.91.0 and its strategies.
