# Documentalist — where it stands (2026-10-01, afternoon)

How finished the role is: what is built and tried on real repositories, what
is only built, what is missing. tried.md has the story of each try; this page
is the summary to start from.

**In one line:** usable every day on a GitHub repository that takes pull
requests — docs judged on each pull request and gardened at night, by Claude
in CI, fixes committed by a GitHub App and reviewed on the forge. Beta: the
catch-up of a large backlog in parts has run once on DomoticsCore, right but
for a badge's link; Sonnet often cites lines wrongly, and Opus pays again;
GitLab is untried.

## Built and tried for real

| What | Tried on |
|---|---|
| Finding suspect docs from git history, cascades cut, propagation at the release | workline, DomoticsCore |
| Hygiene without AI: budgets (body only), duplicates, links inside and outside (lychee), identifiers gone, superseded decisions cited, derived blocks, code no doc describes, sources gone | workline, DomoticsCore |
| Adoption (`workline init`): each doc's sources proposed | workline, DomoticsCore (63 of 64 docs) |
| Judging a suspect doc whole, against its sources whole beside the diffs; a fix it cannot vouch for kept, `checked` left (ADR-0012) | evaluation; DomoticsCore |
| A fix may grow a doc by a tenth; a refused doc left out, the others applied | DomoticsCore gardening (PR #106) |
| `checked` after a squash or a rebase: the commit that brought it stands for it | fixture, by hand |
| On each pull request in CI: Claude judges, the App commits the fix, checks rerun, the line skips its own commit | workline PR #10; DomoticsCore PR #108, a real bug-fix pull request |
| Gardening at night: one pull request per task, at most 3 waiting | DomoticsCore, two runs by hand (PR #106, #109) |
| The push counts suspect docs in one line, asks nothing (ADR-0010, 0011) | workline, DomoticsCore |
| Condense, split, merge a card, merge a repeated passage (Opus) | evaluation; workline |
| Staying on the task: a spec's example left as it is when the default it does not show changes | evaluation (`stays-on-the-task`, workline PR #10 replayed), 3 in 3 on Sonnet |

## Built, tried without an agent

- **Not asked again** (ADR-0013): a fix not vouched for records `judged`;
  a gardening task waits while its pull request is open. On a copy of
  DomoticsCore with #109 open: the docs judged whole wait, the parts go.
- **A cap on a run's tokens** (`ai-max-tokens`): conformance only.

## Tried once for real

- **Judging in parts** (ADR-0009): measured on the evaluation (Sonnet in
  parts 12/12 ×3, at five times the tokens), then on DomoticsCore's
  README.md (2026-10-01, pull request #113): 8 parts and a fix, 180k
  tokens, the fix right but for a badge's link.

## Missing, in the order it matters

1. **Sonnet cites lines a line off in long docs** (twice in two runs on
   DomoticsCore): such a hunk is now placed where its quoted lines are,
   replayed on both answers; to watch on the next runs.
2. **Docs needing more than 8 parts** on DomoticsCore (five, 10 to 16
   parts): their sources to narrow, or the doc to split. The two small
   faults of that run are caught now: a version replaced on part of a line
   (`replaced-in-part`), a file name taken for a name of the code.
3. **A solo repository without pull requests**: built (ADR-0010, accepted)
   — `workline docs` from a ref it moves, the release held by docs suspect
   since the last tag, `doctor` saying where docs are judged; tried in
   conformance only, not yet on a real solo repository.
4. **Forks and GitLab**: a fork's pull request is commented from the
   repository's side (ci/github/workline-fork.yml, built, untried until a
   fork exists); the GitLab template now asks for its tokens unprotected, so
   they reach a merge request's branch, and lets Claude judge — untried on
   a live GitLab.
5. **workline's own docs**: README, usage and most specs are suspect, far
   behind; gardening on workline with Claude and parts would catch them up.
6. Smaller: budgets merge one level deep; Opus's notes see more than the
   task asks (unused); Copilot CLI on a free plan unmeasured (needs the
   owner to turn Copilot Free on).

## Measures

tests/evaluation/results.tsv; `go run ./tests/evaluation/summary`. The
documentalist cases all at their best on Sonnet 5.5 (2026-09-30 and
10-01): fixes a doc made false, confirms one still true, never confirms a
stale doc made false, propagates at the release, opens an issue on a spec,
merges a passage and a card, splits a card, condenses (Opus), finds
planted defects whole and in parts.
