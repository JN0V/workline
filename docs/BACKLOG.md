# Backlog

## Next, in order

The committer and the documentalist first: two roles that prove their worth on
this repository before any other role is added.

1. **Committer: rewrites that can be trusted, and seen.** The hook shows the
   message it committed, not only "message rewritten". A rewrite keeps the
   type, the scope and whatever was not refused (a subject too long is
   shortened, not reworded), checked by `post`. This repository allows its own
   references (`ADR-0001`) in subjects.
2. **The line on the machine, before a push.** A `pre-push` hook runs the
   `merge-request` line on the commits being pushed, with the person's agent:
   the committer checks the range, the documentalist has suspect docs judged
   and proposes its patches, reviewed before the push goes out.
3. **Documentalist: act on what it reports.** The condense and split tasks,
   judged mechanically (no MUST or SHOULD lost, whole parts moved, links in
   place); derived blocks (`workline:derive`) for facts typed by hand; sources
   declared in `.workline/config.yaml` for docs that cannot carry frontmatter,
   like the root README.
4. **Evaluation of both roles.** The "Evaluation" level of
   docs/spec/conformance.md: a real agent on real cases — this repository's own
   refused messages and suspect docs — graded, and the score kept over time.

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
