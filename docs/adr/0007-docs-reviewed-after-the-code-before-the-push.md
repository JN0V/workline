# ADR-0007: Docs are judged after the code is done, reviewed before the push, and every push is approved by a person

- **Status:** accepted
- **Date:** 2026-09-29

## Context

A project routing `pre-push` to the documentalist had every suspect doc of
the repository judged by the agent inside the push, its patches written to
the working tree, the push stopped. Tried on DomoticsCore after adoption
(2026-09-28): 53 docs suspect, about 430k tokens to judge them all, five
calls a push, nine or ten pushes to get through. Three faults:

- **Too late.** At the push, the person has validated their code; doc changes
  land after, in a stopped push, and are not reviewed as they should be.
- **Too wide.** The documentalist never read the pushed range: it judged
  docs no pushed commit had touched, against the rule to stay on the task.
- **Too often.** A typical DomoticsCore push makes about 13 docs suspect
  through wide `sources`; committing its own patches made the docs indexing
  `docs/` suspect again on the next push.

And AI agents push to remotes with no person having looked. No doc agent
runs an AI at commit or push (Swimm, Mintlify, Dosu, CodeRabbit, Promptless:
docs/research/documentalist.md): detection is cheap and mechanical, the
judgement runs on the merge request or after it, its changes in a commit of
their own, reviewed per doc. A push confirmation read from the terminal is
the usual guard against agents (agent-git-guard; docs/research).

## Decision

- **Every push is approved by a person.** The global `pre-push` hook, after
  the roles, lists the commits leaving and asks `Push? [y]es / [N]o / [v]iew /
  [d]ocs` on the terminal; `v` opens a page workline writes itself — the
  commits and their diff — with the system's default browser, whatever the
  editor. No terminal (an agent, an editor's button): the push is refused,
  saying so. It is on wherever the global hooks are; a person turns it off in
  their own config only, which a project cannot.
- **The documentalist judges only what the commits changed.** Given a range,
  a doc is judged only if a commit of the range touched one of its sources;
  the other suspect docs are reported, left for gardening. A change to a doc's
  header alone (`checked`, `verified`) changes nothing it says.
- **Docs are judged when the person says the code is done:** `workline docs`,
  or `d` at the push, on the commits not pushed yet. The proposals are
  reviewed per doc on the same page; those accepted become one `docs:`
  commit. The push never runs an agent: it lists the docs the commits made
  suspect and not reviewed yet, and lets the person decide.
- **The rest goes to gardening**, on a schedule, one merge request per task
  (ADR-0006): docs suspect before these commits, stale ones, the backlog of
  an adoption.
- **Code no doc describes is reported**: a file the commits add, matching the
  project's `documented` globs, under no doc's sources. So is a source that
  no longer exists.
- **The header is a comment** (`<!-- workline … -->`): unseen in a preview, on
  a forge and in a PDF. A doc with a frontmatter of its own keeps it there.

## Consequences

- A push costs no tokens; judging costs them once per piece of work, on
  what it changed.
- A person can push with docs not reviewed: the push says which, and does
  not block. Gardening catches up.
- `git push --no-verify` still skips everything: the approval stops an agent
  pushing by habit, not one set on it. Protected branches on the forge
  remain the barrier for what matters.
- Pushing from an editor's button is refused: a person pushes from a
  terminal, or turns the approval off.
