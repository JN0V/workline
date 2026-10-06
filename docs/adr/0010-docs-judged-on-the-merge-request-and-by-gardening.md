# ADR-0010: Docs are judged on the merge request and by gardening; the push only counts

- **Status:** accepted; built 2026-10-01; amended 2026-10-02 (where the
  merge requests go)
- **Date:** 2026-09-30
- **Supersedes:** ADR-0007 on judging at the push (`d`, the docs listed in
  the question). Its range, the review per doc and the push approval stay.

## Context

ADR-0007 had the docs a push's commits made suspect judged from the push,
with `d`. In use (2026-09-29): a push on DomoticsCore took three minutes
before a word; the person, told four docs were suspect, took it as a step to
follow, answered `d` in the editor's box, where only a terminal can judge,
and the push stopped. Questions at every push that cannot be acted on where
the person is wear the process out.

No doc agent judges at the push (docs/research/documentalist.md): they act
on the merge request, or after it, like Renovate on a schedule. Reviewed from
three sides — a solo developer, a team with forks, the tokens
(docs/research/ci-and-forge.md): the model holds, with gaps: nothing judges
a repository pushed to main directly; the templates cannot commit to a
merge request on GitLab, nor comment on a fork's on GitHub; a bot's commit
waits for its checks; `checked` after a squash merge; re-judging at every
push of a merge request.

## Decision

- **At the push**: the committer and the person's approval (ADR-0008). The
  documentalist runs with no agent and says one line — how many docs the
  commits made suspect since they were last judged, and `workline docs` —
  never a question. `d` goes.
- **On each merge request, in CI**: the docs the merge request made suspect
  are judged, and the fix is one `docs:` commit on its branch, pushed with a
  GitHub App's token, so the checks run again unattended. Never on a
  commit of its own (the loop guard, built, not only specified); verdicts
  kept by the blobs of the doc and its sources, so a push of the merge
  request pays only for what changed; a cap on calls and tokens a run, past
  which the docs left are listed, never passed. From a fork: a comment. No
  agent, no key or no quota: the docs are listed in the comment and the
  check does not block — a person judges.
- **Gardening, on a schedule, at night**: one merge request per task
  (ADR-0006); the docs far behind judged in parts there (ADR-0009); the
  docs an open merge request holds skipped.
- **Locally, when the developer says so**: `workline docs`, judging from
  where the docs were last judged — a ref it moves — to HEAD, pushed or not.
- **At the release**: docs suspect since the last tag block it until judged:
  where there is no merge request, this is the moment docs are made true.
- **Main takes merge requests** on workline and DomoticsCore: 0 approvals
  required, the checks green and up to date, auto-merge; no one forces.

## To settle before building

- ~~What `checked` names once a merge request is squashed~~: tried
  2026-09-30 on the `documented` fixture. A merge commit keeps it true. A
  squash made the doc suspect again, then, the branch gone, blocked the
  documentalist (`unknown`, bad revision); a rebase the same. The commit
  that brought the `checked` to main now stands for it: settled, with three
  conformance cases; the sources' blobs were not needed.
- The templates: a token GitLab gives a merge request's branch; the fork
  comment on GitHub; merge trains and queues left out.
- The AI in CI: the owner's Claude token first, its volume capped; Copilot
  CLI on a free plan measured through the evaluation, to compare.

## Tried

- 2026-09-30, a throwaway pull request on workline (#10): Claude judged the
  doc the change made false, the App committed the fix to the branch, the
  checks ran again unattended, the loop guard held
  (roles/documentalist/docs/tried.md). Left: `checked` names the head of the
  branch, not GitHub's merge commit; the agent stays off an example.

## Consequences

- A push costs seconds and asks one question, the approval.
- Docs are fixed where the change is reviewed, in the same merge request.
- A solo project with no CI is caught up at the release, or when its
  developer runs `workline docs`; the push line keeps the count in sight.
- `workline init` keeps routing the documentalist to `pre-push`, for its one
  line counting the docs; doctor's `documentalist-not-before-push` became
  `docs-judged`: where docs are judged — a merge request in CI, gardening,
  the release — and `docs-judged-nowhere` when none (2026-10-01).

## Amendment (2026-10-02)

The merge requests and comments above go where the project lives
(ADR-0016): GitHub and GitLab natively, another forge through a command
(`forge: cmd:<command>`), and a repository with no forge — pushed to main,
with no merge request — keeps them in its clone with `forge: local`:
gardening's merge requests are local branches there, recorded in
`.git/workline/merge-requests/`, read with `workline issues`. The decision
is unchanged.
