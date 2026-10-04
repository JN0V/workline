---
sources: [internal/line, internal/routing, internal/work, routing.default.yaml]
checked: 1bab30d
verified: agent:claude-code
---
# Routing — v1 (draft)

Who works next is decided by a file, not by a model. Routing reads an event and
the state of the work, and names the roles and gates to run, in order. It is a
pure function: same inputs, same answer, no AI.

## Work starts from a clear need

People own both ends of the line: they state the need and how the result will be
judged, and they accept or reject what comes out. The machine builds in between.
So no development starts without a clear expression of the need. That does not
mean pages of specification; it means the work is no longer research.

A work item — an issue on the forge, or a file in `.workline/work/` without one
— moves through these states:

```
to-refine ──► ready ──► in-progress ──► review ──► validation ──► done
    ▲            │
    └────────────┘  (the need turns out unclear: back to refining)
```

**Ready** means four things are written, even in one line each:

| Field | Question |
|---|---|
| Need | Which user need does this answer — who needs what, and why? |
| Verification | How will the machine prove it works: which tests, checks, thresholds? |
| Validation | How will a person accept it: who, looking at what? |
| Scope | What part of the project does it touch? |

Verification is what the line checks on its own; validation is what a person
decides at the end. Writing both before starting is what keeps the machine
from grading its own work. The engine checks that the four are present and
not empty; it does not judge their quality. A person moves an item to
`ready`; no role does yet. Development roles only take `ready` items, and
the item's **scope** becomes the run's scope (see "Stay on the task" in the
role contract). *Not built yet: the scope is given with `--scope`; no role
takes items yet.*

`review` is the machine's part: reviewers and gates run the verification.
`validation` is the person's part: the item reaches `done` only when the person
named in *Validation* accepts it. A rejection sends it back to `in-progress`
with the reason, or to `to-refine` if the need itself was wrong.

An item that needs exploration stays in `to-refine`. Research, spikes and
prototypes happen there, and their outcome is a refined item, not merged code.

## The line

workline ships a default line (`routing.default.yaml`); a project changes it in
the `routing:` section of `.workline/config.yaml`. An event it lists replaces
the default one; a `handoffs` list it gives replaces the default list — an
empty list forbids every handoff. `workline route <event>` runs an event's
steps; the git hooks run `commit-msg` and `pre-push` through the line too.
`pre-push` is not routed by default: the global hook reaches every repository
on the machine, and a project opts in.

```yaml
routing:
  events:                                 # event -> what runs, in order
    commit-msg:    [committer]
    merge-request: [committer, documentalist, gate:merge]
    schedule:      [documentalist]
    release:       [gate:release, documentalist]   # before the release tool tags

  handoffs:                             # the only handoffs a role may ask for
    - {from: my-role, to: documentalist}  # a project's own role; none shipped asks
```

workline cuts no releases: a release tool does (release-please,
semantic-release…), and `release` runs the documentalist before it, so the
docs due at the release gate it (ADR-0010, ADR-0017). A tool that opens a
release pull request needs no `release` step: on `merge-request`, a role
that runs on `release` runs as on `release` when the merge request comes
from a release tool's branch (`release.branches`). A routing that still
names the retired `release-manager` is refused, saying which tool to use.

- Steps run in the listed order. The first step that ends in `block`,
  `human`, `blocked-external` or an error stops the sequence. It fails closed.
- A `handoff` intention is applied only if its edge is declared here. Anything
  else is refused, like any invalid intention.
- A chain of handoffs is capped (`max-handoffs`, default 3). A loop stops with a
  verdict, not with a timeout.
- Events caused by the engine's own writes (its comments, labels, commits) do
  not trigger routing, so two roles cannot wake each other forever. Its
  commits carry a `Workline-Role: <role>` trailer, and a `Workline-Model:
  <agent>:<model>` for each agent and model whose answer they hold, so a
  later reader can stand apart from them (ADR-0005); a line given a range
  whose last commit carries one judges nothing.

### On a forge: judge, then apply

`workline route <event> --no-apply` runs the whole line in a job that holds no
write token, with the forge and target of the merge request (`--forge`,
`--target`; `--input` and `--scope` go to every step). Each step judges the tree
as it is — no step sees what an earlier one proposed — and nothing is applied.
The result lists the runs to apply, in the line's order (`pending`) — a
step that went round, proposing a merge request a round, gives one run a
round (docs/spec/role-contract.md, "Again");
`workline apply --line <result>` applies them in the job that holds the token
and no AI key, and stops at the first that does not pass.

Applying runs no role, so a handoff proposed under `--no-apply` is recorded
and not run: both steps say so (`handoff-deferred`), with the command that runs
it where an agent may judge.

## Where state lives

On a forge, the state of a work item is a label (`workline:ready`,
`workline:in-progress`…), readable by humans and by any CI. Without a forge, it
is the `state:` line of the item's file in `.workline/work/<id>.md`, whose
`## Need`, `## Verification`, `## Validation` and `## Scope` sections
`workline item ready <id>` checks before moving it. Either way, a state changes only through the
engine, one transition at a time, and each transition is logged. *Built so far:
the move to `ready`, and its check; transitions are not logged yet.*

## Not in this version

The developer, reviewer and tester roles, which will consume `ready` items;
refining items, which the product owner does not do yet.
