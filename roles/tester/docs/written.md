---
sources:
  - roles/tester/code/write.go#Pre
  - roles/tester/code/write.go#testsDone
  - roles/tester/code/write.go#lintTests
  - roles/tester/code/write.go#lintFound
  - roles/tester/code/said.go#refuseIssue
  - roles/tester/code/said.go#refusePost
  - internal/engine/engine.go#unmoved
  - internal/backlog/backlog.go#Readied
  - internal/kit/verdict/outcome.go#machineKinds
checked: e81df7b34
verified: agent:claude-code
---
# When the tests are written

Part of [the hand-over](hand-over.md).

The issue waits for its tests labelled `workline:needs-tests`, set by
the product owner; an issue labelled ready by hand, with no Draft, is
taken too. `workline test write --issue <n>` on an issue neither
waiting for its tests nor ready, or whose Validation names no scenario,
blocks before any agent is asked, said on the issue with what comes
next: ready it, or name its scenarios ([refusals](refused.md)).

Routed on `issue` (`issue: [tester, developer]`), the issue line runs
the tester on it at once, and moves a ready issue whose scenarios no
Draft nor proof kind stands for back to its tests, by rule, said on it
([the line of states](../../../docs/spec/routing.md#room-in-the-next-state)).
Once the Draft is proven, the engine moves the issue on, no agent asked
([ADR-0059](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#3-tests-wakes-the-tester-its-result-makes-ready)):

- **Each scenario** has its row in `## Tests`, or its reason under
  not tested.
- **The project's lint** passes on the tests: each gate routed on
  `merge-request`, only what they add counted, and
  [the reviewer's rules](../../reviewer/README.md) when it is routed
  there: a test once frozen is no longer the tester's to edit
  ([#1040](https://github.com/JN0V/workline/issues/1040)). A finding there is the
  agent's to mend: it is asked once more, each finding named by its file
  and line
  ([#964](https://github.com/JN0V/workline/issues/964)). Still there,
  the run blocks (`tests-lint`): no Draft, the finding said on it, and
  the line runs the tester again on the issue, told the findings, as
  for any [machine error](../../../docs/spec/routing.md#once-a-state-the-record)
  (a bounded number of tries), before the issue
  is blocked for a person, the last finding named
  ([#1738](https://github.com/JN0V/workline/issues/1738)). A gate that
  cannot run at all (`tests-lint-gate-not-run`) blocks at once: no new
  run mends it. A lint's error repeating the build's own
  error the test is red by ([how](build-errors.md#the-lint)) is not
  counted.
- **Then ready**: `workline:ready` on, the other states' labels off,
  said in the tester's comment on the issue (`state-moved`). An issue
  already ready keeps its labels.
- **No Draft, no move**: a Draft that does not open (its commit
  refused, say) undoes the move, the reason on the issue; a run told
  to open none (`--open-merge-request=false`) is refused before its
  agent is asked, nothing applied
  ([#1280](https://github.com/JN0V/workline/issues/1280)).
