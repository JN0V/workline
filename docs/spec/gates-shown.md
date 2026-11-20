---
sources:
  - internal/report/report.go#gateItem
  - internal/report/report.go#Item.level
  - internal/report/report.go#Item.rule
  - internal/report/report.go#Item.fingerprint
  - internal/report/report.go#CodeQuality
  - internal/report/report.go#ForCodeScanning
  - internal/report/report.go#maxAnnotations
  - internal/report/gates.go#placesShown
  - internal/report/gates.go#Gates
  - internal/report/gates.go#GatesShown.note
  - internal/report/gates.go#GatesShown.Comment
  - internal/report/gates.go#inRepo
  - internal/report/gates.go#inert
  - internal/forge/annotate.go#GatesCheck
  - internal/forge/annotate.go#GatesMarker
  - internal/forge/forge.go#Marker
  - internal/forge/annotate.go#annotationsPerCall
  - internal/engine/annotate.go#ShowGates
  - cmd/workline/main.go#showGates
  - cmd/workline/main.go#fromMergeRequest
  - defaults/ci/github/workline-apply.yml#jobs.apply
checked: e2e165c2d
verified: agent:claude-code
---
# Gates — the new findings shown on a merge request

What [gates.md](gates.md#shown-on-the-merge-request) says in short: where
the findings a change adds go, and how
([ADR-0049](../adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md),
[ADR-0047](../adr/0047-findings-on-a-merge-request-open-no-conversation.md)).

## A gate step, and the gates alone

[A gate step's result, and the gates alone](gates-alone.md):

- **What a gate step holds**: on a merge request, each check's count
  and each new finding, named by a fingerprint holding no line number.
- **The gates alone** (`--gates-only`), in a job with no write token;
  the proposing job reads their verdicts (`--gates-from`).

## Where they go

- **GitLab, Code Quality** (`--code-quality`): each new finding, its
  `check_name` `gate:<gate>/<check>/<rule>`, its severity the tool's
  level (`error` major, `warning` minor, `note` info), its fingerprint
  the one above. GitLab compares the report with the target branch's.
- **GitHub, a check and one comment** (`apply --gates <file>
  --from-merge-request <n> --head <sha>`), in the job holding the write
  token and no AI key:
  - the annotations of the check `workline gates` on the head judged,
    fifty a call, 500 at most, the rest counted;
  - one summary comment (`<!-- workline:sticky=gates/summary -->`),
    edited on each push: a row per check, a line per rule with its first
    three places, as many rules as the reviewer's `findings-max`. On
    the developer's pull request, it wakes a review round, which asks
    each line ([the review loop](../../roles/developer/docs/review-loop.md#a-round)).
- **Never code scanning** on a merge request: `--sarif` leaves the gates'
  findings out, and the inspector's triage of them.
- **Nothing to say**: with no new finding and every check passing, no
  check is written; the comment is only edited where an earlier push
  left one.
- **A check with `fix`**: its findings are one line of the comment, the
  command with them, and no annotation.

## The run's data

- **Its places** must be inside the repository; its texts are cut and
  their mentions defused.
- **The merge request it names** must have the head judged as its head,
  else nothing is shown (`moved-on`).
- **Said, never an error**: `gates-shown`, `gates-not-shown` (a refused
  check, a comment not written), `moved-on`.
