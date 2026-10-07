# Reviewer — where it stands (2026-10-07)

**In one line:** it reviews code on a machine (`workline review`) and on a
merge request (opt-in): rules first, lenses as parts, quotes found again,
the change's findings told from the rest by where their cause lies, each
important one judged; built and in conformance, tried with a real agent on
a copy of workline and live on a sandbox pull request (tried.md),
released in v0.9.0. It reads a spec too, a file or an issue, on a machine;
on the forge, after the product owner refines an issue, its findings hold
the issue from `ready` (#128).

## Built

| What | Proof |
|---|---|
| The rules on the comments a change adds: a bug's story, an internal code, a block too long; while they block, no agent | cases `bug-story-in-comment-blocks-without-ai`, `ticket-code-in-comment`, `long-comment-block-reported` |
| Only code reviewed; commits reviewed whole not asked again | `docs-only-asks-nobody`, `reviewed-commits-not-asked-again` |
| A push changing no code asks nobody, its commits recorded; a lens gets only what the new commits change | `push-without-code-asks-nobody`, `lens-given-new-commits-files` |
| On workline's own pull requests in CI, forge writes on: the summary comment edited in place, the record, the lenses in turn (tried.md) | workline#132 |
| Lenses as parts, answering findings only; a patch refused; an answer that does not read asked again once (#138) | `never-approves-nor-patches`, `lens-failed-not-clean`, `lens-answer-asked-again-once` |
| Quotes found again; the change's findings told from the rest by the cause | `finding-without-quote-dropped`, `finding-cause-in-diff-is-fixed-by-author`, `finding-cause-outside-diff-becomes-issue` |
| A removal's exposed lines are the change's (#224): a kept line within three of a hunk taking lines away, or a removed line named, goes to the author; a kept line far from it stays outside (tried.md: the guard removed from `Page` now lands on the change) | `finding-beside-removed-lines-is-fixed-by-author`, `finding-far-from-removed-lines-becomes-issue`, `TestExposedBesideARemoval` |
| A judge for each important finding, the level said; a no drops it | `judged-no-not-reported`, `independence-level-in-verdict` |
| The judge asks the lens's question (#223): for the tests lens, whether a test exercises the behaviour, shown the tests touching the cause's file, a no naming the test; correctness and edge cases keep theirs (tried.md) | `tests-judge-asked-if-a-test-exercises`, `tests-judge-names-the-test-that-exercises`, `correctness-judge-keeps-its-question`, `TestTestsTouching` |
| Warn until measured; caps; an agent unreachable | `new-ai-rules-warn`, `findings-capped-rest-counted`, `agent-unreachable-is-blocked-external` |
| `ai-findings` by lens (#222): a lens set to block fails the run, the others warn, a lens not named warns, one value still sets every lens; a misspelt lens or value stops the run; of two findings on one line, the blocking lens leads (tried.md, planted) | `ai-findings-by-lens-blocks`, `ai-findings-by-lens-warns`, `ai-findings-lens-unnamed-warns`, `ai-findings-one-value-every-lens`, `ai-findings-misspelt-refused`, `ai-findings-blocking-lens-leads-merge` |
| A blocked merge request still gets the comment (#226): what blocks first, marked; the record moved when every lens answered; a rule that blocks comments too; applied by the applying job, the judging one failing (tried.md, planted) | `blocked-run-still-comments`, `blocked-run-moves-the-record`, `blocked-run-judged-apart-still-blocks`, `rules-block-still-comments`, `TestGitLabApplyRunsWhenTheJudgeFails`, `TestGitHubApplyRunsWhenTheJudgeFails` |
| `workline review`, findings as JSON; findings on one line merged | `local-review-outputs-json` |
| Intent (#126): a change saying `Closes #N` (a commit or the merge request) has N read from the forge, its Need, Verification and Scope given to the `intent` lens; a part left out quoted from the issue (`#N`), found again there, judged shown the issue and the whole merge request's change, so a part an earlier commit did is refused; no issue closed, not asked, said; one unread, said (`issue-unread`). Live on the sandbox: #39's token-age part found, verified (tried.md) | `intent-need-left-out-found`, `intent-not-asked-without-an-issue`, `intent-issue-unread-said`, `intent-judge-reads-the-whole-merge-request`, `TestCloses`, `TestIssueText` |
| The author's claims contested (#126): the commit messages whole and the merge request's title and body given as testimony; the `claims` lens quotes the claim, found again in what the author said or dropped, its cause the contradicting line. Live: "No change in behaviour." over `>` turned `>=`, verified (tried.md) | `claim-contradicted-is-a-finding-on-the-line`, `merge-request-body-given-as-testimony`, `TestGitHubMergeRequest` |
| A decision for a person (#126): a finding holding `decision` — a trade-off, what the issue leaves open — asked as a question under **Questions for a person**, its cause quoted; never important, judged, blocking nor an issue; one outside the change dropped. Live: workline-sandbox#42, the lock's length #41 leaves open asked, not reported as a defect (tried.md) | `decision-asked-of-a-person-not-judged`, `decision-in-its-own-section-on-the-merge-request` |
| The diff alone (#126): `diff-alone`, a call of its own given the change only, no file whole, no message, no issue; its findings quoted, judged and routed as a lens's; `--lenses diff-alone` asks it alone. Off by measure: on the evaluation's twelve cases, 10 of 11 planted shown, nothing false, 82k tokens; nothing the lenses missed (tried.md) | `diff-alone-task-holds-only-the-change`, `diff-alone-finding-judged-like-the-others`, `diff-alone-off-asks-nothing-more`, `diff-alone-asked-by-name-alone` |
| The floor on finders only (#126): a lens under `finder-floor` answering nothing leaves the review clean, nothing judged, the commits recorded | `finder-floor-nothing-found-is-clean` |
| Findings on one line each judged by their own lens's question (#229): grouped on one row, each with its verdict; a refused leader dropped alone, the verified one leading and blocking if its lens blocks (tried.md: restock) | `merged-findings-each-judged-by-their-lens`, `merged-finding-lead-refused-other-shown` |
| A quote found spaces and line breaks aside, a tab-indented one read | `quote-over-lines-found` |
| A full review within a token budget (#147): the lenses in one call, each file once (whole, the change marked, or by its hunks), a judge shown the change near the cause and 300 lines of tests; `ai-max-tokens` 200000, the summary saying what each call used; once spent, nothing more asked and the commits not recorded. #146's review: 453k tokens in estimated before, 142k after; the real run 96.1k in, 8.6k out, two of the three true findings verified (tried.md) | `lenses-asked-together-in-one-call`, `lenses-together-finding-naming-no-lens-kept`, `lens-given-each-file-once`, `judge-shown-the-change-near-the-cause`, `tests-judge-reads-tests-up-to-a-cap`, `review-says-the-tokens-of-each-call`, `budget-spent-lens-not-asked`, `budget-spent-findings-not-judged` |
| A judge reads the code the finding stands on (#127): the cause's function whole, then the functions the finding names, those it calls and those calling it, up to `judge-lines-max` (200) lines, the rest named; no build (Go parsed, other languages by pattern). #146's two findings refused "for want of code" both verified by Opus, 16.6k tokens in; the judges' prompts about 25% larger on #146, 10% on #90's cases (tried.md) | `judge-reads-the-function-the-cause-calls`, `judge-reach-capped-rest-named`, `TestReach`, `TestEnclosingByLanguage` |
| Its measure: twelve evaluation cases on the `shop` fixture, three defects planted a lens, two clean changes and untested code outside one; scored by lens with no agent, recall, findings nothing planted, the judge's refusals, the finder floor, tokens | tests/evaluation/cases/reviewer, `TestScoreReview`, `TestReviewerWithFakeAgent`, `TestReviewerCasesPointRight` |
| A spec on a machine (#128): `workline review --spec <file>` or `--issue <n>`, the `spec` event; the spec lenses (ambiguous, unverifiable, out-of-scope, contradicted) together, each judged by its own question; causes found again in the spec (`file:line`, `#n:line`), a symptom in the code; the code the spec names given, none: contradicted not asked; no agent: not reviewed, said; no record. Live on workline-sandbox#43: both planted defects found and verified, 22.5k tokens (tried.md) | `spec-verification-unchecked-found`, `spec-issue-read-by-number`, `spec-contradicted-judge-reads-the-code`, `spec-quote-not-found-dropped`, `spec-naming-no-code-contradicted-not-asked`, `spec-without-ai-not-reviewed`, `TestMendQuotedThenMore` |
| A spec on the forge (#128): on `schedule` after the product owner, one refined issue a run — kept by it, four sections, not ready, its body not read as it is —; one comment on it, its record read back by the product owner's `ready`; five rounds, the sixth a question to a person with no agent, the issue no longer read. Live on the sandbox, planted (#44 held, answered, released; #45 asked at the sixth round) and real (#12, #46: 19.2k and 23.3k tokens a review, tried.md) | `spec-on-the-forge-reads-a-refined-issue`, `spec-read-as-it-is-not-read-again`, `spec-sixth-round-asks-a-person`, `TestSpecReviewReadsBack` |
| An issue outside the change opened through the one way every role shares: a subject an issue holds, open or closed, not opened again (ADR-0018) | backlog `issue-*` cases; the sandbox, live (#9) |

Conformance: tests/conformance/cases/reviewer.

## Cost, measured

Each measure's story is in tried.md.

- **Code, every lens**: #146's eight commits, 142k tokens in, estimated;
  the lenses' call 73.6k (430k before #147). The judges reading whole
  functions (#127): about 160k.
- **Intent and claims** (#126): the lenses' prompt 0.7% larger, 3.0% when
  the change closes an issue; a decision for a person, 0.4% more, no judge
  call.
- **The diff-alone facet**, off: one call more, the change's size (about
  4.6k tokens on the evaluation's cases, 53k on #146's).
- **A spec on a machine** (#128): one call, 9k to 13k characters measured
  (#128's body; a spec naming a Go file), at most the spec and
  `code-lines-max` lines of code; on workline-sandbox#43, 22.5k tokens with
  two judges.
- **A spec on the forge**: one spec a run, 19.2k and 23.3k tokens on the
  sandbox, three and four judges.

## Missing, in the order to build it

1. **Spec review, what is left** (#128): one spec a run (a backlog
   refining five a night waits); a real answer by the product owner
   rewriting its own section, then read again (planted only); the
   spec lenses measured on more specs, false findings counted (#46: four
   important refused by the judge, three nits shown); on GitLab and in
   CI, judged then applied (a round takes two nights, untried).
2. **The developer's loop** (#117): a handoff to the developer, five rounds,
   then `human`; `judge-at-least: model` for the developer's merge requests,
   `context` for a person's, told apart by the author (#81).
3. **Measured** (#90): step 1 done, once, Sonnet finding and Opus judging
   (tried.md, 2026-10-06): correctness found 3 of 3 with nothing false
   shown; edge cases found 3 of 3 but one was sent outside the change
   (since routed to the change, #224), and it showed one finding a clean rename did not cause; the
   judge refused four true findings of the tests lens, which its own
   question now verifies (#223, #229, tried.md). The
   maintainer decides from it: `ai-findings: {correctness: block}` (by
   lens since #222; workline's own line still warns), or step 2 on the
   lenses in doubt. Not measured yet: the
   finder floor's worth (a run with `finder-floor: false`), five runs a
   case (ADR-0014).
4. **Inline comments** on the forge's own review (#81: the bot's identity).
5. **Every lens once when a merge request becomes ready**, from the
   templates (`--input lenses=all` exists; no template passes it yet).
6. **The tests lens outside the change**: a test missing for code the
   change did not touch was opened as an issue (tried.md, runs 9 to 11);
   in #90's step 1 the tests lens left such code alone (0 of 1), while the
   edge-cases lens had three weak overflow findings outside the change
   verified, each an issue on a forge.
7. **A removal's reach past three lines**: a removed guard whose defect
   shows further down (a slice ten lines on) is still sent outside the
   change; the window is a diff's context, not the code's flow.
8. **The lenses together, measured** (#147): validated on #146's
   commits, 104.7k tokens against 430k, two of the three true findings
   verified, the third refused by the judge for want of material — since
   #127, verified (tried.md); the lenses together not yet measured against
   the lenses apart on the evaluation's cases.
9. **The judge's false negatives, counted again** (#90, #127): the
   evaluation's cases with the judges reading whole functions, by a real
   agent; and a call across languages other than by name (a script's
   JSON read by Go) is found only when the finding names the function.
10. **#126, measured further**: intent, claims and decisions by #90 (on
    by default until then); a decision on workline's own pull requests,
    not tried. The diff-alone facet, off: worth trying again on changes
    whose context explains a wrong line away (a helper elsewhere that
    seems to guard it), which the evaluation's cases do not plant, and
    with five runs (ADR-0014).
