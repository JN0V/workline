# Judge — where it stands (2026-10-14)

**In one line:** asked on real use by the documentalist (a fix it
vouches for), the reviewer (each important finding) and the product owner
(a closing as obsolete), at the best independence available from the
agent judged ([ADR-0005](../../../docs/adr/0005-independence-takes-the-best-level-available.md)). Beta.

A try since 2026-10-13 is told in its pull request's `## Tried`
([ADR-0043](../../../docs/adr/0043-a-try-is-told-in-its-pull-request.md)).

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: met, 2026-10-14. The judge reads
   the code it needs before saying no, on by default
   ([#362](https://github.com/JN0V/workline/issues/362)). A judge of
   another provider and each defect it let through as an evaluation case
   are parked ([#80](https://github.com/JN0V/workline/issues/80)): any
   command can already be the judge, and using one waits on a second
   provider.
2. **The main issue validated on real use**: open. No issue of the role
   holds a Validation section the maintainer has signed.
3. **AI verdicts measured**: met on real use, 2026-10-14: 40 of its 229
   verdicts on workline's pull requests read again in the code, its "yes"
   right 19 of 20, its "no" 12 of 20
   ([the judge, measured](../../../docs/research/judge-measured.md)).
   Measured again on the same 40, 2026-10-14, the judge that reads
   against today's: it keeps 6 of the 8 true findings dropped and lets 2
   of 12 false ones through, shown not verified
   ([replayed](../../../docs/research/judge-measured.md#replayed-2026-10-14));
   on by default since
   ([ADR-0045, amended](../../../docs/adr/0045-the-judge-asks-for-the-code-it-needs.md#amendment-2026-10-14-on-by-default)). Also
   inside the other roles' measures: the documentalist's `drifted` cases
   ([ADR-0014](../../../docs/adr/0014-checked-is-earned-by-what-was-read.md)),
   the reviewer's bench.

## Built

No cases folder of its own: it is asked inside the other roles' cases.

- **A finding judged, the level said**: reviewer's
  `judged-no-not-reported`, `independence-level-in-verdict`.
- **A judge of the code asks for what it needs**, its no quoting the code,
  or unsure, on by default
  ([ADR-0045](../../../docs/adr/0045-the-judge-asks-for-the-code-it-needs.md)):
  reviewer's `judge-reads-*`, `judge-no-*`,
  `judge-unsure-shown-not-verified`; its 40 measured questions kept to
  replay ([the judge's replay](../../../docs/spec/conformance/evaluation.md#the-judges-replay)).
- **A doc read against its sources, never by the voucher**:
  documentalist's `sample-never-the-voucher`,
  `sample-judge-from-the-environment`.
- **A split refused**: documentalist's `split-refused-by-the-judge`.
- **A closing as obsolete**: product owner's
  `obsolete-closed-after-silence-and-judge`,
  `obsolete-judge-disagrees-kept`.

## Missing

- **Parked**: a judge of another provider
  ([#80](https://github.com/JN0V/workline/issues/80)).
