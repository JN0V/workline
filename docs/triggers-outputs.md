---
sources: [cmd/workline, internal/line, internal/hooks, internal/forge/gitlab.go, ci/gitlab/workline.gitlab-ci.yml, ci/github/workline.yml, ci/github/workline-gardening.yml, ci/github/workline-sample.yml, routing.default.yaml]
checked: 87a81d4a
verified: agent:claude-code
---
# Jobs' needs and outputs

What a CI job running workline needs, and what it gives back. The
commands each event runs: [running each role from any trigger](triggers.md).

## What each job needs

| Job | Clone | Variables | Tools |
|---|---|---|---|
| judging (`--no-apply`, `sample --out`) | the whole history (`GIT_DEPTH: 0`, `fetch-depth: 0`), the base commit fetched | the agent's credentials, whichever agent you chose (Claude Code: `CLAUDE_CODE_OAUTH_TOKEN` from `claude setup-token`, or `ANTHROPIC_API_KEY`); none is fine: `--ai none` | the engine, gitleaks, your agent's CLI (Claude Code: `claude`, from npm) |
| judging gardening, the product owner, a merge request with the reviewer, or the sample | as above | a token that reads issues, merge requests and their notes: `GITLAB_TOKEN` with `read_api` (`CI_JOB_TOKEN` reads no issue); on GitHub `GH_TOKEN`, `issues: read`, `pull-requests: read`; on a release's, `checks`, `statuses` and `actions` read too ([what holds a release](release-holds.md)) | as above |
| applying (`apply`, `sample --apply`) | a branch, not a detached HEAD, for gardening | `GITLAB_TOKEN` (below); a git identity (`user.name`, `user.email`); `origin` with the token, to push; git set to sign where the forge refuses unsigned commits ([how](gitlab/signed-commits.md)) | the engine; `ssh-keygen` to sign with an SSH key |

The image `ghcr.io/jn0v/workline:<version>` holds the engine, gitleaks,
git, jq, npm and, from v0.38.0, `ssh-keygen`.
`workline version` in a job's log says which engine ran.

## GitLab tokens

The role and scopes each needs:

| Use | Token | Role | Scopes |
|---|---|---|---|
| apply on a merge request, gardening, sample | project access token (a bot of this project) | Developer | `api`, `write_repository` |
| the product owner only | project access token | Reporter (not Planner: it may not give a split its parent) | `api` |
| judging jobs that read the forge | any of the above, or a narrower one | Reporter | `read_api` |
| several projects of a group | group access token (Premium on gitlab.com; every tier self-managed) | the same | the same |

A group access token is used the same way, as `GITLAB_TOKEN`; it has not
been tried with workline yet. `CI_JOB_TOKEN` reaches no issue: never enough
to write. Store the token as a masked variable, not protected if merge
requests from unprotected branches must apply.

## Exit codes

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

## Outputs

- **stdout**: the verdict; with `--json`, status, summary, findings, each
  agent call with its model and tokens, and for a line its steps, their
  seconds, its moment and the runs `pending` for `apply`.
- **Reports**: `--sarif <file>` (GitHub code scanning), `--code-quality
  <file>` (GitLab: `artifacts: reports: codequality`, shown on the merge
  request, every tier). The paths are yours to choose.
- **Job summary**: `--summary <file>` (`route`, `run-role`, `apply`,
  `sample`): the verdict, each step's verdict, findings and seconds, what
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
  tracking issues, the docs' and the product owner's acts'; a reviewer's
  finding outside
  the change, `needs-triage`) and labels.
