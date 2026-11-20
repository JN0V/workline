---
sources:
  - roles/tester/code/proof.go#proofOf
  - roles/tester/code/proof.go#proved
  - roles/tester/code/proof.go#sentBack
  - roles/tester/code/needs.go#needsOf
  - roles/tester/code/needs.go#needed
  - roles/tester/code/needs.go#neededFiles
  - roles/tester/code/write.go#writeReading
  - roles/tester/code/write.go#assumesWord
  - roles/tester/code/write.go#assumed
  - roles/tester/code/write.go#draftBody
  - internal/agent/agent.go#ReadsFiles
  - internal/backlog/proof.go#ProofKinds
  - internal/backlog/proof.go#ProofNote
  - internal/backlog/proof.go#NoTestStands
  - roles/tester/code/repro.go#notShown
  - roles/tester/code/repro.go#unreproduced
  - roles/tester/code/repro.go#bugProof
  - roles/product-owner/role.yaml#settings.acts.ask
checked: c46a93d6c
verified: agent:claude-code
---
# Proof kinds and bugs not reproduced

Part of [the hand-over](hand-over.md).

## A proof kind, not always a new test

When no new test fits, the tester's agent names the proof that does
([ADR-0059](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#4-a-proof-kind-not-always-a-new-test)):

| Kind | When | Proof |
|---|---|---|
| `flaky-n-of-n` | a flaky test to make reliable | the named test passes N runs of N |
| `existing-suite` | a refactor | the project's suite, unchanged and green |
| `docs-checks` | a docs change | the documentalist's checks |

- **Recorded**, the kind and its reason, in the tester's comment on the
  issue; no Draft. The issue moves to ready (`proof-recorded`); one
  labelled ready by hand keeps its state.
- **The developer** takes it as it takes one with a Draft, the kind in
  its task ([the pick](../../developer/docs/pick.md#which-issue)).
- **`none`**, no test possible: the issue goes back to the product
  owner's state, refine, the reason said and recorded
  (`no-test-possible`). The product owner then picks a kind, or
  rewrites the text and sends it back to its tests, or, the reason
  being the tester's own, sends it back unchanged once (its next
  `none` recorded `retried`) — or, none of these, the line blocks it
  ([refine](../../../docs/spec/backlog-acts/refine.md#no-test-possible-a-proof-kind-picked));
  with no agent, a person of the project names one in a comment,
  `proof: <kind> — why`.
- **No reason, no answer**: never "no test possible"
  ([how](refused.md)).
- **Read tools** (`claude`): the agent finds what it lacks in a
  read-only copy, writes the test, and lists what it could not confirm
  in `assumes:` notes, shown on the Draft as **Assumed**
  ([ADR-0063 §8](../../../docs/adr/0063-every-open-issue-is-due-waiting-or-done.md#8-one-code-picker-and-tools-where-the-agent-has-them)).
- **Short of code** (`needs: <paths> — why`), an agent without read
  tools: it lacks a file
  its test needs, never "no test possible". Its task already holds, as
  declarations, what the test and the Scope include or import, two
  steps deep, and asks it to name every file it lacks in one note.
  The issue keeps its state,
  the files said on it (`needs-code`); the line gives them to the
  tester, run again at once, those files first in its code
  ([the code picker](../../../docs/spec/code-picker.md)); then a person
  if still lacking
  ([the record](../../../docs/spec/routing.md#once-a-state-the-record)).
- **Sent back unchanged**: the record keeps the digest of the body the
  tester read. The same body back in its tests, no kind picked, goes to
  `workline:blocked` with no agent, once, said on it
  (`sent-back-unchanged`): no back and forth until the hop cap
  ([#815](https://github.com/JN0V/workline/issues/815)). A product
  owner's retest lifts it once: the agent is asked again
  ([#1053](https://github.com/JN0V/workline/issues/1053)).
- **`flaky-n-of-n` names its test**, `test: <name>` under the note,
  and its runs, `runs: <N>`; both kept in the record. The developer's
  proof runs that test N times
  ([N runs of N](../../developer/docs/proof.md#a-flaky-test-n-runs-of-n)).
- **Refused**: a kind it does not know (`proof-kind-unknown`), one with
  no reason (`proof-reason-missing`), a `flaky-n-of-n` naming no test
  (`proof-test-missing`); the issue keeps its state.

## A bug not reproduced

A bug is reproduced before it is fixed
([ADR-0060](../../../docs/adr/0060-the-person-is-on-the-loop.md#4-a-person-validates-by-the-issues-kind)):
its tests must fail on the base. Its kind is its label
([the kind](../../product-owner/docs/kind.md)).

- **Its tests red on the base**: the Draft, as any issue. The fix
  then merges by the policy once every gate passes
  ([the merge](../../developer/docs/merge.md)).
- **Every test green on the base** (`bug-not-reproduced`): no Draft,
  never ready. The issue keeps its state and gets
  `workline:needs-repro`; the tester's comment asks its reporter how to
  reproduce it.
- **Then, with no agent**, the line reads it
  ([not reproduced](../../../docs/spec/routing.md#a-bug-not-reproduced)):
  - its steps edited: the tester runs again;
  - its reporter answers: back to refine, for the product owner;
  - no answer within the product owner's delay (`acts.ask.wait`, 24 h
    at `normal`): Blocked, reason "not reproduced".
- **Never another kind**: a bug not shown is not turned into a feature,
  nor fixed blind.
- **A proof kind on a bug**: only new tests; `flaky-n-of-n`,
  `existing-suite` or `docs-checks` are refused
  (`bug-proof-new-tests`).
