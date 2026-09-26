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
   graded one's provider, and any command can be that agent (`cmd:`). Left: a
   judge that really is of another provider — the adapter, or a `cmd:` wrapper,
   for the one you use — tried on the cases and set in the weekly run. Each
   defect a role lets through becomes a case.

Then, once both work well here:

- **Release manager, merge-request flow** — a release MR kept up to date, the
  tag on merge; the natural flow in a team.
- **Publishing** — tagged, signed binaries, so CI installs a pinned version;
  the repository is public, so `go install` works, but the templates still
  follow `main`.
- **More agents** — Codex, Antigravity, OpenCode adapters (any command already
  runs as `cmd:`, prompt in, proposals out); `independent-of` in the engine;
  generate the model grid from models.dev and Epoch (docs/spec/model-grid.md).
- **Next roles** — reviewer (independent vendor), architect, tester.

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
- **Release notes must not claim what is not built.** The first trial release
  listed the documentalist as working, because its role *definition* was
  committed as `feat(documentalist): …`. A change that only specifies a role is
  `docs`, not `feat`; and the release manager's notes should be checked against
  what the code can actually do (the "capability lie" Make My Dreams detected).
- **Documentalist, what is left.** What roles/documentalist/README.md lists
  as not built yet, and the task that merges cards too short.

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
- **Patches in the forge's apply job.** A `patch` applied there is written to
  the job's checkout and lost: it should become a commit on the merge
  request's branch, or a suggestion. Today the templates run without AI, so
  nothing proposes one.
- **An unknown option exits 2.** Go's option parser exits 2, which the exit
  codes reserve for `human`; the CLI should say 64, as for other misuse.
- **Scheduled runs in the CI templates.** The README says `schedule` runs the
  documentalist; neither template has a scheduled job yet.
- **Your config folder on macOS.** `config.yaml` and the global hooks follow
  Go's config folder (`~/Library/Application Support/workline` on macOS), but
  your own facets are looked for in `~/.config/workline/roles/`: one folder
  for all, or both named in docs/usage.md.
