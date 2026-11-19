---
sources:
  - roles/committer/code/committer.go#Pre
  - roles/committer/code/committer.go#rewriting
  - roles/committer/code/committer.go#Post
  - roles/committer/code/committer.go#Trailers
  - roles/committer/code/committer.go#ownLine
  - roles/committer/code/committer.go#ownLines
  - internal/kit/commitmsg/commitmsg.go#KeepsHeader
  - cmd/workline/main.go#hookCmd
  - internal/engine/engine.go#withTrailers
  - internal/engine/engine.go#runState.trailers
  - internal/engine/engine.go#Rewriter
  - internal/engine/engine.go#madeBy
  - roles/committer/role.yaml
checked: c46a93d6c
verified: agent:claude-code
---
# Committer — the rewrite

Part of [the committer](../README.md).

## Rewrite (the agent, 30 s at most)

- **On by default**, with an agent
  ([ADR-0050](../../../docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md)):
  the agent gets the findings and the staged diff, and proposes one
  `commit-message`, or a `note` to split the commit.
- **30 s at most**: the hook says which agent rewrites, and how long at
  most, before it starts. Past `ai-max-seconds`, every call and retry
  together, the message is refused with its reasons, and kept.
- **Its seconds said**: `14 s in all — committer 14 s (rewrite 13 s)`;
  over its 30 s, [said once](../../../docs/spec/routing.md#each-steps-seconds-a-moments-budget);
  each call's in its run's `out/calls.jsonl`.
- **`rewrite: false`**, the project's or yours, turns it off; so does a
  project's `ai: none`, and no agent at all. The hook then refuses in about
  a second, keeps your message in the file git named, and prints
  `git commit -e -F <that file>` to take it up again.

## Judge (`post`, no AI)

- The rewrite must pass the same checks, but `possible-internal-code`: the
  agent judged it.
- It must keep every trailer of the original (`Co-Authored-By:`, `Refs:`…), on
  its own line in the last paragraph (`trailer-dropped`).
- A rewrite keeps the author's type, scope and breaking mark, when the header
  was not what was refused; adding a scope the author left out is allowed
  (`header-changed`).
- It adds no line git could read as a `Workline-` trailer, however its
  key is cased or spaced: those are the engine's (`trailer-added`).
- One commit, one message: several proposals are refused (`several-messages`).
- A `note` ends the run as `human`: a person splits the commit.

A refused rewrite is asked once more, of a stronger model (`promote-after: 1`).
A rewrite that passes replaces the message, and the commit goes on; the hook
prints the message committed, since it is no longer the one you wrote.

- **It says it is a rewrite**: the engine adds `Workline-Role: committer`
  and `Workline-Model: <agent>:<model>` to its trailers, so the
  [weekly sample](../../../docs/ci.md#the-weekly-sample) draws it
  ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md)).
- **Never your words**: nothing of the message it replaced reaches the
  commit, its internal codes above all.
- **Still your commit**: workline never takes a rewritten commit for one
  it made itself; the line judges it as yours.
