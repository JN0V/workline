---
type: reference
sources: [internal/backlog/backlog.go#hidden, internal/backlog/backlog.go#drafts, internal/backlog/backlog.go#theirs, internal/backlog/backlog.go#moves, internal/backlog/backlog.go#PathsIn, internal/backlog/backlog.go#builder, internal/backlog/backlog.go#named, internal/backlog/backlog.go#IsDraft, internal/backlog/backlog.go#AcceptedSays, internal/backlog/backlog.go#bullet, internal/backlog/backlog.go#questions, internal/backlog/backlog.go#question, internal/backlog/backlog.go#holds, internal/backlog/assume.go, internal/backlog/proof.go, internal/backlog/plain.go, internal/backlog/repeat.go, internal/backlog/write.go, internal/work/headings.go, internal/work/work.go, internal/work/scenarios.go, internal/work/scope.go, internal/backlog/scenarios.go, internal/backlog/verification.go, internal/engine/backlog.go#report]
status: draft
checked: a19e8cb0
verified: agent:claude-code
---
# Refining to ready — what a refine keeps and writes

Part of [Refining to ready](refine.md): what a refine reads before ready, how it says each thing once, and what it writes.

- **An answer read before ready**
  ([#1331](https://github.com/JN0V/workline/issues/1331)): the agent's
  `ready` on an issue whose run read a person's comment, its body as the
  role last read it, is held once (`answer-unwritten`) unless the run
  drafts a section of the spec on it (not the Verification alone) that
  is not dropped before the ready. The issue stays in
  refine, those comments unheard again (`heard`), the hold kept in the
  state (`answer-held`, their count). The next run quotes them again
  and says the ready was held; its `ready` is taken. Never held: the
  engine's own ready, a person's yes (`workline:accepted`).
- **Each thing said once** (ADR-0039, amended). The text above an
  issue's sections — its description — is read by whose it is, the
  engine telling so, never the agent (`backlog.RolesTop`): a role's when
  the issue holds an import's key or a finding's `opened-by`, and its
  description, spaces aside, matches the digest the state kept at
  opening (`wrote: {Description: …}`), an import's key, or an earlier
  engine's rewrite in plain words (`plain`); opened before that digest
  was kept, a finding opened by a bot (a GitHub app, a GitLab bot)
  counts as unedited while its body is unchanged since the role last
  read it. Otherwise it is a person's.
  - **A role's text on top becomes the Need draft**: when the refine
    writes a `need` — added, filled, or rewritten to answer a comment —,
    that text leaves the top; the engine's lines there — where it comes
    from ("Opened from …", "Found by …"), its hidden markers — stay, in
    their order, and the text before is in the forge's edit history.
    A block the role folded there (`<details>` … `</details>`, a cause
    and its fix) moves whole to the end of the last builder's section
    the refine writes (the Scope), or stays on top when
    it writes none ([#841](https://github.com/JN0V/workline/issues/841)).
    Never on an outsider's issue proposed in a comment. It follows the
    refine's mode and level as the drafts do: at `cautious`, proposed.
    The issue's one line says "Rewrote: the text on top as the Need, in
    plain words."
  - **A person's text on top, with no Need section, is the Need**, a
    person's: `ready` reads it so, and the task says it. A Need draft is
    written, and read instead, only when it does not say that text again.
  - **A draft that repeats is not written** (`backlog.Repeated`):
    containment, the share of its words already said above it, over
    one-word shingles, common words out. Compared: a Need draft with a
    person's text on top; an example or a bug's steps with all above
    them as the issue will read — the text on top, unless taken into the
    Need, and the Need. Never Validation, which restates the case by
    design. Past `repeated-percent-max` (40 by default, 1 to 100), the
    draft is left out, the rest of the refine kept (`repeated`, said),
    and the issue's line says "Not written: the Example, which said
    again what is above it." A person's text is only read; what a
    person said yes to — a proposal accepted with the label, a text the
    reporter agreed to — is written as they read it, measured when it
    was proposed, the top taken only if it said so.
  - A `description` given apart, as an earlier engine asked, is left out
    (`description-kept`, said).
- **What it writes is readable**, where the engine can tell
  (`backlog.LinkDecisions`, `backlog.MarkClosed`): a decision cited bare,
  `ADR-0038`, becomes a link to its page on the forge, when the commit
  holds it (a numbered page in a folder named `adr`, `adrs` or
  `decisions`); an issue cited that is closed is followed by "(closed)",
  once. Neither touches code — a code span, a fenced block. The task says, for each issue read, the issues it cites, open or
  closed and why (`Cites:`), and how a file of the repository is linked
  (`Links:`, the forge's file pages at `HEAD`).
- **An issue read and left with nothing that refines it** — no draft,
  no question, no sources nor closing; a kind, an order, a link or a
  title alone count for nothing — though it lacks a section is said in the
  summary (`nothing-proposed`) and on the issue, in the role's comment
  ([ADR-0059](../../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md), decision 1):
  the sections it lacks, each act of the agent's dropped on it with why,
  and the agent's note on it (`silent` in its state), kept until the
  role acts on it. Its next read tells the agent so: refine it now, or
  ask. The line wakes it again in refine
  ([ADR-0059, amended](../../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-03-an-issue-left-unrefined-is-woken-again)).
  The task lists the repository's
  folders for an issue that names no file: the agent names its sources
  from them, no line quoted. A `sources` naming only code the run did
  not give — no path in or holding one its state, title or body names
  (`unshown`) — needs no line quoted either
  ([#980](https://github.com/JN0V/workline/issues/980)). One mixing code
  given and not given, no line quoted, names the code not given and
  keeps the sources its state names; code given new to it is left out,
  said (`sources-unquoted`, info)
  ([#1151](https://github.com/JN0V/workline/issues/1151)).
