---
sources: [roles/reviewer/role.yaml, roles/reviewer/lenses, internal/builtin/reviewer, internal/backlog/spec.go, routing.default.yaml]
checked: 245015c
verified: agent:claude-code
---
# Reviewer — a spec before it is built

Part of [the reviewer](../README.md). A spec is read before it is built
([#128](https://github.com/JN0V/workline/issues/128)): a file or an issue
on a machine, or, on the forge, each issue the product owner refines.

## On a machine

`workline review --spec docs/x.md`, or `--issue 12`, runs the reviewer's
`spec` event (`--input spec=<file>` or `issue=<n>`): the same pipeline as
for code ([a run](run.md)), with lenses of its own.

- **What it reads**:
  - the file as it reads in the working tree, or the issue's body from the
    forge, up to `spec-lines-max` (the rest cut, said: `spec-cut`);
  - for a lens that `needs: code`, the code the spec names, at HEAD: the
    code files it names, whole, then the functions it names (backticked,
    called, or capitalised) the files do not hold, up to `code-lines-max`
    lines, the rest named. None named: that lens is not asked, said.
- **The spec lenses** (`spec-lenses`, `subject: spec` in their front
  matter), together in one call, each judged by its own question:
  - **ambiguous**: words two readings would build or test differently;
    the judge: can they be taken two ways that build different things?
  - **unverifiable**: a Verification that proves nothing; the judge: can
    it not tell whether the Need is met?
  - **out-of-scope**: a Need asking what the Scope leaves out, or the
    reverse; the judge: does the Need ask outside the Scope?
  - **contradicted**: what the spec says of today's code, the code says
    otherwise, its symptom quoted from the code; the judge: does the code
    contradict it? It reads the function the symptom lies in and those it
    reaches, up to `judge-lines-max`.
- **The quotes**: each cause found again in the spec as given
  (`docs/x.md:9`, `#12:3`, a line of the body), a symptom in its file at
  HEAD; not found, dropped (`finding-unfounded`). Every finding is the
  spec's author's: the product owner, or the person who wrote it.
- **The judge, the verdict, the questions** as for code: each important
  finding judged, a no dropped; what the lenses find warns (`ai-findings`,
  by lens too); an open point only a person settles is a `decision`.
- **No record** is kept: a spec is read whole each run.

## On the forge

Opt-in, off by default: the reviewer after the product owner in the
gardening line
([ADR-0020](../../../docs/adr/0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md)):

```yaml
routing:
  events: {schedule: [documentalist, product-owner, reviewer]}
```

- **Which issue**: one a run, the first in the backlog's order the product
  owner keeps (its state comment), not `workline:ready`, its four sections
  written (drafts too), its body not read as it is. None: no agent asked.
- **What it writes**: one comment on the issue, edited each review:
  - what holds it from ready, the findings, the questions for a person;
  - a hidden record (`workline:spec-review`): the body's digest, the
    round, the important findings open and the sections they lie in.

  A nit or a decision holds nothing; a finding its judge refused is
  dropped.
- **The hold** is the product owner's: its `ready` waits until the
  reviewer read the body as it is and no important finding is open, then
  the engine moves it, with no agent
  ([backlog acts](../../../docs/spec/backlog-acts.md#refining-to-ready)).
- **The answer**: the product owner's next refine rewrites the sections
  the findings lie in that are its own; a person's, it asks the reporter.
  The body changed, the reviewer reads it again: one round more.
- **Five rounds** (`spec-rounds`): reviews in a row that leave a finding
  open. The sixth asks no agent: the comment puts a question to a person,
  and the reviewer stops reading it. The person can:
  - accept it as it reads (`workline:accepted`);
  - settle it and set `workline:ready`;
  - or delete the comment, for five rounds more.
- **A review not whole** (a lens failed, the tokens spent) keeps the last
  record, its body unread: read again at the next run, the rounds counted
  as before, ready still held.
- **In CI**: judged, then applied; the reviewer reads what an earlier run
  refined, so a round takes two nights. `forge-writes: false` writes no
  comment: ready then stays held, `spec-not-reviewed`.
- **Without AI**: nothing read, nothing written; ready held. The sixth
  round's question needs no agent.
