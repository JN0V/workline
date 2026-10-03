# Backlog

## Next, in order

Where the documentalist stands, what is measured and what comes next for
it: roles/documentalist/status.md.

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

- **Releases with the project's own tool** (ADR-0017, accepted): no
  release manager; workline works with release-please, releaser-pleaser,
  semantic-release and the like. Done: the role and its cases retired;
  workline's own `workline.yml` judges with the engine built from the
  commit. In order: release-please
  (with the App, GoReleaser after it) cuts one real release of workline;
  the documentalist's release hold on the release pull request a tool
  opens; later, doctor and `init` detect the tool or recommend one.
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
- **Work state lives in the forge's issues, not in commits.** A bug is an
  issue; as its state moves, the issue moves (comments, labels, closed by
  the pull request that fixes it). No handover committed: no dated
  "Handover" paragraph here, no pull request carrying only one. The next
  roles take it into account: which role opens, updates and closes an
  issue, and what a session's "where to restart" becomes (an issue, a
  pinned one, the forge's project board). The foundation exists
  (ADR-0016): any role's `issue` outcome, and the generic forge operations
  — open or keep an issue, label it, comment on it by a marker — on
  GitHub, GitLab, any forge by `cmd:`, or the clone (`local`). Left: which
  role opens, updates and closes an issue for a bug it finds; a fix's pull
  request linked to its issue and closing it; a project's own roadmap
  files replaced by its issues.
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
  `docs`, not `feat`; and the notes — the release tool's changelog since
  ADR-0017 — should be checked against what the code can actually do (the
  "capability lie" Make My Dreams detected).

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
- **The weekly sample, left** (ADR-0015, built 2026-10-02, conformance
  only; tried on GitLab with v0.2.2, roles/documentalist/tried.md). Its
  first run on GitHub; a `checked` found false on a forge, its label and
  merge request. A judge of another provider, when there is one
  (`cmd:`). The sample's verdicts kept as a measure (results.tsv), and a
  read by a person recorded as one. Commits before `Workline-Model` name
  no model: their independence reads `unknown`. A week the job does not
  run is not caught up (`--since` by hand). A `checked` naming another
  repository is put back by a person, not by the merge request.

- **Where workline writes, left** (ADR-0016, built 2026-10-02,
  conformance only). Gitea and Forgejo natively, as GitHub and GitLab are,
  once the command (ci/forgejo/workline-forge.sh) has run on a live
  instance — Codeberg first; only its label operation runs against a mock.
  The doctor checks `gh` for
  GitHub only: a `cmd:` forge or `local` is not looked at, nor `local` in
  a CI the engine does not recognise.

### Documentalist beyond our repositories

ADR-0014's checks run on six public repositories with no agent
(docs/research/documentalist-genericity.md): fitted to C-like code, English,
`.md`, three-part versions. In order:

1. **Paths not ASCII.** Done: every git call reading paths sets
   `core.quotePath=off`, the documentalist's lists use `-z`
   (`path-not-ascii-judged`).
2. **History and decision records by setting.** Done: changelogs and
   release notes however named or foldered, `architecture-decisions/`,
   `adr001-…`; the `history` and `decisions` settings add globs
   (`count-off-not-in-history-by-convention`,
   `superseded-decision-other-conventions`).
3. **The comment test, wider and loud.** Done: docstrings, `#!` scripts,
   `--`, `{{! }}`, `{# #}` and others; a type it does not know is said
   (`comment-style-unknown`; `docstring-not-evidence-refused`).
4. **The 20,000-character cap a setting.** Done: `whole-chars`, in
   characters as the engine measures a task before the call, 20,000 by
   default; the task grows with it (`whole-chars-setting`). Raising it
   costs tokens each night: measure on the evaluation before raising the
   default.
5. **Doc-like files the globs skip, reported.** Done: `docs-not-read`,
   when gardening, on `init` and by the doctor, with how to include a
   README (`docs-not-read-reported`, `doctor-docs-not-read`). Reading
   `.rst`, `.mdx`, `.adoc` waits for a header form for each.
6. **`value-left`: history markers and version formats.** Done: markers
   skipped as ranges are; `versions.pattern` and `versions.files`
   (`value-left-not-a-marker`, `value-left-version-pattern`,
   `value-left-version-file`).
7. **Words by language.** Done for the removal rule: typography made
   plain, glue and fact words in English and French, by `language` or
   read from the doc (`removal-rule-reads-french`). Count and limit
   words (`count-off`) are still English: "1 200 lignes" unread.
8. **Sources proposed without an agent.** Done: by name, on `init --ai
   none`, never written (`init-without-ai-proposes-by-name`); 261 of
   backstage's 452 docs, about half right, 0 to 5 elsewhere.
