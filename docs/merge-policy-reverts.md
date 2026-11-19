---
sources: [internal/engine/demote.go#demotions, internal/engine/demote.go#holds, internal/engine/demote.go#demote, internal/engine/demote.go#mergedAlone, internal/engine/demote.go#demotedTitle, internal/engine/demote.go#revertWindow, internal/forge/merge.go#ReadStatus, internal/forge/merge.go#github.mark, internal/forge/merge.go#github.marked, internal/forge/github_merge.go#github.ownStatus, internal/forge/local_merge.go#Local.mark, internal/forge/local_merge.go#Local.marked, internal/forge/local_merge.go#Local.closing, defaults/ci/github/workline-merge.yml#on, defaults/ci/github/workline-merge.yml#jobs.follow.if]
checked: c46a93d6c
verified: agent:claude-code
---
# Reverts at `normal`

Part of [the merge policy at `normal`](merge-policy-normal.md).

## A revert demotes its kind

- **A revert demotes its kind**
  ([why](adr/0042-the-engine-merges-by-the-projects-policy.md#each-merge-recorded-a-revert-demotes-its-kind)):
  `follow` reads every revert on the base once, opens an issue "workline:
  its `<kind>` merge requests wait for a person's yes again (revert …)",
  and that kind needs the label until a person who may push and merge
  **closes the issue**; on GitHub, `follow` runs at once on that close.
  A revert it cannot tie to a pull request holds every kind.
- **What it read is the App's mark** on the base, a status only the App's
  counts. None found, while its history holds a merge workline made
  alone: every kind waits, in an issue, until such a person closes it.
  None found and no such merge on the base's last 1000 commits — a first
  install —: the App marks the base read, and nothing waits; a merge
  request whose comments it cannot read counts as such a merge
  ([#1018](https://github.com/JN0V/workline/issues/1018)).

## Set up

- As at `cautious` ([set up](merge-policy.md#set-up)); the App marks each
  base commit it read for reverts with the status `workline/merge-read`.
- **The local forge**: the marks and the demotion issues are kept in the
  clone; a demotion issue you close is lifted
  ([ADR-0042](adr/0042-the-engine-merges-by-the-projects-policy.md#amendment-2026-11-10-1322-the-local-forge-merges-itself)).
- **Another forge** (`cmd:`): besides the operations `cautious` needs,
  `app`, `merge-state`'s `pushers`, `closing`, `mark`, `marked`,
  `comments`, `comment`, `issue` and `open-issue`
  ([forge command](spec/forge-command.md)).

## What the summary says

As at `cautious` ([each word](merge-policy.md#what-the-summary-says)),
and:

| Said | When |
|---|---|
| `merge-demoted`, `demotion-unrecorded` | a revert of one merged alone read: its kind demoted, its issue opened or not |
| `revert-unread` | a revert tied to no merge request: every kind demoted |
| `merge-restored`, `restore-ignored`, `restore-unread` | a demotion's issue closed by a person who may lift it; by anyone else; unread |
| `merge-record-lost`, `merge-unmarked` | no mark of the App's on the base's last 100 commits while a merge made alone is in its history, or a forge that keeps none: every kind held |
