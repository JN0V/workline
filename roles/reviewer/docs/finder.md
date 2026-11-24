---
sources: [roles/reviewer/role.yaml#settings, roles/reviewer/prompts/lenses/repository.md, roles/reviewer/code/finder.go, roles/reviewer/code/reads.go, internal/engine/engine.go]
checked: 1221daba6
verified: agent:claude-code
---
# Reviewer — the repository finder

Part of [the reviewer](../README.md), on a merge request; decided in
[ADR-0073](../../../docs/adr/0073-one-finder-reads-the-whole-repository-once.md).

```mermaid
flowchart LR
  lenses["Lenses and judges<br/>nothing holds the merge"] --> finder["Finder<br/>reads the repository"]
  finder --> judge["A judge<br/>apart from it"] --> verdict["The verdict"]
```

The lenses read the files a change touches. The finder reads past
them, once, when nothing else holds the merge request.

## What it does

- **What it is given**: the whole change, base to head, and the
  repository's files by name, up to 400; nothing a lens found nor a
  judge said.
- **What it reads**: what it asks, as [a judge does](judge.md#when-the-judge-reads):
  a file's lines, a definition by its name, the lines naming a word,
  read from git at the head, within `judge-read-rounds` and
  `judge-read-lines-max`.
- **What it looks for**: one defect the change causes outside its own
  lines: a caller, a callee's contract, a test no workflow runs, a
  target platform's limit. Its prompt: `lenses/repository.md`, which a
  project may replace in `.workline/roles/reviewer/prompts/lenses/`.
- **Its answer**: yes, the defect in one sentence and the line causing it
  quoted; no; or unsure.

## What follows

- **A yes's quote found again** at the head is a finding of the lens
  `repository`: on the change, for its author; outside it, an issue, as
  a lens's ([the quotes](run-quotes.md#56-the-quotes-and-whose-finding-it-is)).
  Not found again: dropped, said.
- **Judged** by a judge apart from the finder: whether the code quoted
  fails as it says.
- **Holds the merge** when `ai-findings` blocks it: `{repository: block}`,
  or `block` for every lens. It stands until a commit answers it
  ([a block stands](verdict.md#a-block-stands-until-a-commit-answers-it)).

## When, and how often

- **On a merge request**, once every lens of the run answered and no
  other finding blocks; never on a draft, nor on a machine.
- **A split review**: in the verdict's run when it blocks, in the run
  apart when it warns ([apart](apart.md)).
- **Once a merge request**: the record keeps the head it read
  (`<!-- workline:finder … -->`). Asked again only while its own block
  stands at a newer head; found no more, the block goes.
- **The summary says** what it came to, that it waits for nothing else
  to hold the merge, or that it did not run: then the next run asks it.
- **Cost**: one call, up to two more for its reads, one judge for a
  finding; within `ai-max-tokens`.
- **Off**: `repo-finder: false`.
