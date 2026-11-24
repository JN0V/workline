---
sources:
  - roles/developer/role.yaml
  - roles/developer/code/developer.go#Pre
  - roles/developer/code/developer.go#Post
  - roles/developer/code/conflict.go#conflictPre
  - roles/developer/code/conflict.go#draftBehind
  - roles/developer/code/proof.go
  - roles/developer/code/bounds.go
checked: e81df7b34
verified: agent:claude-code
---
# Developer — what it does, step by step

Part of [the developer](../README.md), which says it in one line a step.

- **Writes the code**, with an agent, in a worktree of the code: it
  reads, edits and runs the tests there ([the agent](agent.md)).
  Its diff is held to its bounds: a frozen acceptance test, a file the
  project's `CODEOWNERS` lists, an agent file or a gate's are reverted,
  no agent judging; another role's prompt is not; on its
  pull request, the same bounds are a check, whoever pushed
  ([the bounds](bounds.md)). A test that does not fit, it objects
  to on the tester's Draft. The Scope is a hint: a file beyond it is
  applied, and listed on the pull request with why, for the reviewer
  ([beyond the Scope](bounds.md#beyond-the-scope)).
- **Proves it** on the project's tests, the tester's `test-command`,
  then opens its pull request: ready when green with no cap reached,
  else a Draft ([the proof and the caps](proof.md)).
- **Opens it on the tester's Draft**, the acceptance tests with it,
  its code and tests in commits apart, its record hidden on it; in CI,
  the agent holds no forge token ([the pull request](pull-request.md)).
  When the tester found that no acceptance test applies, it builds from
  the issue's text, and its pull request says no test was frozen.
- **Runs when routed**: when an issue is labelled ready, and on the
  schedule ([the triggers](triggers.md)).
- **Fixes the review** of its pull request, a fresh run a verdict, a
  gate or a test red, a test the tester rewrote, or a person's comment; the five whys at round three, a person past
  round five ([the review loop](review-loop.md)).
- **Its fixes merge alone** at the merge policy's `normal`, every gate
  passed, the reviewer's verdict on its head, a bug once a test that
  failed on the base showed it; a feature waits for a person's yes. The
  issue's labels tell which, never the
  agent ([the merge](merge.md)).
- **Behind main, in conflict**: one round merges `main` in, settling
  the markers only; never a rebase ([behind main](conflict.md)).
- **A Draft behind main**: the default branch merged into it in the
  run's clone before the agent starts; in conflict, no agent is asked and
  the issue is blocked, the files named
  ([the pull request](pull-request.md#a-draft-behind-the-default-branch)).
