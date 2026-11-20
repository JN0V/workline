---
sources:
  - internal/builtin/documentalist/documentalist.go#notVouched
  - internal/builtin/documentalist/judge.go#judgedFixes.placesSaid
  - internal/builtin/documentalist/documentalist.go#readScope.String
  - internal/builtin/documentalist/documentalist.go#outcomes
  - internal/builtin/documentalist/documentalist.go#mergeRequest
  - internal/builtin/documentalist/values.go#valuesLeft
  - internal/builtin/documentalist/values.go#replacedVersions
  - internal/builtin/documentalist/values.go#versions
  - internal/builtin/documentalist/values.go#versionsIn
  - internal/builtin/documentalist/values.go#sourceFilesOf
  - internal/builtin/documentalist/values.go#versionRange
  - internal/builtin/documentalist/values.go#versionMarker
  - internal/builtin/documentalist/values.go#versionLater
  - internal/builtin/documentalist/values.go#saidAsNowAt
  - internal/builtin/documentalist/values.go#tag
checked: e2e165c2d
verified: agent:claude-code
---
# Documentalist — what the finding says

Part of [the check](judge.md).

## What the finding says

- A doc judged and not vouched for: what the run did, counted in its body
  — "fixed in 2 places", a code example apart ("and 1 code example"), or
  "nothing found wrong" when the patch only records `judged`; "applied in
  part" when places were withheld. Never
  "fixed" for a header alone: workline's first nightly, 2026-10-02, said
  "what the agent found wrong is fixed" of a doc it found true.
- A doc whose sources did not all go whole: "nothing found wrong in what
  was read", then `read:` what the task gave (what changed in a source
  since `checked`, or [the doc it follows](tasks.md#when-checked-may-move))
  and `not read:` the rest. Never
  "nothing found wrong" of what was not read
  ([#120](https://github.com/JN0V/workline/issues/120)). Gardening reads
  such a doc [in parts](parts.md) instead, when parts
  can ([#462](https://github.com/JN0V/workline/issues/462)).
- The fix's own merge request, gardening's (the scheduled run, nightly
  or weekly), lists each doc judged with that outcome —
  fixed and in how many places, still true, nothing found wrong, or left
  as it was — whether `checked` moved or `judged` was recorded, and, for
  a doc not given whole, what was read and not read; then the places
  withheld, left as they were, for a person.

## After the fixes pass

These findings are added, never refusing (the `count-off` a fix brought
right: [the run's findings](judge-applied.md#what-the-runs-findings-say)):

- **`example-unverified`**, **`example-checked`**: a code example a fix
  writes, unverified or checked by the project's command; a failing one
  is withheld ([code examples](examples.md)).
- **`value-left`** (step 2): a version a fix replaces is still said in the
  doc fixed, or in another doc declaring one of its source files that now
  says the new version. Reported with the lines; not fixed by the engine:
  an old version may be said on purpose.
  - A version: three parts, or as the project's `versions.pattern` writes
    one (a calendar `26.2`).
  - Replaced: said fewer times in the lines a fix adds than in those it
    removes, a new one written in its place.
  - The files of `versions.files` (a `pyproject.toml`, a `package.json`)
    are every doc's sources there.
  - Left out: history docs; a version given as a range (`>=1.4.1`,
    `^1.4.1`), what a dependant accepts; a version after a history marker
    ("First available in v1.9.0", "Added in", "since", "from", "after",
    "New in version", "Deprecated in", `versionadded::`), which says when
    something came, in prose or a code block's comment, or given as a
    minimum ("1.4.1 or later", "and newer"). Reported as pins:
    `<code>v1.4.1</code>`, `> v1.4.1`, and `since: v1.4.1` in code
    ([#569](https://github.com/JN0V/workline/issues/569),
    [#571](https://github.com/JN0V/workline/issues/571)). A release's bump
    holds a doc by the same reading ([the checks](checks.md#suspects),
    [#346](https://github.com/JN0V/workline/issues/346)).
