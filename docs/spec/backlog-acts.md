---
type: reference
sources: [roles/product-owner/role.yaml]
status: draft
checked: 5afbb282
verified: agent:claude-code
---
# Acts on the backlog

What the engine does when a role acts on a project's issues rather than on
its code (ADR-0018). The product owner proposes the acts; the engine checks
each one against the code and the forge, then does it, proposes it, or drops
it. These pages are the contract; the acts built are closing, naming an
issue's sources, putting it in a milestone, ordering it, opening one from
a file, refining one to ready — and back —, splitting one — and
reporting on a parent as its parts close —, renaming it, naming what it
waits on, and flagging the issues built on a need that changed. Since
ADR-0038, what the role proposes, and a person's answer, are on each
issue ([On the issue](backlog-acts/on-the-issue.md),
[A person's answers](backlog-acts/answers.md)); milestones, ordering,
splitting, renaming, what an issue waits on, closing as obsolete and a
changed need are off by default, their code kept.

| Page | What it holds |
|---|---|
| [The issue's state](backlog-acts/state.md) | The comment the engine keeps on each issue. |
| [Opening issues](backlog-acts/opening.md) | The one way every role opens an issue. |
| [Reading](backlog-acts/reading.md) | Which issues a run reads, and in what order. |
| [A changed need](backlog-acts/changed-need.md) | The issues flagged when what they were built on changes. |
| [Importing a file](backlog-acts/import.md) | A file moved to issues once, and [the map](backlog-acts/import.md#the-map-every-item-to-its-issue-or-why-not) of its items. |
| [Closing](backlog-acts/closing.md) | A duplicate closed, and an issue [obsolete: announced, then closed](backlog-acts/closing.md#obsolete-announced-then-closed). |
| [Milestones and ordering](backlog-acts/order.md) | [Milestones](backlog-acts/order.md#milestones), and [Ordering](backlog-acts/order.md#ordering): an issue's priority. |
| [Refining to ready](backlog-acts/refine.md) | `refine`, `ready`, `unready`, `ask`; [a body's footer](backlog-acts/refine.md#a-bodys-footer); [naming its sources](backlog-acts/refine.md#naming-its-sources). |
| [A spec read before ready](backlog-acts/spec-review.md) | The reviewer's findings on a refined issue, holding its `ready`. |
| [Splitting and renaming](backlog-acts/split.md) | [Splitting](backlog-acts/split.md#splitting) an issue into its parts, and [Renaming](backlog-acts/split.md#renaming) it. |
| [A parent and its parts](backlog-acts/parent.md) | A parent reported on as its parts close. |
| [What an issue waits on](backlog-acts/depend.md) | `depend` and `undepend`: the issues an issue waits on. |
| [The order, and what is next](backlog-acts/next.md) | [The order, and what waits](backlog-acts/next.md#the-order-and-what-waits); [what is next, what is stuck](backlog-acts/next.md#what-is-next-what-is-stuck). |
| [The conversation with the reporter](backlog-acts/conversation.md) | The rounds written to a reporter, and their agreement. |
| [Autonomy and caps](backlog-acts/autonomy.md) | Each kind's mode and cap, and [the levels](backlog-acts/autonomy.md#the-levels). |
| [On the issue](backlog-acts/on-the-issue.md) | What each issue and the job's summary show. |
| [A person's answers](backlog-acts/answers.md) | A person's gestures on an issue, and what the engine does. |
| [Trust](backlog-acts/trust.md) | The acts a person undoes, and [the weekly sample](backlog-acts/trust.md#the-weekly-sample-of-its-acts) of those done alone. |
