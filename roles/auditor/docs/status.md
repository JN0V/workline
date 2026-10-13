# Auditor — where it stands (2026-10-13)

**In one line:** reads, each week, one in ten of the docs the
documentalist vouched for (`workline sample`); tried on a GitLab sandbox,
its first scheduled run on workline on 2026-10-05. It audits the docs
only; the product owner's acts are drawn for a person, with no agent.
Beta.

A try since 2026-10-13 is told in its pull request's `## Tried`
([ADR-0043](../../../docs/adr/0043-a-try-is-told-in-its-pull-request.md)).

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: open. Every role's acts
   ([#205](https://github.com/JN0V/workline/issues/205)), to build or
   park. The weekly sample's other gaps are parked
   ([#108](https://github.com/JN0V/workline/issues/108)): a judge of
   another provider, a week not run caught up.
2. **The main issue validated on real use**: open.
   [#155](https://github.com/JN0V/workline/issues/155) collects the weeks;
   no Validation signed by the maintainer yet.
3. **AI verdicts measured**: open. Its verdicts are not kept as a measure
   yet ([#276](https://github.com/JN0V/workline/issues/276)); a `checked`
   found false on a forge is untried
   ([#155](https://github.com/JN0V/workline/issues/155) collects it when
   one turns up).

## Built

- **The weekly sample of the docs vouched for**: documentalist's
  `sample-*`, forge's `*-sample-written`, engine's
  `job-summary-of-a-sample`.
- **The product owner's acts drawn for a person**: product owner's
  `sample-*`.

## Missing

- **Every role's acts re-checked**
  ([#205](https://github.com/JN0V/workline/issues/205)).
- **Its verdicts kept and counted**
  ([#276](https://github.com/JN0V/workline/issues/276)).
- **The weeks, read**
  ([#155](https://github.com/JN0V/workline/issues/155)).
