---
sources:
  - roles/documentalist/role.yaml#settings.parts-move-checked
  - roles/documentalist/code/parts.go#partsFor
  - roles/documentalist/code/parts.go#fixTaskHeader
  - roles/documentalist/code/parts.go#judgedInPartsPatch
  - roles/documentalist/code/parts.go#holdJudgedInParts
  - roles/documentalist/code/parts.go#held
  - roles/documentalist/code/parts.go#vouchFor
  - roles/documentalist/code/parts.go#unsupported
  - roles/documentalist/code/parts.go#vouched
  - roles/documentalist/code/parts.go#docSources
  - roles/documentalist/code/parts.go#partsHeader
  - roles/documentalist/code/parts.go#partsSupportedWhole
checked: e81df7b34
verified: agent:claude-code
---
# Documentalist — the fix after parts

Part of [judged in parts](parts.md).

## The fix

- One last call (`fix`) patches what is wrong:
  [what it is given](parts-claims.md#reading-the-claims-no-ai).
- What the judge refuses of it:
  [moving `checked`](judge.md#moving-checked).
- A quote a part judged partial is no proof
  ([#467](https://github.com/JN0V/workline/issues/467)): words taken out
  on it alone are withheld for a person (`citation-unchecked`), the rest
  applied ([the judge's claims](judge-claims.md#removed-words-need-a-claim)).
  The task says so.
  - Why: on a C++ project, three parts called a line partial; the fix
    cited one of their quotes and made the line false.
  - Measured on every fix after parts kept, 2026-10-01 to 10-17: the two
    such removals both took out true words.
- The header records the commit it was judged in parts at,
  `judged-in-parts`, found wrong or not, and a doc fixed stays suspect,
  saying so: the fix saw only the parts' excerpts.
- It is not put before an agent again until one of its sources changes
  after that commit; a person clears it by reading it whole and moving
  `checked`.
- When gardening, the fix, or only the record, goes on a merge request of
  its own, `workline/documentalist/fix-<first doc>`; the doc waits while
  it is open ([gardening](gardening-merge-requests.md#one-merge-request-per-task)).

## When `checked` moves after parts

On by default since five evaluation runs held the bars it was to be
adopted on
([ADR-0009, amended](../../../docs/adr/0009-docs-far-behind-are-judged-in-parts.md#amendment-2026-10-18-checked-after-parts-measured-again-on);
[#482](https://github.com/JN0V/workline/issues/482));
`parts-move-checked: false` leaves every such doc to a person. The engine
moves `checked`, never the fix, when the parts found nothing wrong, in
conflict or to be read together, and:

- no claim was dropped (a part's note is one), no line is `uncovered`,
  no source is a doc;
- each word of every line stating something is quoted, where it stands,
  in a part's support from the code: each part is asked to quote a
  passage it supports whole. Headings, fences, derived blocks, HTML
  comments and link definitions state nothing; a code example does.

On a merge request's branch the engine moves nothing: the commit it would
name is one a squash merge drops, so `checked` stays, saying so, and a
person moves it to a commit of the default branch once merged
([#2029](https://github.com/JN0V/workline/issues/2029)).

Elsewhere, the header then sets `checked` and `judged-in-parts` to that commit, and
`verified: agent:documentalist`: the doc leaves the list for a person,
and the weekly sample, which cannot read it whole, lists it for a person
([weekly sample](../../../docs/weekly-sample.md)). Otherwise the finding
says what kept it ("no part supports line 14"), or that the engine's
patch is not applied: a set of proposals refused whole takes it out
([role contract](../../../docs/spec/role-contract.md#one-run), "Apply").
