---
sources:
  - roles/documentalist/role.yaml#settings.ai-max-turns
  - roles/documentalist/role.yaml#settings.whole-chars
  - roles/documentalist/code/documentalist.go#preRun.giveSourcesNow
  - roles/documentalist/code/documentalist.go#preRun.writeTask
  - roles/documentalist/code/documentalist.go#readsTree
  - roles/documentalist/code/documentalist.go#readingTask
  - roles/documentalist/code/documentalist.go#readByLog
  - roles/documentalist/code/documentalist.go#holdJudgedDocs
checked: dd48ff276
verified: agent:claude-code
---
# Documentalist — the agent reads the repository

Part of [the agent's tasks](tasks.md), in
[the documentalist](../README.md).
[ADR-0074](../../../docs/adr/0074-agents-read-the-repository-the-engine-stops-preparing-their-input.md),
step 3: the agent reads each doc's sources itself, whatever their size.

## With `claude`

- **The task gives the doc whole** with its line numbers, why it is
  suspect, and **names its sources**: never their text, never a diff.
- **A read tree**: a copy of the repository at `HEAD`, read-only. Its
  tools are `Read`, `Glob` and `Grep`: no shell, no edit, no network.
- **Turns**: `ai-max-turns` (40), told in the task; time,
  `model.timeout`.
- **What it reads is logged** with the call, in `out/calls.jsonl`: each
  file's path and blob.
- **`checked` is earned by that log**: it moves only for a doc whose
  every source file the log holds. A source under a folder: each file
  in it.
  - **A source not read**: the fix is kept, `checked` stays, `judged`
    records when, and the doc is said for a person (`checked-unread`).
  - **A source in another repository** is not in the tree: it is given
    in the task, as below.

## With another agent

A `cmd:` agent, or a fake, does not read the repository:

- each doc is given its sources in the task, whole when they fit
  `whole-chars` (20,000 characters), else what changed in them, and
  `checked` cannot move;
- a doc too far behind for either goes to a person;
- every run says so: `input-cut`, a warning.

## A doc already judged

- A doc whose `judged` names a commit after which no source changed is
  not asked again, whatever its sources' size: a person reads it.
- A `judged-in-parts` a doc still records holds it the same way: read
  where written, never written again.
