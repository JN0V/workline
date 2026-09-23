# Backlog

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
- **Project name.** `assembly-line` is provisional; "Assemblyline" is a known
  malware-analysis platform. `takt` and `andon` are taken.
- **Rebuild Make My Dreams on this framework**, once it works.
- **Scope of a task.** Decide where the scope comes from (issue, spec, command
  line, branch) and how fine it is (paths, modules, components). The contract
  already refuses out-of-scope patches and adds an `issue` intention.
