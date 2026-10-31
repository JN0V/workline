---
sources:
  - internal/developer/proof.go
  - internal/developer/developer.go
  - internal/tester/onchange.go
  - internal/tester/missing.go
  - internal/tester/fresh.go
  - roles/developer/role.yaml
checked: 8d58a704
verified: agent:claude-code
---
# The proof and the caps: ready or a Draft

Part of [the developer](../README.md). Step 3 of
[ADR-0054's plan](../../../docs/adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md#it-proves-locally-first).

```mermaid
flowchart LR
  code["Its code,<br/>within bounds"] --> tests["The project's<br/>tests on it"]
  tests -->|"green,<br/>no cap"| ready["Pull request<br/>ready"]
  tests -->|"red, not run,<br/>a cap"| draft["A Draft"]
```

## The proof

- **Where**: before anything is pushed, its code laid on a fresh
  checkout of the commit the run read ([below](#a-fresh-tree)); no
  agent.
- **What runs**: the tester's `test-command`, once (N times for a
  [flaky test](#a-flaky-test-n-runs-of-n)), its JUnit report
  read ([the tester's settings](../../tester/role.yaml)). One setting
  for both roles: the project's tests.
- **The acceptance tests**: they run with the rest, its code laid on
  the tester's Draft when one is open ([the pull request](pull-request.md)).
- **On the touched folders**: [below](#what-runs).
- **Green**: tests reported, none red, the command exited 0.
- **Red**: a test failed, errored or was skipped, or the command failed.
- **Not run whole**: no `test-command`, a patch that does not apply, no
  report or one naming no test: not green, never a pass.

## What runs

- **`proof-command` set**, holding `{packages}`: it runs in place of
  `test-command`, `{packages}` replaced by the touched folders, as the
  tester's curator takes them
  ([its removals](../../tester/docs/removals.md)).
- **The touched folders**: those of its code and of the acceptance
  tests. Each file counts its own folder, else the nearest above it
  holding a test file (`**/*_test.go`, not `tests/**`). A conformance
  case counts its runner's package.
- **The whole suite stays**: the pull request's own CI runs it. A
  package that only imports a touched one is proven there, not here.
- **Whole here too**, said in the Verification and the record: no
  `proof-command`, one with no `{packages}`, or no touched folder
  holding tests.
- **E.g., Go**: `gotestsum --junitfile {report} -- {packages}`.

## A flaky test: N runs of N

For an issue whose tester recorded `flaky-n-of-n` in place of a Draft
([proof kinds](../../tester/docs/hand-over.md#a-proof-kind-not-always-a-new-test)):

- **N runs**: the command above runs N times on the one fresh tree;
  N is the tester's record's, else `flaky-runs` (10).
- **Green only N of N**: each run green, the named test passing in
  each. It stops at the first run that is not; red or not run, the run
  said (`run 2 of 10`).
- **The named test**: matched in the report by its name whole, or its
  last part (`TestRetry` for `pkg.TestRetry`). Not reported: not
  proven, never green.
- **`{test}`** in `proof-command` is the test's name, to run it alone:
  `gotestsum --junitfile {report} -- -run '^{test}$' {packages}`.
  Another issue gets `""`.
- **`proof-seconds`** bounds the N runs together.
- **A review round** runs the same N runs again.

## A fresh tree

Step 3 of
[ADR-0056](../../../docs/adr/0056-the-developers-agent-works-in-a-sandbox-with-tools.md#the-proof-stays-the-engines).

- **Its agent's runs prove nothing**: it runs the tests in its worktree
  to iterate; a green it reports is not read.
- **Never in its agent's worktree**: the proof checks the start commit
  out in a repository of its own, apart, its code laid on it.
- **What the agent left is not there**: an ignored or untracked file
  outside its diff, a cache, a build output.
- **Nothing of the store the worktree shares is read**: it borrows the
  commits' contents only, never the store's refs, replacements, config,
  attributes or hooks, nor the machine's git config. A commit the agent
  put in the start commit's place changes neither its diff nor the proof.
- **The repository's hash**: the tree takes its object format, SHA-1
  or SHA-256.
- **Not whole, not run**: a start commit holding a submodule, or a
  file whose attributes name a filter (git-lfs, an encryption), is not
  checked out whole there. The proof does not run: unproven, the file
  named, a Draft. Never a red or a green on a pointer file.
- **Not apart**: the job's environment outside the tree (the home
  folder, tool caches, the path) is the one the agent ran in. In CI,
  the project's setup steps make it the project's
  ([the triggers](triggers.md#the-projects-environment)); a local run
  uses the person's machine as it is. The proof on today's default
  branch: [#701](https://github.com/JN0V/workline/issues/701).

## The caps

| Cap | Default | Reached |
|---|---|---|
| `ai-max-tokens` | 500000 | the run's agent calls spent it in fresh tokens, as the agent reports them |
| `ai-max-turns` | 300 | one agent call took them all, as it reports it |
| `ai-max-seconds` | 3600 | the engine stops waiting and kills the agent |
| `proof-seconds` | 900 | the tests stopped past it |

- **Fresh tokens**: the input not read from cache, cache writes
  included, and the output. An agent that loops reads its context
  back each turn: its cache reads, a tenth of the price, dominate its
  total; kept apart in the claim's record (`cached`), never counted.
- **A run's time**: the agent's wait and the tests' together, 75 min
  at most, sized for an issue-sized task
  ([ADR-0056](../../../docs/adr/0056-the-developers-agent-works-in-a-sandbox-with-tools.md#caps)).
- **Turns**: told to each call whole, as `--max-turns` is.

How each agent is held ([ADR-0056](../../../docs/adr/0056-the-developers-agent-works-in-a-sandbox-with-tools.md#caps)):

| Cap | `claude` | `cmd:<any>` |
|---|---|---|
| time | killed with the commands it started | killed, its process group whole; `WORKLINE_MAX_SECONDS` tells it |
| turns | `--max-turns`; its `num_turns` read | `WORKLINE_MAX_TURNS` tells it; `turns` in `WORKLINE_CALL` read |
| tokens | read from its JSON after the run, its cache reads apart | `tokens-in`, `tokens-cached`, `tokens-out` in `WORKLINE_CALL` read |

- **Not reported, not checked**: a call that writes no turns or
  tokens is held by its time alone; the run says so (`cap-not-checked`).
- **No live token cap**: tokens are read after the run; turns and time
  hold the agent meanwhile.
- **A cap reached**: a Draft, even when the tests are green. What the
  agent did so far is its code: bounded, proven, pushed. A run that
  needed more than it may spend is a person's to read.
- **Its agent failed** (exited in error, its answer unread): a Draft
  too, the reason in its Verification and its record (`agent-failed`).

## What it opens

- **Branch**: `workline/developer/<issue>`, the engine committing the
  code within bounds.
- **Ready**, green and no cap; **a Draft** otherwise, which wakes no
  agent ([ADR-0050](../../../docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md)).
- **Its body**: `Closes #<n>` — the pick skips the issue while it is
  open — and a `## Verification` saying what ran, each test red, each
  cap. Its other sections: [the pull request](pull-request.md).
- **A tool not found**: when red, the test output's "not found" or
  "not installed" names are listed, in the red finding and the
  Verification. A red from the environment, not the code: the
  project's [setup steps](triggers.md#the-projects-environment) install
  it.

## The record

- **In the claim's comment** on the issue: `proof` (result, command,
  tests passed, those red, why it did not run, `tree: fresh` and the
  commit checked out, `at`, `scope` touched or whole, the `packages`
  run, why `whole`, the names `missing`), `caps` (setting, spent, allowed), and the
  `turns` and `tokens` its agent reported using; a review round's, in
  its round.
- **Read by the auditor**
  ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md)).
