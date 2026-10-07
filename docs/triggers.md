---
sources: [cmd/workline, internal/line, internal/hooks, internal/forge/gitlab.go, ci/gitlab/workline.gitlab-ci.yml, ci/github/workline.yml, ci/github/workline-gardening.yml, ci/github/workline-sample.yml, routing.default.yaml]
checked: 13d335d
verified: agent:claude-code
---
# Running each role from any trigger

workline listens for nothing: whatever starts the work — a git hook, a
scheduler, a CI pipeline, an internal tool — runs one command, the same on
a laptop and in a job. The CI templates ([ci.md](ci.md)) are these commands
wrapped; this page gives them bare. GitLab pipelines started by the API, a
trigger token or another pipeline: [gitlab-trigger.md](gitlab-trigger.md).

## On your machine

| What | Command | Agent |
|---|---|---|
| each commit's message | the global `commit-msg` hook (`workline hooks install --global`) | yours, to rewrite a refused message |
| before a push | the global `pre-push` hook, when the project routes `pre-push`; by hand: `workline route pre-push --ai none --input range=origin/main..HEAD` | never |
| docs made suspect | `workline docs` (`--since <rev>`; `--review` with no agent) | yours |
| a branch's code | `workline review` (`--base`, `--lenses`, `--json`) | yours |
| a spec, before it is built | `workline review --spec <file>` or `--issue <n>` | yours |
| gardening | `workline route schedule` (`--forge local --open-merge-request` keeps its merge requests as local branches) | yours |
| the backlog | `workline run-role product-owner --event schedule --forge gitlab` | yours |
| a roadmap file to issues | `workline issues import <file>`, then the same with `--apply`; in CI, `--json > line.json`, then `workline apply --line line.json` | yours |
| before tagging a release | `workline route release` | yours |
| the weekly sample | `workline sample --out s.json`, then `workline sample --apply s.json` | the judge |

`--ai none` on any of them runs every check with no agent. With `--forge
gitlab` on a laptop, the token is `GITLAB_TOKEN`, else glab's; the project,
`GITLAB_HOST` and the `origin` remote.

## In CI: one command per event

Each event that writes runs in two jobs (ADR-0016, principle 6): one judges
with the agent and no write token (`--no-apply`), one applies with the
write token and no AI key. `WORKLINE_RUNS_DIR` names where the runs are
kept; carry that folder and the `--json` result from the first job to the
second, at the same path.

**Merge request** — the committer, the documentalist, the reviewer if
routed:

```sh
workline route merge-request --ai "$ai" --no-apply --push-to-merge-request \
  --forge gitlab --target merge-request:<iid> --branch <source-branch> \
  --input range=<base-sha>..<head-sha> --code-quality gl-code-quality.json --json > line.json
workline apply --line line.json          # second job: commits the doc fixes to the branch
```

`--input lenses=all` has the reviewer read every lens instead of one.

**Gardening** — the `schedule` line (the documentalist; the product owner
when routed), one merge request per task:

```sh
git checkout -q -B <branch>              # CI checks out a detached HEAD
workline route schedule --ai "$ai" --no-apply --forge gitlab --open-merge-request --json > line.json
workline apply --line line.json
```

**The product owner alone**, on its own trigger:

```sh
workline run-role product-owner --event schedule --ai "$ai" --no-apply --forge gitlab --json > po.json
workline apply "$(jq -r '."run-dir"' po.json)"
```

**Release** — before your release tool tags; a non-zero exit holds it.
On a machine, `workline route release`; in CI, `workline route release
--ai "$ai" --no-apply --forge gitlab --open-merge-request --json >
line.json`, then `workline apply --line line.json`, puts the docs' fix on
a merge request of its own (untried this way).

**The release fix, on a push to the default branch** — `workline follow
--base <branch> --forge gitlab` (token, no agent) rebuilds that merge
request, `workline/documentalist/release`, on the branch's new tip, so it
stays mergeable as the branch moves (ADR-0034); `user.name` and
`user.email` set first, as for `apply`.

**Weekly sample** — `workline sample --out sample.json` (agent, writes
nothing; `--week 2026-W40`), then `workline sample --apply sample.json
--forge gitlab` (token, no agent), which also draws the product
owner's acts of the week from its report (ADR-0033).

**Import** — `workline issues import <file> --apply` has the agent and the
write token in one process: run it by hand. In a pipeline, split it as
any other role: `workline issues import <file> --json > line.json`
(agent, a read token, writes nothing), then `workline apply --line
line.json` (write token, no agent); [ci.md](ci.md#importing-a-file).

`--sarif <file>` beside `--code-quality` writes the findings for GitHub's
code scanning. The `gate:<name>` steps of a line, or `workline gate
<name>`, run your own scanners.

## What each job needs

| Job | Clone | Variables | Tools |
|---|---|---|---|
| judging (`--no-apply`, `sample --out`) | the whole history (`GIT_DEPTH: 0`, `fetch-depth: 0`), the base commit fetched | the agent's token (`CLAUDE_CODE_OAUTH_TOKEN` from `claude setup-token`, or `ANTHROPIC_API_KEY`); none is fine: `--ai none` | the engine, gitleaks, the `claude` CLI (npm) |
| judging gardening, the product owner, or a merge request with the reviewer | as above | a token that reads issues, merge requests and their notes: `GITLAB_TOKEN` with `read_api` (`CI_JOB_TOKEN` reads no issue); on GitHub `GH_TOKEN`, `issues: read`, `pull-requests: read` | as above |
| applying (`apply`, `sample --apply`) | a branch, not a detached HEAD, for gardening | `GITLAB_TOKEN` (below); a git identity (`user.name`, `user.email`); `origin` with the token, to push | the engine |

The image `ghcr.io/jn0v/workline:<version>` holds the engine, gitleaks,
git, jq and npm. `workline version` in a job's log says which engine ran.

**GitLab tokens**, the role and scopes each needs:

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
`workline sample` exits 2 when a doc is left for a person and 3 when the
judge cannot be reached; `workline doctor`, 1 on an error only. The
templates keep a job's status and still apply what passed, and the
comment of a reviewer that blocks (`artifacts: when: always`,
`if: always()`, `when: always`).

## Outputs

- **stdout**: the verdict; with `--json`, status, summary, findings, each
  agent call with its model and tokens, and for a line its steps and the
  runs `pending` for `apply`.
- **Reports**: `--sarif <file>` (GitHub code scanning), `--code-quality
  <file>` (GitLab: `artifacts: reports: codequality`, shown on the merge
  request, every tier). The paths are yours to choose.
- **Job summary**: `--summary <file>` (`route`, `run-role`, `apply`,
  `sample`): the verdict, each step's verdict and findings, what was
  applied and what waits to be, in Markdown, or HTML for a `.html` file,
  added to the file ([ADR-0035](adr/0035-the-engine-writes-the-jobs-summary.md)).
  GitHub: `$GITHUB_STEP_SUMMARY`; GitLab: an artifact the job's page links
  ([ci.md](ci.md#what-a-job-shows)).
- **Runs**: `$WORKLINE_RUNS_DIR` (else `.git/workline/runs/`), each with
  `out/calls.jsonl`, the agent's answer and what was refused.
- **On the forge**: a comment on the merge request (the documentalist's list
  for a person, the reviewer's summary), edited on each run; doc fixes
  committed to its branch (`Workline-Role: documentalist`); gardening's
  merge requests on `workline/<role>/<task>`; a release's fix on
  `workline/<role>/release`; issues (the product owner's report, "Backlog —
  product owner"; the sample's tracking issues, the docs' and the product
  owner's acts'; a reviewer's finding outside
  the change, `needs-triage`) and labels.
