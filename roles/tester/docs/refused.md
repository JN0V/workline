---
sources:
  - roles/tester/code/write.go#Post
  - roles/tester/code/said.go#refuseIssue
  - roles/tester/code/said.go#refusePost
  - roles/tester/code/said.go#sayBlock
  - roles/tester/code/write.go#toMend
  - roles/tester/code/notrun.go#notRun
  - roles/tester/code/kept.go#guardedSaid
  - internal/kit/verdict/outcome.go#machineKinds
checked: c46a93d6c
verified: agent:claude-code
---
# When the tester refuses an issue

Part of [the hand-over](hand-over.md).

An issue it does not take is refused before any agent is asked
([which](written.md)). The refusal is written on the issue too
([ADR-0059](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#1-every-refusal-claim-wait-or-block-is-on-the-issue)):

- **One comment of the tester's**, edited on each refusal: why, and
  what comes next.
- **Its own tests refused too**: a test green on the base, a scenario
  with no test: asked again once first, told which; still refused, said
  in the same comment, the run blocks. In
  `workline:needs-tests`, the line then blocks the issue, `workline:blocked`,
  the reason on the board
  ([routing](../../../docs/spec/routing.md#once-a-state-the-record)).
- **No test written**, the agent's answer read and no kind named: no
  test applies, its notes the reason, handed over to the developer as
  [`none`](proof-kinds.md#a-proof-kind-not-always-a-new-test) is
  ([#794](https://github.com/JN0V/workline/issues/794)).
- **No reason, no answer**: an agent that names no kind, writes no
  test and gives no reason, or stops at its turn cap first, did no
  work: asked again once, then `no-test-written`, a machine error the
  line retries, never "no test possible"
  ([#1194](https://github.com/JN0V/workline/issues/1194),
  [one run](../../../docs/spec/role-contract.md#one-run)).
- **A scenario kept as today** ("as today", "unchanged"): the agent
  says it not tested, naming the existing test that guards it; its test
  would pass on the base and be refused. A name the base's tests do not
  hold, or none, and the Draft says plainly that no existing test was
  found to check it
  ([#1359](https://github.com/JN0V/workline/issues/1359)).
- **An answer that does not read** (a list item indented one space
  too far): asked again once, with the reader's error; still
  unreadable, refused and said, no third call
  ([#761](https://github.com/JN0V/workline/issues/761)). Prose before
  the list, or a `file:` beside a patch naming its own file, is read
  ([#1148](https://github.com/JN0V/workline/issues/1148),
  [one run](../../../docs/spec/role-contract.md#one-run)).
- **A test the project's runner skips** (a suite its `test_filter`
  does not list): the block names the file and the filter, and the next
  run tells the agent ([#1149](https://github.com/JN0V/workline/issues/1149)).
- **Not a job's log alone**: a person on the issue sees why no Draft
  came.
