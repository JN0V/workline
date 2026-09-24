# Documentalist

What runs, for humans. The AI never reads this file.

## Prepare (`pre`, no AI)

1. **Suspects.** For each doc, list commits since its `checked` commit that
   touched its `sources`. Follow the chain: a suspect technical section makes
   the product docs depending on it suspect too.
2. **Budgets.** Lines per doc, words per section (its own text, up to the next
   heading), card size (min and max: too small is fragmentation), lines per
   folder, lines of the agents' entry points at the root (`AGENTS.md`,
   `CLAUDE.md`).
3. **Duplicates.** Paragraphs of at least `min-words` words compared by their
   three-word sequences; `similarity` 0.85 catches a passage copied and then
   lightly edited.
4. **Links.** Every link to a file of the repository, and to a heading in it,
   must lead somewhere. Links to other sites are counted and said to be
   unchecked: that needs the network.
5. **Identifiers gone.** A name in a code span that was in the code when the
   doc was last edited, and is gone now (DOCER). A name never in the code is
   not reported: it may be a product term.

Not built yet: derived blocks between `workline:derive` markers, style (vale),
links to other sites (lychee), docs citing a superseded ADR, freshness.

| Finding | Level |
|---|---|
| `suspect`, `pending`, `unchecked`, budgets, `duplicate`, `dead-link`, `identifier-gone`, `links-not-checked`, `nothing-tracked` | reported; the run passes |
| `unknown` (a source that could not be read), `setting-missing` (a budget or threshold not set, so a check did not run) | blocks |
| a source repository that cannot be reached | `blocked-external` |

Findings are reported rather than blocking while they are new (warn before
block, docs/spec/role-contract.md); a project turns one off with `enforce`.

**Cascades are cut.** A change of code can make a technical doc suspect, which
makes a product doc suspect, and so on. Only the edges marked `now` are handled
inside the change. Every other suspect goes on a pending list — one tracking
issue, updated in place — with the moment it is due: the release gate refuses
to release while product docs due `at release` are still pending, and the
release manager hands them to the documentalist in one batch. Nothing is
forgotten; nothing drags a small fix into a rewrite of the user guide. (Built so
far: the pending findings. Not yet: the tracking issue and the release gate.)

**Code never rewrites the authority.** When the code disagrees with a doc marked
as the truth (a spec, an ADR, the architecture), the doc is not updated to
match: the documentalist opens an `issue` for the architect, because the code
may be the one that is wrong. (Not built yet: the `truth` setting is not read.)

Suspect docs become the agent's task, each with its lines numbered, what
changed in its sources, and the commits its `checked` must name: at most
`ai-max-calls` docs per run, and no more than fits the role's context budget.
Docs left out stay suspect for a person or a later run. Not built yet: stopping
when `max-open-merge-requests` are waiting, and the other kinds of task
(propagate, condense, duplicates, split).

## Judge (`post`, no AI)

A patch is refused, and the agent asked again with the reasons, when it:

- is not a unified diff, or git cannot apply it;
- touches a doc that was not put before the agent;
- quotes lines that are not at the numbers its hunks cite — git apply alone
  would find them elsewhere and apply anyway;
- changes lines between `workline:derive` markers;
- makes a doc's body longer (the frontmatter does not count);
- leaves `checked` short of the commits given, so the doc would stay suspect;
- brings a budget, link or duplicate problem the tree did not have, or makes
  one worse.

The judge reads a diff as its lines read, whatever counts its hunk headers
announce, and compares its reading with git's: the engine applies with
`git apply --recount`, so what was judged is what is applied.

## Without AI

All checks still run. Suspect docs are reported for a person, who clears each
one by updating `checked`. Not built yet: the checklist in the merge-request
comment.

## Tried for real

On 2026-09-24, with Claude (sonnet, then opus when asked again), on copies of
the `documented` fixture:

- tokens moved from one hour to two, three runs: the agent's answer was right
  each time, in one call; the first run lost the fix to the defect below, the
  two after it applied it;
- a function added, the doc still true: the first answer was refused for
  growing the doc's body; the second, one tier up, moved `checked` and nothing
  else, and opened an issue for a comment the new code contradicts;
- two defects found this way, now guarded: a hunk header counting fewer lines
  than it holds made git drop the fix itself (hence `--recount`), and a commit
  like `11180e1` was read by YAML as a number, so the doc passed for unchecked.

On workline's own repository, without AI: no false positive; one real finding
(docs/spec/role-contract.md is over its line budget).
