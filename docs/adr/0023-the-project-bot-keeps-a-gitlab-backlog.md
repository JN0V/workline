# ADR-0023: On GitLab, the project's bot keeps the backlog; Planner and above are of the project

- **Status:** accepted
- **Date:** 2026-10-05
- **Builds on:** ADR-0016 (writes go where the project lives), ADR-0018
  (an outsider's issue is theirs), ADR-0021 (the conversation with the
  reporter), ADR-0022 (a split's children are GitLab tasks); principles 6,
  12, 14

## Context

The product owner wrote to GitLab with whatever token it found, and took
every reporter there for an outsider — GitLab's issue says nothing of its
author's access, where GitHub's says `author_association`. So on GitLab
every refine became a comment to the reporter, and a split, a rename or a
move to ready only a proposal (roles/product-owner/tried.md, 2026-10-05).
docs/research/ci-and-forge.md ("A bot writing GitLab issues from CI")
found: CI's job token reaches neither issues nor GraphQL; a project access
token is a bot user, a member of that project only, with a role and
scopes; the Planner role (GitLab 17.7) edits issues, labels and
milestones but, tried live, may not give a task its parent; only a
note's author, or a Maintainer, edits a note.

## Decision

- **Of the project** on GitLab is a member, direct or inherited, active,
  with the **Planner** role or above (access level 15: Planner, Reporter,
  Developer, Maintainer, Owner): who may set a label, so may accept a
  draft with `workline:accepted` — GitHub's owner, member or collaborator
  being who may triage there. A Guest, a Minimal Access member, or anyone
  not a member is outside. The members are read once a run
  (`members/all`); a token that may not read them fails the run, loud,
  rather than taking every reporter for an outsider (principle 12).
- **Bots**: a project or group access token's user
  (`project_<id>_bot_…`, `group_<id>_bot_…`) counts by its role, as any
  member, for the issues it opens; its comment never agrees to a proposal
  (ADR-0021). The engine's own comments are told by their marker, never
  by their author: with a person's token, the line writes as that person.
- **The token**: `WORKLINE_GITLAB_TOKEN`, a **project access token** —
  its own bot user, non-billable, a member of this project only, revoked
  in one click — scope `api` (issues, notes, labels, milestones, the
  members, GraphQL's tasks; `read_api` writes nothing), role **Reporter**
  for a project where workline only keeps the backlog, **Developer** with
  `write_repository` where the same token pushes gardening's branches and
  opens their merge requests (the CI template's one token). Not Planner:
  it may not give a split's child its parent. Never the job token: it
  reaches no issue. Where project access tokens are not offered, a
  personal access token, `api`, of an account made for the line and
  added to the project at that role; the person's own as a last resort.
- **Set up in one command** (docs/ci.md): `glab token create … | glab
  variable set WORKLINE_GITLAB_TOKEN --masked`, the token never shown.
- **A note only its author edits**: the engine edits the last note
  carrying a marker; GitLab refusing (403) — a note another token wrote,
  before the bot took over —, it writes the note anew after it, and the
  last one is read. The old one stays, stale, as a person's would.
- **Closing**: GitLab keeps no close reason. A duplicate is closed with
  GitLab's own `/duplicate #n`, which links the original; anything else
  closes as closed, its reason in the engine's comment. "Not planned" is
  a person's on both forges (ADR-0018).

## Consequences

- On GitLab, a member's issue is refined in place, split, renamed and
  moved to ready as on GitHub; a Guest's or a stranger's is proposed to
  its reporter (ADR-0021).
- A project moving from a person's token to the bot keeps its states:
  each is written anew once, beside the old.
- A role lower than the one advised shows as a refusal: Planner's split
  lists its children in the parent's body (ADR-0022's fallback), and says
  so in the report.
- Left: the Planner role's refusal reported as such, not only as the
  fallback; a group access token, and service accounts, untried; a
  self-managed instance untried.
