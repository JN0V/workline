# Reviewer — where it stands (2026-10-14)

**In one line:** reviews code on a machine (`workline review`) and on a
merge request (opt-in): rules first, lenses as parts, quotes found again,
the change's findings told from the rest by where their cause lies, each
important one judged. It reads a spec too, on a machine and, after the
product owner refines an issue, on the forge. Released in v0.9.0; beta.

Each try: up to 2026-10-13 in [tried.md](tried.md), an archive; since, in
its pull request's `## Tried`
([ADR-0043](../../../docs/adr/0043-a-try-is-told-in-its-pull-request.md)).

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: open. The judge reads the code it
   needs ([#362](https://github.com/JN0V/workline/issues/362)), then the
   reviewer is measured before the developer role
   ([#360](https://github.com/JN0V/workline/issues/360)); the developer's
   loop waits for that role
   ([#117](https://github.com/JN0V/workline/issues/117)).
2. **The main issue validated on real use**: open. No issue of the role
   holds a Validation section the maintainer has signed.
3. **AI verdicts measured**: in part.
   [#90](https://github.com/JN0V/workline/issues/90)'s step 1 ran once
   ([tried](tried.md#2026-10-06--the-measure-step-1-90),
   [ADR-0020](../../../docs/adr/0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md#amendment-2026-10-06));
   its judge measured on real use
   ([the judge, measured](../../../docs/research/judge-measured.md));
   five runs a case left
   ([#360](https://github.com/JN0V/workline/issues/360)). Its cost:
   [measured](../../../docs/research/code-review.md#the-reviewers-cost-measured).

## Built

Cases in
[tests/conformance/cases/reviewer](../../../tests/conformance/cases/reviewer);
its measure in
[tests/evaluation/cases/reviewer](../../../tests/evaluation/cases/reviewer).

- **Rules on the comments a change adds**, no agent while they block:
  `bug-story-in-comment-blocks-without-ai`, `ticket-code-in-comment`,
  `long-comment-block-reported`.
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
  `one-lens-a-push-once-every-lens-read`, `one-lens-a-push-on-a-draft`,
  `every-lens-again-once-a-draft-again`,
  `record-unread-asks-one-lens-on-a-ready-merge-request`; a failed run
  on ready asks every lens again once, then says it:
  `every-lens-owed-once-after-a-failed-run-on-ready`,
  `every-lens-on-the-push-after-a-failed-run-on-ready`,
  `every-lens-not-whole-said-after-a-second-failure`,
  `every-lens-run-on-ready-unreachable-writes-nothing`.
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
- **A judge for each important finding**, asking the lens's question:
  `judged-no-not-reported`, `independence-level-in-verdict`,
  `tests-judge-*`, `correctness-judge-keeps-its-question`,
  `merged-findings-each-judged-by-their-lens`,
  `merged-finding-lead-refused-other-shown`.
- **A judge reads the code the finding stands on**:
  `judge-reads-the-function-the-cause-calls`,
  `judge-reach-capped-rest-named`.
- **Warn until measured, `ai-findings` by lens, caps**:
  `new-ai-rules-warn`, `ai-findings-*`, `findings-capped-rest-counted`,
  `agent-unreachable-is-blocked-external`.
- **A blocked merge request still gets the comment**: `blocked-run-*`,
  `rules-block-still-comments`.
- **A block stands until a commit answers it**: `block-stands-*`,
  `block-goes-*`, `block-found-again-shown-once`,
  `passed-stays-passed-on-the-same-commits-again`.
- **`workline review`**, findings as JSON: `local-review-outputs-json`.
- **Each finding on its line**, in the forge's own review:
  `line-comment-*`.
- **Intent, claims, a decision for a person, the diff alone, the finder
  floor**: `intent-*`, `claim-contradicted-is-a-finding-on-the-line`,
  `merge-request-body-given-as-testimony`, `decision-*`, `diff-alone-*`,
  `finder-floor-nothing-found-is-clean`.
- **A full review within a token budget**:
  `lenses-asked-together-in-one-call`, `lens-given-each-file-once`,
  `judge-shown-the-change-near-the-cause`,
  `tests-judge-reads-tests-up-to-a-cap`,
  `review-says-the-tokens-of-each-call`, `budget-spent-*`.
- **A spec, on a machine and on the forge**: `spec-*`.
- **A record that does not read is said**: `spec-record-unread-*`,
  `record-unread-reviews-every-commit-again`.
- **An issue outside the change**, opened the one way every role shares,
  written for a person first: the backlog's `issue-*` cases,
  `finding-cause-outside-diff-becomes-issue`.
- **Its measure**: twelve evaluation cases on the `shop` fixture,
  `TestScoreReview`, `TestReviewerWithFakeAgent`,
  `TestReviewerCasesPointRight`.

## Missing, in the order to build it

1. **The judge reads the code it needs** before saying no: its true
   findings dropped for want of the proof
   ([#362](https://github.com/JN0V/workline/issues/362)).
2. **Measured before the developer**, on that judge: five runs a case, the
   finder floor's worth, the lenses together against apart; intent, claims
   and decisions; the diff alone as a cheap pass
   ([#243](https://github.com/JN0V/workline/issues/243)); then the
   maintainer's choice for correctness
   ([#360](https://github.com/JN0V/workline/issues/360)).
3. **A removal's reach past three lines**: a guard removed, its defect
   further down ([#357](https://github.com/JN0V/workline/issues/357)).
4. **A file moved**, read as all new lines
   ([#358](https://github.com/JN0V/workline/issues/358)).
5. **Reviews of one pull request at once** overwrite each other
   ([#359](https://github.com/JN0V/workline/issues/359)).
6. **Spec review on workline**, turned on after the product owner's trial
   (2026-10-16); then more specs measured, the product owner's real answer
   read again, on GitLab and in CI. Several specs a night parked until a
   queue shows. No issue yet.
7. **The tests lens outside the change**: untested code the change did not
   touch. Waits for the inspector's static-analysis results
   ([#202](https://github.com/JN0V/workline/issues/202)); decided then.
8. **The developer's loop**: a handoff to the developer, five rounds, then
   a person. Waits for the developer role, after inspector and security
   ([#117](https://github.com/JN0V/workline/issues/117)).
9. **Tries on real use**: a draft marked ready, a run on ready a rule
   held, GitLab (the job token reading the notes, the next pipeline), a
   block standing on a forge, an issue outside the change written by a
   real agent; every lens on ready tried on
   [#307](https://github.com/JN0V/workline/pull/307). No issue yet.
