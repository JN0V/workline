# Auditor — where it stands (2026-10-14)

**In one line:** reads, each week, one in ten of the docs the
documentalist vouched for (`workline sample`); tried on a GitLab sandbox,
its first scheduled run on workline on 2026-10-05. The same sample draws
one in ten of every other role's acts, per kind, and lists them for a
person: it re-checks only the docs yet. Beta.

A try since 2026-10-13 is told in its pull request's `## Tried`
([ADR-0043](../../../docs/adr/0043-a-try-is-told-in-its-pull-request.md)).

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: open. Every role's acts
   ([#205](https://github.com/JN0V/workline/issues/205)) are being built,
   its verdicts kept and counted
   ([#276](https://github.com/JN0V/workline/issues/276)) with it, as
   decided on 2026-10-14; how, decided in
   [ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md).
   The weekly sample's other gaps are parked
   ([#108](https://github.com/JN0V/workline/issues/108)): a judge of
   another provider, a week not run caught up.
2. **The main issue validated on real use**: open.
   [#155](https://github.com/JN0V/workline/issues/155) collects the weeks;
   no Validation signed by the maintainer yet.
3. **AI verdicts measured**: open. Its verdicts are not kept as a measure
   yet ([#276](https://github.com/JN0V/workline/issues/276), built with
   [#205](https://github.com/JN0V/workline/issues/205)); a `checked`
   found false on a forge is untried
   ([#155](https://github.com/JN0V/workline/issues/155) collects it when
   one turns up). The bar:
   [ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md#leaving-beta).

## Built

- **The weekly sample of the docs vouched for**: documentalist's
  `sample-*`, forge's `*-sample-written`, engine's
  `job-summary-of-a-sample`.
- **The product owner's acts drawn for a person**: product owner's
  `sample-*`.
- **Every role's acts drawn**, one in ten per role and kind, from the
  records each keeps, listed for a person
  ([#205](https://github.com/JN0V/workline/issues/205)): auditor's
  `sample-*`; a `checked` naming a commit a squash left off the branch,
  read from its merge request's head: documentalist's `sample-checked-*`.

## Missing

- **Every role's acts re-checked**
  ([#205](https://github.com/JN0V/workline/issues/205)): drawn, not read
  yet. Left, as
  [ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md):
  a verdict per act; the records it completes — the product owner's
  `at`, `model` and `body`, the reviewer's line per finding judged, the
  committer's trailers.
- **Its verdicts kept and counted**
  ([#276](https://github.com/JN0V/workline/issues/276)): built with
  [#205](https://github.com/JN0V/workline/issues/205) (decided
  2026-10-14).
- **The weeks, read**
  ([#155](https://github.com/JN0V/workline/issues/155)).
