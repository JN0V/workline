# Documentalist — where it stands (2026-10-01)

How finished the role is: what is built and tried on real repositories, what
is only built, what is missing. tried.md has the story of each try; this page
is the summary to start from.

**In one line:** usable every day on a GitHub repository that takes pull
requests — docs judged on each pull request and gardened at night, by Claude
in CI, fixes committed by a GitHub App and reviewed on the forge. Beta: the
catch-up of a large backlog in parts runs for the first time on DomoticsCore,
caps on tokens are still coarse, GitLab is untried.

## Built and tried for real

| What | Tried on |
|---|---|
| Finding suspect docs from git history, cascades cut, propagation at the release | workline, DomoticsCore |
| Hygiene without AI: budgets (body only), duplicates, links inside and outside (lychee), identifiers gone, superseded decisions cited, derived blocks, code no doc describes, sources gone | workline, DomoticsCore |
| Adoption (`workline init`): each doc's sources proposed | workline, DomoticsCore (63 of 64 docs) |
| Judging a suspect doc whole, against its sources whole beside the diffs; a fix it cannot vouch for kept, `checked` left (ADR-0012) | evaluation; DomoticsCore |
| A fix may grow a doc by a tenth; a refused doc left out, the others applied | DomoticsCore gardening (PR #106) |
| `checked` after a squash or a rebase: the commit that brought it stands for it | fixture, by hand |
| On each pull request in CI: Claude judges, the App commits the fix, checks rerun, the line skips its own commit | workline PR #10; DomoticsCore installed |
| Gardening at night: one pull request per task, at most 3 waiting | DomoticsCore, one run by hand |
| The push counts suspect docs in one line, asks nothing (ADR-0010, 0011) | workline, DomoticsCore |
| Condense, split, merge a card, merge a repeated passage (Opus) | evaluation; workline |

## Built, measured, not yet tried for real

- **Judging in parts** (ADR-0009): measured — Sonnet in parts 12/12 ×3, as
  judging whole, at five times the tokens; Opus 12, 11, 12; Haiku not for
  parts. Turned on for DomoticsCore's nightly gardening by its PR #107: the
  first real parts run there is the next thing to watch (tokens, the pull
  requests' quality, `sources-too-wide`, `uncovered`).

## Missing, in the order it matters

1. **Token caps a run**: `ai-max-calls` counts docs per call, not calls; no
   cap on tokens a run, no verdict cache keyed by the doc's and sources'
   blobs. A pull request pushed five times is judged five times.
2. **A solo repository without pull requests** (ADR-0010): the release gate
   on suspect docs, and `workline docs` judging from a ref it moves — not
   built. `workline doctor` still warns when the documentalist is not routed
   before a push, and `workline init` still routes it there: both predate
   ADR-0010.
3. **Forks and GitLab**: a fork's pull request gets the job summary only,
   no comment; the GitLab template keeps its token protected, so it never
   reaches a merge request's branch — untried on a live GitLab.
4. **Staying on the task**: the agent once aligned an example of a spec to
   the default (workline PR #10). An evaluation case should catch it.
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
