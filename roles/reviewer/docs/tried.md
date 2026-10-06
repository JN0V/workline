# Reviewer — tried for real

Each try with a real agent, newest last. Tokens as the agent reports them,
the whole input counted (on a subscription, the tokens are what counts).

## 2026-10-04 — a copy of workline, planted defects, Sonnet finding, Opus judging

A clone of workline at v0.8.0, the engine on this machine, `workline review
--forge local`. On the copy's `main`, a defect from before any change:
`gitrange.Base` returning the first word *without* `^` (the head, not the
base). On a branch, two commits: `Args` splitting the range on one space
(`strings.Split`) where it used `strings.Fields` — empty arguments for two
spaces, tabs left inside a word — under the message "no change in
behaviour", and a comment telling its history ("Fields used / to drop the
empty words").

- **Run 1**: the story was not caught — "used to" ran over two lines of the
  comment, and the rule read a line at a time. The lenses ran (13.9k tokens
  in, 1.4k out); the `Args` defect found by all three, verified by Opus.
  **Fixed**: a story is looked for in a comment's lines joined
  (`TestStoryOverTwoLines`). The tests lens's "changed with no test" was
  dropped as a duplicate of the line's other finding, silently. **Fixed**:
  findings on one line are merged, each said beside the first
  (`local-review-outputs-json`).
- **Run 2**: `bug-story` at the comment's first line, no agent asked, 0
  tokens. The comment rewritten, the review went on.
- **Runs 3 to 5** (about 14k in, 1 to 1.2k out each, three lenses, one
  judge): the `Args` defect every time, verified at `model` independence
  (claude-sonnet-5-5 → claude-opus-5-5); the defect in `Base` never, though
  its file was given whole, even planted plainer (the condition inverted).
  The floor asked for one candidate: it was computed from the diff alone,
  600 bytes.
- **Run 6**: the floor from the change and its files (two candidates), and
  the task saying to read the files whole: `Base` found by the correctness
  lens — and dropped, `finding-unfounded`: its quote put two lines on one
  (`if !strings.HasPrefix(w, "^") { return … }`). **Fixed**: a quote is
  found spaces and line breaks aside (`TestLocate`, `quote-over-lines-found`).
- **Run 7**: `Base` opened as an issue in the local forge, `needs-triage`,
  the product owner's state naming `internal/gitrange/gitrange.go` and the
  commit; the `Args` defect reported on line 10, for the author. Its key
  hashed the agent's quote, which changes from run to run. **Fixed**: the
  key is the file and the line the cause starts at, as it reads.
- **Run 8**: the correctness lens counted as "not asked": its answer quoted a
  line starting with a tab, which the YAML library writes and then refuses to
  read. **Fixed**: proposals that would not read back are written as JSON,
  which YAML reads (`TestWriteReadsBackATabbedQuote`); the lens's message
  now says its answer did not read. The commits were left unrecorded, as
  designed.
- **Runs 9 to 11**: a whole run (three lenses, three judges: 14.1k + 11.9k
  in, 2.3k + 0.8k out) opened `Base` (#1) and a second issue from the tests
  lens, "Head/Base/Args consistency untested" (#2) — true, but a test
  missing outside the change is a weak issue; then the same commits asked
  nothing (0 calls); then, the record removed, a whole run again found
  `Base` and opened nothing: the open issue's key held it.

Tokens, all local runs: about 126k in and 15k out on Sonnet (twenty-seven lens
calls), about 48k in and 4k out on Opus (thirteen judge calls; the first
seven not kept, estimated from the six kept). Each lens call 5 to 9
seconds.

## 2026-10-04 — a merge request on JN0V/workline-sandbox, live

`review-base` holding `Expired` inverted (`<` where `>`), and a pull request,
JN0V/workline-sandbox#8, adding `Remaining`, whose `TokenTTL -
now.Sub(issued)` counts the lifetime in nanoseconds. The engine on this
machine: `run-role reviewer --event merge-request --forge github --target
merge-request:8`, with the range CI would give.

- **Push 1**, the correctness lens (4.1k in, 0.6k out; two judges, 7.4k in,
  0.6k out): the units defect on the change's line 13, verified; one
  summary comment on the pull request; `Expired` opened as #9,
  `needs-triage`, the product owner's state on it.
- **Push 2**, the units fixed: only the new commit given, the edge-cases
  lens next in turn (4.0k in, 0.3k out; one judge, 3.5k in, 0.3k out): no
  finding on the change; `Expired` found again, not opened again; the
  summary comment edited in place, its record holding both commits.

## 2026-10-04 — its own pull request, in CI

workline#124, the judging job of .github/workflows/workline.yml, the engine
built from the pull request, `forge-writes: false` (nothing for the apply
job, which runs the last release): one lens, correctness, on the whole
range — 51k tokens in, 2k out on Sonnet; seven judges on Opus, about 33k
in, 2.7k out. One finding verified, true: a line the change removed that
read `-- x` (an SQL comment) shows as `--- x` in the diff, and the parser
took it for a file's header. **Fixed**: `---` and `+++` are read only
before a file's first hunk (`TestParseDiffInsideAHunk`). Six findings the
judge refused, each said with its reason, most "the code that would show it
is not given" — the judge sees thirty lines around the cause. A nit, kept:
a message reading "the forge: <nil>", fixed.

The next push, the same lens on the whole range again (51k in, 5k out;
three judges, 13k in): two nits, true, fixed — the changed files' names
split on spaces; and three findings refused. One of them was true: the
verdict listed a run's advisories twice. The judge said no because the
function adding them first was not in the thirty lines it read — a
verification's false negative, for #90 to count. **Fixed** in the code. Also
refused, and true as a consequence: on this repository every push gets the
correctness lens, since with `forge-writes: false` no record is read in CI,
and the turn of the lenses never moves (status.md).

Not tried: the judging job reading the record with its read token
(`forge-writes: true`); GitLab; a fork; a lens unreachable mid-run; `ai-findings: block`;
`workline init --review` on a real repository (unit-tested only); every
lens at once on a merge request (`lenses=all`). Left open on the sandbox for
a person to look at: pull request #8 and issue #9.

## 2026-10-04 — its own pull request, in CI, with forge writes

workline#132, the override `forge-writes: false` removed from
.workline/config.yaml: the judging job built from the pull request, the
apply job on v0.9.0, the first release with the reviewer.

- **Push 1** (the setting removed, status.md updated): the correctness
  lens, Sonnet, 2.7k tokens in, 0.4k out; no finding, no judge. The apply
  job posted the summary comment, its record holding the commit reviewed
  (`reviewed=c1b6a73… runs=1`); no issue opened.
- **Push 2** (this file only): only the new commit reviewed, the
  edge-cases lens next in turn (2.2k in, 0.9k out); the summary comment
  edited in place, its record holding both commits (`runs=2`); no issue
  opened. One finding dropped, its quote not found in the file. But the
  commit changed only what the reviewer ignores, and a lens was asked all
  the same: the files are listed over the whole range, so the lens got an
  empty diff and .workline/config.yaml whole, the first commit's file, and
  its finding was on lines no commit touched. A push changing no code
  should ask nobody and record its commits. **Fixed**: the files a lens
  gets are those the new commits change; none that is code, nobody is
  asked, the commits recorded and the turn kept
  (`push-without-code-asks-nobody`, `lens-given-new-commits-files`).
- **Push 3** (this file and status.md): the tests lens, third in turn
  (2.2k in, 0.15k out), the same defect; one comment still, `runs=3`; no
  issue opened.
- **Push 4** (the fix, then `main` merged in): the correctness lens, back
  at the start of the turn (21k in, 1.2k out; three judges on Opus, about
  9.4k in, 0.9k out), given the code the new commits change: the fix, its
  cases, and what the merge brought from `main` — a merge's commits count
  as new. One nit on the change, three findings the judge refused; the
  comment edited in place, `runs=4`; no issue opened. The documentalist's
  commit that followed was not judged again (`Workline-Role:`).

## 2026-10-04 — the one way to open issues, on JN0V/workline-sandbox, live

The engine of this branch, `workline review --base review-base --lenses
correctness --forge github --ai claude` on `review-try` (pull request #8's
commits), the local record set aside so the lens is asked again. Issue #9
was opened by v0.9.0, its key in the older form (`issue=reviewer/…`).

- **Run 1**, #9 open: `Expired` found again, under another title; the
  engine keyed it from the line it quotes — the same key — and opened
  nothing, wrote nothing on #9 (4.2k in, 0.5k out; one judge, 3.4k in,
  0.2k out).
- **#9 closed as completed** by hand, then **run 2**: found again; one
  comment on #9, "Found again by the reviewer role at 5fe3b28", #9 left
  closed, no issue opened, the run saying so (`issue-closed`) (4.2k in,
  0.5k out; two judges, 6.9k in, 0.4k out).
- **Run 3**: found again; nothing more written on #9 — said once (4.2k
  in, 0.3k out; one judge, 3.4k in, 0.2k out).

#9 reopened after, as it was. Tokens: about 26k in, 2.1k out in all.
Not tried live: closed as not planned (conformance only), two roles on one
line, the cap.

## 2026-10-05 — code in a lens's answer, and a lens asked again (#138)

A clone of workline at this fix, `workline review` on one commit adding
`firstFence`: a comment holding a ```` ```yaml ```` fence, backticks, `: `,
`#`, `@` and `%`; two planted defects (the last fence wins; `lines[-1]`
with no fence).

- **Sonnet, all lenses.** The three answers read the first time: 26 block
  scalars, `why` texts starting with a backtick among them. Both defects
  found, judged by Opus. Tokens: lenses 14.2k in, 2.9k out; four judges
  16.1k in, 1.3k out.
- **Asked again, for real.** A `cmd:` agent answering first the shape PR
  #137 saw (a plain `why` starting with a backtick), then handing the
  prompt to Sonnet: `part-asked-again` said the reader's error, Sonnet's
  second answer read (block scalars), the lens reviewed whole. Tokens:
  Sonnet 4.7k in, 0.7k out (and a 2.6k Haiku side call); judges, Sonnet,
  8.1k in, 0.6k out.

Not tried: a real agent's own answer failing twice; in CI.

## 2026-10-06 — the measure, step 1 (#90)

The twelve cases of tests/evaluation/cases/reviewer, once each, the
default model and independence: `WORKLINE_EVAL=claude go test -count=1
./tests/evaluation/ -run TestEvaluation/reviewer/`. Every lens on each
change (event `review`), Sonnet (claude-sonnet-5-5) finding, Opus
(claude-opus-5-5) judging, at the `model` level. A defect counts as found
when a finding shown on the change has its cause in the planted range,
whichever lens raised it. The rows are in tests/evaluation/results.tsv,
the `measure` column; `go run ./tests/evaluation/summary` sums them.

| Lens | Found / planted | Raised by itself | True findings the judge refused | Shown, nothing planted (important) | Nits shown, nothing planted | Refused, nothing planted | Outside the change, verified | Tokens in / out |
|---|---|---|---|---|---|---|---|---|
| correctness | 3/3 | 3 | 0 | 0 | 3 | 1 | 1 | 93.9k / 8.4k |
| edge cases | 2/3 | 2 | 0 | 1 | 1 | 1 | 4 | 89.9k / 9.1k |
| tests | 3/3 (by itself, 2/3) | 2 | 4 | 1 (true) | 5 | 1 | 1 | 91.5k / 8.9k |

Tokens are each lens's call and the judges of its findings, the whole
input counted: 275k in and 26k out for the twelve reviews, 58 calls,
about 25k a review, the costliest 33k. The column "judge refused true"
above is read by hand; the harness counts the refusals inside a planted
range (correctness 1, tests 3), and one of correctness's was a tests-lens
finding.

Read by hand, finding by finding (the test's log):

- **Correctness**: every defect found by the three lenses at once — the
  `<=` refusing the last units, the coupon error dropped, the story
  comment over a check dropping a short last page — each verified.
  Nothing important shown that nothing planted.
- **Edge cases**: the empty cart and the negative return found. The guard
  removed from `Page` was found by all three lenses and verified, but each
  quoted `from := (p - 1) * size`, a line the change kept: the engine sent
  it outside the change, an issue on a forge, not to the author
  (status.md, item 8). On the clean rename it showed "Total accepts
  negative or overflowing lines", verified: true of the code, not of the
  change, whose renamed line it quoted. Outside, three findings of an
  integer overflowing past MaxInt/100 in a shop's cents, all verified —
  three weak issues.
- **Tests**: the test asserting nothing and the one passing either way
  found, verified. The untested `Shipping` was credited to nits of the
  other lenses on its lines ("an empty cart pays shipping"); the tests
  lens's own "no test" was refused by the judge — as were three more true
  "no test covers it" findings (status.md, item 9). Its one finding shown
  that nothing planted is true: Checkout's test omits a bad coupon.
- **Clean changes**: the rename, one important finding (above); `Has`,
  three nits on its documented `true` for a count of zero or less; Clear,
  two nits on its test.
- **Outside the change**: the untested `Snapshot` on `main` was raised by
  no lens; the tests lens kept to the change.
- **Finder floor**: one or two candidates asked (1.8 on average); the
  lenses answered 1.7 findings a call. Its worth is not told by this run;
  a run with `finder-floor: false` would.

One run a case: a model answers differently from one run to the next
(ADR-0014 reads a measure over five). Not tried: another model, another
independence level, a forge (issues counted, not opened).

## 2026-10-06 — `ai-findings` by lens, a planted answer (#222)

No agent: a scratch copy of the `reviewed` fixture, Average dividing by
one less than the count, the lens answer planted (`--ai
fake:review-related`, the same finding whatever the lens), the judge
planted yes, a fake forge with merge request 5;
`ai-findings: {correctness: block, edge-cases: warn, tests: warn}`.

- **Correctness**, `--input lenses=correctness` on the merge request:
  `block`, exit 1, the finding at calc/calc.go:11 an `error` in SARIF.
  Nothing applied, as for any block: no summary comment on the merge
  request, the finding only in the job's verdict and SARIF.
- **Edge cases**, the same finding: `pass`, a `warning` in SARIF, the
  summary comment posted with it.
- **Both lenses on one line**, `lenses: [edge-cases, correctness]`,
  `review` event: `block`, the correctness finding leading, "also found on
  this line — edge-cases". The try found the second finding's mention lost
  whenever the first was replaced (nit then important too): **fixed**.

Not tried: a real agent; GitHub or GitLab (the fake forge only);
workline's own line switched to block — the maintainer's, after a measure
over five runs a case (ADR-0014).

## 2026-10-06 — a blocked merge request still gets the comment (#226)

No agent: a scratch copy of the `reviewed` fixture, a routing of the
reviewer alone, `ai-findings: {correctness: block}`, the lens answer and
the judge planted, a fake forge with merge request 5; the template's two
jobs by hand, `workline route merge-request --no-apply …` then
`workline apply --line line.json`, the engine of the branch.

- **Push 1**, Average dividing by one less than the count: the judge
  exited 1, one run pending; the apply exited 0 and posted the comment,
  `| **blocks** | correctness | calc/calc.go:11 |` first, the record
  holding the commit, `runs=1`.
- **Push 2**, the fix and a new function: the lens was given only the new
  commit; the same comment edited, the record holding both, `runs=2`.

Not tried: a sandbox merge request (its judge job calls Claude, and the
fix needs a release or a built engine in the workflow); GitLab's
`when: always` and `allow_failure: {exit_codes: [3]}` in a real pipeline
(the template test reads them); a real agent.


## 2026-10-06 — the tests lens judged on whether a test exercises it (#223)

The four tests cases of the evaluation (tests/evaluation/cases/reviewer/
`tests-*`), once, Sonnet finding and Opus judging (`model`), the engine
of the branch; results.tsv holds the rows. Before: #90's step 1, above.

| Case | Before | After |
|---|---|---|
| `tests-shipping-untested` | the tests lens's "no test" refused; credited to other lenses' nits | its "Shipping has no tests" verified (own 1) |
| `tests-rounding-test-passes-either-way` | found; one tests-lens finding refused (judge-no 1) | both verified: the test passing either way, "changed rounding has no failing test" (judge-no 0) |
| `tests-restock-test-asserts-nothing` | found; one tests-lens finding refused (judge-no 1) | the test asserting nothing verified; "Restock edge cases untested" refused, merged on line 48 under the correctness lens's finding and judged by its question (status.md, item 9) |
| `tests-untested-code-outside-the-change` | Snapshot not raised | not raised; two nits on Clear's test, as before |

- **Nothing false shown**: fp 0 for the tests lens in the four cases.
- **Tokens**: 110.8k in all, in and out (shipping 24.3k, rounding 32.7k,
  restock 33.4k, outside 20.4k); the tests lens and its judges 6k to 18k
  in a case, from 6k to 14k before: the judge now reads the tests in the
  cause's folder.
- **Not tried**: five runs a case (ADR-0014); the other lenses' cases
  (their question is today's, conformance `correctness-judge-keeps-its-question`);
  a project whose tests lie in another folder (unit test only).

## 2026-10-06 — a removal's exposed lines are the change's (#224)

The edge-cases case with the guard removed from `Page`
(tests/evaluation/cases/reviewer/`edge-cases-page-loses-its-guard`), once,
Sonnet finding and Opus judging (`model`), the engine of the branch;
results.tsv holds the rows.

| | Before (#90's step 1) | After |
|---|---|---|
| Score | 1/2: the defect not found on the change | 2/2 |
| Where | every lens quoted `from := (p - 1) * size`, kept: an issue (outside 1 a lens) | the same line, `shop/page.go:6`, beside the removal: on the change, verified, by all three lenses (own 1 for edge cases) |
| Tokens | 16.7k in, 1.5k out, 4 calls | 16.5k in, 2.0k out, 4 calls |

- **Also on the change now**: "Overflow in offset computation", an edge-cases
  finding on the same line, verified, merged under the guard's: true of
  the code, weak; not raised in the run before.
- **Not tried**: five runs (ADR-0014); a removal whose defect lies more
  than three lines on (status.md, item 8).

## 2026-10-06 — findings on one line, each judged by its lens (#229)

The restock case (tests/evaluation/cases/reviewer/
`tests-restock-test-asserts-nothing`), once, Sonnet finding and Opus
judging (`model`), the engine of the branch; results.tsv holds the row.

| | Before (#223's rerun) | After |
|---|---|---|
| Score | 2/2 | 2/2 |
| The tests lens on `stock.go:48` | "Restock edge cases untested", merged under correctness, judged by its question: refused | "Restock behaviour untested for accumulation and non-positive quantities": verified |
| The tests lens on `stock_test.go:20` | verified | verified |
| Tokens | 30.8k in, 2.6k out, 6 calls | 32.9k in, 2.0k out, 6 calls |

- **No extra judge call this run**: the other lenses' findings on the
  tests lens's lines were nits, judged by nobody; each extra important
  finding on a shared line costs one judge call (conformance: 3 judge
  calls instead of 1 in `local-review-outputs-json`).
- **Still refused**: an edge-cases finding on `stock.go:49`, judged by its
  own question (judge-no 1, credited to the tests defect it overlaps).
- **Not tried**: five runs (ADR-0014); a run where two important
  findings of two lenses share a line (conformance only).

## 2026-10-06 — a full review's cost, measured offline, then cut (#147)

The 430k run of 2026-10-05: #146's eight commits (`ab5a206..5be9afd`),
every lens, Sonnet finding, Opus judging. Its run folder was kept: four
lens calls of 96.6k tokens in each (one asked again), six judges of 5.3k
to 8.9k.

- **The measure, no tokens**: a scratch clone at `5be9afd`, `workline
  review --base ab5a206 --ai cmd:<a recording agent>`. The agent keeps
  each prompt, answers each lens with what that lens answered in the 430k
  run, and each judge yes: the same findings, the same judge questions.
- **Tokens from characters**: the ten real calls give tokens ≈ 711 +
  0.82 × characters, about 1.2 characters a token on this code and diff,
  not 4. The role's `context.budget` (characters / 4) let a 96.6k call
  through as 29k.

| Call | Before (main) | After |
|---|---|---|
| lenses | 3 calls × 117k chars, 96.5k tokens each: diff 65.0k chars, files 48.2k, lens text 0.5k, facets 2.7k | 1 call, 87.1k chars, 72.0k tokens: hunks 55.5k (files too long to give whole), files whole with the change marked 26.9k, three lens texts 1.4k |
| judges, correctness and edge cases | 5 calls, 33.7k tokens: finding 0.7-0.9k chars, code around 1.0-2.9k, the cause file's whole diff 1.3-6.5k | 5 calls, 26.5k: the hunks within 40 lines of the cause, 1.0-3.1k |
| judges, tests | 2 calls, 129.6k: 68.1k chars of tests each, up to `code-lines-max` | 2 calls, 43.4k: tests up to `tests-lines-max` (300), 18.8k chars |
| **total in** | **452.8k** (the engine of today; 430k in the run, an older engine) | **142.0k** |

- **Cut**: the lenses asked in one call, the change given once
  (`lenses-together`); each file once, whole with the change marked while
  the files fit `code-lines-max` (now 600 lines), else by its hunks; a
  judge shown the change near the cause; a tests judge shown 300 lines of
  tests, the rest named.
- **The budget**: `ai-max-tokens` set for the reviewer, 200000. The
  summary says the tokens used against it; each call's, by lens and by
  judged finding, in `out/review.json` and printed by `workline review`.

**The validation run**, one, the engine of this branch, `ai-max-tokens:
175000` (the cap plus the largest judge call stays under 200k), Sonnet:

- **The lenses' call**: 73.6k tokens in (72.0k estimated), 4.8k out,
  40 seconds. Seven findings, none kept: the agent wrote each lens in
  its `why` ("lens: edge-cases. …"), not as a key, the answer's shape
  having no `lens`; the engine dropped them all, said. No judge was asked.
  **Fixed**: the shape names `lens`; a finding naming none is read as the
  first lens's, said (`lenses-together-finding-naming-no-lens-kept`).
- **Against the three true findings** of the 430k run, by the answer as
  written: a later comment cancelling an agreement (backlog.go) raised;
  the Forgejo comments never flagging bots raised; the Forgejo lookup
  failing silently (`|| echo none`) not raised. Also raised: the local
  forge's notes with no author, so no reply there ever agrees (important,
  not judged); four nits.
- **Spent**: 78.4k of the 200k allowed. A second run with the fix was not
  made: one run was allowed.

**The judged validation run**, one, with the fix (`lens` in the shape),
`ai-max-tokens: 175000`, Sonnet finding, Opus judging (`model`):

| Call | Tokens in | Out |
|---|---|---|
| the lenses, together | 73.7k | 6.6k |
| judge, edge cases, backlog.go:1147 | 6.2k | 0.2k |
| judge, correctness, workline-forge.sh:94 | 5.4k | 0.4k |
| judge, edge cases, workline-forge.sh:90 | 5.5k | 0.8k |
| judge, correctness, workline-forge.sh:74 | 5.3k | 0.5k |
| **total**, 5 calls | **96.1k** | **8.6k** |

- **Cost**: 104.7k in and out, against 430k in and 20k out; the
  commits recorded, every lens having answered and every finding judged.
- **The three true findings**: the Forgejo lookup failing silently
  (`|| echo none`) and a later comment cancelling an agreement, both
  verified; the Forgejo comments never flagging bots, raised by
  correctness and refused by the judge: "the code that would make this a
  defect … is not shown" — Agreement lies in backlog.go, outside the
  thirty lines it reads (status.md, item 7, #127). Two of three shown.
- **Lost by asking the lenses together**: none dropped; every finding
  named its lens: correctness 2, edge cases 4, tests 4 (the 430k run: 2, 4,
  10, three past `findings-max`). The tests lens raised fewer, its four
  the gaps the 430k run showed.
- **Also**: a second correctness finding refused for the same reason
  (Forgejo issues with no author); the local forge never honouring
  `agreed` (nit, raised again); a GitLab sticky creating a note on a 403
  when only an update was asked (nit, new).
- **Spent on the validation**: 78.4k (the first run) and 104.7k, 183.1k.

Not tried: five runs (ADR-0014); the evaluation's cases with the lenses
together (it asks them apart, as it measured them).

## 2026-10-06 — a judge reads the functions the cause reaches (#127)

Offline first, no tokens: the recording agent of #147 on #146's review
(the 430k run's answers, each judge yes), and on the twelve evaluation
cases (a planted finding a defect, each judge yes); origin/main's engine
against this branch's. Tokens from characters by #147's fit (711 + 0.82
× characters a call).

| Judges | Before | After |
|---|---|---|
| #146's review, 7 judges | 79.4k chars, 70.1k tokens | 100.8k chars, 87.7k tokens |
| the evaluation's cases, 9 judges | 33.0k chars, 33.5k tokens | 37.0k chars, 36.7k tokens |

- **What grew**: a judge reads the cause's function whole and up to 200
  lines more (`judge-lines-max`); #146's whole review, about 142k
  estimated before, about 160k after, under `ai-max-tokens`.
- **Noise found and cut on the way**: a capitalised word opening the
  title (`Last`) pulled a function of that name from another package —
  such a word now comes last; an unexported Go function's namesake in
  another package taken for its caller — now its package only.

**The validation run**, one, real: #146's review at `5be9afd`, the lenses
answered with no tokens by the two findings the judged validation run of
#147 (above) had refused "for want of code", the judge Opus
(`WORKLINE_JUDGE=claude:opus`).

| Finding | Before (#147's run) | After |
|---|---|---|
| Forgejo comments never flag bots (workline-forge.sh:94) | refused: "the claimed failure rests on things not shown here" | verified: "`reply.Bot` in Agreement is always false and a write-permission bot passes" |
| Forgejo issues carry no author (workline-forge.sh:74) | refused: "the engine's `reply.Author != is.Author` check … is not shown" | verified, citing `Agreement`'s check |

- **What the judge read**: the 31 lines around the shell cause (no
  function there), `Agreement` (backlog.go, named by the finding, in Go),
  and the script's own functions those lines call (`all`, `api`, `arg`).
- **Tokens**: 16.6k in, 0.8k out, two calls (8.2k and 8.4k in, against
  5.5k and 5.4k before).
- **Not tried**: the evaluation's cases with a real agent (#90 counts the
  judge's false negatives there); a project in another language than Go
  and shell, for real (unit tests only); CI.
