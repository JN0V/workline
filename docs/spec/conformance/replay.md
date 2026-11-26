---
sources: [tests/evaluation/replay_test.go, tests/evaluation/review_test.go#matchDefects, tests/evaluation/summary/main.go#printSets, tests/evaluation/eval_test.go#TestReplay, tests/evaluation/eval_test.go#onDemand, tests/evaluation/eval_test.go#cloneAt, tests/evaluation/meaning/main.go, tests/evaluation/findings_test.go]
checked: 74faf6584
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
| Documentalist | `documentalist/*` | wrong claims fixed, a true page vouched for; one judge check on false edits |
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

## Scored by meaning

The score above is by place: any Important quoted on a planted line earns
the point, and the same finding quoted elsewhere earns nothing. It rewards
an arm that says more, and it cannot compare a reviewer that quotes the
issue with one that quotes the code. The scorer in
`tests/evaluation/meaning` asks what each finding says.

- **The points**: `points.yaml` states each planted defect as the owner
  sent it back, what does not count as saying it, and its kind: out of
  the diff, weak or missing test, issue not delivered, in-diff code. A
  point marked `out` (ambiguous, duplicate) is judged and left out of
  the score.
- **The judge is blind**: it gets a case's points and every finding of
  one run, each with its place, title and body. Never the arm, its
  model, the lens, the severity, nor what the engine's judge said.
- **Per point**: which findings say it, which say a part of it, and the
  words quoted. A run earns the point when an Important shown says it.
  Said only by an Important the engine's judge refused, or only by a
  nit, is counted apart.
- **Clean changes**: the judge reads the repository at the change's head
  and says of each Important shown: true defect, nit or false alarm.
- **Twice**: each judgement is asked twice, and a third time when the
  two differ; the middle verdict holds. Judgements are kept, so scoring
  again asks only what is new.
- **One judge for every arm**: `claude:opus@medium` unless `-judge`
  names another, `claude:<model>@<effort>` or `cmd:<any command>`.

```sh
WORKLINE_EVAL_FINDINGS=runs.jsonl WORKLINE_EVAL_ARM=before WORKLINE_EVAL=claude \
  go test -count=3 -timeout 120m -run 'TestReplay/reviewer/' ./tests/evaluation/
go run ./tests/evaluation/meaning -findings runs.jsonl -out scores/
```

- `WORKLINE_EVAL_FINDINGS` keeps each run's findings whole, a JSON line
  a case run, under the arm `WORKLINE_EVAL_ARM` names. Any reviewer can
  write the same lines: `arm`, `case`, `tokens_in`, `tokens_out`, and
  `findings` (`place`, `severity`, `refused`, `title`, `body`).
- `scores/scores.tsv`: a row per point and run, with both judgements.
- `scores/summary.md`: per arm, the points said a run with their mean
  and spread, by kind of point, the Importants on no point, the false
  alarms on clean changes, the tokens.
- It costs about 4k tokens a judgement on a miss, about 50k on a clean
  change with an Important shown.

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
- `WORKLINE_EVAL_KEEP=<folder>` keeps what each run left, a folder a
  case and run: the run folders (tasks, answers, calls) and the change
  the role made (`change.diff`). The repository goes with the test; why
  a point was lost is read there.
- `given.needs` names what a case needs on the machine: PlatformIO
  (`pio`) for DomoticsCore's developer cases, `$ANDROID_HOME` and
  `java` for TraceGlass's. Missing, the case is skipped ("no
  toolchain"), never scored.
- `TestReplay` runs only when `-run` names it: never in the weekly run
  nor in a plain `go test ./...`.
- With no agent, `TestReplayCasesLoad` reads every case strictly, and
  `TestReplayCasesPointRight` checks each reviewer place's text at its
  head when the mirror is in the cache.

## Cost

Prompt sizes measured with a fake agent (0 tokens), and three cases run
for real on 2026-11-25 (the reviewer on Sonnet, its judge on Opus):

- A reviewer case: 18–53k tokens (lenses' call 8–38k, judges the rest);
  about 38k on average. DomoticsCore #273's prompt is the largest, about
  4 times the median.
- The reviewer's set, one run: about 0.9–1M tokens; at `-count=3`,
  about 3M. Run a part (`kept-`, `clean-`, one case) while iterating;
  the whole set before and after a change.
