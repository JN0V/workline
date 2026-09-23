# Routing — v1 (draft)

Who works next is decided by a file, not by a model. Routing reads an event and
the state of the work, and names the roles and gates to run, in order. It is a
pure function: same inputs, same answer, no AI.

## Work starts from a clear need

No development starts without a clear expression of the need. That does not
mean pages of specification; it means the work is no longer research.

A work item — an issue on the forge, or a file in `.assembly/work/` without one
— moves through these states:

```
to-refine ──► ready ──► in-progress ──► review ──► done
    ▲            │
    └────────────┘  (the need turns out unclear: back to refining)
```

**Ready** means three things are written, even in one line each:

| Field | Question |
|---|---|
| Need | Who needs what, and why? |
| Done when | How will we know it works? |
| Scope | What part of the project does it touch? |

The engine checks that the three are present and not empty; it does not judge
their quality. A person — or a product role, once it exists — moves an item to
`ready`. Development roles only take `ready` items, and the item's **scope**
becomes the run's scope (see "Stay on the task" in the role contract).

An item that needs exploration stays in `to-refine`. Research, spikes and
prototypes happen there, and their outcome is a refined item, not merged code.

## `routing.yml`

```yaml
routing: 1

on:                                   # event -> what runs, in order
  commit-msg:    [committer]
  merge-request: [committer, documentalist, gate:merge]
  merge:         [release-manager]
  schedule:      [documentalist]
  release:       [gate:release, release-manager]

handoffs:                             # the only handoffs a role may ask for
  - {from: release-manager, to: documentalist}
```

- Steps run in the listed order. The first step that ends in `block`,
  `human`, `blocked-external` or an error stops the sequence. It fails closed.
- A `handoff` intention is applied only if its edge is declared here. Anything
  else is refused, like any invalid intention.
- A chain of handoffs is capped (`max-handoffs`, default 3). A loop stops with a
  verdict, not with a timeout.
- Events caused by the engine's own writes (its comments, labels, commits) do
  not trigger routing, so two roles cannot wake each other forever.

## Where state lives

On a forge, the state of a work item is a label (`assembly:ready`,
`assembly:in-progress`…), readable by humans and by any CI. Without a forge, it
is a field in the item's file. Either way, a state changes only through the
engine, one transition at a time, and each transition is logged.

## Not in this version

The developer, reviewer and tester roles, which will consume `ready` items; the
product role that helps refine them.
