---
sources: [tests/evaluation/replay_test.go, tests/evaluation/eval_test.go#TestReplay, tests/evaluation/eval_test.go#onDemand, tests/evaluation/eval_test.go#cloneAt]
checked: af68b1bb5
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
| Reviewer misses | `reviewer/<repo>-<n>-*` | recall: points found; false Importants (`fp`) |
| Reviewer kept catches | `reviewer/kept-*` | a true finding the line caught then, then fixed: lost cases |
| Reviewer clean | `reviewer/clean-*` | merged, nobody sent it back: false Importants |
| Documentalist | `documentalist/*` | wrong claims fixed; one judge check on false edits |
| Product owner | `product-owner/*` | the kind the owner corrected it to |
| Developer | `developer/*` | the missing work done; no "Closes" over work left out |

Misses alone would reward a role that says more: a change is kept only
when it raises recall without losing a kept catch or adding a false
Important on a clean change.

## Run one role's set

```sh
WORKLINE_EVAL=claude go test -count=3 -timeout 120m \
  -run 'TestReplay/reviewer/' ./tests/evaluation/
go run ./tests/evaluation/summary     # per case: runs, mean, range
```

- `-run 'TestReplay/<role>/'` picks a set; `TestReplay/reviewer/kept-`
  or `clean-` a part of one; one case by its name.
- `-count=3`: each case three times; read the worst run (the range's
  low end), never the best.
- Each run appends a line to `results.tsv`, with its tokens.
- `WORKLINE_JUDGE` is needed for the documentalist's judge check.
- The developer's cases need the project's own test tools on the
  machine: PlatformIO for DomoticsCore, an Android SDK
  (`ANDROID_HOME`) for TraceGlass; without them the run ends "build
  broken".
- A finding that quotes the issue (`#139`) counts for every point placed
  on that issue: read an intent point's `finding:` lines, not only the
  score.
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
