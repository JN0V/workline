# Round 6 root causes — why the tester and the line keep taking fixes (2026-11-08)

**Question.** Round 6 of the trial ([#757](https://github.com/JN0V/workline/issues/757#issuecomment-6719368180)),
on a Kotlin/Android copy, opened six defects
([#1192](https://github.com/JN0V/workline/issues/1192) to
[#1197](https://github.com/JN0V/workline/issues/1197)) and missed five
targets. `internal/tester` and `internal/codepick` had each taken five
fix pull requests in the 48 h before it, which trips the stop on fixes of
[ADR-0063](../adr/0063-every-open-issue-is-due-waiting-or-done.md#working-agreements).
What few causes lie under all of them?

**Verdict.** Five causes, and one reason the fixes repeat:

| # | Root cause | Explains |
|---|---|---|
| [RC1](#rc1-the-red-first-proof-is-shaped-on-one-toolchain) | The red-first proof is shaped on one toolchain; others are taught one message at a time | [#1192](https://github.com/JN0V/workline/issues/1192); 3 of 4 tester misses |
| [RC2](#rc2-a-cut-of-pushed-code-is-read-as-a-signal) | A cut of the code pushed into a prompt is read as a signal, the size of the issue among them | [#1193](https://github.com/JN0V/workline/issues/1193); every size crossing; tester tasks 55–120 KB |
| [RC3](#rc3-the-size-judgement-shipped-unmeasured) | The product owner's split-or-grant judgement shipped untried on a real agent, without the facts it needs | feature granted, atomic bug split |
| [RC4](#rc4-the-line-still-decides-by-a-fingerprint) | The line still decides "due" by a fingerprint, and reads facts per caller; the step that replaces it was passed over | [#1194](https://github.com/JN0V/workline/issues/1194) (stall), [#1195](https://github.com/JN0V/workline/issues/1195), [#1196](https://github.com/JN0V/workline/issues/1196); product owner calls 3/15/7 |
| [RC5](#rc5-a-run-has-caps-per-call-not-per-run-and-its-end-is-untyped) | A run has caps per call, not per run, and how an agent stopped is not part of its outcome | [#1197](https://github.com/JN0V/workline/issues/1197), [#1194](https://github.com/JN0V/workline/issues/1194) (turn cap) |
| [M](#m-why-the-fixes-repeat-tried-below-the-real-path) | Fixes are tried below the role's real path; the trial is the first end-to-end run | the 24 tester and picker commits since 2026-11-04 |

Read-only, on `main` at [`41983fc9`](https://github.com/JN0V/workline/tree/41983fc9),
the engine the round pinned. Method: five whys
([root-cause.md](root-cause.md)), each chain ended on a mechanism seen
in the code or the run artifacts, never on "a missing case".

## Evidence used

- The round's report and the six issues.
- Run notes and artifacts (`line.json`, `calls.jsonl`) of the 98 agent
  calls, kept by the trial's operator.
- `git log --since=2026-11-04 -- internal/tester internal/codepick`:
  24 commits. Sorted by what they fixed:
  - **a toolchain's shape**, 10:
    [#1066](https://github.com/JN0V/workline/pull/1066),
    [#1114](https://github.com/JN0V/workline/pull/1114),
    [#1155](https://github.com/JN0V/workline/pull/1155),
    [#1180](https://github.com/JN0V/workline/pull/1180),
    [#1181](https://github.com/JN0V/workline/pull/1181),
    [#1182](https://github.com/JN0V/workline/pull/1182),
    [#1184](https://github.com/JN0V/workline/pull/1184),
    [#1186](https://github.com/JN0V/workline/pull/1186),
    [#1188](https://github.com/JN0V/workline/pull/1188),
    [#1190](https://github.com/JN0V/workline/pull/1190);
  - **what code a prompt holds**, 9:
    [#1024](https://github.com/JN0V/workline/pull/1024),
    [#1049](https://github.com/JN0V/workline/pull/1049),
    [#1060](https://github.com/JN0V/workline/pull/1060),
    [#1106](https://github.com/JN0V/workline/pull/1106),
    [#1108](https://github.com/JN0V/workline/pull/1108),
    [#1110](https://github.com/JN0V/workline/pull/1110),
    [#1139](https://github.com/JN0V/workline/pull/1139),
    [#1154](https://github.com/JN0V/workline/pull/1154),
    [#1163](https://github.com/JN0V/workline/pull/1163);
  - **the line**, 5:
    [#1042](https://github.com/JN0V/workline/pull/1042),
    [#1052](https://github.com/JN0V/workline/pull/1052),
    [#1092](https://github.com/JN0V/workline/pull/1092),
    [#1170](https://github.com/JN0V/workline/pull/1170),
    [#1173](https://github.com/JN0V/workline/pull/1173).

## RC1: the red-first proof is shaped on one toolchain

**Five whys** on [#1192](https://github.com/JN0V/workline/issues/1192),
3 of the round's 7 tester runs (170,966 fresh tokens):

1. Why blocked? `report-missing`: the test command wrote no report.
2. Why no report? Gradle stops at `compileTestKotlin`: a test that does
   not compile writes no JUnit XML, in any module, with
   `./gradlew testDebugUnitTest`.
3. Why is that a block, not "not built there"? `runIn` refuses when no
   report exists
   ([tester.go:507](https://github.com/JN0V/workline/blob/41983fc9/internal/tester/tester.go#L507))
   before `buildErrorsIn` reads the compiler's lines
   ([tester.go:146](https://github.com/JN0V/workline/blob/41983fc9/internal/tester/tester.go#L146)).
   The order assumes a runner that writes a report even when the build
   fails, as `gotestsum` does for Go.
4. Why would the lines not have saved it anyway? A missing name is an
   allow-list: two Kotlin messages
   ([role.yaml:139](https://github.com/JN0V/workline/blob/41983fc9/roles/tester/role.yaml#L139)).
   Any other error in a test of the change, `Too many arguments`,
   `'nextIndex' overrides nothing`, or a knock-on `Cannot infer type`, is
   a broken build.
5. Why one message at a time? Each ecosystem's shape is taught after a
   trial meets it: 10 of the 24 commits above. Each fix widens one row,
   path or pattern; none changes the default that an unknown shape
   blocks.

**Structural change.**

- **Read the compiler's lines whenever the red run fails**, with a
  report or without one. A plain fix.
- **Hand them back to the tester for one more try**, as
  [ADR-0063 §6](../adr/0063-every-open-issue-is-due-waiting-or-done.md#6-typed-outcomes)
  decided ("a machine error changes the input of its retry"). A plain
  fix.
- **Invert the default for errors in a test of the change**: not built
  there unless the error is a syntax or parse error, a short deny-list
  per language. Then a signature or a type the Scope adds needs no row.
  **A design decision, for a panel**: the risk is a test red for a wrong
  reason read as red for the right one.
- Closes [#1192](https://github.com/JN0V/workline/issues/1192). Makes
  most new `missing-name` rows unnecessary.

## RC2: a cut of pushed code is read as a signal

**Five whys** on [#1193](https://github.com/JN0V/workline/issues/1193),
the size crossings, and the tester's 55–120 KB tasks:

1. Why was #4 split? Its tester run "crossed its size".
2. Why did it cross? Its prompt was cut: `TracingViewModelTest.kt`, 162
   of 2,726 lines. The issue had spent 227,973 of its 400,000 normal
   fresh tokens.
3. Why is a cut a crossing?
   [ADR-0064 §2](../adr/0064-size-is-a-target-not-a-wall.md#2-two-checkpoints)
   lists "a cut of the code" beside a prompt over budget and a token cap.
4. Why is a cut a poor signal? It is constant, not rare:
   - every product owner call in the round was cut at 16,000 bytes;
   - every spec review was cut at 600 lines;
   - one 2,700-line file named as a source is cut at any tier, so #7
     crossed again at ×2 (688 of 2,747 lines);
   - the tester has read tools under `claude`
     ([#1110](https://github.com/JN0V/workline/pull/1110)): on #5 it made
     41 turns of Read, Grep and Glob. A cut of its seed does not stop
     it from reading the file.
5. Why is a cut there at all? The picker pushes whole files and
   excerpts by line count, sized to fit the prompt, and the agent then
   pulls the same code again with its tools: 5.4 M cache-read tokens for
   7 tester calls. Nine of the 24 commits tuned what is pushed.

**Structural change.**

- **Where the agent has read tools, the seed is a map, not the code**:
  paths, declarations, the region of the one nearest test, capped in
  bytes (the 30 KB target). A cut of it is told, never a crossing.
- **A size crossing comes only from what the issue spent**: fresh
  tokens against its normal
  ([ADR-0064 §1](../adr/0064-size-is-a-target-not-a-wall.md#1-normal-size-is-measured-per-project)),
  and a change too large to review whole.
- **A design decision, for a panel**: it amends ADR-0064 §2 and
  [ADR-0063 §8](../adr/0063-every-open-issue-is-due-waiting-or-done.md#8-one-code-picker-and-tools-where-the-agent-has-them).
  Agents with no tools keep the seeded code and the cut as today.
- Closes [#1193](https://github.com/JN0V/workline/issues/1193) (its
  first scenario); may replace
  [#829](https://github.com/JN0V/workline/issues/829) (a large Scope
  folder's files given by name order).

## RC3: the size judgement shipped unmeasured

**Five whys** on the feature granted `size/large` where "Thirds" alone
was a vertical slice, and the atomic bug split though its text said its
parts fail apart:

1. Why the wrong call both times? The product owner judged without the
   facts that decide it.
2. Which facts? At a crossing its task says "the tester's run on it is
   too big for that role"
   ([productowner.go:2003](https://github.com/JN0V/workline/blob/41983fc9/internal/builtin/productowner/productowner.go#L2003)).
   It does not say that one file was cut, nor that 228k of 400k were
   spent. "Too big" invites a split.
3. Why was that not seen before the round? The split and grant were
   tried with `cmd:` stubs only:
   [#1173](https://github.com/JN0V/workline/pull/1173)'s Tried says
   "not tried: a real agent's prediction or split".
4. Why no real agent? The evaluation suite has two product owner cases,
   none on size
   ([tests/evaluation/cases/product-owner](https://github.com/JN0V/workline/tree/41983fc9/tests/evaluation/cases/product-owner)).
5. Why no case? A judgement act is built as an engine feature: its
   Done is conformance, which checks the plumbing, not the judgement.

**Structural change.**

- **The size task names what crossed and what was spent**: which file
  was cut, the issue's fresh tokens against its normal, and the issue's
  own words on its parts.
- **Two evaluation cases from this round**, run before `split: act`
  stays on past a size: #4 (atomic, expect a grant) and #5 (a vertical
  slice, expect a split).
- **A plain fix**, plus cases. Covered by
  [#1193](https://github.com/JN0V/workline/issues/1193)'s second
  scenario.

## RC4: the line still decides by a fingerprint

**Five whys** on [#1196](https://github.com/JN0V/workline/issues/1196),
[#1195](https://github.com/JN0V/workline/issues/1195), the stall of
[#1194](https://github.com/JN0V/workline/issues/1194), and the product
owner's 3/15/7 calls against a target of 2:

1. Why did a person's `workline:accepted` wake nothing? The issue was
   `state-acted`
   ([states.go:337](https://github.com/JN0V/workline/blob/41983fc9/internal/line/states.go#L337)).
2. Why? `due` compares the state, a hash of the body, the spec mark,
   and the date the *state label* was put on
   ([record.go:313](https://github.com/JN0V/workline/blob/41983fc9/internal/line/record.go#L313),
   [states.go:159](https://github.com/JN0V/workline/blob/41983fc9/internal/line/states.go#L159)).
   Another label is not "labelling". The same rule:
   - **misses a person**: an accepted proposal (#1196); a tester verdict
     whose intentions were never applied leaves the same fingerprint, so
     nobody runs again (#1194);
   - **wakes a role on its own write**: on #3, call 2 of 3 was woken by
     the product owner's own body edit and did nothing
     ("spec-not-reviewed"); call 3, 20,227 tokens in and 217 out, only
     moved it to ready after a clean review, an act with no judgement.
     On #4, 7 calls in refine for 3 review rounds.
3. Why are #6 and #7 proposals at all? `reporter-outside`: issues the
   workline App opened read as opened "by someone without write access"
   (run artifact of #6, `13608387399`). GitHub gives an App's issue no
   `OWNER`, `MEMBER` or `COLLABORATOR` association
   ([github.go:343](https://github.com/JN0V/workline/blob/41983fc9/internal/forge/github.go#L343)),
   and a split's child does not inherit its parent's reporter. One of
   nine separate demotions to "propose"
   ([backlog.go:1050](https://github.com/JN0V/workline/blob/41983fc9/internal/backlog/backlog.go#L1050)).
4. Why still a fingerprint and facts read per caller? These are root
   causes 1 and 2 of
   [ADR-0063](../adr/0063-every-open-issue-is-due-waiting-or-done.md#four-root-causes),
   accepted on 2026-11-04. Its steps 6–8 (the view, `Next` in shadow,
   `Next` decides) are not built, and no issue tracks them.
5. Why not built? ADR-0064's size acts were built on 2026-11-08
   ([#1170](https://github.com/JN0V/workline/pull/1170),
   [#1173](https://github.com/JN0V/workline/pull/1173)) on top of
   `stateOf`: a size crossing, a split's children, a grant. Each is a new
   path through the rule ADR-0063 replaces, each with its own stranding
   case.

**Structural change.**

- **ADR-0063 steps 6–8 next, before any new act on the line**;
  #1194, #1195, #1196 as simulator traces in their Done. Sequencing of
  an accepted ADR: no panel.
- **Amend ADR-0063 §3, the input of a step**: a person's labels
  (`accepted`, `size/*`) are input; a role's own write is not new input
  for that role. **A design decision, for a panel**: as written, §3
  would still wake the product owner on its own body edit, and does not
  list labels.
- **Plain fixes, kept small under the stop**:
  - the App's own issues, and a split's children, are insiders: the
    child takes its parent's reporter;
  - ready after a clean spec review is moved by the engine, with no
    agent call (principle 4).
- Closes [#1195](https://github.com/JN0V/workline/issues/1195),
  [#1196](https://github.com/JN0V/workline/issues/1196), the stall half
  of [#1194](https://github.com/JN0V/workline/issues/1194); related to
  [#1157](https://github.com/JN0V/workline/issues/1157) and
  [#1091](https://github.com/JN0V/workline/issues/1091), same input
  question.

## RC5: a run has caps per call, not per run, and its end is untyped

**Five whys** on [#1197](https://github.com/JN0V/workline/issues/1197)
(70 min hung, 18 runs queued behind it) and the turn cap of
[#1194](https://github.com/JN0V/workline/issues/1194) (75,112 tokens):

1. Why did nothing end the hung run? No cap covers the whole run.
2. Why? Caps are per call:
   - the agent has `ai-max-seconds` and a process-group kill;
   - the red-first check's test command runs with no deadline: `deadline`
     is set only for the curator and the developer's proof
     ([curate.go:483](https://github.com/JN0V/workline/blob/41983fc9/internal/tester/curate.go#L483),
     [onchange.go:92](https://github.com/JN0V/workline/blob/41983fc9/internal/tester/onchange.go#L92));
     `suite-max-seconds` is a measure, a warning
     ([measure.go:210](https://github.com/JN0V/workline/blob/41983fc9/internal/tester/measure.go#L210));
   - the CI issue job has no `timeout-minutes`
     ([workline-develop.yml](https://github.com/JN0V/workline/blob/41983fc9/ci/github/workline-develop.yml)),
     so GitHub's 6 h applies;
   - the step prints nothing per phase, so which one hung is not known.
3. Why was #5's tester read as "no test possible"? Its run ended at 41
   of 40 turns with an empty answer. `error_max_turns` is read as an
   answer
   ([claude.go:162](https://github.com/JN0V/workline/blob/41983fc9/internal/agent/claude.go#L162)),
   and an answer with no test as `no-test-possible`
   ([proof.go:101](https://github.com/JN0V/workline/blob/41983fc9/internal/tester/proof.go#L101)).
4. Why is a turn cap an answer?
   [ADR-0056](../adr/0056-the-developers-agent-works-in-a-sandbox-with-tools.md)
   chose it for the developer, whose edits in its worktree are its work.
   The tester answers in text: an empty text is no work.
5. Why not a machine error?
   [ADR-0063 §6](../adr/0063-every-open-issue-is-due-waiting-or-done.md#6-typed-outcomes)
   types a run's outcome, but how the agent stopped (answered, turn cap,
   time cap, empty) is not one of its inputs.

**Structural change.**

- **One deadline per run**, the sum of its caps, passed to every child
  process (agent, test command, git), with a phase line on stderr each
  step, and the CI job's `timeout-minutes` above it.
- **The agent's stop reason in its result**: a role whose answer is
  text, ending empty or at its cap, is a `machine-error`, retried with
  the cap said, never a verdict.
- **Plain fixes**: both apply decisions already taken (principle 12,
  ADR-0063 §6). Whether `ai-max-turns` scales with a size tier is asked
  in #1194.
- Closes [#1197](https://github.com/JN0V/workline/issues/1197) and the
  turn-cap half of [#1194](https://github.com/JN0V/workline/issues/1194).

## M: why the fixes repeat: tried below the real path

- **The round's prep fixes were tried below the role's real path**:
  - [#1181](https://github.com/JN0V/workline/pull/1181), Kotlin build
    errors: `workline test check`, no agent, one module's task;
  - [#1188](https://github.com/JN0V/workline/pull/1188), the picker: the
    picker called directly;
  - [#1170](https://github.com/JN0V/workline/pull/1170) and
    [#1173](https://github.com/JN0V/workline/pull/1173), the size acts:
    `cmd:` stubs, on a small Python repository.
- **None ran `route issue` with a real tester on the Kotlin copy**,
  with the copy's own `test-command`. The round was the first end-to-end
  run, and found each gap at trial speed: RC1's order and RC2's crossing
  both show on the first real call.
- **Proposal**: before a round, one real `route issue` per role on the
  copy, on one issue, told under `## Tried`. Its outputs (the build's
  output, the agent's answers) frozen as simulator inputs
  ([#1079](https://github.com/JN0V/workline/issues/1079)). **A design
  decision, for a panel**: it amends ADR-0063's working agreements and
  costs tokens before each round. Tracked in
  [#1198](https://github.com/JN0V/workline/issues/1198).

## Summary of proposals

| Change | Cause | Closes or replaces | Panel |
|---|---|---|---|
| Read compiler lines with no report; hand them back once | RC1 | [#1192](https://github.com/JN0V/workline/issues/1192) | no |
| An error in a new test is "not built there" unless syntax | RC1 | most new `missing-name` rows | **yes** |
| Seed a map where the agent has tools; cross on tokens only | RC2 | [#1193](https://github.com/JN0V/workline/issues/1193), [#829](https://github.com/JN0V/workline/issues/829) | **yes** (ADR-0064 §2) |
| Size task states what crossed; two evaluation cases | RC3 | [#1193](https://github.com/JN0V/workline/issues/1193) second scenario | no |
| ADR-0063 steps 6–8 before any new line act | RC4 | [#1195](https://github.com/JN0V/workline/issues/1195), [#1196](https://github.com/JN0V/workline/issues/1196), [#1194](https://github.com/JN0V/workline/issues/1194) stall | no (accepted) |
| Labels are input; a role's own write is not | RC4 | the 3/15/7 calls | **yes** (ADR-0063 §3) |
| App's issues insiders; ready after clean review with no agent | RC4 | [#1195](https://github.com/JN0V/workline/issues/1195); calls | no |
| One deadline per run; stop reason typed | RC5 | [#1197](https://github.com/JN0V/workline/issues/1197), [#1194](https://github.com/JN0V/workline/issues/1194) turn cap | no |
| A real rehearsal per role before a round | M | [#1198](https://github.com/JN0V/workline/issues/1198) | **yes** |
