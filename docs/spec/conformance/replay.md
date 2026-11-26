---
sources: [tests/evaluation/replay_test.go, tests/evaluation/review_test.go#matchDefects, tests/evaluation/summary/main.go#printSets, tests/evaluation/eval_test.go#TestReplay, tests/evaluation/eval_test.go#onDemand, tests/evaluation/eval_test.go#cloneAt, .github/workflows/eval-replay.yml]
checked: c6853bf82
---
# Replay set — real misses, played again

The fixtures ([Evaluation — four measures](evaluation.md)) plant defects
a person invented. The replay set holds what the line really got wrong
on DomoticsCore, TraceGlass and workline, at the commits it happened on,
so a change to a role is measured on it before and after.

## What a case is

- An evaluation case (same `run`, `grade`, `review`), under
  `tests/evaluation/replay/<role>/`.
- `given.clone: {repo, base, head}`: the repository on github.com; the
  run sees `main` at `base` and the branch `change` at `head`. It is
  cloned from a mirror in the user's cache
  (`~/.cache/workline/eval-mirrors/`), made and fetched on the first run.
- `given.forge`: the issue and the pull request as they read then.
- `about` names each point, who found it, and what the line said then.

## The sets, and their score

| Set | Files | Score |
|---|---|---|
| Reviewer misses | `reviewer/<repo>-<n>-*` | recall: points found; unplanted Importants as `extra`, not failed |
| Reviewer kept catches | `reviewer/kept-*` | a true finding caught then, then fixed: lost cases |
| Reviewer clean | `reviewer/clean-*` | merged, nobody sent it back: false Importants fail |
| Documentalist | `documentalist/*` | wrong claims fixed; one judge check on false edits |
| Product owner | `product-owner/*` | the kind the owner corrected it to |
| Developer | `developer/*` | the missing work done; no "Closes" over work left out |

A reviewer replay is scored stricter than a fixture:

- **An Important only**: a nit at a planted place finds nothing.
- **One finding, one point**: findings and points are paired, each at
  most once, a finding of the point's own lens first.
- **Outside the change too**: an Important at a planted place off the
  changed lines counts, unless the judge refused it.
- **An issue place is the intent lens's**: a finding quoting `#139`
  finds only an intent point.
- **False Importants fail a clean change only**: on a miss or a kept
  catch they are `extra` in the measure, unverified.

Misses alone would reward a role that says more: a change is kept only
when it raises recall without losing a kept catch or adding a false
Important on a clean change.

## Run one role's set

```sh
WORKLINE_EVAL=claude go test -count=3 -timeout 120m \
  -run 'TestReplay/reviewer/' ./tests/evaluation/
go run ./tests/evaluation/summary     # per case and per set, a workline commit each
```

- `-run 'TestReplay/<role>/'` picks a set; `TestReplay/reviewer/kept-`
  or `clean-` a part of one; one case by its name.
- `-count=3`: each case three times. The summary says, per case,
  how many runs earned every point (`2/3`) and the worst; per set, the
  cases full in every run. A commit's runs are never pooled with
  another's: a before and an after stay apart. The commit is the last
  to change the engine, the roles, the cases, the replays or the code
  scoring them; a fixture's runs stay pooled.
- Each run appends a line to `results.tsv`, with its tokens.
- `WORKLINE_JUDGE` is needed for the documentalist's judge check.
- `given.needs` names what a case needs on the machine: PlatformIO
  (`pio`) for DomoticsCore's developer cases, `$ANDROID_HOME` and
  `java` for TraceGlass's. Missing, the case is skipped ("no
  toolchain"), never scored.
- `TestReplay` runs only when `-run` names it: never in the weekly run
  nor in a plain `go test ./...`.
- With no agent, `TestReplayCasesLoad` reads every case strictly, and
  `TestReplayCasesPointRight` checks each reviewer place's text at its
  head when the mirror is in the cache.

## On GitHub's runners

[eval-replay.yml](../../../.github/workflows/eval-replay.yml), by hand,
runs a set off a person's machine, each run its own job:

```sh
gh workflow run eval-replay.yml -f ref=<branch|refs/pull/<n>/head|sha> \
  -f run='TestReplay/developer/' -f eval=claude -f judge=claude:opus -f runs=3
```

- The toolchains are there: PlatformIO, a JDK 21, the runner's Android SDK.
- `env` takes more `WORKLINE_*` variables, `KEY=VALUE` separated by
  spaces or `;`.
- `cases` is `go test -parallel`: 2 cases at once in a job by default.
- Each job's artifact holds its `results.tsv` lines and its log; the
  runs kept too when the branch reads `WORKLINE_EVAL_KEEP`,
  `WORKLINE_EVAL_FINDINGS` or `WORKLINE_FREE_TRANSCRIPTS`.
  `eval-replay-rows` holds every job's lines, to append to `results.tsv`.
- The ref's code runs with the agent's login: dispatch only a ref whose
  code you trust, never a fork's pull request head.

## Cost

Prompt sizes measured with a fake agent (0 tokens), and three cases run
for real on 2026-11-25 (the reviewer on Sonnet, its judge on Opus):

- A reviewer case: 18–53k tokens (lenses' call 8–38k, judges the rest);
  about 38k on average. DomoticsCore #273's prompt is the largest, about
  4 times the median.
- The reviewer's set, one run: about 0.9–1M tokens; at `-count=3`,
  about 3M. Run a part (`kept-`, `clean-`, one case) while iterating;
  the whole set before and after a change.
