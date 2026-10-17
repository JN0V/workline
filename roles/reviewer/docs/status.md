# Reviewer — where it stands (2026-10-17)

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
   needs, on since 2026-10-14
   ([#362](https://github.com/JN0V/workline/issues/362)); the reviewer
   measured on it, five runs a case, correctness blocking on workline
   itself ([#360](https://github.com/JN0V/workline/issues/360)); the developer's
   loop waits for that role
   ([#117](https://github.com/JN0V/workline/issues/117)).
2. **The main issue validated on real use**: open. No issue of the role
   holds a Validation section the maintainer has signed.
3. **AI verdicts measured**: measured. Five runs a case, every lens
   together, the judge reading: 55 of 55 planted defects found, no false
   finding verified
   ([the reviewer, measured](../../../docs/research/reviewer-measured.md));
   its judge measured on real use
   ([the judge, measured](../../../docs/research/judge-measured.md)).
   Its cost:
   [measured](../../../docs/research/code-review.md#the-reviewers-cost-measured).

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
- **One review at a time per merge request**: a push waits for the
  review before it to be written, which the apply writes even once the
  branch moved on; each apply merges the record it finds
  ([#375](https://github.com/JN0V/workline/issues/375)):
  `review-applied-when-a-push-lands-before-it*`,
  `record-merged-when-two-runs-judged-before-either-applied*`.
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
- **Several specs a night** on the forge, `specs-per-run` (5), the rest
  the next night; measured with no model on workline's 10 waiting, two
  nights instead of ten
  ([#433](https://github.com/JN0V/workline/issues/433)):
  `spec-several-read-in-one-run`, `spec-rest-read-at-the-next-run`,
  `spec-tokens-spent-later-specs-wait`,
  `spec-question-to-a-person-takes-no-place`,
  `spec-without-ai-questions-still-asked`,
  `spec-agent-unreachable-questions-wait`.
- **A record that does not read is said**: `spec-record-unread-*`,
  `record-unread-reviews-every-commit-again`.
- **An issue outside the change**, opened the one way every role shares,
  written for a person first: the backlog's `issue-*` cases,
  `finding-cause-outside-diff-becomes-issue`.
- **Its measure**: fourteen evaluation cases on the `shop` fixture,
  intent and claims among them; every lens together, apart, one lens or
  the diff alone ([#360](https://github.com/JN0V/workline/issues/360)):
  `TestScoreReview`, `TestReviewerWithFakeAgent`,
  `TestReviewerCasesPointRight`, `TestIntentAndClaimsCasesGiveTheirLens`.

## Missing, in the order to build it

1. **Correctness blocking by default**, once its first real blocks on
   workline's pull requests have been watched (above). Measured on
   [#360](https://github.com/JN0V/workline/issues/360): five runs a case,
   intent and claims; lenses apart not needed, together missing nothing;
   the diff alone does not win
   ([#243](https://github.com/JN0V/workline/issues/243)). Not measured:
   the finder floor's worth, a decision for a person (no case plants
   one), one lens a push whichever lens is in turn, the cheaper push over
   five runs. No issue yet.
2. **Spec review on workline**, turned on after the product owner's trial
   (2026-10-16); then more specs measured, the product owner's real answer
   read again, on GitLab and in CI. No issue yet.
3. **The tests lens outside the change**: untested code the change did not
   touch. Waits for the inspector's static-analysis results
   ([#202](https://github.com/JN0V/workline/issues/202)); decided then.
4. **The developer's loop**: a handoff to the developer, five rounds, then
   a person. Waits for the developer role, after inspector and security
   ([#117](https://github.com/JN0V/workline/issues/117)).
5. **Tries on real use**: a draft marked ready, a run on ready a rule
   held, GitLab (the job token reading the notes, the next pipeline), a
   block standing on a forge, an issue outside the change written by a
   real agent; every lens on ready tried on
   [#307](https://github.com/JN0V/workline/pull/307). No issue yet.
