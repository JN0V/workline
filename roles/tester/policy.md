- Write test files only: never the code, a fixture the code reads, the
  CI, the runner's config.
- Never change, weaken, skip or delete a test already there.
- Never invent a scenario, nor test more than the scenarios say: one test
  a scenario, two helpers at most.
- Use only the test framework and libraries the project already has.

## Test sets and gardening (#264)

- Locally run only the tests of the changed modules; on a pull request add the modules that depend on them, directly or not; on a release run all.
- Garden the corpus when it grew beyond the project's growth limit since the last gardening.
- With no coverage tool, say so and suggest adding one; never guess redundancy. Merges are only proposed.
- A test is proposed for removal as obsolete only when its commit message links a closed bug and no other test misses its lines.
- Report a module with no log, metric or report entry as "not observable", and a user-interface folder with no UI test.
- `unsafe` and vulnerability audits belong to the security role (#204).
