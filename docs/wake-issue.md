---
sources:
  - internal/wake/issue.go#Title
  - internal/wake/issue.go#Kept
  - internal/wake/issue.go#Issue
  - internal/wake/issue.go#limit
  - internal/wake/issue.go#render
  - internal/wake/wake.go#Read
  - internal/wake/wake.go#From
  - internal/wake/wake.go#readForge
  - cmd/workline/wake.go#wakeCmd
checked: e2e165c2d
verified: agent:claude-code
---
# The wake's issue

Part of [The wake](wake.md).

- **One issue**, *workline: where the line may be drifting*, opened when
  something is listed; edited in place when what it lists changes;
  never commented on.
- **Each trigger is kept 7 days**, the newest read of it: someone
  reading once a week reads each one.
- **Under the forge's size limit**: past it, the oldest triggers are
  left out, and the issue says how many. An edit the forge refuses
  anyway is a warning on a GitHub Actions run, not only a log line.
- **No merge skipped**: a read starts where the issue's last read ended,
  when that is earlier on the branch than `--range`'s start: a run the
  forge dropped from its queue is read by the next.
- **The cord**: anyone, a person or a role, puts `step-back` on an issue
  or a merge request; the next merge lists it. The wake makes the label
  when it is missing.
- **What was not read** is said under *Not read*, never taken for
  nothing found: no forge, a read refused, a forge that keeps no
  rulesets or no `workline/accepted` (GitLab). Closing links unread, no
  merge is said to close none; its issue unread, the issue is left as it
  is.

