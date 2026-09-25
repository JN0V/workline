# Backlog

## Next, in order

The committer and the documentalist first: two roles that prove their worth on
this repository before any other role is added.

1. **Evaluation, run often.** The evaluation exists (tests/evaluation/), keeps
   the exact models, efforts and tokens of each score, can force one model and
   effort, and summarises the runs; its scores are only worth their number of
   runs, since a model answers differently from one run to the next. Still to
   do: pin exact model ids per tier, so a new generation comes in only once
   measured; run it on a schedule; set each role's tier and effort on what the
   runs show; add a case for each defect a role lets through; grade with a
   judge from another provider what no check can.

Then, once both work well here:

- **Release manager, merge-request flow** — a release MR kept up to date, the
  tag on merge; the natural flow in a team.
- **Publishing** — tagged, signed binaries, so CI installs a pinned version;
  the repository is public, so `go install` works, but the templates still
  follow `main`.
- **More agents** — Codex, Antigravity, OpenCode adapters, and a generic one
  running any command given as the agent (prompt in, proposals out), so an
  existing factory can plug its own; generate the model grid from models.dev
  and Epoch (docs/spec/model-grid.md).
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
- **Verdicts as SARIF.** The role contract says findings map to SARIF; nothing
  writes it yet. It would put workline's findings in GitHub code scanning and
  GitLab Code Quality, next to the other tools of an existing pipeline.
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
