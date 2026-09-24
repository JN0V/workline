# Backlog

## Next, in order

1. **Documentalist, second slice** — size budgets (per doc, section, folder;
   a minimum card size too), duplicates, dead links, identifiers gone from the
   code, and the AI call that judges a suspect doc (docs/research/documentalist.md).
2. **Release manager, merge-request flow** — a release MR kept up to date, the
   tag on merge; the natural flow in a team.
3. **Publishing** — tagged, signed binaries and a public repository, so CI can
   install workline (`go install` fails on a private module).
4. **More agents** — Codex, Antigravity, OpenCode adapters; generate the model
   grid from models.dev and Epoch (docs/spec/model-grid.md).
5. **Next roles** — reviewer (independent vendor), architect, tester.

## Parked

Topics raised and parked, so they are not lost. Newest last.

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
- **Granularity of a task's scope.** The scope comes from the `ready` work item
  (routing spec). Still open: paths, modules or components, and how a module is
  declared.
- **Release notes must not claim what is not built.** The first trial release
  listed the documentalist as working, because its role *definition* was
  committed as `feat(documentalist): …`. A change that only specifies a role is
  `docs`, not `feat`; and the release manager's notes should be checked against
  what the code can actually do (the "capability lie" Make My Dreams detected).
