---
sources:
  - internal/engine/merge.go#personsCommentOpen
  - internal/engine/merge.go#heldByConversations
  - internal/engine/merge.go#mergePolicy
checked: 3f432ab14
verified: agent:claude-code
---
# What holds a pull request

Part of [the merge policy](merge-policy.md): what keeps an accepted pull
request from merging, each found by `merge.go`.

## A person's comment with no answer

- **Holds it**: nothing merges, auto-merge stays off, whatever the
  policy.
- **Answered by a role's reply that names a test**.
- **Answered by the yes of the person who wrote it**, put at or after it
  and still on. The trace written before a yes put in their name
  ([ADR-0068 §7](adr/0068-the-line-steps-back-on-its-own-drift.md#7-a-persons-yes-is-assisted-never-delegated))
  never holds that yes.
- **A conversation on a line** is answered by that yes only when every
  person's comment in it is theirs, before it.

## Open conversations, said

- **When**: auto-merge on, its checks green, the forge not merging, and
  its base requiring every conversation resolved.
- **Said**: `follow` lists each open one, linked, in its summary and
  once on the pull request, from the run that turns auto-merge on.
- **Who resolves**: the reviewer its own, once the code answers them
  ([lines](../roles/reviewer/docs/lines-again.md#not-repeated-found-again));
  a person theirs.

## Nothing turned on, said in the job's summary

- **Why**: a verdict or a check red, a review short.
- **Each check counts by its latest run on the head**: a failure re-run
  green since holds nothing.
- **A run the forge cancelled is no result**: the latest other run of
  that check counts — on GitLab, the newest other pipeline of that head
  —, else it waits.
- **One not judged yet** waits, named.
