# Backlog

## Next, in order

**Handover (2026-10-02).** The documentalist vouched for docs it
never read: the work is **ADR-0014, accepted**, in its order. Step 0 is
built: the engine refuses a `checked` whose sources were not all given
whole; the `checked` already moved is put back to `judged` — workline's 10
docs (0669684), DomoticsCore's 16 on its local branch
docs/undo-unearned-checked, not pushed. Its release waits for the person to
merge #35 and this branch (fix/partial-settled-by-another); a doc a split
creates is born without `checked` there too. The 16 reviewed verdicts are
confirmed by a second, independent check
(docs/research/documentalist-fixes-reviewed.md). Step 1 is built — the
`drifted` fixture, two cases grading both sides, four grades, pass rates,
the held-out set (docs/spec/conformance.md) — and its **baseline is
measured** (roles/documentalist/tried.md, 2026-10-01: five rounds on Sonnet
on each engine, 0.85M tokens; never `checked` over a falsehood 3/5 and 2/5
on the version bump, a frozen line count and a test count vouched for).
To measure again, as it was run (`-v` for the notes):

```sh
for i in 1 2 3 4 5; do WORKLINE_EVAL=claude:sonnet go test -count=1 -timeout 60m -run 'TestEvaluation/documentalist/drifted' ./tests/evaluation/; done
# the summary pools every run of a case: keep one engine's rows apart (the workline column)
awk -F'\t' 'NR==1 || $2 == "<commit>"' tests/evaluation/results.tsv > "$TMPDIR/runs.tsv" && go run ./tests/evaluation/summary "$TMPDIR/runs.tsv" | grep drifted
```

**Step 2 is built**: `count-off`, `value-left`, the removal rule and a
comment is not evidence (a `claim` beside a patch cites each word it takes
out), measured with no agent (roles/documentalist/tried.md: the three wrong
workline fixes refused, 11 of 12 right ones pass, 172493e refused on a
comment). **Step 3's gate is passed** with Sonnet (tried.md; 1.33M tokens):
`drifted` five rounds, every point 4/5 and 5/5 (0/5 at the baseline),
claims given at the first answer; eight reviewed fixes replayed, nothing
false vouched for or written, so no quote-checked claims ADR. What remains
is fixes not made (docs/research/documentalist-fixes-reviewed.md). Two
are met before step 4, conformance only, no agent run on them yet: the
task and every refusal say a count the engine gives needs no claim; a
refusal names only the places refused and the rest of the patch to send
again; and each run keeps the agent's answers and the accepted claims
(`out/agent-answer.txt`, `out/claims.yaml`), so step 4 can read them. The
others are parked below (Engine and CI, "Left by step 3"). **Step 4's
acceptance does not hold** (roles/documentalist/status.md, "Measures";
tried.md, 2026-10-02; 4.63M tokens): on the held-out set, five runs each,
nothing false vouched for or written, every `count-off` and `value-left`
the rules cover reported, fewer false alarms than real, the versions and
flags made — but a judged doc costs 12.8% more than at the baseline
(a96306c), past the tenth: one DomoticsCore run in five asked twice,
`removal-uncited` on a word beside a count. The weekly one-in-ten sample
waits for it. Its three costs are met, conformance and replay only
(status.md, "after step 4"): no claim for the rest of a line whose count
the engine fixes, nor for glue taken out of a line reworded; a table's
total a `count-off`; a line a diff adds may be quoted; a patch git cannot
apply refused saying which line it misquotes; a doc in parts recorded
though claims were dropped; a history doc never condensed. **Step 4
run again** on a15fc0d (tried.md, 2026-10-02; 3.96M tokens, the same
method, the baseline's first nights reused): **does not hold**, bar 6
only — +13.9% a judged doc the first night, +5.5% without DomoticsCore's
one run in five asked twice (the HeapTracker pitfall rewritten with no
claim, refused rightly, withdrawn). The three costs are met as measured:
docs in parts recorded and not asked again, CHANGELOG never picked, no
word beside a count refused; nothing false let through by the narrowed
rule. **What a re-ask costs is cut**, conformance and replay only
(status.md, "after step 4 again"): a place refused by the removal rule,
a hunk misquoting, or a move of `checked` nobody can vouch for is no
longer asked for again — the rest of the fix is applied, the doc
recorded `judged`, the place reported for a person (the run's findings,
the gardening request's body); only a patch of which no hunk quotes
right is asked again. `count-off` reads "line count (607)", "line
count: 607" and "a 216-line `X.h`". **Step 4 holds**, run a third
time on d890be5 (tried.md, 2026-10-02; 3.56M tokens): all six bars,
+5.7% a judged doc the first night, no re-ask in 65 nights; **ADR-0014
steps 0 to 2 are accepted**. **Next**: the release — the person merges
#35 and this branch (fix/partial-settled-by-another), then a tag; then
DomoticsCore's pin bumped to it and its branch docs/undo-unearned-checked
(16 `checked` put back) pushed for review; then the weekly one-in-ten
sample of the docs vouched for (ADR-0014, Amendment for who reads). To
watch on the next runs: a misquoted hunk withheld with a right fix in it
(core's counts, index's footer version: once each in five runs, where a
re-ask had mended 3 in 4); a right claim naming no `doc` in a two-doc
answer, unread, and refused as "no claim"; an answer whose YAML breaks,
reported (`agent-invalid-output`) but not asked again, the doc judged
again the next night. DomoticsCore: branch docs/narrow-wide-sources (two
docs narrowed) waits, not pushed; its main runs workline v0.1.1.
The night's gardening, on DomoticsCore and on workline: record it in
tried.md, against the review. Sandboxes for trying CI for real:
github.com/JN0V/workline-sandbox and its fork jn0v-lab/workline-sandbox;
gitlab.com/JN0V/workline-sandbox, its two tokens set as masked variables
(the GitLab one expires about 2026-10-31).

The committer and the documentalist first: two roles that prove their worth on
this repository before any other role is added.

1. **Evaluation, the judge.** The evaluation keeps the exact model, effort and
   tokens of each score, compares models (`WORKLINE_EVAL=claude:<model>@<effort>`),
   summarises the runs (`go run ./tests/evaluation/summary`) and runs every
   week (tests/evaluation/schedule/, docs/spec/conformance.md); models follow
   the aliases and a change is noticed (ADR-0004); tiers were set on its
   measures (the documentalist condenses on `frontier`); a `judge` check asks
   what no check can grade of an agent `WORKLINE_JUDGE` names, never of the
   graded one's provider, and any command can be that agent (`cmd:`). The judge
   takes the best independence available and says which (ADR-0005); the
   weekly run judges on `claude:sonnet`. Left: a judge of another provider,
   when there is one. Each defect a role lets through becomes a case.
2. **The review is on the merge request** (ADR-0011, 2026-09-30): the push
   approval (ADR-0007, 0008) is off unless a person asks for it; an agent
   pushes a branch and opens the pull request, never merges. Left: tell an
   agent from a person on the forge — a bot account or a GitHub App with no
   right to merge — so the rule is a check, not a wish; the editor and a
   dialog on macOS and Windows, for those who ask for the approval.
3. **Docs judged on the merge request and by gardening** (ADR-0010,
   accepted and built): on GitHub, GitLab, a fork's pull request, and a
   repository without pull requests. Where it stands, and what is missing
   in order: roles/documentalist/status.md.
4. **Judge docs far behind in parts** (ADR-0009, accepted on its measures;
   ADR-0012, a doc judged against its sources whole): turned on for
   DomoticsCore's nightly gardening (its PR #107), tried once (#113); on
   workline's own docs nightly since #27. Next: watch those nights and
   record them in roles/documentalist/tried.md.
5. **Try the install and the adoption on the other machine**, where their
   gaps were found (2026-09-28): `workline setup`, `workline doctor`,
   `workline init` there, on a real repository.

Then, once both work well here:

- **Release manager, merge-request flow** — a release MR kept up to date, the
  tag on merge; the natural flow in a team.
- **Install without Go in CI.** Done (v0.1.0, 2026-10-01): releases with
  their binaries (GoReleaser), the image ghcr.io/jn0v/workline the GitLab
  template runs in, the GitHub templates downloading a pinned release;
  GitLab judge 112 → 28 s, apply 182 → 15 s. Left: a reusable action
  (`uses: JN0V/workline@v1`) so a repository's workflow is a few lines and
  is updated by its tag; DomoticsCore's workflows moved onto a release.
- **More agents** — Codex, Antigravity, OpenCode adapters (any command already
  runs as `cmd:`, prompt in, proposals out); `independent-of` in the engine;
  generate the model grid from models.dev and Epoch (docs/spec/model-grid.md).
- **Next roles** — reviewer, architect, tester. The reviewer does not wait for
  another provider: it takes the best independence available and says which
  (ADR-0005). Among its first rules: a comment says why the code is there,
  readable in five years without the bug; the bug's story belongs in the
  commit message. The mechanical part (long comment blocks added, ticket
  codes, "used to", "the bug was") before any AI.
- **Which model reviews which.** Evaluation cases with defects planted on
  purpose — a wrong edge case, a comment telling a bug's story — reviewed by
  Opus, Sonnet and Haiku, at each independence level of ADR-0005: what each
  one catches, and at what cost. Before the reviewer is built.

## Parked

Topics raised and parked, so they are not lost; removed once done. Newest
last.

### Roles and method

- **Consolidate the test corpus.** Tests should prove behaviour over time, not
  the one fix just made. Before adding a test, look for one covering the same
  behaviour and extend it; periodically merge redundant tests, using coverage
  overlap and mutation score. A tester or test-curator role.
- **Five whys as a shared skill.** Root cause before any countermeasure in the
  defect ledger; used by reviewer, developer, tester, architect; triggered when
  a defect is recorded or a role blocks repeatedly.
- **Context and memory management.** Long agent sessions fill their context and
  degrade (compaction, context rot). Make My Dreams addressed it; decide whether
  the framework handles it, e.g. what a role writes down for the next run
  instead of keeping it in context.
- **Rebuild Make My Dreams on this framework**, once it works.
- **Style (vale).** Parked 2026-09-28: without AI, the risk for a doc is to
  be wrong or out of reach, which the checks already cover; a style linter
  adds a binary, rules to keep and findings people learn to ignore. Worth it
  when a project already keeps a vale config: run vale when `.vale.ini`
  exists, as lychee is run, and report without blocking.
- **Release notes must not claim what is not built.** The first trial release
  listed the documentalist as working, because its role *definition* was
  committed as `feat(documentalist): …`. A change that only specifies a role is
  `docs`, not `feat`; and the release manager's notes should be checked against
  what the code can actually do (the "capability lie" Make My Dreams detected).

### Engine and CI

- **Granularity of a task's scope.** The scope comes from the `ready` work item
  (routing spec). Still open: paths, modules or components, and how a module is
  declared.
- **Let a project raise a rule to block.** `enforce` only lowers a rule
  (block → warn → off); the documentalist's hygiene findings are reported and
  cannot be made to block.
- **Settings merge one level deep.** A project that sets
  `budgets: {doc-lines: 300}` drops the other budgets; the documentalist says
  so (`setting-missing`), but a deep merge would be less surprising.
- **An unknown option exits 2.** Go's option parser exits 2, which the exit
  codes reserve for `human`; the CLI should say 64, as for other misuse.
- **A finding of an earlier round outlives what it said.** The engine keeps a
  round's finding that no later round reports again; `nothing-tracked`, true
  before the first round applied, was still shown after it. `init` no longer
  reports it; the rule itself holds for any finding a round makes untrue.
- **Installing lychee on Linux without brew or cargo.** `setup` then only
  names the page; its release binaries could be fetched, pinned.
- **Your config folder on macOS.** `config.yaml` and the global hooks follow
  Go's config folder (`~/Library/Application Support/workline` on macOS), but
  your own facets are looked for in `~/.config/workline/roles/`: one folder
  for all, or both named in docs/usage.md.
- **New code no doc describes** (raised 2026-10-01, left out of ADR-0014).
  A file added outside every doc's sources is ignored unless `documented`
  is set, and neither workline nor DomoticsCore sets it. Deriving
  `documented` from the declared sources is circular (a file outside them
  never matches); by their folders (`dir/**`) it works, but a source like
  `library.json` must not widen to a whole component. Start without AI: the
  count by folder, tests, examples, vendored and generated files left out;
  a migration for repositories already adopted (`doctor`); only then an
  agent proposing the doc a file belongs to, its "none" recorded so it is
  not asked again, and a file attached only if the doc stays within
  `parts-max`.
- **Left by step 3** (ADR-0014; docs/research/documentalist-fixes-reviewed.md),
  none a falsehood vouched for or written; to watch in step 4:
  - *A line no part claimed* (3cd196d's line 60): judged in parts, no part
    spoke to it, so the fix never saw it. The parts' coverage (ADR-0009):
    `uncovered` reports a line naming a name from the code no part speaks
    of; a line naming none, that no part claims, goes unseen.
  - *A fixed bug left for a person* (92feb08, d784398): neither rewritten
    nor removed, `value-left` reporting its old version; not vouched for.
  - *A `supported` part verdict resting on a comment* (d43b3f2, two
    claims): the removal rule checks a fix's claims, not a part's. Harmless
    while a doc judged in parts never moves `checked`.
  - *A claim reaches three lines either way*: one claim that holds, given
    for line 10, also cites a removal at line 12 its quote says nothing of
    (found writing the case refused-part-named-the-rest-kept). A claim is
    checked to exist, not to support the change.
- **Your allow-list blocks the bot's commits** (reproduced 2026-10-01).
  With `~/.config/workline/allowed-identities` holding your address alone,
  `workline run-role committer --event merge-request --input
  range=d43b3f2~1..d43b3f2 --ai none` blocks with `identity`, author and
  committer: the App's commits are
  `336169029+workline-jn0v[bot]@users.noreply.github.com`, and GitHub's
  web merges `noreply@github.com`; any local range holding them is refused
  (a branch carrying the App's fix pushed again after a rebase, a merge
  request checked by hand). Not seen in CI, which has no user list. Fix:
  the user adds the bot's pattern, or the committer allows the forge's own
  identities a project names (its App, `noreply@github.com`) by setting.
