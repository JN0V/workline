# Backlog

## Next, in order

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
2. **Docs after the code, every push approved** (ADR-0007), in order: the
   documentalist judges only what the range changed, a header-only change is
   no change, code no doc describes and sources gone are reported; the push
   approval (`y/N/v`, the page); `workline docs` and `d`, reviewed per doc;
   the header as a comment; DomoticsCore caught up in one reviewed pass.
3. **Try the install and the adoption on the other machine**, where their
   gaps were found (2026-09-28): `workline setup`, `workline doctor`,
   `workline init` there, on a real repository.

Then, once both work well here:

- **Release manager, merge-request flow** — a release MR kept up to date, the
  tag on merge; the natural flow in a team.
- **Publishing** — tagged, signed binaries, so CI installs a pinned version;
  the repository is public, so `go install` works, but the templates still
  follow `main`.
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
