---
sources:
  - internal/engine/merge.go#theYes
  - internal/engine/merge.go#releaseRebuild
  - internal/engine/merge.go#mergePolicy
  - internal/engine/queue.go#queueGroup
checked: 3f432ab14
verified: agent:claude-code
---
# The yes covers one head

Part of [the merge policy](merge-policy.md). A person's yes covers the
head they read; a push after it takes it back, but for the pushes below
that add nothing a person must read.

## A push takes the yes back

- **A push after the yes takes it back**: auto-merge off, the label off,
  one comment.
- **When each head was pushed** is the forge's word: GitHub's repository
  activity, GitLab's versions.
- **Each yes label by its own pushes**: `workline:accepted` put before a
  push and `workline:accepted-whole` after it, only the stale one goes
  off; the other covers the head and stands.

## A release tool's rebuild is no push

The release tool force-pushes its pull request on each merge to the
base. Its yes holds on the rebuilt head when the rebuild adds only what
the line merged by itself:

- **Every push after the yes is the release tool's own**, its App's: one
  a person pushed, or by a hand the forge does not name, needs a yes.
- **The rebuilt head is one commit on the base**, and every base commit
  between the covered head's base and the new one came from a merged
  pull request that carried no person's yes label.
- **Needs a yes**: a feature, a task, a commit with no merged pull
  request (a direct push, a hand merge), or one that cannot be read.
- **Its tree is the covered head's change redone on the new base**, but
  for the files the release tool's own commit changed on the covered
  head (its version and changelog files); any other file changed takes
  the yes off.
- **Said**: when the yes is taken off, the comment names the added
  commit or the file that caused it.
- **Not a release**: any push after the yes takes it back, as above.

## workline's own commit is no push

Checked by its tree, never by its trailer alone. The committer checks
every commit all the same:

- **A role's branch** (`workline/…`) rebuilt on the base: one commit,
  exactly the covered change redone on it.
- **A merge of the base**, never of another branch, with workline's
  trailer: exactly what workline's own merge gives.
- **A record**, with workline's trailer: one commit changing, against its
  parent, only lines workline writes in the docs — a header's
  `checked`, `judged`, `judged-in-parts` or `verified`, a derived block
  as its command gives it
  ([ADR-0041](adr/0041-workline-settles-its-own-lines-when-the-base-moves.md)).
- **Still a push**: a docs fix changing a doc's text or another key, and
  any other commit with no workline trailer, whatever it changes.
- **A rewritten message**: one whose message the committer rewrote
  (`Workline-Role: committer`) is a person's.

## Held by the forge, not by a race

- **GitHub keeps auto-merge on** after a writer's push. So the base
  requires the status `workline/accepted` from the App, which `follow`
  sets on the head a yes covers: a new head lacks it and cannot merge.
- **A base that does not require it**, or from any source, gets nothing
  turned on, said.
- **On GitLab**, the merge waits for the new head's pipeline, which holds
  `workline:merge`.
- **A merge queue** tests a group's commit against the same checks:
  `follow --merge-group` sets `workline/accepted` there as the head the
  group ends with holds it. The queue's run checks only the gates, the
  yes staying the head's ([a merge queue](merge-queue.md)).

## Where it is read

- **The yes and the pushes after it**: `merge.go` finds the person's yes
  on the label events, and takes it back on a push.
- **A release tool's rebuild, no push**: `merge.go` tells it from a push.
- **The queue**: `queue.go` groups the pull requests that wait to merge.
