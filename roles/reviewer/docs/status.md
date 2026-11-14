# Reviewer — where it stands (2026-11-12)

**In one line:** reviews code on a machine (`workline review`) and on a
merge request (opt-in): rules first, lenses as parts, quotes found again,
the change's findings told from the rest by where their cause lies, each
important one judged. It reads a spec too, on a machine and, after the
product owner refines an issue, on the forge. The developer fixes its
findings in review rounds. In the V1 acceptance round it reviewed every
request of 9 issues, and their specs in refine, on three real
repository copies
([round 3](https://github.com/JN0V/workline/issues/1266#issuecomment-6803875764)).
Released in v0.9.0; beta.

Each try: up to 2026-10-13 in [tried.md](tried.md), an archive; since, in
its pull request's `## Tried`
([ADR-0043](../../../docs/adr/0043-a-try-is-told-in-its-pull-request.md)).

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: open. The judge reads the code it
   needs, on since 2026-10-14
   ([#362](https://github.com/JN0V/workline/issues/362)); the reviewer
   measured on it, five runs a case, correctness blocking on workline
   itself ([#360](https://github.com/JN0V/workline/issues/360)); the
   developer's loop is built and ran on real use (Built). Left: the
   gaps the V1 rounds found, kept for 1.0 in [Missing](#missing).
2. **The main issue validated on real use**: awaits the maintainer. No
   issue of the role holds a Validation section the maintainer has
   signed.
   - **The evidence**: V1 round 3, every request of 9 issues reviewed
     at its head and green, the developer's review rounds run by the
     line
     ([round 3](https://github.com/JN0V/workline/issues/1266#issuecomment-6803875764));
     on workline, a developer's pull request fixed in a review round,
     then merged ([#1141](https://github.com/JN0V/workline/pull/1141)).
   - **What the maintainer would sign**: a Validation for the role,
     such as "the reviewer's findings on a real change are fixed in
     review rounds before the merge", met by the evidence above; or name
     another.
3. **AI verdicts measured**: met. Five runs a case, every lens
   together, the judge reading: 55 of 55 planted defects found, no false
   finding verified
   ([the reviewer, measured](../../../docs/research/reviewer-measured.md));
   its judge measured on real use
   ([the judge, measured](../../../docs/research/judge-measured.md)).
   Its cost:
   [measured](../../../docs/research/code-review.md#the-reviewers-cost-measured).
   On real use, the auditor's count of its weekly samples: of 42 dropped
   findings drawn, 25 stand and 1 is false; of 40 kept, 16 stand and 1
   is false; the rest unproven or not read, no week answered by the
   maintainer yet ([the count](https://github.com/JN0V/workline/issues/155#issuecomment-6265992275)).

**Correctness blocks on workline itself** (the maintainer, 2026-10-15,
[#360](https://github.com/JN0V/workline/issues/360)):

- **Set**: `ai-findings: {correctness: block}` in workline's
  `.workline/config.yaml`; the other lenses warn.
- **Why**: on the 70 reviews measured, a block on correctness held 29
  merges, all on true findings; no false block; no correctness defect
  let through.
- **The role's default stays `warn`** until the first real blocks on
  workline's pull requests have been watched; then it becomes the
  default.

## Built

Cases in
[tests/conformance/cases/reviewer](../../../tests/conformance/cases/reviewer);
its measure in
[tests/evaluation/cases/reviewer](../../../tests/evaluation/cases/reviewer).

- **Rules on the comments a change adds**, no agent while they block:
  `ticket-code-in-comment`, `long-comment-block-reported`.
- **A story word in a comment judged, never blocking**:
  `bug-story-warns-without-ai`, `bug-story-word-in-a-right-comment-not-blocked`,
  `bug-story-judged-by-the-comments-lens`,
  `bug-story-judged-on-its-push-whatever-the-turn`.
- **Only code reviewed**, commits reviewed whole not asked again:
  `docs-only-asks-nobody`, `reviewed-commits-not-asked-again`,
  `push-without-code-asks-nobody`, `lens-given-new-commits-files`.
- **A rebase not reviewed again**: `rebased-commits-not-asked-again`,
  `rebased-commit-changed-by-a-conflict-reviewed-again`,
  `new-commit-after-a-rebase-reviewed-alone`.
- **A merge of the base read for what it settled**:
  `merge-of-main-asks-no-lens`,
  `merge-resolution-alone-given-to-the-lens`.
- **Every lens once when a merge request becomes ready**, told by its
  draft state and the record too, so a run on ready a push cancelled is
  not lost ([#359](https://github.com/JN0V/workline/issues/359)):
  `every-lens-once-when-*`, `one-lens-on-the-push-after-ready`,
  `run-on-ready-held-by-a-rule-owes-every-lens`,
  `every-lens-owed-read-on-the-next-push`,
  `every-lens-when-the-run-on-ready-was-cancelled`,
  `one-lens-a-push-once-every-lens-read`,
  `every-lens-again-once-a-draft-again`,
  `record-unread-asks-one-lens-on-a-ready-merge-request`; a failed run
  on ready asks every lens again once, then says it:
  `every-lens-owed-once-after-a-failed-run-on-ready`,
  `every-lens-on-the-push-after-a-failed-run-on-ready`,
  `every-lens-not-whole-said-after-a-second-failure`,
  `every-lens-run-on-ready-unreachable-writes-nothing`.
- **A lens that did not answer owes its commits**, asked first at the
  next run, whosever turn it is; `read=all` waits for it
  ([#588](https://github.com/JN0V/workline/issues/588)):
  `warn-lenses-failed-read-their-commits-when-run-again`,
  `lens-that-blocks-failed-reads-its-commits-when-run-again`,
  `warn-lenses-answered-read-nothing-when-run-again`,
  `warn-turn-of-a-lens-that-blocks-records-what-none-owes`.
- **One review at a time per merge request**: a push waits for the
  review before it to be written, which the apply writes even once the
  branch moved on; each apply merges the record it finds
  ([#375](https://github.com/JN0V/workline/issues/375)):
  `review-applied-when-a-push-lands-before-it*`,
  `record-merged-when-two-runs-judged-before-either-applied*`; applied
  out of order, the newer run's text kept
  ([#598](https://github.com/JN0V/workline/issues/598)):
  `newer-review-kept-when-applied-out-of-order-on-gitlab`.
- **The record read only from workline's own account's comment**,
  another's carrying its marker never read, the range read again
  ([ADR-0051](../../../docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md),
  step 1): `record-of-another-author-not-read`; nor one another edited
  last, as the forge's GraphQL says (step 9):
  `record-edited-by-another-not-read*`.
- **No commit skipped when what its own pipeline runs changes**
  (`.github/`, `.gitlab-ci.yml`, `.workline/`, `pipeline-paths`;
  ADR-0051, step 9): `line-change-skips-no-commit`,
  `pipeline-paths-skip-no-commit`, `ci-change-skips-on-the-record`.
- **The lenses that warn read apart from the verdict**, each run its own
  comment and record
  ([ADR-0050](../../../docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md),
  [apart](apart.md)): `warn-apart-verdict-waits-for-no-lens-that-warns`,
  `warn-apart-a-lens-that-blocks-still-holds-the-verdict`,
  `warn-finding-posted-after-the-verdict`,
  `warn-lens-failing-changes-no-verdict`,
  `warn-only-takes-up-the-record-of-a-review-not-split`,
  `warn-only-every-lens-blocks-writes-nothing`.
- **No lens on a draft**, the last review kept, the AI review said to
  wait for ready
  ([ADR-0050](../../../docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md),
  [ready](ready.md)): `no-lens-on-a-draft`,
  `draft-with-no-agent-waits-for-a-person`,
  `every-lens-once-when-a-draft-no-lens-read-is-marked-ready`,
  `warn-apart-no-lens-on-a-draft`, `routing/draft-line-asks-no-agent`;
  a draft pushed, then marked ready, tried on
  [#533](https://github.com/JN0V/workline/pull/533).
- **After ready, only what is new**: a push reads the commits no review
  read, a merge of main nothing again; the lenses that warn five new
  commits a push, the rest said and read next; a finding judged once
  keeps its verdict on the same patch
  ([ADR-0050](../../../docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md),
  [after ready](ready.md#after-ready-only-what-is-new),
  [judged once](once.md)): `after-ready-*`, `commits-*`,
  `finding-judged-once-not-judged-again`,
  `finding-judged-again-once-its-file-changes`; a push and a merge of
  main after ready tried on [#534](https://github.com/JN0V/workline/pull/534).
- **Lenses as parts**, answering findings only; an answer that does not
  read asked again once: `never-approves-nor-patches`,
  `lens-failed-not-clean`, `lens-answer-asked-again-once`.
- **Quotes found again; the change's findings told from the rest**:
  `finding-without-quote-dropped`,
  `finding-cause-in-diff-is-fixed-by-author`,
  `finding-cause-outside-diff-becomes-issue`, `quote-over-lines-found`.
- **A removal's exposed lines are the change's**:
  `finding-beside-removed-lines-is-fixed-by-author`,
  `finding-far-from-removed-lines-becomes-issue`.
- **A removal's reach**, a later use of a name it tested or assigned
  ([#357](https://github.com/JN0V/workline/issues/357)):
  `finding-on-a-removed-guards-name-is-fixed-by-author`,
  `finding-on-a-name-the-removal-never-tested-becomes-issue`.
- **A file moved read as moved**, its kept lines not the change
  ([#358](https://github.com/JN0V/workline/issues/358)): `moved-file-*`,
  `moved-and-rewritten-file-read-as-new`,
  `block-stands-when-its-file-is-moved`.
- **A judge for each important finding**, asking the lens's question:
  `judged-no-not-reported`, `independence-level-in-verdict`,
  `tests-judge-*`, `correctness-judge-keeps-its-question`,
  `merged-findings-each-judged-by-their-lens`,
  `merged-finding-lead-refused-other-shown`.
- **A judge reads the code the finding stands on**:
  `judge-reads-the-function-the-cause-calls`,
  `judge-reach-capped-rest-named`.
- **A judge asks for the code it needs**, on since its
  [replay](../../../docs/research/judge-measured.md#replayed-2026-10-14)
  ([ADR-0045, amended](../../../docs/adr/0045-the-judge-asks-for-the-code-it-needs.md#amendment-2026-10-14-on-by-default)):
  `judge-reads-by-default`,
  `judge-reads-a-function-then-answers`, `judge-reads-past-its-caps`,
  `judge-reads-the-commit-only`, `judge-reads-stopped-by-the-budget`,
  `judge-no-quotes-the-code-finding-dropped`,
  `judge-no-quote-not-found-shown-not-verified`,
  `judge-unsure-shown-not-verified`.
- **Warn until measured, `ai-findings` by lens, caps**:
  `new-ai-rules-warn`, `ai-findings-*`, `findings-capped-rest-counted`,
  `agent-unreachable-is-blocked-external`.
- **A blocked merge request still gets the comment**: `blocked-run-*`,
  `rules-block-still-comments`.
- **A block stands until a commit answers it**: `block-stands-*`,
  `block-goes-*`, `block-found-again-shown-once`,
  `passed-stays-passed-on-the-same-commits-again`.
- **A line per finding judged**, every run's, on GitHub and GitLab, the
  oldest gone first past the comment's size, for the weekly sample
  ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md)):
  `judged-line-for-each-finding-judged`, `judged-lines-*`.
- **`workline review`**, findings as JSON: `local-review-outputs-json`.
- **Each important finding on its line**, in the forge's own review:
  `line-comment-*`. A nit on no line, listed last, asking nothing
  ([ADR-0051, cut 4](../../../docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-4-no-fix-round-for-nits)):
  `nit-in-the-summary-asks-nothing`.
- **Intent, claims, a decision for a person, the diff alone, the finder
  floor**: `intent-*`, `claim-contradicted-is-a-finding-on-the-line`,
  `merge-request-body-given-as-testimony`, `decision-*`, `diff-alone-*`,
  `finder-floor-nothing-found-is-clean`.
- **Simplicity, a warning only**: more than the issue needs, judged
  against it, until its false alarms are counted
  ([ADR-0068](../../../docs/adr/0068-the-line-steps-back-on-its-own-drift.md#next-each-decided-when-reached)):
  `simplicity-*`, two planted cases in the evaluation.
- **A full review within a token budget**:
  `lenses-asked-together-in-one-call`, `lens-given-each-file-once`,
  `judge-shown-the-change-near-the-cause`,
  `tests-judge-reads-tests-up-to-a-cap`,
  `review-says-the-tokens-of-each-call`, `budget-spent-*`.
- **A change too large for one call, in parts**, past `parts-max` no
  lens asked ([#446](https://github.com/JN0V/workline/issues/446)): `change-*`.
- **A spec, on a machine and on the forge**: `spec-*`.
- **The code a spec names, read as Markdown**: the functions it writes
  as code, a fenced block's too, never a plain word; a function in a
  file cut still shown
  ([#425](https://github.com/JN0V/workline/issues/425)); measured with
  no model on the 10 specs waiting, 113k tokens in down to 92k
  ([#477](https://github.com/JN0V/workline/pull/477)):
  `spec-fenced-quote-names-no-prose-word`,
  `spec-long-file-named-its-function-shown`,
  `spec-capitalised-words-name-no-code`.
- **Spec review on workline**, on since 2026-10-17, after the product
  owner's trial: the reviewer after it in workline's `schedule` line,
  up to 5 specs a night (`specs-per-run`); a night's cost estimated
  offline in [#424](https://github.com/JN0V/workline/pull/424).
- **Several specs a night** on the forge, `specs-per-run` (5), the rest
  the next night; measured with no model on workline's 10 waiting, two
  nights instead of ten ([#433](https://github.com/JN0V/workline/issues/433)):
  `spec-several-read-in-one-run`, `spec-rest-read-at-the-next-run`,
  `spec-tokens-spent-later-specs-wait`,
  `spec-question-to-a-person-takes-no-place`,
  `spec-without-ai-questions-still-asked`,
  `spec-agent-unreachable-questions-wait`.
- **A record that does not read is said**: `spec-record-unread-*`,
  `record-unread-reviews-every-commit-again`.
- **The record read with GitLab's job token**, which may not list the
  project's members: `record-read-without-the-members-on-gitlab`; tried
  on gitlab.com ([#565](https://github.com/JN0V/workline/pull/565)).
- **An issue outside the change**, opened the one way every role shares,
  written for a person first: the backlog's `issue-*` cases,
  `finding-cause-outside-diff-becomes-issue`.
- **Its measure**: fourteen evaluation cases on the `shop` fixture,
  intent and claims among them; every lens together, apart, one lens or
  the diff alone ([#360](https://github.com/JN0V/workline/issues/360)):
  `TestScoreReview`, `TestReviewerWithFakeAgent`,
  `TestReviewerCasesPointRight`, `TestIntentAndClaimsCasesGiveTheirLens`.
- **The code a spec names, by the shared picker**
  ([the code picker](../../../docs/spec/code-picker.md)). Cases
  `reviewer/picker-*`. Tried on
  [#1106](https://github.com/JN0V/workline/pull/1106#tried).
- **The developer's loop**: a fresh developer run fixes each review,
  the five whys at round three, a person past round five
  ([the developer's review loop](../../developer/docs/review-loop.md));
  the developer reads every finding's full text from the review's
  record ([#1312](https://github.com/JN0V/workline/pull/1312)); a thread
  its fix answered resolved
  ([#1045](https://github.com/JN0V/workline/pull/1045),
  [#1051](https://github.com/JN0V/workline/pull/1051)); a test finding
  sent to the tester
  ([#1300](https://github.com/JN0V/workline/pull/1300)), its "none
  needed" weighed ([#1316](https://github.com/JN0V/workline/pull/1316)). Ran on
  workline ([#1141](https://github.com/JN0V/workline/pull/1141)) and in
  the V1 rounds
  ([round 3](https://github.com/JN0V/workline/issues/1266#issuecomment-6803875764)).
- **A merge alone only after a review of its head**
  ([#1264](https://github.com/JN0V/workline/pull/1264)).
- **A spec read in refine's own run**, not at night
  ([#787](https://github.com/JN0V/workline/pull/787)).

## Missing

Gaps found on real use and in the V1 rounds
([round 3](https://github.com/JN0V/workline/issues/1266#issuecomment-6803875764)),
kept for 1.0:

- **The judge drops a defect** the code's comment calls intended
  (kept for 1.0, [#1357](https://github.com/JN0V/workline/issues/1357)).
- **`code-lines-max` cuts** a typical Android ViewModel, one lens left,
  a cut warning on every head
  (kept for 1.0, [#1334](https://github.com/JN0V/workline/issues/1334)).
- **Answers to a spec finding** missed or read twice in three edge cases
  (kept for 1.0, [#1346](https://github.com/JN0V/workline/issues/1346)).

Parked:

- **Correctness blocking by default**, once its first real blocks on
  workline's pull requests have been counted (above): a default to
  change, not a gap. Measured on
  [#360](https://github.com/JN0V/workline/issues/360): five runs a case,
  intent and claims; lenses apart not needed, together missing nothing;
  the diff alone does not win
  ([#243](https://github.com/JN0V/workline/issues/243)). Not measured:
  the finder floor's worth, a decision for a person (no case plants
  one), one lens a push whichever lens is in turn, the cheaper push over
  five runs
  (parked, [#1389](https://github.com/JN0V/workline/issues/1389)).
- **The tests lens outside the change**: untested code the change did not
  touch. Waits for the inspector's static-analysis results; decided then
  (parked, [#202](https://github.com/JN0V/workline/issues/202)).
- **Tries on real use**, nothing to build: spec review over more specs,
  on GitLab and in CI (the V1 rounds read every spec on a local forge);
  a run on ready a rule held, GitLab (the next pipeline with an agent),
  a block standing on a forge, an issue outside the change written by a
  real agent, a change read in parts; every lens on ready tried on
  [#307](https://github.com/JN0V/workline/pull/307)
  (parked, [#1391](https://github.com/JN0V/workline/issues/1391)).
