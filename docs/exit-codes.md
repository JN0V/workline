---
sources:
  - cmd/workline/main.go#exitFor
  - cmd/workline/cli.go#parseFlags
  - cmd/workline/cli.go#misuse
  - cmd/workline/main.go#gateCmd
  - cmd/workline/main.go#sampleCmd
  - internal/sample/sample.go#draw
  - cmd/workline/main.go#doctorPrint
  - internal/doctor/doctor.go#Run
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:propose
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:apply
  - defaults/ci/github/workline-apply.yml#jobs.follow.if
checked: e81df7b34
verified: agent:claude-code
---
# Exit codes

Part of [jobs' needs and outputs](triggers-outputs.md).

| Code | `route`, `run-role`, `review`, `apply` | In CI |
|---|---|---|
| 0 | `pass` | go on |
| 1 | `block`, or an error | fail the job |
| 2 | `human`: a person decides | fail, or allow and read the comment |
| 3 | `blocked-external`: the agent or the forge failed | a warning; run again later |
| 64 | the command was misused | fix the job |

An unknown option exits 64 too, never 2. `workline gate` exits 0 or 1;
`workline sample` exits 2 when a doc or an act is left for a person and 3
when the judge or the forge cannot be reached; `workline doctor`, 1 on an error only. The
templates keep a job's status and still apply what passed, and what a
role that blocks names to say why, such as its comments
([`on-block`](spec/role-contract.md#one-run); `artifacts: when: always`,
`if: always()`, `when: always`).

