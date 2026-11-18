---
sources: [internal/engine/routine.go#routineCandidate, internal/engine/routine.go#weighRoutine, internal/engine/routine.go#notOwn, internal/engine/routine.go#notAllowed, internal/engine/routine.go#routineKind, internal/engine/routine.go#admit, internal/engine/routine.go#alone, internal/engine/merge.go#OwnCommit, internal/engine/merge.go#docPage, internal/engine/merge.go#rolePage, internal/forge/local_merge.go#LocalApp, internal/forge/local_merge.go#Local.app, internal/forge/local_merge.go#localAuthor, internal/engine/routine.go#readMark, internal/engine/routine.go#normal.asked, internal/engine/routine.go#RoutineKinds, internal/engine/routine.go#mergeKinds]
checked: c46a93d6c
verified: agent:claude-code
---
# Routine work at `normal`

Part of [the merge policy at `normal`](merge-policy-normal.md).

## Routine work with no yes

- **Merged with no label**: workline's own pull requests, as the forge
  says it — a commit says anything of itself:
  - from a role's branch (`workline/<role>/…`), opened by workline's App;
  - every push to it the App's, every commit committed by it and its own
    by its tree;
  - on the local forge, with no App: a role's branch, every commit a
    role's by its trailer and tree, whatever its committer — the roles
    commit under your name there;
  - only docs of the documentalist's, none of
    [what never merges alone](merge-policy-normal.md#what-never-merges-alone);
  - in a role's folder, only its README and `docs/` pages, never what
    the agent is given — persona, knowledge, instruction, policy, lenses
    ([the layout](spec/role-contract.md#layout));
  - their head's verdict passed, every agent's review whole.
- **Four kinds, by what changed** (and the developer's `code`,
  [its page](merge-policy-code.md)):
  - `records`: only when docs were read (`judged`) — gardening's
    `workline/documentalist/records` among them, for the docs judged on
    merged merge requests
    ([more](../roles/documentalist/docs/gardening.md#records-kept-on-merged-merge-requests));
  - `derived`: derived blocks, as their commands give them;
  - `docs`: doc text a role wrote — gardening's;
  - `reader`: pages an agent reworded for their reader, which wait for a
    yes ([below](#the-readers-rewordings-wait-for-a-yes)).
- **The same path as a yes**: `workline/accepted` set on the head, auto-merge
  on. The veto is the forge's, as
  [at `cautious`](merge-policy.md#the-veto-is-the-forges).
- **Each one recorded**: a comment on it — no person's yes, its kind,
  why —, and the job's summary, which says which kinds merge
  alone and which wait. Its head's checks still running, the comment
  says it merges once they pass, never before.
- **A revert demotes its kind**, until a person closes the issue that
  says it ([reverts](merge-policy-reverts.md)).
- **Never alone**: the release tool's, a fork's, a person's, another
  bot's, a Draft, one changing a guarded file
  ([what never merges alone](merge-policy-normal.md#what-never-merges-alone)); with a person's token, nothing.
- **The governance line**: a change an AI wrote and an AI reviewed merges
  with no person's approval — past the two-person rule of SLSA and SOC 2.
- **The App's key, out of a pull request's own run**: none opens a
  `workline/` pull request as the App ([ADR-0044](adr/0044-the-apps-key-stays-with-the-default-branchs-runs.md)).
- **Tried on workline's own repository first**
  ([#322](https://github.com/JN0V/workline/issues/322); the
  [criteria](adr/0042-the-engine-merges-by-the-projects-policy.md#the-veto-what-never-merges-alone-the-criteria)).

## The reader's rewordings wait for a yes

- **The kind `reader`**: the documentalist's
  [reader task](../roles/documentalist/docs/reader.md) has an agent
  reword pages for their reader. Told by the `reader:` line it writes in
  each page's header, never by the branch's name.
- **It waits for a person's yes**, the label `workline:accepted`, as at
  `cautious`: a rewording may change what a page says, and only a person
  reading it sees that
  ([why](adr/0042-the-engine-merges-by-the-projects-policy.md#amendment-2026-10-17-438-an-agents-rewording-waits-for-a-yes)).
- **Said twice**: in the job's summary, and once on the pull request, a
  comment naming the label and the setting below.
- **To let it merge alone**, a project's choice:

  ```yaml
  merge: {autonomy: normal, alone: [reader]}
  ```

  It then merges as the other kinds, recorded; a revert demotes it.

## What the summary says

As at `cautious` ([each word](merge-policy.md#what-the-summary-says)),
and:

| Said | When |
|---|---|
| `waits-for-a-yes` | a push, a commit or a page not workline's; a kind demoted; `reader`, not let merge alone |
