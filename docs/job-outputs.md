---
sources:
  - internal/line/line.go#Result
  - internal/engine/engine.go#Result
  - internal/agent/agent.go#Call
  - internal/engine/engine.go#callAgent
  - internal/report/report.go#Write
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:propose.artifacts
  - internal/report/summary.go#AppendSummary
  - internal/report/summary.go#Summary
  - internal/engine/engine.go#RunsDir
  - internal/forge/forge.go#Forge
  - internal/forge/review.go#LineReviewer
  - internal/engine/engine.go#pushToMergeRequest
  - internal/engine/engine.go#branchPrefix
  - internal/backlog/ledger.go#LabelProposed
  - internal/sample/apply.go#IssueTitle
  - internal/backlog/opening.go#LabelTriage
checked: e2e165c2d
verified: agent:claude-code
---
# What a job gives back

Part of [jobs' needs and outputs](triggers-outputs.md).

- **stdout**: the verdict; with `--json`, status, summary, findings, each
  agent call with its model and tokens, and for a line its steps, their
  seconds, its moment and the runs `pending` for `apply`; on the line of
  states, the moves left to `apply` (`states`), and with `--input
  reconcile=list` the issues due (`due`) and review rounds (`reviews`).
- **Reports**: `--sarif <file>` (GitHub code scanning), `--code-quality
  <file>` (GitLab: `artifacts: reports: codequality`, shown on the merge
  request, every tier). The paths are yours to choose.
- **Job summary**: `--summary <file>`: the verdict, each step's verdict, findings and seconds, what
  was applied and what waits to be, in Markdown, or HTML for a `.html` file,
  added to the file ([ADR-0035](adr/0035-the-engine-writes-the-jobs-summary.md)).
  GitHub: `$GITHUB_STEP_SUMMARY`; GitLab: an artifact the job's page links
  ([ci.md](ci.md#what-a-job-shows)).
- **Runs**: `$WORKLINE_RUNS_DIR` (else `.git/workline/runs/`), each with
  `out/calls.jsonl`, the agent's answer and what was refused.
- **On the forge**: a comment on the merge request (the documentalist's list
  for a person, the reviewer's summary), edited on each run; the
  reviewer's findings on their lines, in the forge's review; doc fixes
  as a diff in a comment, nothing pushed to its branch; gardening's
  merge requests on `workline/<role>/<task>`; issues (the product owner's comment and
  label `workline:proposed` on each issue it proposes on; the sample's
  tracking issue, of every role's acts; a reviewer's
  finding outside
  the change, `needs-triage`) and labels.
