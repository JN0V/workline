---
type: reference
sources: [internal/backlog/ledger.go, internal/builtin/productowner/productowner.go, internal/backlog/spec.go]
status: draft
checked: 9fcf6cf6
verified: agent:claude-code
---
# A person's answers

Part of [Acts on the backlog](../backlog-acts.md): what a person does on an issue, and what the engine reads of it.

At each run, acts or not, with or without an agent, the engine reads on
each open issue what a person did since the last (ADR-0038):

| A person | The engine |
|---|---|
| labels it `workline:accepted` (either spelling; GitLab Free keeps it beside `workline::proposed`) | each act its state proposes, as recorded, done whatever its mode or cap — but a closing only advised, a proposal past its rounds, or a slip with no milestone to go to, left to a person — a closing as obsolete closed at once, no announcement nor second judge —, said "Accepted by @x" (the label's last setter, from the forge's label events; "a person of the project" when it does not say); its drafts made theirs and the issue moved to ready when complete; `workline:proposed` taken off by the role |
| comments, being its reporter or of the project, on an issue waiting on them | "revise": the issue read first, the agent told so; a `refine` rewrites the sections still the role's own (`backlog.Own`: a draft, or a text as the role wrote it), never a person's; 👀 on each comment read (`React`; GitLab's "already taken" is done); one line in reply (`<!-- workline:product-owner/reply=<id> -->`); `revisions` counted — one a run, whatever the comments; at `acts.ask.rounds` less one, a new comment is left to a person (`left-to-a-person`), nothing written |
| takes `workline:proposed` off, without a yes | "not now": its proposals dropped, `aside` kept in its state with its body's digest, and what they wrote up to the label's removal heard with it (`heard`, from the forge's label events); nothing proposed nor done on it (`set-aside`) but the engine's upkeep — a `keep`, a closed blocker's link taken off, a slip moved — until its body changes or a counted comment comes; never a demotion |
| comments on an issue with nothing waiting on them — set aside, read and left, never read | read first, the agent told that what they say decides; no 👀, no reply. One set aside, its body as it was then, is said "Brought back by a comment from @x on <day>" (its body changed, it is back for that, said as any other comment), the comment quoted, never "set aside": the agent treats it as any issue at its level — completes, proposes, moves to ready — unless the comment says otherwise ("not now", "leave it", "I'll do it"). Set aside, it is so no more once the run decides an act on it; nothing decided, it stays so, said in the summary (`set-aside`) |
| edits its body | read again; a section edited is theirs; a section the role wrote and they deleted (`deleted`) is never written again |
| closes it | its proposals settled; never reopened |

**Who counts**: the reporter and the project's people — GitHub's author
association of owner, member or collaborator; GitLab's members from the
Planner role (ADR-0023) — never a bot, an author the forge does not name,
nor a bare "+1" (`backlog.Counted`). Which comment is the role's is told
by its marker, never by its author: the role may write with a person's
token. Only their comments bring an issue back into a run (`heard`); a stranger's is given to the agent as context
when the issue is read for another reason. An act whose quote is found
only in a comment of someone who does not count is proposed, never done
alone (`outsider-evidence`).

**Slowing down**: from `proposals-max` issues waiting on a person, a run
reads only the issues a person commented on (`proposals-waiting`).
Silence is never a yes. Nor does a run put more than `proposals-max`
issues on a person: the label goes on those it decided on first, then by
number, while fewer wait; the others keep their proposal, said
(`proposal-held`), labelled once fewer wait (`Plan.labelWithin`, #428).
Each issue labelled is named in the summary (`proposed`), whether this
run proposed on it or an earlier one, a draft or a migrated report's
record.
