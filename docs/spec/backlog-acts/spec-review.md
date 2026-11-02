---
type: reference
sources: [internal/backlog/spec.go, internal/backlog/backlog.go, internal/routing/routing.go, internal/builtin/productowner/productowner.go]
status: draft
checked: a7467d4f
verified: agent:claude-code
---
# A spec read before ready

Part of [Acts on the backlog](../backlog-acts.md): the reviewer's reading of a refined issue, and the ready it holds.

When the project's line runs the reviewer after the product owner
(`schedule: [documentalist, product-owner, reviewer]`; `routing.Follows`),
the reviewer reads one refined issue a run and keeps what it found in one
comment on it, `<!-- workline:sticky=reviewer/spec -->`, hiding a record
(`backlog.SpecReview`, #128, ADR-0020):

```
<!-- workline:spec-review body=3f9a1c0e2b7d round=2 open=1 in=Verification -->
```

- the body's digest it read (`BodyDigest`; empty when the review was not
  whole), the round, the important findings open, the sections they lie
  in, and `stopped` once the rounds are spent.
- **A section is named by its heading** in the body, as the state's
  `wrote` and `deleted` are, read by meaning (ADR-0039): any heading,
  each escaped as a URL's query escapes a value (`in=Steps+to+reproduce`,
  a comma `%2C`). A heading of letters only is written as it always was,
  so a record written before reads the same.
- **A record that does not read** (#302): the issue passed over, ready
  held, said (`spec-record-unread`) until a person deletes the comment or
  accepts the issue.
- **`ready` is held** by the engine's check (`backlog.SpecHold`), after
  the four sections, whoever proposed it — the agent, or the engine
  itself — and at every level: `spec-not-reviewed` (no record, or
  one for another body, its rounds not spent: a refine in the same run changes it),
  `spec-findings-open`, `spec-rounds-spent`. A person's
  `workline:accepted` lifts it; `workline:ready` set by hand is theirs.
- **Read again**: an issue with findings open on its body as it is, a
  review its state has not `answered` — kept by the body that review
  read, never by its round, which restarts at 1 after a clean review —,
  is read again, the comment given to the agent (3000 characters at
  most) with the sections it may rewrite.
- **Answered**: a `refine` may rewrite a section with text when a finding
  open lies in it and the one rule says it may (`backlog.Rewritable`,
  [ADR-0060](../../adr/0060-the-person-is-on-the-loop.md)):
  - the builder's part, Verification and Scope, always — even a text a
    person wrote, at every level; "Changed: Verification" says it, the
    text before in the forge's edit history;
  - the person's part, Need, Validation, an example, a bug's steps, only
    while it is the role's own: a draft no person accepted, or a text
    whose digest is the one the state's `wrote` holds — recorded by every
    refine for what it wrote, and by a split for its children's four
    sections. A person's is kept (`section-kept`): the agent asks its
    reporter. Issues refined before #128 have no `wrote`: their sections
    read as a person's, but a Need or Validation still marked a draft,
    the role's.
- **Answered under the rule before**: the state's `revised` keeps the
  sections the role was given with a review (`answered`). A review it
  answered by asking a person for a section it may rewrite now
  (`backlog.Unrevised`) is read again once; the line counts refine's
  step due again for it, once a review (its record's `revise`).
- **Released**: once the record holds the body as it is, nothing open,
  not stopped, and the four sections pass, `pre` proposes `ready` itself
  (`own`), with no agent, under `acts.ready`'s mode and cap.
- **Off** (the reviewer not after it in a line): nothing changes.
