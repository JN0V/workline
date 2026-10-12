# ADR-0011: The review is on the merge request, not the push

- **Status:** accepted; amended 2026-10-12 (the engine merges by the
  project's policy, [ADR-0042](0042-the-engine-merges-by-the-projects-policy.md))
- **Date:** 2026-09-30
- **Amends:** ADR-0007, whose approval of every push was on for everyone;
  ADR-0008, whose channels stay for whoever asks for it

## Context

ADR-0007 had a person approve every push, so that no agent pushes with no
person having looked; ADR-0008 took the question to the editor. In use
(2026-09-29 and 30): the editor's box went unseen, then an Enter on it
stopped the push — VS Code writes "Press 'Enter' to confirm" under every
box, while an empty answer is a no; and a push waited on a question for
every branch sent. Meanwhile `main` of workline and DomoticsCore now takes
merge requests only, with the checks green (docs/research/ci-and-forge.md):
nothing reaches it without a merge request.

A review at the push sees commits in a box; a review on the merge request
sees the diff, the checks and the discussion, where the forge already asks
people to look. It is where the person wants to be made to look.

## Decision

- **The push approval is off by default.** A person who wants it sets
  `approve-push: true` in their own config; the channels of ADR-0008 then
  apply, `approve-push-via` their order. The box says "Type y then Enter to
  push; Enter alone stops".
- **The review is on the merge request.** `main` takes merge requests, the
  checks green; a team requires approvals there, a solo maintainer none.
- **An agent pushes a branch and opens the merge request; it never merges,
  and never turns auto-merge on**: the person does, on the forge. Written in
  AGENTS.md for agents working on workline.

## Consequences

- A push asks nothing; it takes seconds.
- An agent could still merge with the person's forge token: the rule is a
  wish until the forge tells an agent from a person — a bot account or a
  GitHub App for agents, with no right to merge. Listed in docs/BACKLOG.md.
- Where no merge request is required, nothing reviews an agent's push:
  such a project turns `approve-push` on, or protects its branch.

## Amendment (2026-10-12): the engine merges by the project's policy

[ADR-0042](0042-the-engine-merges-by-the-projects-policy.md), built by
[#321](https://github.com/JN0V/workline/issues/321) at its `cautious` level.

- **An agent never merges, never turns auto-merge on, and never puts
  `workline:accepted`**: kept.
- **The engine does, by the project's policy** (`merge:`): `workline
  follow`, holding the write token and no AI key, turns the forge's own
  auto-merge on for a pull request a person labelled, its head's verdict
  passed. The forge merges, checking its own rules.
- **The review stays the merge request's**; the person's yes is a label
  on it. Without `merge:`, the person merges, as before.
