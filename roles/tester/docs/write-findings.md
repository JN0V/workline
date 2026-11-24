---
sources:
  - roles/tester/role.yaml
  - roles/tester/code
  - internal/kit/testmark
  - defaults/ci/github/workline-objection.yml
  - defaults/ci/github/workline-answer.yml
  - defaults/ci/github/workline-gardening.yml
checked: e81df7b34
verified: agent:claude-code
---
# What a write refuses

Part of [the tester](../README.md), after [writing an issue's tests](write.md).
Each finding below opens nothing.

## Before the agent writes

| Finding | When; nothing is opened |
|---|---|
| `issue-not-ready` | the issue is labelled neither `workline:needs-tests` nor `workline:ready` |
| `tests-lint` | the tests do not pass the project's lint, still after the agent is asked once more with each finding; the line runs the tester again, a bounded number of times, before a person is asked |
| `tests-lint-gate-not-run` | a lint gate cannot run at all: the issue is blocked at once, no retry mends it |
| `no-named-scenario` | its Validation names no scenario: asked of the product owner, never invented |
| `scenario-without-test` | a scenario has no `covers` entry, no test nor reason; asked again once with the names first |

## After the agent answers

| Finding | When; nothing is opened |
|---|---|
| `declared-test-missing` | a declared test is none of the new tests red on the base, nor of the test files written; asked again once, then blocked: the product owner reads a blocked issue again |
| `declared-test-ambiguous` | a declared id is the end of several new tests (`test_x` for `a.test_x` and `b.test_x`); asked again once for a longer id, then blocked |
| `test-line-changed` | a patch changes or removes a line of a test file the base holds |
| `base-test-skipped` | a test that ran on the base is skipped with the tests laid on it |

## After the check on the base

| Finding | When; nothing is opened |
|---|---|
| `intention-refused` | a patch reaches a file outside `test-paths` |
| `test-green-on-base` | a test is not proven red; asked again once, told how to mend it (one passing on the base may keep its scenario as today), then blocked. A feature's answer keeping every scenario as today is `none`, handed over to the developer |
| `no-new-test` | the report names no test of the files written: the project's runner did not run them, the filter that may skip them named (a PlatformIO `test_filter`); the next run gives the agent this reason |
| `bug-not-reproduced` | a bug's tests all pass on the base: `workline:needs-repro`, its reporter asked ([how](proof-kinds.md#a-bug-not-reproduced)) |
| `no-test-written` | no test proposed, the answer unread or refused whole; an answer that does not read is asked again once first. An answer read with no test nor kind, but a reason, is `none`, handed over to the developer: no acceptance test applies |

Each refusal is written on the issue ([how](refused.md)).
