---
type: reference
sources: [internal/backlog/opening.go, internal/engine/engine.go, internal/engine/backlog.go]
status: draft
checked: 6dc52a5b
judged: 8b6aecc
verified: agent:claude-code
---
# Opening issues

Part of [Acts on the backlog](../backlog-acts.md): the one way every role opens an issue.

Every role opens an issue through one way (ADR-0018, `backlog.Openings`),
with no AI:

| Step | What the engine does |
|---|---|
| Key | A finding on code quotes its line (`at: {path, text}`); found again in the file, the key is the file and that line, spaces aside (`issue=<path>#<8 hex>`). Otherwise, or the line not found (`issue-quote-not-found`, said), its title (`issue=title#<8 hex>`). A text a backlog file is imported from keys its own (`import=<path>:<hex>`). Hidden in the body. |
| Looked for | In every issue, open and closed, read once a run (`AllIssues`); the key a role wrote before the way was shared (`issue=<role>/<key>`) counts too. |
| Open, holding it | Left as it is: nothing written, whichever role opened it. |
| Closed as not planned or duplicate; for an import's text or a split's child, closed at all | A person's no, or a text settled: nothing written (`issue-closed`, said in the run). |
| A finding's, closed otherwise | Done, or no reason kept (GitLab, the local forge): one comment, once, "found again" with the commit; left closed (`issue-closed`). |
| New, past `issues-max` | Not opened, counted (`issues-capped`); found again, opened at a later run. Three by default; the product owner's import has its own cap (`open`). |
| New | Opened: the role named in a line — an import's hidden, `<!-- workline:opened from `ROADMAP.md`, lines 7 to 8 by the product-owner role -->`; a finding's, where the role found it when it says (`found`): "Found by the reviewer role while reviewing ccffcb2, outside that change."; for a finding, `<!-- workline:opened-by=<role> -->` and `needs-triage`; the product owner's state, written last, keeping a digest of the text it was opened with (`wrote: {Description: …}`) — a run stopped before it, resumed, finds the issue open and writes what it lacks. |
