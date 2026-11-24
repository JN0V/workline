---
sources:
  - internal/line/line.go#GatesInput
  - internal/line/line.go#foundFor.read
  - roles/inspector/code/record.go#loadRecord
  - roles/inspector/code/record.go#localRecord
  - internal/forge/own.go#Kept.Last
  - roles/inspector/code/record.go#task
  - roles/inspector/code/inspector.go#answered
  - roles/inspector/code/inspector.go#settled
  - roles/inspector/code/inspector.go#current
  - roles/inspector/code/record.go#shownMax
  - roles/inspector/code/record.go#comment
  - roles/inspector/role.yaml#on-moved-on
  - defaults/ci/github/workline-warn.yml#jobs.warn.concurrency
  - defaults/ci/github/workline-apply.yml#jobs.apply.concurrency
  - internal/engine/review.go#applier.mergedRecord
  - internal/forge/at.go#At
  - roles/inspector/code/inspector.go#Post
  - internal/report/report.go#ForCodeScanning
checked: e81df7b34
verified: agent:claude-code
---
# Inspector — a run, in short

Part of [the inspector](../README.md): what one run does, step by step.

1. **The gates' result**: the gate steps before it in the line, or the
   file `--gates-from` gives, only for the commits the run judges.
2. **The record**: hidden in its comment on a merge request, read only
   from the one workline's own account wrote and no one else edited last
   ([ADR-0051](../../../docs/adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md));
   on a machine, in `.git/workline/inspector-record`. A finding there is
   not asked again.
3. **One call**: the findings left, each with the head's code around it;
   those triaged before named for their place in the order only.
4. **Read back**: a triage naming no finding asked, saying not why it
   matters, or calling a false positive without why is not kept, said,
   and asked again at the next push.
5. **The comment**: every new finding triaged, in the order to fix them,
   25 at most, the run's summary holding the rest; the record hidden
   below. With nothing triaged to show, only a comment an earlier push
   left is edited. A push landing before it is posted: it is still
   posted, and the push's run, which waits for it, asks only what is
   new; both runs' records are merged
   ([#580](https://github.com/JN0V/workline/issues/580)). Applied out
   of order, the newer run's text stays
   ([#598](https://github.com/JN0V/workline/issues/598)).
6. **The verdict**: always a pass; each triage a warning in the run's
   summary. Never in code scanning: workline posts it itself.
