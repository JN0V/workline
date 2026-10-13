# Judge — where it stands (2026-10-13)

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

1. **Nothing buildable in Missing**: met. A judge of another provider and
   each defect it let through as an evaluation case are parked
   ([#80](https://github.com/JN0V/workline/issues/80)): any command can
   already be the judge, and using one waits on a second provider.
2. **The main issue validated on real use**: open. No issue of the role
   holds a Validation section the maintainer has signed.
3. **AI verdicts measured**: in part. Its verdicts are measured only
   inside the other roles' measures: the documentalist's `drifted` cases
   ([ADR-0014](../../../docs/adr/0014-checked-is-earned-by-what-was-read.md)),
   the reviewer's bench, where it refused true findings ([reviewer
   status](../../reviewer/docs/status.md), Missing 3 and 4).

## Built

No cases folder of its own: it is asked inside the other roles' cases.

- **A finding judged, the level said**: reviewer's
  `judged-no-not-reported`, `independence-level-in-verdict`.
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
