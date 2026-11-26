---
sources: [roles/reviewer/role.yaml, roles/reviewer/code/reviewer.go#ask, internal/engine/engine.go#askParts]
checked: 9cc037b76
---
# Reviewer — a change too large for one call

Part of [a run on code](run.md), step 4. Why:
[ADR-0074](../../../docs/adr/0074-agents-read-the-repository-the-engine-stops-preparing-their-input.md).

A lens's task holds the change as a plain diff while it fits:
`diff-lines-max` lines (1500), and half the role's context budget
(100,000 tokens), room kept for the lenses, what the author says and
the issue. Past either, what happens depends on the agent.

## An agent that reads the repository (claude)

- **The diff is a file** in its copy of the repository,
  `workline-change.diff`, the engine's, not the project's.
- **Its task names it**, and every file the change touches.
- **It reads it whole**, as it reads the rest, its turns capped
  (`ai-max-turns`, 40).
- **Never cut, never in parts**: one call reads the whole change.

## Another agent (cmd:, a stand-in)

- **No lens is asked**, no token spent: it reads nothing but its task,
  and a cut diff would be a review not whole.
- **Said**: `review-not-whole`, with the lines the diff holds; the
  summary says the change was not reviewed whole, and to split it.
- **The commits are not recorded** as reviewed: a smaller change, or an
  agent that reads, reviews them.
- **Read whole by a person**: `workline:accepted-whole` lets it merge
  ([merge policy](../../../docs/merge-policy.md#what-it-does)).

## What stays the same

- **The diff-alone facet**, off by default, is given the change only,
  never the repository; cut past `diff-lines-max`, said.
- **The judges** read what they read before: the code near the cause,
  and the reads they ask
  ([the judge](judge.md)).
