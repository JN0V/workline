# Documentalist — where it stands (2026-10-07)

How finished the role is: what is built and tried on real repositories, what
is only built, what is missing. tried.md has the story of each try; this page
is the summary to start from.

**In one line:** usable every day on a GitHub or GitLab repository that
takes merge requests — docs judged on each one and gardened at night, by
Claude in CI, from a release of workline (v0.2.3, docs/ci.md), fixes
committed to the branch and reviewed on the forge. Beta: the catch-up of a
large backlog in parts has run once on DomoticsCore, right but for a
badge's link; a fix in parts once followed a stale code comment; a
repository without merge requests is built but untried for real.

Sandboxes for trying CI for real: github.com/JN0V/workline-sandbox and
its fork jn0v-lab/workline-sandbox; gitlab.com/JN0V/workline-sandbox, its
two tokens set as masked variables (the GitLab one expires about
2026-10-31).

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: open. The docs to split and the
   default of judging in parts (Missing 3, 4) are still to do; open bugs
   [#120](https://github.com/JN0V/workline/issues/120) and
   [#121](https://github.com/JN0V/workline/issues/121) to fix or park.
2. **The main issue validated on real use**: open. No issue of the role
   holds a Validation section the maintainer has signed.
3. **AI verdicts measured**: in part. ADR-0014's step 4 holds on the
   `drifted` cases (Measures, below); the other evaluation cases have one
   run each, not five.

## Built and tried for real

| What | Tried on |
|---|---|
| Finding suspect docs from git history, cascades cut, propagation at the release | workline, DomoticsCore |
| Hygiene without AI: budgets (body only), duplicates, links inside and outside (lychee), identifiers gone, superseded decisions cited, derived blocks, code no doc describes, sources gone | workline, DomoticsCore |
| Adoption (`workline init`): each doc's sources proposed | workline, DomoticsCore (63 of 64 docs) |
| Judging a suspect doc whole, against its sources whole beside the diffs; a fix it cannot vouch for kept, `checked` left (ADR-0012) | evaluation; DomoticsCore |
| A fix may grow a doc by a tenth; a refused doc left out, the others applied | DomoticsCore gardening (PR #106) |
| `checked` after a squash or a rebase: the commit that brought it stands for it | fixture, by hand |
| A shallow clone lacking a doc's `checked`: `shallow-clone`, never a silent pass; the GitLab template fetches the whole history; the doctor warns | gitlab.com JN0V/workline-sandbox, probe pipeline, no agent; the new engine on copies only (tried.md, 2026-10-06) |
| On each pull request in CI: Claude judges, the App commits the fix, checks rerun, the line skips its own commit | workline PR #10; DomoticsCore PR #108, a real bug-fix pull request |
| Gardening at night: one pull request per task, at most 3 waiting | DomoticsCore, two runs by hand (PR #106, #109); gitlab.com JN0V/workline-sandbox, its schedule played (!3, issue #1) |
| The push counts suspect docs in one line, asks nothing (ADR-0010, 0011) | workline, DomoticsCore |
| A fork's pull request: judged without secrets, its report commented from the repository's side, one comment kept (ci/github/workline-fork.yml) | JN0V/workline-sandbox #1, from jn0v-lab's fork |
| On a GitLab merge request: Claude judges, the token commits the fix to its branch, the next pipeline skips the line's own commit | gitlab.com JN0V/workline-sandbox !1; a fork's merge request untried |
| The weekly sample on GitLab: Opus reads a doc Sonnet vouched for, true, one comment on the tracking issue, none more on a rerun | gitlab.com JN0V/workline-sandbox #2 (v0.2.2); GitHub untried, a `checked` found false untried |
| Condense, split, merge a card, merge a repeated passage (Opus) | evaluation; workline |
| A release tool's pull request held as the release: every doc suspect since the last release, the fix on a merge request of its own into `main` (ADR-0017) | a copy of JN0V/workline-sandbox, `local` forge, Claude; on GitHub with release-please untried |
| That merge request follows `main` (ADR-0034, `workline follow`): rebuilt on main's new tip when main moved `checked` beside its `judged`, the pull request kept and mergeable; left alone once a person committed to it | JN0V/workline-sandbox #38, run by hand, no agent; the CI job on a push to `main` and GitLab (mock only) untried |
| Staying on the task: a spec's example left as it is when the default it does not show changes | evaluation (`stays-on-the-task`, workline PR #10 replayed), 3 in 3 on Sonnet |

## Built, tried without an agent

- **Not asked again** (ADR-0013): a fix not vouched for records `judged`;
  a gardening task waits while its pull request is open. On a copy of
  DomoticsCore with #109 open: the docs judged whole wait, the parts go.
- **A cap on a run's tokens** (`ai-max-tokens`): conformance only.
- **A doc too large to condense or split** (`too-large-to-condense`,
  `too-large-to-split`): reported for a person, the next doc condensed.
  Replayed on a copy of workline at 14f0db4, a recording agent: the task
  that stopped the night of 2026-10-07 is not built, the next doc is the
  task (tried.md, 2026-10-07).
- **`checked` earned by what was read** (ADR-0014, step 0): a doc whose
  sources did not all go whole into its task is told `checked` cannot move,
  and a fix moving it is refused (`checked-unread`), on every task the
  documentalist judges — a merge request, gardening, `workline docs`, the
  release, and condensing, splitting or merging, which give no source; a
  doc they create is born without `checked`. Conformance only, no agent
  run on it yet. The `checked` already moved without being earned are put
  back to `judged`: workline's 10 docs (0669684), DomoticsCore's 16
  (0d86737 on its main) (docs/research/documentalist-fixes-reviewed.md).
- **A partial claim settled by another part** (workline #29, where a stale
  code comment won over the setting): a passage one part finds partial and
  another supports is not handed to the fix. Conformance only; to watch on
  the next runs in parts.
- **The first scheduled nightly's two faults** (workline, 2026-10-02,
  tried.md): a doc found true but not vouched for is said "nothing found
  wrong", a fix "fixed in N places", in the findings and beside each doc
  in the gardening pull request; and a CI night that proposes the docs
  judged whole goes round again, the parts judged the same night on the
  `fix` pull request (ADR-0013, amended). Conformance only: the next
  nights to watch, with a second task's tokens.
- **The weekly sample of the docs vouched for** (ADR-0014, step 4;
  ADR-0015; `workline sample`): one in ten of the docs whose `checked`
  the documentalist moved in the last week, drawn the same on a rerun,
  each read whole against its sources at the commit `checked` names by the
  project's judge (`sample.judge`, `WORKLINE_JUDGE`), never the model the
  commit's new `Workline-Model` trailer names; every quote checked, a
  comment no evidence; sources too large for a read whole reported as an
  unearned `checked`. Written, with no agent, to one tracking issue a
  comment a week; a false `checked` labels it `documentalist-step-0` and
  a merge request puts it back. In CI weekly (workline-sample.yml, the
  GitLab template's second schedule); workline's read by Opus. Tried on
  GitLab (above); its three faults met since: a commit written unquoted
  in `sample.after` failed every run (read now, or refused when YAML
  would mangle it), the role GitLab asks to set a schedule's variables
  (docs/ci.md), and the week chosen in CI (`WORKLINE_SAMPLE_WEEK`). The
  rest conformance only (fake readers).
- **Where it writes** (ADR-0016): on the project's forge, wherever it
  lives. `forge: local` keeps the comments, issues, labels, gardening's
  merge requests (local branches, nothing pushed) and the sample's writes
  in the clone (`.git/workline/`), read with `workline
  issues`; `forge: cmd:<command>` plugs another forge by a JSON contract
  (docs/spec/forge-command.md), with a sample for Forgejo and Gitea
  (ci/forgejo/). With no forge, an issue is refused as every other write;
  in CI, the local forge refuses its writes. Conformance only (fifteen
  cases, a fake command); the Forgejo script's labels run against a mock
  of its API (ci/forgejo/workline_forge_test.go), no live instance.
- **Beyond our repositories** (docs/research/documentalist-genericity.md,
  fixes 1 to 3): a path that is not ASCII is read; changelogs, release
  notes and decision records are known by the names other projects give
  them, and by the `history` and `decisions` settings; the comment test
  reads Python docstrings, `#!` scripts, SQL, Handlebars and Django
  templates, and says a type it does not know. Fixes 4 to 8: the cap on
  sources judged whole is `whole-chars`; the doc-like files the role does
  not read (`.rst`, `.mdx`, package READMEs) are said, when gardening, on
  `init` and by the doctor; `value-left` skips history markers ("First
  available in", `versionadded`) and reads the project's version pattern
  and version files; the removal rule compares plain typography, with
  French glue and fact words; `init` without an agent proposes sources by
  name. Conformance and the probe's public copies, no agent; `whole-chars`
  raised costs tokens no agent run has measured yet.

- **Docs for their reader** ([#193](https://github.com/JN0V/workline/issues/193)):
  when gardening, with no agent, user pages are checked for decisions,
  docs and issues named without a link, paragraphs and table cells too
  long, a flow with no diagram, a page no link reaches (docs/checks.md,
  12). Run on workline's own pages (tried.md, 2026-10-07). Not built: the
  agent reading the pages they point to
  ([#256](https://github.com/JN0V/workline/issues/256)).

## Tried once for real

- **Judging in parts** (ADR-0009): measured on the evaluation (Sonnet in
  parts 12/12 ×3, at five times the tokens), then on DomoticsCore's
  README.md (2026-10-01, pull request #113): 8 parts and a fix, 180k
  tokens, the fix right but for a badge's link.
- **A solo repository without pull requests** (ADR-0010): a copy of
  WaterMeter, pushed to a local remote (2026-10-01, tried.md): adopted by
  `workline init`, the push counting the docs with no agent, `workline
  docs` judging them, the release held by those left. Three defects found
  and guarded.

## Missing, in the order it matters

1. **A fix vouches for what it did not check.** Reviewed on 2026-10-01:
   every version the agent wrote was right, but moving `checked` vouched
   for frozen numbers still false beside them — line counts, test counts,
   on five DomoticsCore docs whose sources the agent never saw whole (16c660b
   and three more), the judge accepting it — and a bug long fixed was
   rewritten rather than removed (92feb08); the same
   version was left stale in sibling docs. The plan, reviewed by seven
   independent reviewers: ADR-0014 (accepted). Its step 0 is built — the
   engine refuses a `checked` the agent could not have earned, and the
   `checked` already moved are put back (above), released in v0.2.0. Step 1 is
   built and its baseline measured (below, "Measures"). Step 2 is built:
   `count-off`, `value-left`, the removal rule and "a comment is not
   evidence", measured with no agent (below). Step 3's gate is passed
   with a real agent (below): nothing false vouched for or written, so no
   new mechanism; the fixes it left unmade met by wording where they could
   be (below, after step 3). Step 4's acceptance, on the held-out set, does
   **not hold** (below): nothing false vouched for or written, every
   count reported, but the tokens a judged doc costs are 12.8% over the
   baseline, past the tenth the bar allows. The three costs behind it are
   met, and measured again (below): the same five bars hold, and bar 6
   fails again, +13.9%, one run in five asked twice for a rewrite given
   no claim. The re-ask, which sent the whole task again, is cut: what
   holds of a fix is applied, the place refused reported (below, "After
   step 4 again"). Run a third time, **step 4 holds** (below): +5.7% a
   judged doc the first night, no re-ask; steps 0 to 2 accepted. The
   weekly one-in-ten sample is built (above), tried on a GitLab sandbox,
   not run on workline yet. Next: its first weeks, watched on its issue.
2. **Sonnet cites lines a line off in long docs** (twice in two runs on
   DomoticsCore): such a hunk is now placed where its quoted lines are,
   replayed on both answers; to watch on the next runs.
3. **Docs needing more than 8 parts** on DomoticsCore: six (10 to 16
   parts). Two narrowed to 8 (branch docs/narrow-wide-sources, not pushed
   yet); four to split, each holding more than one doc:
   reference/eventbus-architecture.md, components/core/project-context.md,
   architecture/component-configuration-pattern.md,
   components/webui/project-context.md.
4. **Judging in parts off by default**: on a solo repository far behind
   (WaterMeter, tried.md), 7 of 10 suspect docs were too large to be judged
   whole and went to a person. Measured since (status above): to decide
   whether it is on by default.
5. **workline's own docs**: README, usage and most specs are suspect, far
   behind; gardened nightly in parts since #27, first night to watch.
   roles/documentalist/README.md, which needed 15 parts, is split into
   four pages of 3 to 6 parts each, their sources narrowed to their files.
6. Smaller: Opus's notes see more than the task asks (unused); Copilot CLI
   on a free plan unmeasured (needs the owner to turn Copilot Free on).

## Measures

tests/evaluation/results.tsv; `go run ./tests/evaluation/summary`, which
gives pass rates — the runs earning every point — and marks fewer than five
runs (ADR-0014). On Sonnet 5.5 (2026-09-30 and 10-01), read that way: finds
planted defects whole 6/6, in parts 3/6; stays on the task 3/3; fixes a doc
made false, confirms one still true, never confirms a stale doc made false,
propagates at the release, opens an issue on a spec, merges a passage, one
run each, all passed; condenses on Opus 6/6. Too few runs to be a measure,
and none of these cases planted what went wrong for real.

ADR-0014 step 1 (2026-10-01): the `drifted` fixture, real shapes from the
review (docs/spec/conformance.md, "Real shapes, both sides graded"), and
two cases on it — a version bump on a merge request, and gardening — each
grading never `checked` over a planted falsehood and `checked` on a clean
control; the grades `finding`, `no-finding`, `judged-is-head`,
`checked-unchanged`; a held-out set kept out of the design. Proved without
an agent (fake `cmd:` agents on every `go test`). **The baseline**
(tried.md, 2026-10-01; five runs on Sonnet each, on this engine and on the
one before step 0, within a run of each other): the clean control `checked`, the right fixes
made and what is true kept, 5/5 everywhere; never `checked` over a planted
falsehood 5/5 in gardening, 3/5 before and 2/5 after on the version bump — a
frozen line count vouched for (2 runs each), a test count no source shows
(1 run); about 42k tokens a run. The three `count-off` findings were lost
in every run, step 2 not being built; the engine reports them now.

To measure the `drifted` cases again, as they were run (`-v` for the
notes):

```sh
for i in 1 2 3 4 5; do WORKLINE_EVAL=claude:sonnet go test -count=1 -timeout 60m -run 'TestEvaluation/documentalist/drifted' ./tests/evaluation/; done
# the summary pools every run of a case: keep one engine's rows apart (the workline column)
awk -F'\t' 'NR==1 || $2 == "<commit>"' tests/evaluation/results.tsv > "$TMPDIR/runs.tsv" && go run ./tests/evaluation/summary "$TMPDIR/runs.tsv" | grep drifted
```

ADR-0014 step 2, no agent (tried.md, 2026-10-01): `count-off` on a copy
of DomoticsCore, 60 reports, all real, the three known counts among them;
none on workline, whose docs state no file's line count. `value-left`,
the bot's fixes on DomoticsCore replayed: 51 reports, 47 real, 4 false
(a version marking when a feature came, a version-history row), after a
version range was tuned out; the badge's link and MQTT's siblings found.
Not yet run with a real agent: the tokens and the fixes it makes once
told the counts are step 4's to measure.

The removal rule and "a comment is not evidence" (tried.md, 2026-10-01):
every word a fix takes out is cited in a `claim` beside it — a quote the
engine finds in a source file outside its comments, or a name gone from the
code — or refused. The reviewed bot fixes replayed with the claims they
could honestly give: the three wrong ones on workline refused (d43b3f2
uncited; d38a0e4 and #29 resting on a comment, each comment reported); 11
of 12 right ones pass, 172493e refused, its only evidence a template's
header comment (it undid d38a0e4, which the rule refuses). Conformance and
replay only, no agent run on it yet.

ADR-0014 step 3, with a real agent (tried.md, 2026-10-01; Sonnet, 1.33M
tokens): the `drifted` cases, five rounds, every point 4/5 in gardening
and 5/5 on the version bump (0/5 each at the baseline), never `checked`
over a planted falsehood 5/5 on both (2/5 on the bump before), the claims
the removal rule needs given at the first answer in nine runs of ten, at
about the baseline's tokens (bump +1%; gardening +17% on average, one run
asked again). Eight of the reviewed bot fixes replayed, one run each: no
`checked` moved over anything false, no true claim removed, no comment
winning, no fixed bug rewritten; the right versions made. What remains is
fixes not made: line counts the engine gives left by the agent (3 of 3
runs that had them), a right fix withdrawn after a refusal, a line no
part claimed (docs/research/documentalist-fixes-reviewed.md).

After step 3 (conformance only, no agent run on it yet): the task, the
facet and every refusal say a line count the engine reports off is
brought to its number with no claim, the engine's count being the
evidence — the stated number and its "~" or "about" exempt, in prose, a
table or a listing, another number not; a refusal names only the places
refused and the rest of the doc's patch, to be sent again unchanged; each
run keeps the agent's answers as they came and the accepted claims
(`out/agent-answer.txt`, `out/refused-<n>-answer.txt`, `out/claims.yaml`).
The line no part claimed, the fixed bug left for a person, and a part's
`supported` resting on a comment are parked (#106).

ADR-0014 step 4, the acceptance (tried.md, 2026-10-02; Sonnet, 4.63M
tokens): the held-out set — WaterMeter, DomoticsCore after #114,
workline at ffdb06d — five runs of the gardening path each, graded
against the code by another agent. Per bar:

1. **Held.** `checked` moved five times (two WaterMeter wiring guides in
   two runs, roles/documentalist/README.md once), each doc read whole:
   nothing false. None moved on DomoticsCore, whose sources never fit.
2. **Held.** No true claim removed, no comment winning; a version-history
   row thinned (Wifi's "1.4.1 | Current release" became 1.7.0).
3. **Held.** `count-off` 60 reports, all real, none missed within the
   rule; `value-left` 57 real and 25 false a full run, none missed.
4. **Held, read for the held-out set.** No doc there is both right and
   shown by its sources: the docs `checked` could fairly move on are
   vouched for in some runs and `judged` in others (WaterMeter 2 of 5, as
   on the baseline engine; the README 1 of 5). The clean control is the
   evaluation's (step 3: 5/5).
5. **Held for versions and flags** (5/5 on every doc reached, GPIO34 5/5,
   the line counts 5/5 where the baseline made none); one other right fix
   less often: HeapTracker's pitfall 3/5, the baseline 5/5.
6. **Failed.** The first night, the same task on both engines, five runs:
   +12.8% a judged doc (+3% WaterMeter, +6% workline, +26% DomoticsCore,
   one run asked twice and its doc left out; +5% without it).

What sets it back (ADR-0014: each step's bar measured before the next):
step 2's removal rule asks again for a word reworded beside a count
("Watch the 800-line limit"), and a fix withdrawn rather than cited (the
pitfall, OTA's total). Judging in parts costs 0.23M to 0.32M a night and,
`claims-dropped` keeping the doc unrecorded, is asked again every night
(8 of 11 nights; the baseline engine too).

After step 4, the three costs (conformance and replay only, no agent run
on them yet; step 4 to run again): the rest of a line whose count the
engine fixes, but for a fact of its own (another number, a version, a
name, a negation, a quantifier, a conjunction, a tense), and glue taken
out of a line reworded ("the", "per"; never such a fact), need no claim; a
table's total is a `count-off` the engine sums; the task says a line a
diff adds may be quoted; a patch git cannot apply is refused saying which
line it misquotes (the run asked twice had skipped a blank line). The
reviewed bot fixes replayed again: the three wrong ones refused, 11 of 12
right ones pass, as before; the step 4 answers refused — OTA's total,
"Watch" beside 930 lines — pass. A doc in parts whose claims were dropped
is recorded, the drop said for a person, and not asked again until a
source changes; a history doc (a changelog) is never picked for
condensing.

ADR-0014 step 4 again, on a15fc0d (tried.md, 2026-10-02; Sonnet, 3.96M
tokens; the same method, copies and caps, the baseline's first nights
reused): **does not hold**, on bar 6 again. Bars 1 to 5 hold: `checked`
moved only on WaterMeter's two wiring guides (4 runs of 5), nothing
false; no true claim removed, and every line taken out without a claim
is an engine count, the line otherwise unchanged; `count-off` 61, all
real, OTA's `Total` among them; the version fixes identical in every
run, OTA's `Total` 5/5 (0/5), HeapTracker's pitfall 4/5 (3/5; the
baseline 5/5). The three costs are met as measured: a doc in parts is
recorded, claims dropped or not (9 of 9 nights in parts), and not asked
again; CHANGELOG.md is never picked; no word beside a count refused. Bar 6:
+13.9% a judged doc the first night (+3% WaterMeter, +7% workline, +28%
DomoticsCore), +5.5% without the one run in five asked twice — a
rewrite given no claim, refused rightly, then withdrawn. A re-ask
sends the whole task again (two docs, 36k) for one place; over every
night, a doc judged whole costs 22.0k, as before (21.8k).

After step 4 again (conformance and replay only, no agent run on it yet;
step 4 to run again): **what holds of a fix is applied**. A place the
removal rule refuses (`removal-uncited`, `citation-unchecked`,
`comment-not-evidence`), a hunk git cannot apply for misquoting the doc,
or a move of `checked` nobody can vouch for (`checked-unread`,
`checked-over-count-off`) is no longer asked for again: the rest of the
doc's fix is applied, the place left as it was, `checked` kept and
`judged` recorded, and the place reported for a person — the run's
findings, and the body of a gardening request. Only a patch of which no
hunk quotes the doc right is asked again: re-asks after a citation
refusal brought the place back right 1 time in 5 (d784398's versions;
16c660b, HeapTracker's pitfall and OTA's `Total` and `OTA.cpp` count
withdrawn), after a misquote 3 in 4. `post` narrows the patches; the
engine applies them as narrowed, held to the same catalogue and bounds
(docs/spec/role-contract.md). `count-off` reads "`OTA.cpp` line count
(607)", "line count: 607" and "a 216-line `X.h`": on DomoticsCore at
91100ed, 59 reports before, 60 after, the one added real (`OTA.cpp`, 759
lines); none on workline, WaterMeter's 2 unchanged. The reviewed bot
fixes replayed: the three wrong places still withheld, 11 of 12 right
ones pass, as before, and 3cd196d's right line 60, lost with the wrong
change when it was reverted, now kept. Step 4's refused answers replayed
through `post`: HeapTracker's applies its two counts, the pitfall
withheld and reported; OTA's 607 passes whole; the misquoted OTA answer
applies all but its misquoted hunk — which held a right count, mended by
the re-ask then, a person's now.

ADR-0014 step 4 a third time, on d890be5 (tried.md, 2026-10-02; Sonnet,
3.56M tokens; the same method, the baseline's first nights reused):
**holds**, all six bars. `checked` moved only on WaterMeter's two wiring
guides (3 runs of 5), nothing false; no doc with a place withheld moved
it. `count-off` 62 places, all real; `value-left` more real than false.
The first night costs +5.7% a judged doc (+3.1% WaterMeter, +7.2%
workline, +7.4% DomoticsCore); over every night 20.8k a doc judged whole
(22.0k before); no re-ask in 65 nights. Three places withheld and
reported in the gardening request: HeapTracker's pitfall once (a right
claim naming no `doc`, unread in a two-doc answer), and two misquoted
hunks that carried right fixes — core's two counts, index's footer
version — now a person's, where a re-ask had mended 3 in 4. To watch:
those losses, and an answer whose YAML breaks (one night in 65), which
is reported but not asked again, the doc judged again the next night.

After step 4 the third time (conformance and replay only, no agent run on
them): those three losses are met. **Step 4 was measured at d890be5,
before them**; these fixes only recover right fixes that were refused,
they refuse nothing new. A claim with no `doc:` in an answer patching
several docs is read for the only doc whose patch changes its lines with
its quote under that doc's sources (or its name in the lines removed);
one that could stand for none or several is reported
(`claim-unattributed`), the place refused saying a claim was given
without `doc:`, and the task says each claim names its doc. A hunk
misquoting is mended where its place is beyond doubt: context differing
by blank lines alone, fitting the doc at one place only; else each run
of changed lines placed by the lines it removes, found once — never
where in doubt. An answer that is not valid YAML is asked again once, on
the same tier, with the reader's error. Replayed: the reviewed bot fixes
give the same verdicts as before (the three wrong refused, 11 of 12
right pass); step 4's 64 answers judged again, the only change five
places now applied, each right and cited — HeapTracker's pitfall, core's
two counts, index's footer version (twice), and a README line in parts
(check_versions.py checks a tag whenever one is supplied).
