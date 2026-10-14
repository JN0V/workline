---
sources: [internal/engine/routine.go, internal/engine/demote.go, internal/engine/merge.go, internal/forge/merge.go]
checked: c6af84e6
verified: agent:claude-code
---
# The merge policy at `normal`

Part of [the merge policy](merge-policy.md): at `normal`, workline's own
routine pull requests merge with no person's yes; everything else waits
for one, as at `cautious`
([ADR-0042](adr/0042-the-engine-merges-by-the-projects-policy.md)).

```yaml
merge: {autonomy: normal}   # a project's choice, never a default
```

## Routine work with no yes

- **Merged with no label**: workline's own pull requests, as the forge
  says it — a commit says anything of itself:
  - from a role's branch (`workline/<role>/…`), opened by workline's App;
  - every push to it the App's, every commit committed by it and its own
    by its tree;
  - only docs of the documentalist's that agents are not told to read
    (not AGENTS.md, CLAUDE.md, a page they link to, a prompt…);
  - their head's verdict passed, every agent's review whole.
- **Three kinds, by what changed**:
  - `records`: only when docs were read (`judged`);
  - `derived`: derived blocks, as their commands give them;
  - `docs`: doc text a role wrote — the release's docs fix, gardening's.
- **The same path as a yes**: `workline/accepted` set on the head, auto-merge
  on, in line. The veto is the forge's, as
  [at `cautious`](merge-policy.md#the-veto-is-the-forges).
- **Each one recorded**: a comment on it — no person's yes, its kind,
  why —, and the job's summary, whose first line says which kinds merge
  alone.
- **A revert demotes its kind**
  ([why](adr/0042-the-engine-merges-by-the-projects-policy.md#each-merge-recorded-a-revert-demotes-its-kind)):
  `follow` reads every revert on the base once, opens an issue "workline:
  its `<kind>` merge requests wait for a person's yes again (revert …)",
  and that kind needs the label until a person who may push and merge
  **closes the issue**. A revert it cannot tie to a pull request holds
  every kind.
- **What it read is the App's mark** on the base, a status only the App's
  counts. None found — the first run at `normal` too —: every kind waits,
  in an issue, until such a person closes it.
- **Never alone**: the release tool's, a fork's, a person's, another
  bot's, a Draft; with a person's token, nothing.
- **The governance line**: a change an AI wrote and an AI reviewed merges
  with no person's approval — past the two-person rule of SLSA and SOC 2.
- **The App's key, out of a pull request's own run**: none opens a
  `workline/` pull request as the App; a release's fix only when the App
  alone pushed to its branch ([ADR-0044](adr/0044-the-apps-key-stays-with-the-default-branchs-runs.md)).
- **Tried on workline's own repository first**; agents' pull requests come
  later ([#322](https://github.com/JN0V/workline/issues/322); the
  [criteria](adr/0042-the-engine-merges-by-the-projects-policy.md#the-veto-what-never-merges-alone-the-criteria)).

## Set up

- As at `cautious` ([set up](merge-policy.md#set-up)); the App marks each
  base commit it read for reverts with the status `workline/merge-read`.
- **Another forge** (`cmd:`): besides the operations `cautious` needs,
  `app`, `merge-state`'s `pushers`, `closing`, `mark`, `marked`,
  `comments`, `comment`, `issue` and `open-issue`
  ([forge command](spec/forge-command.md)).

## What the summary says

As at `cautious` ([each word](merge-policy.md#what-the-summary-says)),
and:

| Said | When |
|---|---|
| `waits-for-a-yes` | the release tool's; a push, a commit or a page not workline's; a kind demoted |
| `merge-app-unknown` | the token is no App's: nothing merges alone |
| `merge-demoted`, `demotion-unrecorded` | a revert of one merged alone read: its kind demoted, its issue opened or not |
| `revert-unread` | a revert tied to no merge request: every kind demoted |
| `merge-restored`, `restore-ignored`, `restore-unread` | a demotion's issue closed by a person who may lift it; by anyone else; unread |
| `merge-record-lost`, `merge-unmarked` | no mark of the App's on the base's last 100 commits, or none kept: every kind held |
