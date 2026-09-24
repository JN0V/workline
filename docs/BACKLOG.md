# Backlog

## Next, in order

1. **Routing drives the forge too** — `workline route` gains `--forge`,
   `--target` and `--no-apply`, and the CI templates call
   `workline route merge-request` instead of each role, so one routing governs
   the machine and the forge. Conformance cases first.
2. **Release manager, merge-request flow** — a release MR kept up to date, the
   tag on merge; the natural flow in a team.
3. **Publishing** — tagged, signed binaries, so CI installs a pinned version;
   the repository is public, so `go install` works, but the templates still
   follow `main`.
4. **More agents** — Codex, Antigravity, OpenCode adapters, and a generic one
   running any command given as the agent (prompt in, proposals out), so an
   existing factory can plug its own; generate the model
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
- **Documentalist, what is left.** Derived blocks (`workline:derive`), style
  (vale), links to other sites (lychee), docs citing a superseded ADR,
  freshness, the pending list as one tracking issue and its release gate,
  backpressure (`max-open-merge-requests`), the checklist comment without AI,
  and the other kinds of task: propagate, condense, duplicates, split.
- **Let a project raise a rule to block.** `enforce` only lowers a rule
  (block → warn → off); the documentalist's hygiene findings are reported and
  cannot be made to block.
- **Settings merge one level deep.** A project that sets
  `budgets: {doc-lines: 300}` drops the other budgets; the documentalist says
  so (`setting-missing`), but a deep merge would be less surprising.
- **docs/spec/role-contract.md is over its budget** (314 lines for 200), found
  by the documentalist on this repository: split it into cards.
- **Local events beyond `commit-msg`.** The global hook forwards every git
  hook but runs roles only on `commit-msg`; `pre-push` could run the
  documentalist before anything leaves the machine.
- **Verdicts as SARIF.** The role contract says findings map to SARIF; nothing
  writes it yet. It would put workline's findings in GitHub code scanning and
  GitLab Code Quality, next to the other tools of an existing pipeline.
