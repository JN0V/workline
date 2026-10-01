# Documentalist — where it stands (2026-10-01, late night)

How finished the role is: what is built and tried on real repositories, what
is only built, what is missing. tried.md has the story of each try; this page
is the summary to start from.

**In one line:** usable every day on a GitHub or GitLab repository that
takes merge requests — docs judged on each one and gardened at night, by
Claude in CI, from a release of workline (v0.1.1, docs/ci.md), fixes
committed to the branch and reviewed on the forge. Beta: the catch-up of a
large backlog in parts has run once on DomoticsCore, right but for a
badge's link; a fix in parts once followed a stale code comment; a
repository without merge requests is built but untried for real.

## Built and tried for real

| What | Tried on |
|---|---|
| Finding suspect docs from git history, cascades cut, propagation at the release | workline, DomoticsCore |
| Hygiene without AI: budgets (body only), duplicates, links inside and outside (lychee), identifiers gone, superseded decisions cited, derived blocks, code no doc describes, sources gone | workline, DomoticsCore |
| Adoption (`workline init`): each doc's sources proposed | workline, DomoticsCore (63 of 64 docs) |
| Judging a suspect doc whole, against its sources whole beside the diffs; a fix it cannot vouch for kept, `checked` left (ADR-0012) | evaluation; DomoticsCore |
| A fix may grow a doc by a tenth; a refused doc left out, the others applied | DomoticsCore gardening (PR #106) |
| `checked` after a squash or a rebase: the commit that brought it stands for it | fixture, by hand |
| On each pull request in CI: Claude judges, the App commits the fix, checks rerun, the line skips its own commit | workline PR #10; DomoticsCore PR #108, a real bug-fix pull request |
| Gardening at night: one pull request per task, at most 3 waiting | DomoticsCore, two runs by hand (PR #106, #109) |
| The push counts suspect docs in one line, asks nothing (ADR-0010, 0011) | workline, DomoticsCore |
| A fork's pull request: judged without secrets, its report commented from the repository's side, one comment kept (ci/github/workline-fork.yml) | JN0V/workline-sandbox #1, from jn0v-lab's fork |
| On a GitLab merge request: Claude judges, the token commits the fix to its branch, the next pipeline skips the line's own commit | gitlab.com JN0V/workline-sandbox !1; a fork's merge request untried |
| Condense, split, merge a card, merge a repeated passage (Opus) | evaluation; workline |
| Staying on the task: a spec's example left as it is when the default it does not show changes | evaluation (`stays-on-the-task`, workline PR #10 replayed), 3 in 3 on Sonnet |

## Built, tried without an agent

- **Not asked again** (ADR-0013): a fix not vouched for records `judged`;
  a gardening task waits while its pull request is open. On a copy of
  DomoticsCore with #109 open: the docs judged whole wait, the parts go.
- **A cap on a run's tokens** (`ai-max-tokens`): conformance only.
- **`checked` earned by what was read** (ADR-0014, step 0): a doc whose
  sources did not all go whole into its task is told `checked` cannot move,
  and a fix moving it is refused (`checked-unread`), on every task the
  documentalist judges — a merge request, gardening, `workline docs`, the
  release, and condensing, splitting or merging, which give no source; a
  doc they create is born without `checked`. Conformance only, no agent
  run on it yet. The `checked` already moved without being earned are put
  back to `judged`: workline's 10 docs (0669684), DomoticsCore's 16 on its
  local branch docs/undo-unearned-checked, not pushed
  (docs/research/documentalist-fixes-reviewed.md).
- **A partial claim settled by another part** (workline #29, where a stale
  code comment won over the setting): a passage one part finds partial and
  another supports is not handed to the fix. Conformance only; to watch on
  the next runs in parts.

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
   `checked` already moved are put back (above); its release waits for the
   person to merge #35 and fix/partial-settled-by-another. Step 1 is
   built and its baseline measured (below, "Measures"). Step 2 is built:
   `count-off`, `value-left`, the removal rule and "a comment is not
   evidence", measured with no agent (below). Step 3's gate is passed
   with a real agent (below): nothing false vouched for or written, so no
   new mechanism; the fixes it left unmade met by wording where they could
   be (below, after step 3); next, step 4's acceptance on the held-out set.
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
6. Smaller: budgets merge one level deep; Opus's notes see more than the
   task asks (unused); Copilot CLI on a free plan unmeasured (needs the
   owner to turn Copilot Free on).

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
`supported` resting on a comment are parked (docs/BACKLOG.md).
