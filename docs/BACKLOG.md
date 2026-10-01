# Backlog

## Next, in order

**Handover (2026-10-01, evening).** The documentalist is being finished
before any other role; roles/documentalist/status.md says where it stands
and what is missing, in order. Since the afternoon: GitLab reached through
its API alone and tried on gitlab.com, a fork's pull request commented,
`workline docs` from a ref and the release held by suspect docs (ADR-0010),
and releases — **v0.1.1** (v0.1.0 is retracted: a tag pushed is never
moved, the Go proxy keeps its first commit), the templates on it, the setup
guide docs/ci.md. To check first: DomoticsCore #114 (its workflows on
v0.1.1) merged; the night's gardening, on DomoticsCore and on workline's
own docs, in parts for the first time (#27) — record both in tried.md.
Then status.md's "Missing" from the top. Sandboxes for trying CI for real:
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
