# Round 6 root causes — why the tester and the line keep taking fixes (2026-11-08)

**Question.** Round 6 of the trial ([#757](https://github.com/JN0V/workline/issues/757#issuecomment-6719368180)),
on [JN0V/TraceGlass-trial](https://github.com/JN0V/TraceGlass-trial), a
Kotlin/Android copy, opened six defects
([#1192](https://github.com/JN0V/workline/issues/1192) to
[#1197](https://github.com/JN0V/workline/issues/1197)) and missed five
targets. `internal/tester` and `internal/codepick` had each taken five
fix pull requests in the 48 h before it, which trips the stop on fixes of
[ADR-0063](../adr/0063-every-open-issue-is-due-waiting-or-done.md#working-agreements).
What few causes lie under all of them?

**Verdict.** Six causes, and one reason the fixes repeat. Issues
written "copy #N" are the trial copy's, not workline's.

| # | Root cause | Explains |
|---|---|---|
| [RC1](#rc1-the-red-first-proof-is-shaped-on-one-toolchain) | The red-first proof is shaped on one toolchain; others are taught one message at a time | [#1192](https://github.com/JN0V/workline/issues/1192); 3 of 4 tester misses |
| [RC2](#rc2-an-untyped-block-with-a-cut-becomes-a-size-question) | `report-missing` is not typed, and a blocked run with a cut is read as a size question | [#1193](https://github.com/JN0V/workline/issues/1193), downstream of [#1192](https://github.com/JN0V/workline/issues/1192); every size crossing |
| [RC3](#rc3-the-size-judgement-shipped-unmeasured) | The product owner's split-or-grant judgement shipped untried on a real agent, without the facts it needs | feature granted, atomic bug split |
| [RC4](#rc4-the-line-still-decides-by-a-fingerprint) | The line still decides "due" by a fingerprint; the step that replaces it waited | [#1196](https://github.com/JN0V/workline/issues/1196), the retry half of [#1194](https://github.com/JN0V/workline/issues/1194); product owner calls 3/15/7 |
| [RC5](#rc5-the-apps-own-issues-read-as-an-outsiders) | The App's own issues, a split's children among them, read as an outsider's | [#1195](https://github.com/JN0V/workline/issues/1195) |
| [RC6](#rc6-a-runs-end-is-not-checked) | A run's end is not checked: caps per call, the agent's stop untyped, a pass's intentions dropped silently | [#1197](https://github.com/JN0V/workline/issues/1197), the turn cap and the stall of [#1194](https://github.com/JN0V/workline/issues/1194) |
| [M](#m-why-the-fixes-repeat-tried-below-the-real-path) | Fixes are tried below the role's real path; the trial is the first end-to-end run | the 24 tester and picker commits since 2026-11-04 |

Read-only, on `main` at [`41983fc9`](https://github.com/JN0V/workline/tree/41983fc9),
the engine the round pinned. Method: five whys
([root-cause.md](root-cause.md)), each chain ended on a mechanism seen
in the code or the run artifacts, never on "a missing case". An
independent review and a panel followed:
[Panel decision](#panel-decision).

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

**A gap in a built step.**
[ADR-0063](../adr/0063-every-open-issue-is-due-waiting-or-done.md#build-plan)
step 2, typed outcomes
([#1087](https://github.com/JN0V/workline/issues/1087), closed), has
RC1's case as its own Given: "a tester's test that does not build …
asked again with the compiler's output". It misses this shape, against
[§6](../adr/0063-every-open-issue-is-due-waiting-or-done.md#6-typed-outcomes):
"an unknown error of a run is a machine error".

**Structural change**, as the panel set it:

- **Read the compiler's lines before refusing a run with no report.**
  A plain fix.
- **Hand an unknown error back to the tester once**, with those lines.
  A plain fix: what step 2 promised.
- **Then, an error in a test of the change is red-first only when it
  quotes a name the issue or its Scope names, or one missing from the
  base tree.** Other errors in that file are knock-ons. An error on a
  name the base has goes back to the tester. No syntax deny-list.
- Closes [#1192](https://github.com/JN0V/workline/issues/1192). Makes
  most new `missing-name` rows unnecessary.

## RC2: an untyped block with a cut becomes a size question

**Five whys** on [#1193](https://github.com/JN0V/workline/issues/1193)
and every size crossing of the round:

1. Why was copy [#4](https://github.com/JN0V/TraceGlass-trial/issues/4)
   split? Its tester run "crossed its size".
2. Why did it cross? Its findings were `input-cut` (a warning:
   `TracingViewModelTest.kt`, 162 of 2,726 lines) and `report-missing`
   (a block) (run artifact `13608212211`).
3. Why do those two make a crossing? A cut alone never crosses:
   `input-cut` is always a warning
   ([verdict/cut.go:15](https://github.com/JN0V/workline/blob/41983fc9/internal/verdict/cut.go#L15)).
   `OutcomeOf` asks the product owner on size for a *blocked* run that
   also carries a cut, "its block may be the cut's"
   ([outcome.go:122](https://github.com/JN0V/workline/blob/41983fc9/internal/verdict/outcome.go#L122)).
4. Why was the block not set aside as a machine error first?
   `report-missing` is not in `machineKinds`
   ([outcome.go:66-72](https://github.com/JN0V/workline/blob/41983fc9/internal/verdict/outcome.go#L66-L72)).
   So RC1's compile error, untyped, reached the size rule.
5. Why does the size rule take any block? It guesses the block's cause
   from a cut beside it, and cuts are common: every product owner call
   of the round was cut at 16,000 bytes, every spec review at 600 lines.
   They crossed nothing only because they did not block.

- **[#1193](https://github.com/JN0V/workline/issues/1193) follows from [#1192](https://github.com/JN0V/workline/issues/1192).** One untyped failure gave both the block
  and, through the cut rule, the split.
- **The same on copy [#7](https://github.com/JN0V/TraceGlass-trial/issues/7)**:
  the product owner granted `size/large` on a block that was a compile
  error; the next run crossed again on the same file, then waited for a
  person.

**Structural change**, as the panel set it:

- **`report-missing` with a failed exit is a machine error.** A plain
  fix.
- **A cut is no longer a size crossing**, for any agent: the fallback
  branch in `OutcomeOf` removed, and one phrase of
  [ADR-0064 §2](../adr/0064-size-is-a-target-not-a-wall.md#2-two-checkpoints)
  ("a cut of the code") struck. A plain fix.
- **A size crossing comes from what the issue spent**: fresh tokens
  against its normal
  ([ADR-0064 §1](../adr/0064-size-is-a-target-not-a-wall.md#1-normal-size-is-measured-per-project)),
  a prompt over budget, a change too large to review whole.
- **Postponed**: seeding a map of the code, not the code, where the
  agent has read tools (the tester's tasks were 55–120 KB, with 5.4 M
  cache-read tokens over 7 calls). It waits for a measure, under
  [ADR-0063 §8](../adr/0063-every-open-issue-is-due-waiting-or-done.md#8-one-code-picker-and-tools-where-the-agent-has-them);
  related to [#829](https://github.com/JN0V/workline/issues/829).
- Closes [#1193](https://github.com/JN0V/workline/issues/1193)'s first
  scenario, with RC1.

## RC3: the size judgement shipped unmeasured

**Five whys** on the feature (copy
[#5](https://github.com/JN0V/TraceGlass-trial/issues/5)) granted
`size/large` where "Thirds" alone was a vertical slice, and the atomic
bug (copy #4) split though its text said its parts fail apart:

1. Why the wrong call both times? The product owner judged without the
   facts that decide it.
2. Which facts? At a crossing its task says "the tester's run on it is
   too big for that role"
   ([productowner.go:2003](https://github.com/JN0V/workline/blob/41983fc9/internal/builtin/productowner/productowner.go#L2003)).
   It does not say that the run blocked on a compile error, that one
   file was cut, nor that 228k of 400k were spent. "Too big" invites a
   split.
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

- **The size task names what crossed and what was spent**: the issue's
  fresh tokens against its normal, and the issue's own words on its
  parts.
- **Two evaluation cases from this round**, run before `split: act`
  stays on past a size: copy #4 (atomic, expect a grant) and copy #5 (a
  vertical slice, expect a split).
- **A plain fix**, plus cases. Covered by
  [#1193](https://github.com/JN0V/workline/issues/1193)'s second
  scenario.

## RC4: the line still decides by a fingerprint

**Five whys** on [#1196](https://github.com/JN0V/workline/issues/1196),
the retry half of [#1194](https://github.com/JN0V/workline/issues/1194),
and the product owner's 3/15/7 calls against a target of 2:

1. Why did a person's `workline:accepted` wake nothing? The issue was
   `state-acted`
   ([states.go:337](https://github.com/JN0V/workline/blob/41983fc9/internal/line/states.go#L337)).
2. Why? `due` compares the state, a hash of the body, the spec mark,
   and the date the *state label* was put on
   ([record.go:313](https://github.com/JN0V/workline/blob/41983fc9/internal/line/record.go#L313),
   [states.go:159](https://github.com/JN0V/workline/blob/41983fc9/internal/line/states.go#L159)).
   Another label is not "labelling".
3. What else does the same rule do?
   - **It misses a person**: an accepted proposal
     ([#1196](https://github.com/JN0V/workline/issues/1196)); after a
     run whose intentions were dropped
     ([RC6](#rc6-a-runs-end-is-not-checked)), the same fingerprint, so
     nobody runs again
     ([#1194](https://github.com/JN0V/workline/issues/1194)).
   - **It wakes a role on its own write**: on copy
     [#3](https://github.com/JN0V/TraceGlass-trial/issues/3), call 2 of
     3 was woken by the product owner's own body edit and did nothing
     ("spec-not-reviewed"). Call 3 only moved it to ready after a clean
     review, an act with no judgement (219 fresh tokens, 20,225 more
     read from cache). On copy #4, 6 product owner calls in refine,
     then the split call, for 2 reviewer runs.
4. Why still a fingerprint? It is root cause 1 of
   [ADR-0063](../adr/0063-every-open-issue-is-due-waiting-or-done.md#four-root-causes),
   accepted on 2026-11-04. Its steps 6–8 (the view, `Next` in shadow,
   `Next` decides) are not built, and no issue tracks them.
5. Why not built? A sequencing choice:
   [ADR-0064](../adr/0064-size-is-a-target-not-a-wall.md), accepted on
   2026-11-06, was built first
   ([#1170](https://github.com/JN0V/workline/pull/1170),
   [#1173](https://github.com/JN0V/workline/pull/1173), 2026-11-08).
   Its acts (a crossing, a split's children, a grant) went through
   `stateOf`, the rule step 8 replaces, each a new stranding path.

**Structural change**, as the panel set it:

- **Finish [ADR-0063 §3](../adr/0063-every-open-issue-is-due-waiting-or-done.md#3-the-input-of-a-step)
  with steps 6–8, before any new act on the line**: record the body
  and labels as they stand after the role's own write, so its own edit
  is not new input; a person's other labels (`accepted`, `size/*`)
  enter as label events. No lookup of who wrote what. Copy issues
  #3, #6, #7 frozen as simulator traces in their Done.
- **Ready after a clean spec review is moved by the engine**, with no
  agent call ([principle 4](../PRINCIPLES.md)). A plain fix.
- Closes [#1196](https://github.com/JN0V/workline/issues/1196) and the
  retry half of [#1194](https://github.com/JN0V/workline/issues/1194);
  related to [#1157](https://github.com/JN0V/workline/issues/1157) and
  [#1091](https://github.com/JN0V/workline/issues/1091), the same input
  question.

## RC5: the App's own issues read as an outsider's

**Five whys** on [#1195](https://github.com/JN0V/workline/issues/1195):

1. Why did copy [#6](https://github.com/JN0V/TraceGlass-trial/issues/6)
   and #7 wait for a person after their drafts timed out? Their `ready`
   was proposed, not done.
2. Why proposed? `reporter-outside`: "opened by someone without write
   access to the project" (run artifact of copy #6, `13608387399`;
   [backlog.go:1050](https://github.com/JN0V/workline/blob/41983fc9/internal/backlog/backlog.go#L1050)).
   Not `draftsDue`, as the issue's Scope guessed.
3. Why an outsider? The workline App opened them. GitHub gives an App's
   issue no `OWNER`, `MEMBER` or `COLLABORATOR` association
   ([github.go:343](https://github.com/JN0V/workline/blob/41983fc9/internal/forge/github.go#L343)).
4. Why does a child not count as its parent's? A split's child does not
   inherit its parent's reporter.
5. Why was it not seen? It is one of about a dozen separate demotions
   to "propose" in `decide`
   ([backlog.go:987-1106](https://github.com/JN0V/workline/blob/41983fc9/internal/backlog/backlog.go#L987-L1106)),
   and no try had the App open an issue.

**Structural change.** The App's own issues, and a split's children,
count as insiders; a child takes its parent's reporter. A plain fix,
independent of RC4. Closes
[#1195](https://github.com/JN0V/workline/issues/1195).

## RC6: a run's end is not checked

**Five whys** on [#1197](https://github.com/JN0V/workline/issues/1197)
(70 min hung, 18 runs queued behind it) and
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
3. Why was copy #5's tester read as "no test possible"? Its run ended at
   41 of 40 turns with an empty answer. `error_max_turns` is read as an
   answer
   ([claude.go:162](https://github.com/JN0V/workline/blob/41983fc9/internal/agent/claude.go#L162)).
   An answer with empty notes takes the "no test file … proposed"
   message as its reason and becomes `KindNone`
   ([write.go:421-428](https://github.com/JN0V/workline/blob/41983fc9/internal/tester/write.go#L421-L428));
   `no-test-possible`
   ([proof.go:101](https://github.com/JN0V/workline/blob/41983fc9/internal/tester/proof.go#L101))
   follows. Why a cap is an answer:
   [ADR-0056](../adr/0056-the-developers-agent-works-in-a-sandbox-with-tools.md)
   chose it for the developer, whose edits are its work; the tester
   answers in text.
4. Why did copy #5 then stall? Its tester step ended `pass`, "the issue
   moves to refine", with `to-apply` unset. The engine returned with no
   intention to apply
   ([engine.go:869](https://github.com/JN0V/workline/blob/41983fc9/internal/engine/engine.go#L869))
   before `ToApply` is set
   ([engine.go:937](https://github.com/JN0V/workline/blob/41983fc9/internal/engine/engine.go#L937)):
   the label move and the comment were dropped, with no finding. Why
   they were not in its list is for a red trace to settle.
5. Why not a machine error?
   [ADR-0063 §6](../adr/0063-every-open-issue-is-due-waiting-or-done.md#6-typed-outcomes)
   types a run's outcome, but how the agent stopped (answered, turn cap,
   time cap, empty) is not one of its inputs, and a pass is never
   checked against what it said it would move.

**Structural change.** Plain fixes, applying decisions already taken
([principle 12](../PRINCIPLES.md), [ADR-0063](../adr/0063-every-open-issue-is-due-waiting-or-done.md) §6):

- **One deadline per run**, the sum of its caps, passed to every child
  process (agent, test command, git), a phase line on stderr each step,
  and the CI job's `timeout-minutes` above it.
- **An empty answer is `no-test-written`**, already a machine kind
  ([outcome.go:69](https://github.com/JN0V/workline/blob/41983fc9/internal/verdict/outcome.go#L69)):
  retried with the cap said, never `KindNone`. Whether `ai-max-turns`
  scales with a size tier is asked in
  [#1194](https://github.com/JN0V/workline/issues/1194).
- **A pass that says it moves an issue, and applies nothing, is a
  machine error**, never silent.
- Closes [#1197](https://github.com/JN0V/workline/issues/1197) and,
  with RC4, [#1194](https://github.com/JN0V/workline/issues/1194).

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
  run: RC1 and RC2 both show on the first real call.
- **Proposal**, as the panel set it: a working agreement. A fix to a
  role runs that role for real on the round's copy, in its own pull
  request, told under `## Tried`; only the roles whose packages changed.
  Its outputs are frozen for the simulator
  ([#1079](https://github.com/JN0V/workline/issues/1079)). Tracked in
  [#1198](https://github.com/JN0V/workline/issues/1198).

## Panel decision

Three independent readers (the engine's code, a person using workline,
prior art) agreed on all four proposals put to them, each with changes
([the panel's comment](https://github.com/JN0V/workline/pull/1199#issuecomment-6719455006)):

- **RC1**: no syntax deny-list. The two plain fixes first; then an
  error is red-first only on a name the issue or Scope names, or one
  the base lacks.
- **RC2**: a cut is no longer a size crossing, for any agent: a plain
  fix. The code map is postponed until measured.
- **RC4**: no lookup of who wrote what. The body and labels as they
  stand after the role's write; a person's other labels as label
  events. It finishes [ADR-0063](../adr/0063-every-open-issue-is-due-waiting-or-done.md#build-plan) §3, steps 6–8.
- **M ([#1198](https://github.com/JN0V/workline/issues/1198))**: a
  working agreement, not a design. A role's fix runs that role for real
  on the copy, in its own pull request.

## Summary of proposals

| Change | Cause | Closes or replaces | Kind |
|---|---|---|---|
| Read compiler lines before refusing; hand unknown errors back once | RC1 | [#1192](https://github.com/JN0V/workline/issues/1192) | plain fix |
| Red-first only on a name the issue names or the base lacks | RC1 | most new `missing-name` rows | panel: agreed |
| `report-missing` a machine error; a cut never crosses | RC2 | [#1193](https://github.com/JN0V/workline/issues/1193) first scenario | plain fix ([ADR-0064 §2](../adr/0064-size-is-a-target-not-a-wall.md#2-two-checkpoints), one phrase) |
| Seed a map where the agent has tools | RC2 | [#829](https://github.com/JN0V/workline/issues/829) | postponed: a measure first |
| Size task states what crossed; two evaluation cases | RC3 | [#1193](https://github.com/JN0V/workline/issues/1193) second scenario | plain fix |
| ADR-0063 steps 6–8, input recorded after the role's write | RC4 | [#1196](https://github.com/JN0V/workline/issues/1196), [#1194](https://github.com/JN0V/workline/issues/1194) retry | [ADR-0063 §3](../adr/0063-every-open-issue-is-due-waiting-or-done.md#3-the-input-of-a-step), panel: agreed |
| Ready after clean review with no agent | RC4 | the 3/15/7 calls | plain fix |
| App's issues and split children are insiders | RC5 | [#1195](https://github.com/JN0V/workline/issues/1195) | plain fix |
| One deadline per run; empty answer `no-test-written`; a silent pass refused | RC6 | [#1197](https://github.com/JN0V/workline/issues/1197), [#1194](https://github.com/JN0V/workline/issues/1194) | plain fix |
| A role's fix runs that role for real on the copy | M | [#1198](https://github.com/JN0V/workline/issues/1198) | working agreement, panel: agreed |
