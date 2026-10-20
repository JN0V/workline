---
sources: [internal/builtin/documentalist/notes.go, internal/builtin/documentalist/values.go, internal/builtin/documentalist/judge.go, internal/builtin/documentalist/documentalist.go]
checked: 42ffe0bd
verified: agent:claude-code
---
# Documentalist — the judge

Part of [the documentalist](../README.md). `post` judges the agent's
answer with no AI, before anything is applied. Words used here:

- **vouched for**: `checked` moved to the commit the task gave — the doc
  read true against its sources;
- **`judged`**: a header field naming the commit a doc was judged at
  without being vouched for; the doc stays suspect for a person;
- **withheld**: one place of a fix refused while the rest is applied
  ([what holds is applied](#what-holds-is-applied));
- **DomoticsCore**: a C++ project the role runs on in CI, where many of
  these rules were found ([tried for real](tried.md));
- **steps 2 to 4**: the steps of
  [ADR-0014](../../../docs/adr/0014-checked-is-earned-by-what-was-read.md),
  `checked` earned by what was read.

## Refusals at a glance

A patch is refused, and the agent asked again with the reasons, when it:

- is not a unified diff, or one it cannot apply, or quotes lines that are
  not at the numbers its hunks cite
  ([hunks](#hunks-that-misquote-the-doc));
- touches a doc that was not put before the agent;
- replaces a version on a line that still says the old one elsewhere — a
  badge's text bumped, its link left (`replaced-in-part`);
- changes lines between `workline:derive` markers;
- makes a doc's body longer by more than a tenth of it (the header does
  not count): a fix may say what the code now does, never pad;
- moves `checked` when it may not
  ([moving `checked`](#moving-checked));
- takes words out of a doc's body with no claim saying why
  (`removal-uncited`, [claims](judge-claims.md));
- rests a change on a comment alone (`comment-not-evidence`,
  [comments](judge-claims.md#a-comment-is-not-evidence));
- brings a budget, link or duplicate problem the tree did not have, or
  makes one worse — but a size already over budget may grow by what a fix
  allows, and stays reported, for condensing: truth before size.

Places marked *withheld* below are not refused with the whole answer: the
rest of the fix is applied without them. Condense, split, merge-card and
dedupe tasks are judged by their own rules ([gardening](gardening.md#condensing)),
the reader task by [its own](reader.md#what-the-engine-checks).

## Hunks that misquote the doc

- **Not a diff git can apply**: the refusal says which line a hunk quotes
  wrong, a blank line skipped or one that is not there, where git says
  only the hunk's line (DomoticsCore, step 4).
- **Mended first**, where its place is beyond doubt, the doc's own lines
  put back as its context. Sonnet skipped a blank line, and quoted a
  context line the doc has not, and the right fixes those hunks carried
  were lost (DomoticsCore, step 4).
  - A hunk whose context differs from the doc by blank lines alone is
    placed where its other lines are, when they fit the doc at one place
    only — a blank line let go in the context, never between two lines it
    removes.
  - Else each run of the lines it changes is placed by the lines it
    removes alone, found once in the doc, the context dropped. A run that
    only adds, or removes only blank lines, cannot be.
- ***Withheld*** when some hunks quote the doc right and the others cannot
  be mended: those are left out. With none quoting right, the agent is
  asked again — a misquote is a slip it mended at the second answer, 3 of
  4 times on DomoticsCore.
- **Lines not at the numbers cited**: refused, since git apply alone would
  find them elsewhere and apply anyway. A hunk a few lines off (3 at most)
  whose quoted lines the doc holds only there is placed there: agents
  miscount lines in long docs, not their content.
- **Read as its lines read**, whatever counts its hunk headers announce,
  and compared with git's reading. The engine applies with `git apply
  --recount --unidiff-zero`, and checks with it, so what was judged is
  what is applied: a last hunk with no context after it is not taken for
  the end of the doc.

## Moving `checked`

- **To another commit than the one given**: refused. A fix that leaves
  `checked` alone is kept, the doc fixed but not vouched for
  ([ADR-0012](../../../docs/adr/0012-a-doc-is-judged-against-its-sources-whole.md)),
  if it sets `judged` to that commit (`judged-not-set`): the doc is not
  put before an agent again until a source changes
  ([ADR-0013](../../../docs/adr/0013-what-was-judged-is-not-asked-again.md)).
- **Sources not all given whole** (`checked-unread`,
  [ADR-0014](../../../docs/adr/0014-checked-is-earned-by-what-was-read.md)):
  judged on diffs, or beside a doc of the same task, nobody read the rest.
  `pre` records the docs whose sources it gave whole (`in/read-whole.yaml`;
  none, when the file is missing), and what it gave of the others
  (`in/read-not-whole.yaml`). *Withheld*: the move of `checked`.
- **A line count still off** (`checked-over-count-off`, step 2): the task
  gave the real count; the fix brings it there, with no claim, or leaves
  `checked`. *Withheld*: the move of `checked`.
- **A doc judged in parts** (`checked-moved-in-parts`): the fix saw only
  the parts' excerpts; refused. Only the engine moves it, when the parts
  support every line stating something and `parts-move-checked` is on, as
  by default ([judged in parts](parts.md#when-checked-moves-after-parts)).
- **A doc the answer's own note doubts** (`checked-over-note`,
  [#449](https://github.com/JN0V/workline/issues/449)): a note names its
  doc in `doc`, else in its text, by path or by a file name no other doc
  of the task shares. *Withheld*: the move of `checked`, the note quoted.

## What holds is applied

A refusal used to send the whole task again — two docs, 35k tokens — for
one place the engine had already named (step 4). Now:

- a place withheld is not asked for again: the doc's fix is applied
  without it, the place left as it was;
- the doc keeps `checked` where it was and records `judged` (its header
  put back, `judged` set), so a person reads it before moving `checked`;
  it is not put before an agent again until a source changes
  ([ADR-0013](../../../docs/adr/0013-what-was-judged-is-not-asked-again.md));
- each place withheld is a finding for a person — the refusal's rule, the
  doc and its lines, and why — in the run's verdict, the merge request's
  comment or the gardening pull request, as other findings; the doc's
  `suspect` finding says it was fixed in part;
- with every place of a doc withheld, the fix is only its record: `judged`
  set, the places reported;
- `post` leaves the narrowed patches in `out/intentions.yaml`, and the
  engine applies those ([role contract](../../../docs/spec/role-contract.md)).

## What the finding says

- A doc judged and not vouched for: what the run did, counted in its body
  — "fixed in 2 places", a code example apart ("and 1 code example"), or
  "nothing found wrong" when the patch only records `judged`. Never
  "fixed" for a header alone: workline's first nightly, 2026-10-02, said
  "what the agent found wrong is fixed" of a doc it found true.
- A doc whose sources did not all go whole: "nothing found wrong in what
  was read", then `read:` what the task gave (what changed in a source
  since `checked`, the doc it follows) and `not read:` the rest. Never
  "nothing found wrong" of what was not read
  ([#120](https://github.com/JN0V/workline/issues/120)). Gardening reads
  such a doc [in parts](parts.md) instead, when parts
  can ([#462](https://github.com/JN0V/workline/issues/462)).
- A doc judged in parts, once the fix is judged: the places it fixed, or
  that it changed nothing of what the parts found.
- On a person's merge request, a doc whose patch only records `judged`:
  its record is kept in the documentalist's comment there, nothing
  committed for it, and the finding says so
  ([more](push.md#a-doc-found-with-nothing-to-fix-no-commit)).
- The gardening pull request (gardening: the scheduled run, nightly or
  weekly) lists each doc it judged with the same outcome — fixed and in
  how many places, still true, or nothing found wrong — whether `checked`
  moved or `judged` was recorded, and, for a doc not given whole, what was
  read and not read.

## Asked again

- For any other refusal, the answer asked again replaces the refused one
  whole: the agent gives again, unchanged, every change not refused.
- Each answer is kept in the run folder as it came (`out/agent-answer.txt`;
  a refused one as `out/refused-<n>-answer.txt`), and the claims of the
  accepted one in `out/claims.yaml`: judged, never applied, kept so that a
  fix's evidence can be checked afterwards.
- Asked again as often as the role allows and still refused, a doc's fix
  is left out (`left-out`, saying why); the other docs' fixes are judged
  again without it and applied. The doc stays as it was, suspect.
- A block the answer did not cause — docs due, a release held, a check
  that could not run — is never asked again: another answer would change
  none of it. A release held only for docs due this run did not settle
  applies the answer's fixes all the same
  ([#543](https://github.com/JN0V/workline/issues/543)).

## After the fixes pass

These findings are added, never refusing:

- **`count-off`** a fix brought right says "fixed in this run".
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
    `^1.4.1`), what a dependant accepts; a version after a
    history marker ("First available in v1.9.0", "Added in", "since",
    "from", "after", "New in version", "Deprecated in", `versionadded::`),
    which says when something came, or given as a minimum ("1.4.1 or
    later", "and newer"). The same reading decides whether a release's
    bump holds a doc ([the checks](checks.md#suspects),
    [#346](https://github.com/JN0V/workline/issues/346)).
