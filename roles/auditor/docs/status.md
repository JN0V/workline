# Auditor — where it stands (2026-10-14)

**In one line:** re-checks, each week, one in ten of every role's acts,
per kind (`workline sample`): the docs the documentalist vouched for, the
product owner's acts, the reviewer's judged findings, the committer's
rewrites. The docs' sample tried on a GitLab sandbox, its first scheduled
run on workline on 2026-10-05; the other acts' verdicts tried with no
real agent yet. Beta.

A try since 2026-10-13 is told in its pull request's `## Tried`
([ADR-0043](../../../docs/adr/0043-a-try-is-told-in-its-pull-request.md)).

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: open. Every role's acts
   ([#205](https://github.com/JN0V/workline/issues/205)) are drawn and
   re-checked, the verdicts kept and counted
   ([#276](https://github.com/JN0V/workline/issues/276)); left, as
   [ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md)
   decided: the records it completes, the product owner's issue folded in.
   The weekly sample's other gaps are parked
   ([#108](https://github.com/JN0V/workline/issues/108)): a judge of
   another provider, a week not run caught up.
2. **The main issue validated on real use**: open.
   [#155](https://github.com/JN0V/workline/issues/155) collects the weeks;
   no Validation signed by the maintainer yet.
3. **AI verdicts measured**: open. Its verdicts are kept and counted
   ([#276](https://github.com/JN0V/workline/issues/276)): the count, a
   comment on [#155](https://github.com/JN0V/workline/issues/155), no
   week answered yet; a `checked` found false on a forge is untried
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
- **Each act re-checked**, one call an act, its kind's question
  (`kinds/*.md`), a finding blind, never on a model that acted, false
  only with a quote found again, within the week's tokens
  ([#205](https://github.com/JN0V/workline/issues/205)): auditor's
  `verdict-*`.
- **Its verdicts kept and counted**
  ([#276](https://github.com/JN0V/workline/issues/276)): a dispute
  settled by three judges, `panel-settles-a-dispute`; the week's rows
  hidden, the maintainer's 👍 or 👎 and named rows read, the count
  rewritten from every week, `count-*`; a true finding the judge
  dropped opened as an issue, three a week at most,
  `dropped-true-finding-opened-capped`. The count:
  [a comment kept on #155](https://github.com/JN0V/workline/issues/155).
- **The reviewer's findings judged, every run's**, from the line its
  summary comment hides for each, read at the head that run read, on
  GitHub and GitLab; the lines past the comment's size said, not drawn
  ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md)):
  auditor's `sample-reads-every-run-judged*`, reviewer's `judged-line*`.
- **The committer's rewrites, drawn at all**: a rewritten message carries
  `Workline-Role: committer` and `Workline-Model`, read on the commit or
  on its merge request's head
  ([#401](https://github.com/JN0V/workline/pull/401)): committer's
  `rewrite-carries-its-trailers`, `rewrite-trailers-join-the-authors`,
  `no-rewrite-no-trailer`; auditor's `sample-draws-every-role`.

## Missing

- **The record ADR-0046 completes last**
  ([#205](https://github.com/JN0V/workline/issues/205)): the product
  owner's `at`, `model` and `body`, so its act is read on the text and
  the code it read
  ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md)).
- **The product owner's issue folded in**
  ([ADR-0033](../../../docs/adr/0033-the-weekly-sample-draws-the-product-owners-acts.md)'s,
  its level suggestion kept), with its fields, after its trial.
- **The weeks, read**
  ([#155](https://github.com/JN0V/workline/issues/155)).
