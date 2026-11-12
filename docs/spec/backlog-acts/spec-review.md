---
type: reference
sources: [internal/backlog/spec.go, internal/backlog/backlog.go, internal/routing/routing.go, internal/builtin/productowner/productowner.go]
status: draft
checked: 8538d558
verified: agent:claude-code
---
# A spec read before ready

Part of [Acts on the backlog](../backlog-acts.md): the reviewer's reading of a refined issue, and the ready it holds.

When the project's line runs the reviewer after the product owner on
any event (`schedule: [documentalist, product-owner, reviewer]`, or
`issue: [product-owner, reviewer, …]`; `routing.Follows`), the reviewer
reads refined issues and keeps what it found in one comment on each, `<!-- workline:sticky=reviewer/spec -->`, hiding a record
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
  the Need, Validation and Scope, whoever proposed it — the agent, or the engine
  itself — and at every level: `spec-not-reviewed` (no record, or
  one for another body, its rounds not spent: a refine in the same run changes it),
  `spec-findings-open`, `spec-rounds-spent`. Findings in the
  Verification alone hold nothing: the tester's
  ([ADR-0066](../../adr/0066-a-role-mirrors-a-human-role-ai-first-where-code-is-rigid.md)). A person's
  `workline:accepted` lifts it; `workline:ready` set by hand is theirs.
- **Read again**: an issue with findings open on its body as it is, a
  review its state has not `answered` — kept by the body that review
  read, never by its round, which restarts at 1 after a clean review —,
  is read again, the comment given to the agent (3000 characters at
  most) with the sections it may rewrite.
- **Answered**: a `refine` may rewrite a section with text when a finding
  open lies in it and the one rule says it may (`backlog.Rewritable`,
  [ADR-0060](../../adr/0060-the-person-is-on-the-loop.md)):
  - the builder's part, the Scope, always — even a text a person
    wrote, at every level; "Changed: Scope" says it, the text before in
    the forge's edit history;
  - never the Verification, the tester's: a finding there is given to
    the tester when it writes the tests (`backlog.SpecForTester`);
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
  not stopped, and the Need, Validation and Scope pass, `pre` proposes `ready` itself
  (`own`), with no agent, under `acts.ready`'s mode and cap.
  Never on an issue the product owner left open itself
  ([#1395](https://github.com/JN0V/workline/issues/1395);
  `backlog.LeftOpen`, said `ready-left-open`): its question to a person
  unanswered, no assumption taken for it
  ([ADR-0060](../../adr/0060-the-person-is-on-the-loop.md));
  or moved back to refine on the body as it is (its state's last `did`
  of `ready` or `unready` an `unready` of that body's digest; `did`
  keeps 35 days of acts). A
  person's comment since is read by the agent first, whose own `ready`
  is taken; the body changed and reviewed again, the engine readies it.
- **At once, on `issue`** ([ADR-0059](../../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md),
  [routing](../routing.md)): refine's step runs the reviewer after the
  product owner on that issue's spec alone; its review, written since,
  makes the step due again, and the product owner reads it then.
- **Off** (the reviewer not after it in a line): nothing changes.
