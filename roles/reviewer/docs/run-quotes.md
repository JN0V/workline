---
sources: [roles/reviewer/role.yaml, roles/reviewer/prompts/instruction.md, roles/reviewer/prompts/policy.md, roles/reviewer/prompts/lenses, roles/reviewer/code]
checked: e81df7b34
verified: agent:claude-code
---
# Reviewer — the quotes, and whose finding it is

Part of [a run on code](run.md): steps 5 and 6.

## 5–6. The quotes, and whose finding it is

### 5. The quotes

The engine finds each cause again, spaces and line breaks aside: in
the file at the head of the range, among the lines the change removed,
or in the issue it names (`#4`, the change's); a symptom, in its file; a
claim, in what the author said.

- A file moved, quoted on its old path, is found on its new one.
- Not found: dropped, said (`finding-unfounded`).
- Two on one line are grouped, never one dropped
  ([#229](https://github.com/JN0V/workline/issues/229)).

### 6. Related or not

- **The change's**: a cause on a line the change added or removed, or
  on a kept line within three of a removal (a hunk taking away more
  lines than it adds, [#224](https://github.com/JN0V/workline/issues/224)).
  Its author's: an important one on that line, a nit in the summary.
- **A removal's reach**
  ([#357](https://github.com/JN0V/workline/issues/357)): a kept line
  further down is the change's too when, in the same function, it uses
  a name a removed line tested or assigned (`if p < 1` removed, then
  `from := (p - 1) * size`). Its comment goes where the removal shows.
- **A claim it does not hold**
  ([#1415](https://github.com/JN0V/workline/issues/1415)): a finding
  of the claims lens is the change's wherever its cause lies, a kept
  line below the lines it added too; on a line the diff does not
  show, it stays in the summary.
- **Elsewhere**: an issue, once, labelled `needs-triage`, with the
  product owner's state; never on the merge request. Its key is the
  file and the cause's line; an issue open or closed holds it, none
  opened ([ADR-0018](../../../docs/adr/0018-the-product-owner.md)). A
  nit there is left, counted.
- **Written for a person first**, as the product owner writes
  ([writing an issue](../../product-owner/docs/writing.md)): what a
  person sees (the lens's `seen`), then the cause, the line and the
  fix folded, then one line, "Found by the reviewer role while
  reviewing ccffcb2, outside that change." The lens and the judge's
  verdict are hidden in the body.
- **A decision for a person**
  ([#126](https://github.com/JN0V/workline/issues/126)): a finding
  holding `decision` — a trade-off, a design choice, what the issue
  leaves open — is a question, not a defect.
  - Its cause found again, in the change or its issue, else dropped,
    said; never an issue.
  - Never important, never judged, never blocking; one a line (in an
    issue, one a quote); `questions-max` a run, the rest counted.
  - Asked in the summary comment under **Questions for a person**, one
    line each, its cause quoted; locally, `decision` (`question`).
  - A reply on the merge request is enough: the reviewer does not wait.
