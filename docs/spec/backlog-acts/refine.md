---
type: reference
sources: [internal/backlog/backlog.go, internal/backlog/plain.go, internal/backlog/write.go, internal/work/headings.go, internal/work/work.go, internal/forge/forge.go, internal/engine/backlog.go]
status: draft
checked: 09c0c9ae
verified: agent:claude-code
---
# Refining to ready

Part of [Acts on the backlog](../backlog-acts.md): writing the sections an issue lacks, moving it to ready and back, and naming its code.

An issue is `ready` when four sections are written in its body — `## Need`,
`## Verification`, `## Validation`, `## Scope` (docs/spec/routing.md) — and
its Need and Validation are a person's (ADR-0018). The product owner writes
what it can and asks for the rest:

```yaml
- refine:
    issue: 12
    scope: "WriteRows in src/export/csv.go, and its test."
    sources: [src/export/csv.go]     # the files the scope names, in the commit
    verification: "A test writes three rows and reads three lines back."
    need: "Every row exported, so a report counts what was sold."   # a draft
    example: "Ann makes 3 sales on 8 Oct and exports that day: 2 rows."   # a draft, never required; a bug: steps
    validation: "The maintainer opens an export of a known day."    # a draft
    description: "Exporting a day drops its last sale: 3 sales, 2 rows."   # a role's text on top, rewritten
    why: "The issue says what is wrong, not how it will be proved."
- ready: {issue: 12, why: "Its four sections are there, Need and Validation the reporter's."}
- ask: {issue: 14, questions: "Which export: CSV or JSON? What should a blank row become?", why: "The need is not clear."}
```

- **`refine`** adds the sections the body does not have, after its text,
  which stays as it is; one there but empty — an issue form's field left
  `_No response_` — is filled in place (an issue form writes its fields as
  `### ` headings, read as `## ` ones). A section with text — a person's,
  or one the role wrote before — is never rewritten (`section-kept`), but
  the role's own to answer the reviewer or a person's comment
  ([A spec read before ready](spec-review.md), [A person's answers](answers.md)).
  **In the reader's order** (the table's, `work.Headings`): the person's part — Need,
  an `example` (`## Example`) or a bug's `steps` (`## Steps to
  reproduce`), Validation — above a thematic break `---`; the builder's —
  Verification, Scope — below it. A section added goes after the nearest
  one before it in that order, else before the nearest one after it, at
  its heading's level, never moving a person's; the break is added when
  every section of the person's part comes before every one of the
  builder's and none is between them; a break alone after the person's
  part, as a footer, is that break. The example and the steps are never
  required. Need, Validation, the example and the steps are drafts: each
  follows a hidden marker alone (`<!-- workline:draft -->`), and the issue gets the label
  `workline:draft`. **A person accepts them with one label,
  `workline:accepted`** — on the issue, or on many at once from the list of
  issues; the engine creates that label with the first draft, so it is
  there to pick; only who may triage sets a label, so it is a person of the
  project's. Its description (`backlog.AcceptedSays`) names its other use:
  on a pull request, a yes to merge it, by the
  [merge policy](../../merge-policy.md), which creates it too. The engine then takes the draft marks out and moves the issue
  to ready, with no agent. **Editing a draft makes it a person's too**:
  its text no longer the one the role wrote (the state's `wrote`, a
  digest per section), the mark left in it or not (`backlog.IsDraft`). A
  draft an earlier engine wrote — a visible line, "*Draft by the product
  owner: …*", around the marker — with no digest kept is told by its
  marker alone; the role now writes the marker alone. Scope names its files or folders (`sources`, 1 to 5, in
  the commit, as for naming its sources), which the issue's state gets when
  it has none; given none, the engine takes the paths the Scope's text
  names that the commit holds (`backlog.PathsIn`). A Scope naming none, or
  a path the commit does not hold, is left out, the rest of the refine
  written (`scope-without-files`, said in the summary with what a person
  may do). Nothing to add drops the act (`nothing-to-refine`). The
  issue gets the label `workline:to-refine`.
- **A role's text on top, in plain words** (ADR-0038, amended). The
  text above an issue's sections — its description — a role opened it
  with and no person edited is rewritten when the agent gives a
  `description`: an import's item, another role's finding. The engine
  tells it is the role's, never the agent (`backlog.Rewrite`): the issue
  holds an import's key or a finding's `opened-by`, and its description,
  spaces aside, matches the digest the state kept at opening (`wrote:
  {Description: …}`); opened before that digest was kept, an import's key
  holds one, and a finding opened by a bot (a GitHub app, a GitLab bot)
  counts as unedited while its body is unchanged since the role last read
  it. A description the role rewrote (`plain`) is again only to answer a
  person's comment on its proposal; edited since, it is theirs. The
  engine's lines there — where it comes from ("Opened from …", "Found by
  …"), its hidden markers — are kept after the new text, in their order;
  the text before is in the forge's edit history. Over 100 words outside
  its folds and fenced code, given on an issue a person opened or edited, or in a
  text proposed to an outsider, it is left out, the rest of the refine
  kept (`description-kept`, said). It follows the refine's mode and
  level as the drafts do: at `cautious`, proposed. The issue's one line
  says "Rewrote: the description, in plain words."
- **What it writes is readable**, where the engine can tell
  (`backlog.LinkDecisions`, `backlog.MarkClosed`): a decision cited bare,
  `ADR-0038`, becomes a link to its page on the forge, when the commit
  holds it (a numbered page in a folder named `adr`, `adrs` or
  `decisions`); an issue cited that is closed is followed by "(closed)",
  once. Neither touches code — a code span, a fenced block. The task says, for each issue read, the issues it cites, open or
  closed and why (`Cites:`), and how a file of the repository is linked
  (`Links:`, the forge's file pages at `HEAD`).
- **An issue read and left with nothing** — no draft, no question, no
  act dropped with its reason — though it lacks a section is said in the
  summary (`nothing-proposed`), never left in silence.
- **`ready`** is checked by the engine, not taken from the agent: the four
  sections there and not empty, no draft of the role's left in Need or
  Validation unless the issue bears `workline:accepted` — read on the
  issue, never in a proposal (`not-ready` otherwise, naming what is
  missing); a mark left in a section a person edited is taken out. Done, the issue gets
  `workline:ready` and loses `workline:to-refine`, as `workline item ready`
  does, and loses `workline:draft` and `workline:accepted`. An issue opened
  by someone without write access to the project is theirs: moving it to
  ready is proposed on the issue, never done (`reporter-outside`) —
  unless a person of the project accepted it with the label. Who is of the
  project: on GitHub, the issue's author association (owner, member,
  collaborator); on GitLab, a member with the Planner role or above, read
  from the project's members (ADR-0023) — a token that may not list them
  fails the run, loud; on a plugged forge, its `insider`. One the forge
  does not say is outside.
- **`unready`** moves a ready issue back to refine: `workline:ready`
  off, `workline:to-refine` on, a comment telling the issue why and who
  accepted it. It is **always proposed**, whatever the level or the
  settings — a ready issue is moved back by a person only (ADR-0032) —,
  done when a person of the project accepts it (`workline:accepted`); one not `workline:ready`
  is dropped (`unready-not-ready`). No quote; the state must read.

  ```yaml
  - unready: {issue: 13, why: "#12 now asks for JSON: its CSV row count no longer proves it."}
  ```

- **`ask`** comments on the issue, naming its reporter, with the agent's
  questions. The answer is a person's comment: the issue is read again at
  the next run.
- An issue whose body a person changed since it was read — a draft
  accepted, a section written — is read again (its state's `body`, a
  digest of the body when it was last read or written by the engine).

## A body's footer

A body's footer belongs to no section (`work.Footer`): what a tool or a
person appends after the last section is not the Scope's text. It is,
outside fenced code (CommonMark's fences: three or more backticks or
tildes open one, a backtick fence's info string holding no backtick; a
line of the same character alone, as long or longer, closes it):

- from a **thematic break** (`---`, `***`, `___`, CommonMark's, a blank
  line above it, so not a heading's underline) after the last heading, to
  the end — one with headings after it, as between the person's part and
  the builder's, is no footer, and is not the text of the section it
  ends (`work.Sections`): a draft's digest is the same with it or without;
- or else the body's **last line, when it is a tool's signature**: alone
  in its paragraph, reading "Generated with …" or "Generated by …", after
  an emoji or a mark if any — `🤖 Generated with [a tool](https://…)`.

Every reading of the sections leaves it out — `ready`'s check, the
state's `sections`, `wrote` and `deleted`, a finding's section —, and a
section `refine` adds, or a parent's `## Sub-issues`, goes above it,
the footer left last as it was.

## Naming its sources

```yaml
- sources:
    issue: 12
    sources: [src/export/csv.go]
    quote: {path: src/export/csv.go, text: "func WriteRows(rows []string, write func(string)) {"}
    why: "#12's WriteRows is defined here."
```

The files, or folders, must be in the commit the run is on (1 to 5;
`sources-unknown` otherwise), and a quote found in one of the files named,
an anchor after `#` aside, or in a file inside a folder named (`no-quote`
otherwise). A
folder is given at the next run as the list of its files. Done, the issue's
state gets them and loses `judged`: it is read again, with that code, at the
next run.
