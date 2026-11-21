---
sources:
  - roles/reviewer/role.yaml#events
  - roles/reviewer/role.yaml#settings.spec-lenses
  - roles/reviewer/role.yaml#settings.spec-lines-max
  - roles/reviewer/role.yaml#settings.code-lines-max
  - roles/reviewer/role.yaml#settings.lenses-together
  - roles/reviewer/prompts/lenses/ambiguous.md
  - roles/reviewer/prompts/lenses/unverifiable.md
  - roles/reviewer/prompts/lenses/out-of-scope.md
  - roles/reviewer/prompts/lenses/contradicted.md
  - roles/reviewer/code/spec.go#prepareSpec
  - roles/reviewer/code/spec.go#readSpec
  - roles/reviewer/code/spec.go#specCut
  - roles/reviewer/code/spec.go#askSpec
  - roles/reviewer/code/spec.go#specIntro
  - roles/reviewer/code/spec.go#inSpec
  - roles/reviewer/code/spec.go#specMaterial
  - roles/reviewer/code/reviewer.go#lensFindings
  - roles/reviewer/code/reviewer.go#Post
checked: 60a7720e
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
  - for a lens that `needs: code`, the code the spec names, at HEAD, up
    to `code-lines-max` lines, the rest named. None named: that lens is
    not asked, said. [The code a spec names](spec-code.md)
    says how it is found.
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
- **The judge, the verdict, the questions** as for code
  ([the judge](judge.md)); an open point only a person settles is a
  finding holding `decision`.
- **No record** is kept: a spec is read whole each run. Nor a line per
  finding judged: the weekly sample draws a merge request's
  ([the code's](judge.md#every-finding-judged-kept-for-the-weekly-sample)).

## On the forge

Opt-in: the reviewer after the product owner reads each issue it
refines, and holds it from ready while an important finding is open.

- **Which issues, how many, in CI**: [a spec on the forge](spec-forge.md).
- **Its comment, the hold, the rounds**: [a spec's hold and rounds](spec-rounds.md).
