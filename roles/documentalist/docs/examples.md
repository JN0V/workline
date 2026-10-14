---
sources: [internal/builtin/documentalist/examples.go, internal/builtin/documentalist/judge.go, roles/documentalist/role.yaml]
checked: 592d886
verified: agent:claude-code
---
# Documentalist — the code examples a fix writes

Part of [the judge](judge.md). The removal rule reads the words a fix
takes out; no word says whether a code example compiles, or runs right.
On DomoticsCore, a fix rewrote an example that compiled and was wrong at
run time; its pull request said "fixed in 8 places", the example among
them ([#121](https://github.com/JN0V/workline/issues/121)).

## Told apart

- **A code example**: a fenced block a fix adds or changes.
- **Not one**: a block moved unchanged; a block whose only change brings
  a line count the engine found off to its number (a fenced listing's
  "Clock.h (569 lines)"): the engine's count checked it.
- **Counted apart**: "fixed in 1 place and 1 code example", never one of
  the places fixed.

## Unverified, unless the project checks it

| The project | The example | Said |
|---|---|---|
| names no command | applied | `example-unverified`, and in the merge request: "unverified (lines 10-12): read it before merging" |
| names one, it passes | applied | `example-checked`: "checked by `<command>`" |
| names one, it fails | withheld, the rest of the fix applied | `example-failed`, with the command's output |

- **The setting**: `examples: {check: <command>}`, in the documentalist's
  settings ([README](../README.md#settings)); the command lives in the
  config, as `derive`'s do, never in a doc.
- **How it runs**: on a checkout of `HEAD` with the fixed docs written in,
  the doc whose examples changed as its argument (`"$@"`), in the job the
  agent runs in: give it nothing the agent may not have. Past 10 minutes,
  it has failed.
- **Withheld**: the doc keeps `checked` and records `judged`, as any place
  withheld ([what holds is applied](judge.md#what-holds-is-applied)); the
  last 10 lines of the output are shown.
- **A command that checks examples**, for instance `python3 -m doctest
  "$@"` for Python's `>>>` examples (a blank line before the closing
  fence, or doctest reads the fence as output), or a project's own script.

No model reads an example; one checking it against the signatures it
calls is parked ([#353](https://github.com/JN0V/workline/issues/353)).
