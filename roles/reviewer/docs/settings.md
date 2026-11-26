---
sources: [roles/reviewer/role.yaml#events, roles/reviewer/role.yaml#apart, roles/reviewer/role.yaml#duties, roles/reviewer/role.yaml#part-intentions, roles/reviewer/role.yaml#without-ai, roles/reviewer/role.yaml#settings, cmd/workline/main.go#reviewCmd, internal/routing/adopt.go#AddReviewer, roles/reviewer/code/reviewer.go#settings, roles/reviewer/code/reviewer.go#ahead, defaults/ci/gitlab/workline.gitlab-ci.yml#workline:propose.script]
checked: e81df7b34
verified: agent:claude-code
---
# Settings

Part of [the reviewer](../README.md). Under `roles: {reviewer: {settings: …}}` in `.workline/config.yaml`
([config reference](../../../docs/config.md)); defaults from role.yaml:

| Key | Default | |
|---|---|---|
| `base` | `main` | where `workline review` starts without `--base` |
| `ignore` | `*.md`, `**/*.md`, `docs/**`, `LICENSE*`, `**/*.txt`, `CHANGELOG*` | not code: a change touching only these asks nobody |
| `lenses` | `[correctness, edge-cases, tests, intent, claims, simplicity, comments]` | in this order, in turn, on a merge request; a lens with nothing to read skipped; `comments` takes no turn, asked whenever `story-words` flags a comment |
| `lenses-per-push` | `1` | `--input lenses=all` asks every one, on the whole change: the templates pass it when the merge request becomes ready |
| `commits-per-push` | `5` | after ready: new commits a lens that warns reads a push, the oldest first; the rest at the next push, said; none left when what its own pipeline runs changes; `0`: every one ([more](ready.md#after-ready-only-what-is-new)) |
| `pipeline-paths` | none | what else the merge request's own pipeline runs, besides `.github/`, `.gitlab-ci.yml` and `.workline/`, as `[ci/, scripts/ci/]`: a change to it skips no commit on the record ([more](run-record.md#3-the-record)) |
| `lenses-together` | `true` | the lenses of a run in one call; `false`: a call each |
| `diff-alone` | `false` | `true`: one more call, given the change only |

## Specs, and comments

| Key | Default | |
|---|---|---|
| `spec-lenses` | `[ambiguous, unverifiable, out-of-scope, contradicted]` | the lenses a spec is read through |
| `spec-rounds` | `5` | on the forge: reviews in a row leaving a finding open, then a question to a person (1 to 20) |
| `specs-per-run` | `5` | on the forge: the specs a run reads, the rest at the next run (1 to 20; [a spec](spec-forge.md)) |
| `comment-block-max` | `8` | lines of one added comment (`long-comment`, a warning) |
| `story-words` | `used to`, `the bug was`, `previously` | a comment that may tell the code's history: a guess the `comments` lens judges; with no agent, or no lens with `hints: bug-story`, a warning |

## What blocks, and who judges

| Key | Default | |
|---|---|---|
| `ai-findings` | `warn` | `block`: a verified important finding blocks; by lens, `{correctness: block}` (below) |
| `judge-at-least` | `context` | `model` or `provider`: the judge's independence |
| `judge-at-least-by-role` | `{developer: model}` | a higher floor for a change a role's commits made: the [developer](../../developer/README.md)'s code judged by another model |
| `forge-writes` | `true` | `false`: no comment, on a line or the summary, nor issue: the verdict only |
| `finder-floor` | `true` | each lens looks for a number of candidates first; never a floor on what is shown |

- **`ai-findings` by lens**: a lens not named warns; of findings on one
  line, a verified one from a blocking lens leads.
- **`lenses`**: intent and simplicity have nothing to read when the
  change closes no issue, and are skipped ([more](run-lenses.md#4-the-lenses)); simplicity's findings are
  nits, never judged nor asked of the developer, until its false alarms
  are counted on past pull requests
  ([ADR-0068](../../../docs/adr/0068-the-line-steps-back-on-its-own-drift.md#next-each-decided-when-reached),
  [#1497](https://github.com/JN0V/workline/issues/1497)).

## Caps

What a run may read and write, and spend. On the line's run on an
issue, the issue's size tier multiplies `diff-lines-max`,
`code-lines-max` and `ai-max-tokens`
([size tier](../../../docs/spec/routing.md#once-a-state-the-record)):

| Key | Default | |
|---|---|---|
| `findings-max` | `10` | findings on the change a run; the rest counted |
| `issues-max` | `3` | issues opened a run for what lies outside it |
| `questions-max` | `3` | questions put to a person a run; the rest counted |
| `diff-lines-max` | `1500` | lines of diff a lens's task holds; past them, a file to read, or no lens asked ([too large](too-large.md)) |
| `ai-max-turns` | `40` | turns of a lens that reads the repository itself |
| `code-lines-max` | `600` | lines of the change a judge is shown, and of the code a spec names |
| `judge-lines-max` | `200` | lines of code a judge is shown, the rest named; `0`: the 31 lines around the cause |
| `judge-read-rounds` | `2` | rounds of reads a judge may ask before it answers, 8 at most ([when the judge reads](judge.md#when-the-judge-reads)); `0`: none, a no drops the finding |
| `judge-read-lines-max` | `400` | lines read for one judge, all its rounds; the rest named; `0`: no read |
| `tests-lines-max` | `300` | lines of tests a tests-lens judge is shown |
| `testimony-lines-max` | `80` | lines of what the author says (commit messages, the merge request) |
| `issue-lines-max` | `160` | lines of the issues the change closes (Need, Example, Steps, Validation, Verification, Scope) |
| `ai-max-tokens` | `300000` | tokens a run may spend, all calls; the call crossing it is paid; `0`: no cap |
| `spec-lines-max` | `300` | lines of a spec given; the rest cut, said |
