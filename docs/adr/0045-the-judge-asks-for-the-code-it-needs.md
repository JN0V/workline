# ADR-0045: The judge asks for the code it needs; the engine reads it at the reviewed commit

- **Status:** accepted; built in the order below, on by default once measured
- **Date:** 2026-10-14, for [#362](https://github.com/JN0V/workline/issues/362)
- **Decided by:** the maintainer, 2026-10-14: caps, budget, `unsure`, scope
- **Builds on:** [ADR-0005](0005-independence-takes-the-best-level-available.md)
  (independence), [ADR-0020](0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md)
  (the judge's material, `ai-max-tokens`); principles 5, 6, 7, 10, 11, 12
- **Research:** [a judge that reads the code](../research/judge-reads-code.md),
  [the judge, measured](../research/judge-measured.md)

## Context

- **Measured on 2026-10-14**: the reviewer's judge's "yes" right 19 of 20,
  its "no" 12 of 20: 8 true findings dropped, 2 worth raising.
- **It refuses what it is not shown**: 140 of its 173 refusals say the
  material was not shown or the finding guesses; 7 of the 8 wrong ones do.
- **Where their proof lay**
  ([read at each commit](../research/judge-reads-code.md#where-the-proof-lay-the-8-refusals-measured)):
  3 in a function longer than `judge-lines-max` (the cause's own, one it
  calls, one calling it); 1 two calls away; 1 in a constant, a setting
  and another package; 2 in a missing test, which only a search shows.
  The eighth weighed code it had.
- **What it gets today**
  ([its material](../../roles/reviewer/docs/judge.md#7-the-judge)): the
  finding, the cause's function and those it reaches up to
  `judge-lines-max` (200) lines, the change near the cause; it answers one
  note, yes or no.
- **How it is called** ([role contract](../spec/role-contract.md), step
  3): `claude` with no tools, outside the repository; a `cmd:` the same
  way, prompt in, answer out. With no agent, no judge.
- **A judge call** took 3.9k to 7.0k tokens in on
  [#280](https://github.com/JN0V/workline/pull/280)'s run; 3.4k to 8.4k on
  [the reviewer's tries](../../roles/reviewer/docs/tried.md).

## Options

- **A. A larger context, chosen by the engine**: a higher
  `judge-lines-max`, the callers' callers, the constants and settings
  named, the tests whole.
  - For: no new protocol; every agent as today.
  - Against: 3 of the 7 at most, with a cap near 600 lines; never a
    missing test, nor a name two calls away. Paid on every call, the
    right "yes" included: about 13k tokens more a judge for 400 lines
    (estimate: 37 characters a line in workline's Go, line numbers added,
    0.82 token a character).
- **B. The judge reads with its agent's own tools**: `claude` given its
  read tools in a checkout at the reviewed commit, turns capped; a `cmd:`
  wires its own agent's.
  - For: what review products and research judges do.
  - Against: only agents with read tools, each with its own names,
    boundary and caps, in turns or dollars, never lines; read-only and the
    cap are the agent's promise, not the engine's (principle 6); the
    engine does not see what was read, so cannot check a quote against it;
    a checkout at the commit when the working tree moved.
- **C. "Cannot tell", shown not verified**: a third answer.
  - For: no token more.
  - Against: "not shown" or "guesses" is in 140 of 173 refusals and in 9
    of the 12 right ones: about four refusals in five would come back to
    a person (estimate from that word match).
- **D. The judge asks, the engine reads** (chosen): the judge may
  answer with reads; the engine serves them from git at the reviewed
  commit, within caps, and asks again.
  - For: any agent that answers text — `claude` as today, a `cmd:`, the
    conformance cases' fake agent; the engine bounds, serves, records and
    checks each read; git only (principles 10, 11).
  - Against: a call more for each judge that reads; a protocol of
    workline's own.

## Decision: D, with C as its last step

### What the judge may ask

- **Where**: the judges of the reviewer's code review first; the task
  says it may read, and what is left.
- **Other judges** (the spec review's, the documentalist's, the product
  owner's) read only once a measure shows they lack the code the same
  way; the widened auditor
  ([ADR-0046](https://github.com/JN0V/workline/pull/385)) supplies those
  measures.
- **Its answer** is a note, or a list of reads, each saying why:
  - a file's lines: `read: {path: internal/engine/engine.go, lines: 242-330}`;
  - a definition by name: `read: {name: Decide}`, found as the material's
    functions are, the cause's file then its folder first;
  - a word's places: `read: {grep: XDG_CONFIG_HOME}`, `path:line` and the
    line each.

### How the engine serves them

- **At the reviewed commit, from git's objects** (`git show`, `git grep
  -w` on the commit): read-only by construction; only what the repository
  tracks, never the working tree, an ignored file, nor a path outside it.
- **Caps**, decided:
  - `judge-read-rounds`, 2: the rounds of reads a judge may ask;
  - `judge-read-lines-max`, 400: lines served to one judge, all rounds
    together; past it, cut, the rest named;
  - a search past 50 places is not served: narrow it, the round counted.
- **Then the judge is asked again**, a new call: the same question and
  material, what it read, what is left. The last round must answer.

### What a no must hold, what the engine checks

- **A no quotes the lines that refute the finding**, with their path.
- **The engine checks**: the finding's own quotes, found again before any
  judge (as today); each read at the commit, inside the repository,
  within the caps, a refused one said to the judge; the no's quote, found
  again at the commit in what the judge was shown or read; the rounds and
  their tokens, in `out/review.json`.
- **A no not found again, or with no quote**, does not drop the finding:
  shown not verified, said (`judge-no-unquoted`).
- **`unsure`**, a third answer after reading: a judge that still cannot
  tell says so. The finding is shown not verified, never dropped, never
  blocking, with why (`judge-unsure`); its rate is measured on the
  replay. A yes stands as today.

### Each agent

- **`claude`**: called as today, no tools, outside the repository, a new
  call a round; its reads are an answer, not tool calls.
- **A `cmd:`**: the same text in and out; one that never asks to read is
  judged as today. A judge of another provider
  ([#80](https://github.com/JN0V/workline/issues/80)) needs no read tools.
- **None**: no judge, as today. **The fake agent**: reads, then a note.

### Independence

- **Picked, and its level said, as today**
  ([ADR-0005](0005-independence-takes-the-best-level-available.md)).
- **It reads only files at the reviewed commit**, chosen by itself: never
  the finder's answer, the lens's prompt, the commit messages, the merge
  request nor its comments; what the author says stays a claim the lenses
  contest ([ADR-0020](0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md)).
- **What it read is said** beside its answer and on a dropped finding's
  line. The files are data: a judge can only ask reads and answer.

### The token budget

- **Each round is a call**, counted in `ai-max-tokens`.
- **The reviewer's budget rises from 200000 to 300000**, so a large
  review is judged whole. A run spends what it uses: one under 200k costs
  the same; the most one run may spend grows by half.
- **Estimates**: a judge that does not read, as today plus about 0.2k
  tokens; one reading 150 lines once, about 18k against 6.5k on the run
  in Context; under 50k at the caps. That run (one lens, five judges,
  64k): about 110k if four read once. A review like
  [#146](https://github.com/JN0V/workline/issues/146)'s (every lens, seven
  judges, about 160k): about 250k if five read once, under 300000.
- **First rounds first**, past that: every judge answers once before any
  read is served. One whose reads the budget stops is taken as `unsure`:
  no finding is left unjudged by a read.

## Measured again before it is on by default

- **The set**: the 40 verdicts of the measure, each question and material
  as asked, kept from the run's artifacts (step 0 below).
- **Two arms**, the same questions and judge model as then: today's judge,
  and the judge that reads, the repository at each commit. Today's arm
  tells the reads' effect from the judge's changes since.
- **It holds when**, against today's arm: fewer of the 8 true findings
  dropped; the yes right 19 of 20 or more; no more of the 12 false
  findings let through. The `unsure` rate, reads and tokens a judge are
  said; the maintainer reads them before the default changes.
- **Its cost**: about 0.8 million tokens in, 2.1 million if every judge
  read to its caps (estimates: 80 first calls near 6.5k, then the reads).
- **Until then** `judge-read-rounds: 0`: today's judge; a project may
  turn it on.

## Built in small pull requests, conformance first

0. **First, before they expire** (about 2027-01-02): the 40 questions
   copied from the `workline-runs` artifacts into `tests/evaluation/`,
   with their commit, lens and truth.
1. **Cases with a fake judge**: it reads a function outside its material
   and answers from it (the issue's first scenario); a read past the cap;
   a path untracked or outside; the commit read, not the working tree; a
   no not found again, and `unsure`, shown not verified; a short budget.
2. **The protocol** (`read`, a no's quote, `unsure`) and its rounds; the
   reads served by the reviewer, as its material finds code; the
   reviewer's `ai-max-tokens` at 300000.
3. **The measure** on the set; then the default.

## Not chosen

- **A, B, C alone**: above. **Several findings to one judge**: a judge
  answers one question ([ADR-0020](0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md)).
- **The judge reading the merge request or its comments**: the author's
  account is the lenses' to contest.
- **Memory of earlier verdicts**: it carried errors from one verdict to
  the next ([research](../research/judge-reads-code.md)).
- **Lowering `judge-lines-max` now**: today's material keeps the yes
  right; the measure may show the reads let it shrink.

## Consequences

- More tokens where the judge would refuse; fewer true findings dropped,
  if the measure says so. A no rests on quoted code, or says it cannot tell.
- A loop other judges may use, each once a measure shows the need.
- The reviewer's measure ([#360](https://github.com/JN0V/workline/issues/360))
  runs on this judge.
