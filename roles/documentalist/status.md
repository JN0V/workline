# Documentalist — where it stands (2026-10-01, evening)

How finished the role is: what is built and tried on real repositories, what
is only built, what is missing. tried.md has the story of each try; this page
is the summary to start from.

**In one line:** usable every day on a GitHub or GitLab repository that
takes merge requests — docs judged on each one and gardened at night, by
Claude in CI, from a release of workline (v0.1.1, docs/ci.md), fixes
committed to the branch and reviewed on the forge. Beta: the catch-up of a
large backlog in parts has run once on DomoticsCore, right but for a
badge's link; a fix in parts once followed a stale code comment; a
repository without merge requests is built but untried for real.

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
| A fork's pull request: judged without secrets, its report commented from the repository's side, one comment kept (ci/github/workline-fork.yml) | JN0V/workline-sandbox #1, from jn0v-lab's fork |
| On a GitLab merge request: Claude judges, the token commits the fix to its branch, the next pipeline skips the line's own commit | gitlab.com JN0V/workline-sandbox !1; a fork's merge request untried |
| Condense, split, merge a card, merge a repeated passage (Opus) | evaluation; workline |
| Staying on the task: a spec's example left as it is when the default it does not show changes | evaluation (`stays-on-the-task`, workline PR #10 replayed), 3 in 3 on Sonnet |

## Built, tried without an agent

- **Not asked again** (ADR-0013): a fix not vouched for records `judged`;
  a gardening task waits while its pull request is open. On a copy of
  DomoticsCore with #109 open: the docs judged whole wait, the parts go.
- **A cap on a run's tokens** (`ai-max-tokens`): conformance only.
- **A partial claim settled by another part** (workline #29, where a stale
  code comment won over the setting): a passage one part finds partial and
  another supports is not handed to the fix. Conformance only; to watch on
  the next runs in parts.

## Tried once for real

- **Judging in parts** (ADR-0009): measured on the evaluation (Sonnet in
  parts 12/12 ×3, at five times the tokens), then on DomoticsCore's
  README.md (2026-10-01, pull request #113): 8 parts and a fix, 180k
  tokens, the fix right but for a badge's link.
- **A solo repository without pull requests** (ADR-0010): a copy of
  WaterMeter, pushed to a local remote (2026-10-01, tried.md): adopted by
  `workline init`, the push counting the docs with no agent, `workline
  docs` judging them, the release held by those left. Three defects found
  and guarded.

## Missing, in the order it matters

1. **A fix vouches for what it did not check.** Reviewed on 2026-10-01:
   every version the agent wrote was right, but moving `checked` vouched
   for frozen numbers still false beside them — line counts, test counts,
   a bug fixed long ago (DomoticsCore 16c660b, 92feb08) — and the same
   version was left stale in sibling docs. Numbers a doc can derive
   (`workline:derive`) or drop should go; a fix should not vouch for a line
   it did not read against a source.
2. **Sonnet cites lines a line off in long docs** (twice in two runs on
   DomoticsCore): such a hunk is now placed where its quoted lines are,
   replayed on both answers; to watch on the next runs.
3. **Docs needing more than 8 parts** on DomoticsCore: six (10 to 16
   parts). Two narrowed to 8 (branch docs/narrow-wide-sources, not pushed
   yet); four to split, each holding more than one doc:
   reference/eventbus-architecture.md, components/core/project-context.md,
   architecture/component-configuration-pattern.md,
   components/webui/project-context.md.
4. **Judging in parts off by default**: on a solo repository far behind
   (WaterMeter, tried.md), 7 of 10 suspect docs were too large to be judged
   whole and went to a person. Measured since (status above): to decide
   whether it is on by default.
5. **workline's own docs**: README, usage and most specs are suspect, far
   behind; gardened nightly in parts since #27, first night to watch.
   roles/documentalist/README.md, which needed 15 parts, is split into
   four pages of 3 to 6 parts each, their sources narrowed to their files.
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
