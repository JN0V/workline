---
sources:
  - internal/builtin/documentalist/judge.go#judgePatches
  - internal/builtin/documentalist/judge.go#replacedInPart
  - internal/builtin/documentalist/documentalist.go#suspectDoc
  - internal/builtin/documentalist/notes.go#doubtedDocs
  - internal/builtin/documentalist/notes.go#doubtSaid
checked: e2e165c2d
verified: agent:claude-code
---
# Documentalist — the check

Part of [the documentalist](../README.md). `post` checks the agent's
answer with no AI, before anything is applied; the judge is the engine's
service it may consult. Words used here:

- **vouched for**: `checked` moved to the commit the task gave — the doc
  read true against its sources;
- **`judged`**: a header field naming the commit a doc was judged at
  without being vouched for; the doc stays suspect for a person;
- **withheld**: one place of a fix refused while the rest is applied
  ([what holds is applied](judge-applied.md#what-holds-is-applied));
- **DomoticsCore**: a C++ project the role runs on in CI, where many of
  these rules were found ([tried for real](tried.md));
- **steps 2 to 4**: the steps of
  [ADR-0014](../../../docs/adr/0014-checked-is-earned-by-what-was-read.md),
  `checked` earned by what was read.

The rest of the judge:

- [hunks that misquote the doc, and asking again](judge-hunks.md);
- [what holds is applied](judge-applied.md), and what the run's findings
  say of it;
- [what the finding says](judge-findings.md), and the findings added once
  the fixes pass;
- [the claims](judge-claims.md) behind a word taken out.

## Refusals at a glance

A patch is refused, and the agent asked again with the reasons, when it:

- is not a unified diff, or one it cannot apply, or quotes lines that are
  not at the numbers its hunks cite
  ([hunks](judge-hunks.md#hunks-that-misquote-the-doc));
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

Places marked *withheld*, here and in [hunks](judge-hunks.md), are not
refused with the whole answer: the rest of the fix is applied without
them. Condense, split, merge-card and dedupe tasks are judged by their own
rules ([gardening](gardening-condense.md#condensing)), the reader task by
[its own](reader.md#what-the-engine-checks).

## Moving `checked`

- **To another commit than the one given**: refused. A fix that leaves
  `checked` alone is kept, the doc fixed but not vouched for
  ([ADR-0012](../../../docs/adr/0012-a-doc-is-judged-against-its-sources-whole.md)),
  if it sets `judged` to that commit (`judged-not-set`): the doc is not
  put before an agent again until a source changes
  ([ADR-0013](../../../docs/adr/0013-what-was-judged-is-not-asked-again.md),
  [what holds](judge-applied.md#what-holds-is-applied)).
- **Sources not all given whole** (`checked-unread`,
  [ADR-0014](../../../docs/adr/0014-checked-is-earned-by-what-was-read.md)):
  judged on diffs, or beside a doc of the same task, nobody read the rest.
  `pre` records which docs had their sources given whole
  ([how](judge-applied.md#what-the-runs-findings-say)). *Withheld*: the
  move of `checked`.
- **A line count still off** (`checked-over-count-off`, step 2): the task
  gave the real count; the fix brings it there, with no claim, or leaves
  `checked`. *Withheld*: the move of `checked`.
- **A doc judged in parts** (`checked-moved-in-parts`): the fix saw only
  the parts' excerpts; refused. Only the engine moves it, when the parts
  support every line stating something
  ([judged in parts](parts-fix.md#when-checked-moves-after-parts)).
- **A doc the answer's own note doubts** (`checked-over-note`,
  [#449](https://github.com/JN0V/workline/issues/449)): a note names its
  doc in `doc`, else in its text, by path or by a file name no other doc
  of the task shares. *Withheld*: the move of `checked`, the note quoted.
