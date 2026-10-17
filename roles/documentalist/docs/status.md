# Documentalist — where it stands (2026-10-17)

**In one line:** usable every day on a GitHub or GitLab repository that
takes merge requests: docs judged on each one and gardened at night, by
Claude in CI from a release of workline ([CI](../../../docs/ci.md)),
fixes committed to the branch and reviewed on the forge. Beta: judging
docs too large in parts, on by default, ran six nights on DomoticsCore
and workline; a fix in parts once followed a stale code comment.

Each try: up to 2026-10-13 in [tried.md](tried.md), an archive; since, in
its pull request's `## Tried`
([ADR-0043](../../../docs/adr/0043-a-try-is-told-in-its-pull-request.md)).
Sandboxes for trying CI: github.com/JN0V/workline-sandbox, its fork
jn0v-lab/workline-sandbox, gitlab.com/JN0V/workline-sandbox (its tokens
masked variables, the GitLab one expiring about 2026-10-31).

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: open. The docs to split are still
   to do.
   [#120](https://github.com/JN0V/workline/issues/120) is fixed in its
   simple version (Built: a doc judged on its diff);
   [#121](https://github.com/JN0V/workline/issues/121) too (Built: code
   examples said unverified, or checked by the project's command), a
   model reading them parked
   ([#353](https://github.com/JN0V/workline/issues/353)).
2. **The main issue validated on real use**: open. No issue of the role
   holds a Validation section the maintainer has signed.
3. **AI verdicts measured**: in part. Step 4 of
   [ADR-0014](../../../docs/adr/0014-checked-is-earned-by-what-was-read.md#4-acceptance-written-before-it-is-measured)
   holds, at the third run, on the held-out set: the bars and runs in
   [research](../../../docs/research/documentalist-fixes-reviewed.md#the-held-out-set-adr-0014-step-4)
   and [tried.md](tried.md) (2026-10-01 to 10-02). The `drifted` cases,
   five runs each, and how to run them again: [the
   evaluation](../../../docs/spec/conformance/evaluation.md#real-shapes).
   The other evaluation cases have one run each, not five.

## Built

Cases in
[tests/conformance/cases/documentalist](../../../tests/conformance/cases/documentalist);
its evaluation in
[tests/evaluation/cases/documentalist](../../../tests/evaluation/cases/documentalist).

- **Suspect docs from git history**, cascades cut, propagated at the
  release: `stale-docs-reported`, `suspect-*`, `range-*`,
  `cascade-is-cut`, `release-propagates-product-docs`.
- **Hygiene without AI**: budgets, duplicates, links (lychee), identifiers
  gone, superseded decisions, derived blocks, code no doc describes,
  sources gone: `budgets-reported`, `duplicates-reported`,
  `dead-links-reported`, `external-links-*`, `identifier-gone-*`,
  `superseded-decision-*`, `derived-block*`, `undocumented-code-reported`,
  `source-gone-reported`.
- **Adoption** (`workline init`), each doc's sources proposed: setup's
  `init-*`.
- **A suspect doc judged whole**, its sources whole beside the diffs; a
  fix it cannot vouch for kept, `checked` left:
  `sources-whole-beside-the-diff`, `suspect-doc-fixed`,
  `stale-doc-confirmed`, `fix-kept-without-vouching`.
- **A fix grows a doc by a tenth at most**; a refused doc left out:
  `fix-may-grow-a-doc-a-little`, `patch-must-not-grow`,
  `refused-doc-left-out-others-applied`.
- **`checked` after a squash or a rebase**:
  `doc-fixed-in-a-squashed-merge-request`,
  `doc-fixed-in-a-rebased-merge-request`,
  `judged-in-a-rebased-merge-request`.
- **A shallow clone said, never a silent pass**: `shallow-clone-*`,
  `release-refuses-a-shallow-clone`, setup's `doctor-shallow-clone`.
- **On each merge request**, GitHub and GitLab, the fix committed to the
  branch, its own commit not judged again; a fork's as a comment:
  `patch-pushed-to-the-merge-request`, `patch-as-comment-from-a-fork`,
  routing's `own-commit-*`.
- **The push counts suspect docs**, asks nothing: routing's
  `pre-push-line`.
- **Gardening at night**, one merge request a task, at most 3 waiting; not
  asked again while it is open, a doc judged in parts while any merge
  request changes it, the next doc going
  ([#356](https://github.com/JN0V/workline/issues/356)): `gardening-*`,
  `judged-not-asked-again`, `task-waits-for-its-merge-request`,
  `parts-go-while-suspect-waits`, `parts-record-waiting-not-judged-again`,
  `parts-judged-once-a-run`, `condense-waits-for-its-merge-request`.
- **Its stale merge requests rebuilt or closed**: `gardening-branch-*`.
- **People's pull requests settled on its own lines**: `persons-branch*`.
- **The merge policy**, `cautious`, and `normal` for workline's own
  routine pull requests
  ([ADR-0042](../../../docs/adr/0042-the-engine-merges-by-the-projects-policy.md)):
  the forge merges what a person accepted, kept up to date, the yes taken
  back on a later push; at `normal`, workline's own records, derived
  blocks and docs with no yes, each recorded, a revert demoting its kind: the
  [merge cases](../../../tests/conformance/cases/merge), routing's
  `forged-merge-is-judged`, `merge-of-another-branch-is-judged`.
- **A release tool's pull request held as the release**, its fix on a
  merge request of its own that follows `main`: `release-*`; on workline,
  [#258](https://github.com/JN0V/workline/pull/258) held,
  [#259](https://github.com/JN0V/workline/pull/259) merged first.
- **Condense, split, merge a card, merge a repeated passage**:
  `condense-*`, `split-*`, `merge-card-*`, `dedupe-*`.
- **Judging in parts**, a partial claim settled by another part:
  `judged-in-parts-*`, `parts-*`, `judge-in-parts-off`. On by default
  ([#82](https://github.com/JN0V/workline/issues/82#issuecomment-6163487736),
  decided 2026-10-14 on six nights; a night capped by `parts-max-per-run`):
  `judged-in-parts-by-default`; a doc too long itself says so at once:
  `suspect-doc-too-large`; a part's support quoted from a comment only is
  no support ([#106](https://github.com/JN0V/workline/issues/106)):
  `judged-in-parts-comment-does-not-*`.
- **A cap on a run's tokens**: `token-cap-between-parts`, engine's
  `token-cap-ends-the-run`.
- **`checked` earned by what was read**: `checked-unread-refused`,
  `checked-moves-when-read-whole`,
  `condense-new-doc-born-checked-refused`. The `checked` moved unearned
  put back: workline's 10 (0669684), DomoticsCore's 16 (0d86737 on its
  main).
- **What it reports and refuses of a fix**: `count-off-*`, `value-left-*`,
  `removal-*`, `comment-not-evidence-refused`,
  `docstring-not-evidence-refused`, `citation-not-a-source-refused`,
  `count-fixed-needs-no-claim`, `refused-place-reported-the-rest-applied`,
  `patch-part-not-applying-rest-applied`, `hunk-*`, `claim-without-doc-*`,
  `answer-not-yaml-asked-again-once`, `gardening-lists-places-withheld`.
- **Code examples a fix writes** told apart: said unverified, or checked
  by the command the project names, withheld when it fails: `example-*`.
- **Says what it read**: "nothing found wrong", and "in what was read"
  for a doc judged on its diff: `judged-nothing-found-wrong-says-so`,
  `judged-on-its-diff-says-what-was-read`.
- **The weekly sample of the docs vouched for**: `sample-*`.
- **Where it writes**: the project's forge, `local` or a command: forge's
  `local-*`, `cmd-*`.
- **Beyond our repositories**: `path-not-ascii-judged`,
  `superseded-decision-other-conventions`,
  `count-off-not-in-history-by-convention`,
  `comment-style-unknown-reported`, `whole-chars-*`,
  `docs-not-read-reported`, `removal-rule-reads-french`, setup's
  `init-without-ai-proposes-by-name`.
- **Docs for their reader**: the checks; with an agent, the pages they
  flag read last in a gardening run, a fix proposed on a merge request
  ([#256](https://github.com/JN0V/workline/issues/256)): `reader-*`.
  That merge request waits for a person's yes even at `normal`, until the
  project lets it merge alone, so the maintainer reads each one for that
  issue's Validation ([#438](https://github.com/JN0V/workline/issues/438)):
  merge's `normal-reader-*`, `alone-*`.
- **Staying on the task**: the evaluation's `stays-on-the-task`.

## Missing, in the order it matters

1. **The weekly sample's first weeks**, watched
   ([#155](https://github.com/JN0V/workline/issues/155)).
2. **DomoticsCore's docs needing more than 8 parts**: two narrowed (branch
   `docs/narrow-wide-sources`, not pushed); four to split
   (`reference/eventbus-architecture.md`,
   `components/core/project-context.md`,
   `architecture/component-configuration-pattern.md`,
   `components/webui/project-context.md`). No issue yet.
3. **workline's own docs far behind**, gardened nightly in parts since
   [#27](https://github.com/JN0V/workline/issues/27), to watch. No issue
   yet.
4. **The agent reads**: a doc judged on its diff against the code as it
   is now, a read each night (no issue yet).
5. **To watch**: Sonnet citing lines a line off in long docs (now placed
   by their quote); a partial claim settled by another part; the first
   nightly's two faults, with a second task's tokens. No issue yet.
6. **The merge policy, what is left**: at `normal`, the developer
   role's pull requests, pushed with the App, built with that role
   ([#117](https://github.com/JN0V/workline/issues/117); decided on
   [#322](https://github.com/JN0V/workline/issues/322#issuecomment-6186910822)).
7. **Smaller**: Opus's notes see more than the task asks (unused); Copilot
   CLI on a free plan unmeasured (the owner must turn Copilot Free on). No
   issue yet.
8. **Tries on real use** (no issue yet):
    - its stale merge requests closed on GitHub and GitLab, and settled
      people's branches pushed there (the App's `workflows` permission),
      each in its CI job;
    - the release fix following `main` in CI and on GitLab;
    - on GitLab, a fork's merge request; the weekly sample on GitHub; a
      `checked` found false;
    - `checked` earned by what was read, with a real agent;
    - the Forgejo sample on a live instance;
    - `whole-chars` raised, its tokens measured;
    - a real agent writing a code example, a real project's
      `examples.check` running on it;
    - the merge policy in CI: its jobs with the GitHub App's token, the
      `verdict` job, a merge starting `follow` and release-please on
      `main`, a release and its docs, a fork's pull request; on GitLab, a
      yes held through the new head's pipeline;
    - the jobs that write from workline-apply.yml, after a pull request's
      run, the App's key in its environment
      ([#331](https://github.com/JN0V/workline/issues/331)).
