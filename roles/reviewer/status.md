# Reviewer — where it stands (2026-10-04)

**In one line:** it reviews code on a machine (`workline review`) and on a
merge request (opt-in): rules first, lenses as parts, quotes found again,
the change's findings told from the rest by where their cause lies, each
important one judged; built and in conformance, tried with a real agent on
a copy of workline and live on a sandbox pull request (tried.md),
released in v0.9.0.

## Built

| What | Proof |
|---|---|
| The rules on the comments a change adds: a bug's story, an internal code, a block too long; while they block, no agent | cases `bug-story-in-comment-blocks-without-ai`, `ticket-code-in-comment`, `long-comment-block-reported` |
| Only code reviewed; commits reviewed whole not asked again | `docs-only-asks-nobody`, `reviewed-commits-not-asked-again` |
| Lenses as parts, answering findings only; a patch refused | `never-approves-nor-patches`, `lens-failed-not-clean` |
| Quotes found again; the change's findings told from the rest by the cause | `finding-without-quote-dropped`, `finding-cause-in-diff-is-fixed-by-author`, `finding-cause-outside-diff-becomes-issue` |
| A judge for each important finding, the level said; a no drops it | `judged-no-not-reported`, `independence-level-in-verdict` |
| Warn until measured; caps; an agent unreachable | `new-ai-rules-warn`, `findings-capped-rest-counted`, `agent-unreachable-is-blocked-external` |
| `workline review`, findings as JSON; findings on one line merged | `local-review-outputs-json` |
| A quote found spaces and line breaks aside, a tab-indented one read | `quote-over-lines-found` |

Conformance: tests/conformance/cases/reviewer.

## Missing, in the order to build it

1. **Spec review**: a file or an issue as input; on the forge after the
   product owner refines, `ready` only with no important finding open, the
   loop bounded at five rounds, then a person.
2. **The developer's loop** (#117): a handoff to the developer, five rounds,
   then `human`; `judge-at-least: model` for the developer's merge requests,
   `context` for a person's, told apart by the author (#81).
3. **Measured** (#90): each lens's recall and false positives on planted
   defects, the finder floor's worth; then `ai-findings: block`.
4. **Inline comments** on the forge's own review (#81: the bot's identity).
5. **Every lens once when a merge request becomes ready**, from the
   templates (`--input lenses=all` exists; no template passes it yet).
6. **A closed issue's key**: only open issues are looked at, so a subject a
   person closed may be opened again.
7. **The tests lens outside the change**: a test missing for code the
   change did not touch was opened as an issue (tried.md); whether it
   should be is for #90's measure.
8. **Tried in CI with forge writes**: workline's own pull requests run it
   with forge writes on (the role's default) since v0.9.0 shipped the
   role to the apply job; the summary comment, the record and the turn of
   the lenses there are being tried (tried.md): comment, record and turn
   work; a push changing no code still asks a lens, given the earlier
   commits' files with an empty diff.
9. **The judge's material**: thirty lines around the cause; it refused a
   true finding whose evidence lay further (tried.md). The functions the
   cause calls or is called by, given whole, are the next step.
