# ADR-0051: A check sits where its catch is worth its cost

- **Status:** accepted; steps 1, 2, 5, 8 and 9 built ([ADR-0013],
  [ADR-0042] and
  [ADR-0020](0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md#amendment-2026-10-21-adr-0051-cut-4-no-fix-round-for-nits),
  amended); step 3's removal of the release's docs built ([ADR-0010],
  [ADR-0017], [ADR-0034] amended); the rest not yet
  ([the build order](#build-order)); one point open, one decided
  ([open](#open))
- **Date:** 2026-10-21
- **Decided by:** the maintainer, on 2026-10-21: the principle and five
  cuts, as a panel of three lenses (latency, reliability, cost) refined
  them against the measures
- **Amends:** [ADR-0050]: its moment table (§3), what it kept (§6) and
  the release's docs at night (§7); it placed each act by the wait
  alone, now by catch and cost too
- **To amend once built:** [ADR-0010] and [ADR-0017] (docs hold the
  release), [ADR-0034] (removed with the release's docs), [ADR-0041]
  (follow's docs merges), [ADR-0042] (up to date, one at a time)
- **Builds on:** [ADR-0011], [ADR-0043]; principles 4, 5, 7, 9, 12, 13
- **Research:** [catch-and-cost.md](../research/catch-and-cost.md)

## Context

Four weeks of workline's own work, measured on 2026-10-21:

- **Escapes stayed flat** as checks came in: 0.42 to 0.67 a product PR.
- **A product PR costs about four times more**: 16.8 → 64.8 M builder
  tokens, 21 → 92 min (S1 → S4); 2.08 extra rounds, 1.23 the line's.
- **Best catch**: Important correctness and edge-cases findings, 94 of
  104 acted on, for 0.30 rounds a PR.
- **Worst**: the release waiting on judged docs, 130 red heads and 61
  docs PRs for one truth fix; strict up to date, 103 rounds, and no test
  red after a merge of main.
- **Upkeep**: 42% of merged PRs in W43; three circles keep coming back.
- **79% of the tokens is the build run**, which no placement reaches.

## Decision

### 1. The principle

Each check runs at the moment where its catch is worth its cost.

- **Local**: fast, cheap and deterministic. A convenience: `--no-verify`,
  web edits and agents with no hooks skip it.
- **Merge request**: deterministic checks always re-run, because local
  hooks can be bypassed; AI judgement once per patch, never redone.
- **Release**: the full battery has had its turn; it blocks only on
  deterministic results of the exact commit.
- **Night or week**: the rest.
- **What an agent judges opens a PR or an issue**; it never holds a
  release.

### 2. The moments

B: its failure holds the merge or the release. W: said, never holding.
Ships: *default* in the templates and `routing.default.yaml`;
*workline* on workline's own line only.

| Moment | What runs | B / W | Ships |
|---|---|---|---|
| Commit | the committer's rules; gitleaks; a refused message rewritten, 30 s at most | B | default |
| Push, opt-in | the committer on the range; docs the range made wrong, listed, no stamp asked | B; docs W | default |
| The builder's own runs | tests; the try, told under `## Tried` | — | workline |
| Merge request, each head | the committer on the range and the squash title; the gates' new findings; the reviewer's rules; the project's tests | B | default; tests the project's |
| Merge request, AI once a patch | correctness and its judge | B | default |
| | edge-cases: its Important findings get one round | W | default |
| | tests, intent; every lens's nits: shown, no round | W | default |
| | the inspector's triage | W, apart | default |
| Merge request, at ready | the documentalist's AI: a comment with the diff, never a push | W | default |
| Main after a merge | one run: the gates' whole report, the whole repository's docs checks; the project's tests; a red main opens one issue | W | default; tests the project's |
| Release, weekly; a patch release when a fix touches a real user | main's tip green, read, not re-run; the release PR's tests and build; an open issue labelled to hold it | B | default; tests and build the project's |
| | `workline version` from the built binary; the candidate, `--ai none`, on the GitHub and GitLab sandboxes | B | workline |
| Night | the product owner, the spec review | W | default, weekly in the templates |
| Week | one docs PR, judged at night; the weekly sample | W | default |
| | the evaluation, a real agent: a fall a second run confirms opens an issue labelled to hold the release | W | workline |

### 3. The five cuts

#### Cut 1: the release, weekly

- **Monday**, after the week's sample and evaluation have run; a patch
  release when a fix touches a real user.
- **It never waits on judged docs.** Deleted: the release-docs task, its
  comment record, and the dry run's comment on the release PR.
- **It blocks on** §2's release rows, run beside the tests, never after
  them (the tests alone take 240 s, over the 3 min budget). The hold is
  the forge's own: an issue and a label the person can remove. A sandbox
  that could not run is said, never read as a pass.
- **The evaluation never runs on the release's path**: a fall a second
  run confirms opens such an issue.

#### Cut 2: strict up to date, dropped

- **A PR merges once its own head is green**, with no "one at a time".
- **The net is main's run**: a red main opens one issue and notes "base
  red, not yours" on open PRs.
- **Red mains are counted a week.** Above one a week, the forge's queue
  goes on: GitHub's merge queue in an organisation, GitLab's merge
  trains.
- **GitLab users keep merged-results pipelines**: without them a merge
  request's pipeline tests its branch alone.

#### Cut 3: no docs obligation per PR

- **No zero-suspect rule, no `checked` stamps pushed to PRs**: the
  builder fixes the prose its change made wrong.
- **One docs PR a week**, judged at night, skipping docs an open PR
  changes. Follow's docs merges then go.
- **The documentalist's AI on the merge request at ready only**: a
  comment with the diff, never a push. It stays as the sensor of what
  this cut gives up.
- **The documentalist's criterion** "nothing false lands" ([#464])
  becomes "nothing false stays past a week's docs PR".

#### Cut 4: no fix round for nits

- **No round for any lens's nits**, nor for the tests and intent
  lenses' findings: shown, no fix asked.
- **Correctness, and the judge, keep blocking.**
- **Edge-cases stays a warning**; its Important findings get one round.
- **A team wanting Important findings to hold the merge** uses the
  forge's "conversations resolved" rule, nits kept in the summary.

#### Cut 5: one record of what was read

- **Trusted only from workline's own account**, never from a comment
  another edited last: the forge says who did, in GraphQL (GitHub's
  `editor`, GitLab's `lastEditedBy`).
- **A change to what the PR's own pipeline runs**, by any of its
  commits, gets no skip: such a PR runs its own workflow, which wrote
  what the record says was read. By default `.github/` (workflows,
  actions), `.gitlab-ci.yml` and `.workline/`; a project whose pipeline
  runs scripts from elsewhere adds them (the reviewer's
  `pipeline-paths`). Not `ci/` by default: in workline it holds product
  templates its own pipeline does not run. 14 of workline's last 60
  merged PRs would be read whole at each push, 21 with `ci/`.
- **Not on the forge's own marker** (a GitHub review's `commit_id`,
  GitLab's diff versions), amended once the design was looked at:
  - **It ends no race**: a push's run reads before the run before it
    is applied, whatever the apply writes to.
  - **One notification a push**: a GitHub review needs a body, so it
    adds an entry on the PR each push; a GitLab note pins a commit only
    on a line.
  - **The record holds more than the commits read**: patch ids, the
    lenses' turn, the lenses owing commits, what blocks, the findings
    judged; the inspector's holds findings, by fingerprint, no commit.
- **A fast second push reads twice** what the push before it read,
  accepted: about 2.3k tokens in on [#580]'s example.

### 4. Also

- **On GitLab, warn-only jobs run in a child pipeline** the merge does
  not wait for: "merge when the pipeline succeeds" waits for
  `allow_failure` jobs too.
- **The merge request checks the squash title** against the committer's
  rules, room left for GitHub's " (#N)": 12% of main's subjects are
  over 72 characters for that suffix.
- **Each row says whether it ships** as a default or is workline's own (§2).
- **A feature turned off on workline is removed** from the product, not
  kept untested: the release's docs, the docs-PR lifecycle's pieces,
  follow's docs merges. 27 of 119 escapes sat in those, 24 found by
  workline's own line.
- **The nightly job lands before the workday** ([#607]).

## Build order

By upkeep removed per unit of work:

1. **Trust the reviewer's record**: today `loadRecord` reads the first
   comment carrying its marker, from any author, while it writes the
   last. First, since the cuts below skip on records.
2. **Drop strict**: 103 rounds, merge series 0.89 a PR. Built ([#612]):
   no one at a time, none brought up to date but a release held for its
   docs; main's run red says "base red, not yours" once on each open PR,
   and its issue counts red mains a week; [ADR-0042] amended.
3. **The release** (cut 1): 130 red heads, most of 61 docs PRs.
4. **Stop stamps on PRs**: no `--push-to-merge-request`; 69 rounds.
5. **No rounds for nits** (cut 4): up to 93 rounds.
6. **The documentalist's AI at ready**, a comment with the diff.
7. **The weekly docs PR**, before the workday.
8. **Fold main's two runs**: `test.yml`'s findings into `route main`.
   Built: the findings job gone, 3.8 min a merge; lychee at night, in
   gardening; main's issue and gardening say what code scanning held.
9. **What was read, trusted** (cut 5): no record another edited last,
   no skip on a change to what the PR's own pipeline runs. Built ([ADR-0013]
   amended); the forge's markers left, why in cut 5.
10. **Remove follow's docs merges**.
11. **The GitLab child pipeline** for what warns.
12. **The squash title** at the merge request.

## Consequences

The panel's estimates from the measures, not tried; a fifth of main
merges assumed to stay for real conflicts.

- **Rounds per product PR**: 2.08 → about 1.0 (the line's: 1.23 → 0.2).
- **Upkeep share of merged PRs**: 42% (W43) → about 4%; 5% with the
  weekly release.
- **CI waited**: about 6 min less a PR; 4–5 min less per PR ahead in line.
- **Cost per issue**: 64.8 M tokens and 92 min → about 58 M and 69 min,
  still about ×3.5 of 09-30's week, because most of it is the build run.
- **Shipped before fixed**: 21 of 61 dated escapes were fixed before the
  Monday after their cause; 3 before the next release at today's pace.
- **Given up**: 9 prose fixes on PRs in four weeks (7 in W40), now a
  week later; one truth fix at a release; strict's unmeasured catch.
  Their sensors: the documentalist's AI at ready, the red-main count.

## Open

- **Forbidden terms in CI**, decided by the maintainer on 2026-10-21:
  the list stays local, not a CI secret. The accepted risk: commits made
  in CI are not scanned for those terms. The merge request's gitleaks
  runs its default rules only; a bypassed hook lets a term through. No
  term is ever written here.
- **The builder's own cost**: 79% of a product PR's tokens, the next
  lever; not decided here.

[#464]: https://github.com/JN0V/workline/issues/464
[#580]: https://github.com/JN0V/workline/issues/580
[#607]: https://github.com/JN0V/workline/issues/607
[#612]: https://github.com/JN0V/workline/pull/612
[ADR-0010]: 0010-docs-judged-on-the-merge-request-and-by-gardening.md
[ADR-0011]: 0011-the-review-is-on-the-merge-request-not-the-push.md
[ADR-0013]: 0013-what-was-judged-is-not-asked-again.md
[ADR-0017]: 0017-the-release-manager.md
[ADR-0034]: 0034-the-release-fix-follows-its-base.md
[ADR-0041]: 0041-workline-settles-its-own-lines-when-the-base-moves.md
[ADR-0042]: 0042-the-engine-merges-by-the-projects-policy.md
[ADR-0043]: 0043-a-try-is-told-in-its-pull-request.md
[ADR-0050]: 0050-each-act-has-its-moment-and-a-wait-budget.md
