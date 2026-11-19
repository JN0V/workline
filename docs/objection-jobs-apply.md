---
sources:
  - internal/engine/fromrun.go#ResumeFrom
  - internal/engine/fromrun.go#outOfLine
  - internal/engine/fromrun.go#issueFor
  - roles/tester/role.yaml
checked: c46a93d6c
verified: agent:claude-code
---
# What apply takes from the judge

Part of [An objection's jobs](objection-jobs.md): the judge ran the
Draft's code, so apply reads its run as data.

- The run's folder, named by the job, is checked by the engine too.
- `workline apply --from-merge-request` reads only its proposals: the
  repository, the roles and the forge are the job's own
  ([`workline apply`](commands.md#workline-apply)).
- Only what the Draft's line may do: its branch, its threads, the issue
  it was opened for (`workline/tester/issue-<n>`); anything else refuses
  the whole run.
- A branch moved on since the head judged applies nothing: the
  objection waits for the next run.
