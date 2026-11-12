# Auditor — where it stands (2026-11-12)

**In one line:** re-checks, each week, one in ten of every role's acts,
per kind (`workline sample`): the docs the documentalist vouched for, the
product owner's acts, the reviewer's judged findings, the committer's
rewrites, the tester's acts, the developer's code, each against what it
was done on. The
docs' sample tried on a GitLab sandbox, its first scheduled run on
workline on 2026-10-05; the tester's acts read by a real judge on stub
acts ([#669](https://github.com/JN0V/workline/pull/669)); the product
owner's, the reviewer's and the committer's acts read by a real judge
on workline each week since W42, the developer's and the
documentalist's drawn, not read yet
([the count](https://github.com/JN0V/workline/issues/155#issuecomment-6265992275)).
Beta.

A try since 2026-10-13 is told in its pull request's `## Tried`
([ADR-0043](../../../docs/adr/0043-a-try-is-told-in-its-pull-request.md)).

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: met (2026-10-17). Every role's acts
   ([#205](https://github.com/JN0V/workline/issues/205)) are drawn and
   re-checked on the records
   [ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md)
   completes, the verdicts kept and counted
   ([#276](https://github.com/JN0V/workline/issues/276)), the product
   owner's issue folded in. The weekly sample's other gaps are parked
   ([#108](https://github.com/JN0V/workline/issues/108)): a judge of
   another provider, a week not run caught up.
2. **The main issue validated on real use**: awaits the maintainer.
   The bar ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md#leaving-beta)):
   a week on workline lists verdicts for the acts of at least three
   roles, and the maintainer agrees with each false one. W45 lists
   verdicts for the product owner, the reviewer and the committer, 9
   false ([W45](https://github.com/JN0V/workline/issues/155#issuecomment-6742605949)).
   The maintainer would sign it with a 👍 on W45, or a 👎 naming the
   rows they contest. [#205](https://github.com/JN0V/workline/issues/205),
   the main issue, and [#155](https://github.com/JN0V/workline/issues/155),
   where the weeks are written, are parked since 2026-11-09.
3. **AI verdicts measured**: open, in two parts
   ([the bar](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md#leaving-beta)).
   - **Planted**: six evaluation cases, a wrong act and its right twin
     for a reviewer's dropped finding, a committer's rewrite and a
     tester's tests, 30 of 30 in five runs (2026-10-29,
     [amendment](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md#amendment-2026-10-29-the-auditor-measured)).
     Left: a product owner's closing and a false doc (kept for 1.0,
     [#1379](https://github.com/JN0V/workline/issues/1379)).
   - **Real use**: awaits the maintainer. The bar is 60 verdicts the
     maintainer answered over four weeks, at most one contested. W42 to W45 read
     90 acts, none answered yet
     ([the count](https://github.com/JN0V/workline/issues/155#issuecomment-6265992275)).
     They would sign it with a 👍 or a 👎 on each week's comment. Its
     own issues closed as not planned are counted in the same comment:
     none opened yet.

## Built

- **The weekly sample of the docs vouched for**: documentalist's
  `sample-*`, forge's `*-sample-written`, engine's
  `job-summary-of-a-sample`.
- **The product owner's acts drawn**, each a person undid said, the
  level they suggest in the week's comment, its own issue folded in and
  its `weekly-sample` said no longer read
  ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md#settled-while-building)):
  product owner's `sample-*`, `weekly-sample-said-gone`; auditor's
  `sample-folds-the-product-owners-issue`.
- **Every role's acts drawn**, one in ten per role and kind, from the
  records each keeps, listed for a person
  ([#205](https://github.com/JN0V/workline/issues/205)): auditor's
  `sample-*`; a `checked` naming a commit a squash left off the branch,
  read from its merge request's head: documentalist's `sample-checked-*`;
  the head's commit that moved a `checked` says whose the move was — a
  person's not drawn, as all of W41's and W42's so far turned out — and
  who vouched ([#466](https://github.com/JN0V/workline/issues/466)):
  documentalist's `sample-squashed-*`.
- **Each act re-checked**, one call an act, its kind's question
  (`kinds/*.md`), a finding blind, never on a model that acted, false
  only with a quote found again, within the week's tokens
  ([#205](https://github.com/JN0V/workline/issues/205)): auditor's
  `verdict-*`. A grant of a size, `kinds/size.md`: false where a split
  was possible
  ([ADR-0064, §6](../../../docs/adr/0064-size-is-a-target-not-a-wall.md#6-every-grant-observed)).
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
- **The product owner's acts, on what they read**: each keeps `at`,
  `model` and `body` in the issue's state; the auditor reads the issue's
  text by that digest — the body now, else GitHub's edit history — and
  the code at `at`, judges apart from `model`, and names an act an older
  engine recorded, read as its issue reads now: product owner's
  `acts-record-what-was-read`; auditor's
  `verdict-reads-what-the-product-owner-read`.

- **The tester's acts** (2026-10-28): its tests, from its commit and
  the record on its Draft, which now keeps the scenarios read; an
  objection answered kept or rewritten, from its reply's hidden line; a
  curator's removal, from its commit and pull request; the five whys a
  third answer gave, drawn apart (step 8 of the tester); each with its
  own question (`kinds/tests.md`, `kept.md`, `rewritten.md`, `whys.md`,
  `removal.md`)
  ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md#amendment-2026-10-28-the-testers-row)): auditor's
  `sample-draws-the-testers-acts`, `verdict-reads-the-testers-acts`.
  Tried with stub tester runs and a real judge, on
  [#669](https://github.com/JN0V/workline/pull/669).
- **The developer's code** (2026-10-29): on its merge requests merged,
  its own commits and the record on its pull request; its question in
  `kinds/code.md`
  ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md#amendment-2026-10-29-the-developers-row)):
  auditor's `sample-draws-the-developers-code`. No real judge yet.

## Missing

- **A product owner's closing and a false doc, planted** in its
  evaluation (kept for 1.0,
  [#1379](https://github.com/JN0V/workline/issues/1379)): a measure for
  criterion 3, not a gap in what is built, so criterion 1 holds.
- **Tries on real use**:
  - **the weeks, read** by the maintainer, W42 to W45 waiting (parked,
    [#155](https://github.com/JN0V/workline/issues/155));
  - **a product owner's act read on GitHub's edit history**, live
    (parked, [#155](https://github.com/JN0V/workline/issues/155));
  - **the developer's code re-checked by a real judge**: 4 drawn from W42 to W45,
    none read yet (parked,
    [#155](https://github.com/JN0V/workline/issues/155)).
