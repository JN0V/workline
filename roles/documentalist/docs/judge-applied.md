---
sources:
  - roles/documentalist/code/documentalist.go#Post
  - roles/documentalist/code/judge.go#asJudged
  - roles/documentalist/code/judge.go#judgedFixes.countFixed
  - roles/documentalist/code/documentalist.go#preRun.holdJudged
  - roles/documentalist/code/documentalist.go#holdJudgedDocs
  - roles/documentalist/code/documentalist.go#held
  - roles/documentalist/code/documentalist.go#readByLog
checked: e81df7b34
verified: agent:claude-code
---
# Documentalist — what holds is applied

Part of [the check](judge.md).

## What holds is applied

A refusal used to send the whole task again — two docs, a costly
call ([measured](../../../docs/research/token-costs-measured.md#documentalist)) — for
one place the engine had already named (step 4). Now:

- a place withheld is not asked for again: the doc's fix is applied
  without it, the place left as it was;
- the doc keeps `checked` where it was and records `judged` (its header
  put back, `judged` set), so a person reads it before moving `checked`;
  it is not put before an agent again until a source changes, unless one
  is in another repository or the doc is due at a later moment (a product
  doc, at gardening)
  ([ADR-0013](../../../docs/adr/0013-what-was-judged-is-not-asked-again.md));
- each place withheld is a finding for a person — the refusal's rule, the
  doc and its lines, and why — in the run's verdict, the merge request's
  comment or the fix's own merge request, as other findings; the doc's
  `suspect` finding says it was fixed in part
  ([what the finding says](judge-findings.md#what-the-finding-says));
- with every place of a doc withheld, the fix is only its record: `judged`
  set, the places reported;
- `post` leaves the narrowed patches in `out/intentions.yaml`, and the
  engine applies those ([role contract](../../../docs/spec/role-contract.md)).

## What the run's findings say

- `pre` records the docs whose sources it gave whole
  (`in/read-whole.yaml`; none, when the file is missing), and what it gave
  of the others (`in/read-not-whole.yaml`): only the first may have
  `checked` moved. With an agent that reads the repository, `post` keeps
  of them only the docs whose every source file its calls' log holds
  ([more](reads.md)).
- On a person's merge request, a doc whose patch only records `judged`:
  its record is kept in the documentalist's comment there, nothing
  committed for it, and the finding says so
  ([more](push-records.md)).
- **`count-off`** a fix brought right says "fixed in this run".
