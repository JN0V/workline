---
type: reference
sources: [internal/backlog/backlog.go, internal/backlog/obsolete.go, internal/builtin/productowner/productowner.go, internal/engine/backlog.go]
status: draft
checked: fac02675
verified: agent:claude-code
---
# Closing

Part of [Acts on the backlog](../backlog-acts.md): closing an issue as a duplicate, or as obsolete.

```yaml
- close:
    issue: 12
    reason: duplicate          # duplicate | obsolete
    duplicate-of: 7            # a duplicate's original
    quote: {issue: 7, text: "the export drops the last row"}
    # or, for obsolete: {path: src/export/csv.go, text: "for i := 0; i <= len(rows); i++"}
    why: "#7 reports the same row dropped by WriteRows."
```

- **Never not planned.** Refusing a need is the person's (principle 1): any
  other reason is dropped (`close-reason`), the issue left open.
- **No quote, no act.** The quote must be found again, as written but for
  spaces, in the file at the commit the run is on, or in the issue's title,
  body or one of its comments. A quote missing or not found drops the act
  (`no-quote`).
- A duplicate is closed with the forge's own reason (GitHub's `duplicate`,
  linking the original), with a comment that quotes the evidence and says
  how to undo: reopen it.

## Obsolete: announced, then closed

An obsolete issue is not closed by the agent's act (ADR-0024). In mode
`act`, the act **announces** it — unless the issue bears an exempt label
(`exempt`; `acts.close-obsolete.exempt`, `pinned` and `security` by
default), is announced already (`announced`), or was kept open on the same
quote (`obsolete-kept`, its file and a digest of its text, in the state's
`kept`):

- the label `workline:obsolete` (created when the project has none), then
  a comment naming its reporter: why, the code quoted, the commit, the day
  from which it may be closed (`days` later, 7 by default), and how to keep
  it open; it ends with a block the engine reads back — the quote, why,
  the commit, the day announced, the model that proposed it — and the
  marker `<!-- workline:product-owner/obsolete -->`.

At each later run, `pre` reads the last announcement of each issue not
settled (its quote not in `kept`), with no agent:

| Found | What the engine does |
|---|---|
| A comment after it, not the engine's nor a bot's — whoever wrote it | `keep`: the label off, the quote added to `kept`; nothing written to them |
| The label taken off, or an exempt label set | `keep`, the same |
| The code quoted no longer in the commit | `keep`, and the issue told why |
| Its day not come | nothing |
| Due | a second judge asked (`in/judge/obsolete-<n>/`), apart, at the best independence from the model that announced it (ADR-0005); yes: the engine's `close`, checked again — the announcement there, due, unanswered, the same quote, the judge's yes — then closed as completed (GitLab, the local forge: closed), the label off, a comment with the evidence, the day announced and the judge with its level; no: `keep`, the judge's reason said on the issue |
| Due, no agent or a judge that did not answer | nothing: it waits (`judge-unavailable`) |

Closing as completed, never as not planned: the code did the work, and
not planned is a person's no, which every role's opening reads as such.
`keep` is never held back by a mode nor a cap; the agent may propose it
too, on an issue announced. In `propose`, the agent's act and the
engine's closing are proposed on the issue for a person, nothing is
announced; accepted, it is closed at once.

**Off** (the default since ADR-0038), a closing the agent writes is not
dropped in silence: it is **said** on the issue for a person to close
(`advice` in its state), its evidence quoted — "Close it yourself if you
agree: what it asks looks done already. … The role does not close it" —
with the label `workline:proposed`. Nothing is announced or closed; the
label `workline:accepted` does not close it; the comment asks the person
to close the issue, or to take the label off to keep it. The task tells
the agent so (ADR-0038, amended 2026-10-09). The same holds for
`close-duplicate` turned off.
