---
sources:
  - roles/documentalist/code/judge.go#parseDiff
  - roles/documentalist/code/judge.go#misquoted
  - roles/documentalist/code/judge.go#notApplying
  - roles/documentalist/code/judge.go#placeWithin
  - roles/documentalist/code/judge.go#placed
  - roles/documentalist/code/judge.go#mended
  - roles/documentalist/code/judge.go#blanksLetGo
  - roles/documentalist/code/judge.go#quotedRight
  - roles/documentalist/code/judge.go#writtenAnew
  - roles/documentalist/code/judge.go#placedPatch
  - roles/documentalist/code/judge.go#gitApplied
  - internal/agent/prompt.go#Prompt
  - internal/engine/engine.go#leaveOut
checked: e81df7b34
verified: agent:claude-code
---
# Documentalist — hunks, and asking again

Part of [the check](judge.md).

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

## Asked again

- For any other refusal, the answer asked again replaces the refused one
  whole: the agent gives again, unchanged, every change not refused.
- Each answer is kept in the run folder as it came (`out/agent-answer.txt`;
  a refused one as `out/refused-<n>-answer.txt`), and the claims of the
  accepted one in `out/claims.yaml`: judged, never applied, kept so that a
  fix's evidence can be checked afterwards
  ([runs](../../../docs/config.md#runs)).
- Asked again as often as the role allows and still refused, a doc's fix
  is left out (`left-out`, saying why); the other docs' fixes are judged
  again without it and applied. The doc stays as it was, suspect
  ([role contract](../../../docs/spec/role-contract.md#one-run)).
- A block the answer did not cause — a check that could not run — is
  never asked again: another answer would change none of it
  ([verdict](../../../docs/spec/role-outcome.md#verdict)).
