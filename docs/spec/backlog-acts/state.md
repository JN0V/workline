---
type: reference
sources: [internal/backlog/backlog.go, internal/backlog/opening.go, internal/backlog/acts.go, internal/builtin/productowner/productowner.go, internal/engine/backlog.go]
status: draft
checked: a88dc872
verified: agent:claude-code
---
# The issue's state

Part of [Acts on the backlog](../backlog-acts.md): what the engine keeps of an issue, in one comment on it.

What the engine knows of an issue lives in one comment it keeps on that
issue, marked `<!-- workline:sticky=product-owner/state -->`:

```yaml
sources: [src/export/csv.go#WriteRows]   # what the issue is about, in the code
confirmed: 1a8e5a4                       # the commit it was last found true at
judged: 1a8e5a4                          # the commit the role last read it at
comments: 2                              # an older engine's count of the comments it read; `heard` now (A person's answers)
body: 3f9a1c0e2b7d                       # a digest of its body, when last read or written
priority: 2                              # the priority the role last set (Ordering)
effort: 1                                # the effort the role set, once; -1, set in a field whose values are hidden (Ordering)
title: The CSV export drops the last row  # the title the role last set (Renaming)
split: [13, 14]                          # the children it was split into (Splitting)
kept: ['src/export/csv.go:b2a6ba892dee'] # announcements as obsolete kept open, by their quote (Closing)
sections:                                # its Need and Scope as last read or written (A changed need)
  Need: Every row exported, so a report counts what was sold.
  Scope: WriteRows in src/export/csv.go.
wrote: {Verification: cb1c26c09a46}      # the sections the role wrote, a digest of each text (A spec read before ready); Description, the text a role opened it with
answered: 3f9a1c0e2b7d                   # the review of its spec last given to the agent, by the body it read
plain: 9b0e7d1c2a44                      # its description as the role rewrote it in plain words (Refining to ready)
deleted: [Validation]                    # sections the role wrote and a person deleted: never written again
proposed:                                # the acts proposed on it, each as decided, since a day (On the issue)
  - {act: rename, line: 'Rename #9 to …', since: '2026-10-08', proposal: {do: rename, issue: 9, title: …}}
done: [{act: ready, level: normal}]      # its acts done alone a person may undo (Trust)
undone: [{act: rename, evidence: …, day: '2026-10-08'}]  # its acts a person undid: that kind proposed here
did: [{act: refine, level: normal, day: '2026-10-08', at: 1a8e5a4, model: 'claude:claude-sonnet-5', body: 3f9a1c0e2b7d, line: …}]  # what the role did, its day, what it read; `yes` when a person's
closed: close-duplicate                  # the kind it was closed with: open again, undone
label: workline:proposed                 # the label the role put on it while it waits on a person
aside: 3f9a1c0e2b7d                      # set aside by a person, at this body: nothing proposed until it changes
heard: 2                                 # the comments that count the role read: one more is a person's word since
revisions: 1                             # its drafts revised for a person's comments
rules: 2                                 # the way of reading it was last read under (backlog.Reading)
```

The comment says, above the state, what the role did and proposes there,
the state folded ([On the issue](on-the-issue.md)). The block's fence is longer than any
run of backticks in it: a text holding a code fence is kept as written.

Each act in `did` keeps what its run read, for the weekly sample to
re-check it on that (ADR-0046): `at`, the commit the run's `pre` ran on;
`body`, the digest of the issue's body `pre` read (`in/read.yaml`, every
open issue's); `model`, the agents and models the run asked
(`<agent>:<model>`, a comma between two), none for an act a person
accepted. An engine before ADR-0046 kept none of them.

`sections`, `wrote` and `deleted` keep each section under the heading it
had in the body, and are read by its meaning (`work.Headings`,
[ADR-0039](../../adr/0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md)):
a section deleted under one heading is never written under another of
the same meaning.

Two carrying the marker, the last is read: on GitLab only a note's author
edits it, so one another token wrote is left, and the state is written
anew after it (ADR-0023).

An issue without that comment, or with one that does not read, is never
acted on: an act on it is dropped (`no-state`, `state-broken`) and nothing is
written on the issue. One that does not read is never taken for none, which
would write a state anew over what the role held there: `pre` says it each
run, a warning (`state-broken`, #302), until a person mends that comment or
deletes it. The role's `pre` gives an issue it takes its first
state, `confirmed` at the commit it read, with no sources until an act
names them. An issue another role opens for what it found outside its task
(the reviewer's, ADR-0020) gets its state from the engine as it is opened —
its sources, and the commit it was seen at — and `needs-triage`; it is
never read yet, so a later run reads it, in its catch-up after a person's
issues, and its agent is told it
is that role's draft to refine ([Opening issues](opening.md)).
