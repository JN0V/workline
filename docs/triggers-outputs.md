---
sources:
  - defaults/ci/gitlab/workline.gitlab-ci.yml#variables
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:claude
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:bot
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:garden
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:apply
  - defaults/ci/github/workline.yml#jobs.propose.permissions
  - defaults/ci/github/workline-gardening.yml#jobs.propose
  - Dockerfile
  - docs/adr/0023-the-project-bot-keeps-a-gitlab-backlog.md#decision
  - docs/gitlab-trigger.md#what-premium-adds
checked: e81df7b34
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
git, jq, npm, `ssh-keygen` (from v0.38.0), and bubblewrap and socat,
which Claude Code's shell needs where the developer's agent runs
commands (from v0.69.0).
`workline version` in a job's log says which engine ran.

## GitLab tokens

The role and scopes each needs:

| Use | Token | Role | Scopes |
|---|---|---|---|
| apply on a merge request, gardening, sample | project access token (a bot of this project) | Developer | `api`, `write_repository` |
| the product owner only | project access token | Reporter (not Planner: it may not give a split's child its parent) | `api` |
| proposing jobs that read the forge | any of the above, or a narrower one | Reporter | `read_api` |
| several projects of a group | group access token (Premium on gitlab.com; every tier self-managed) | the same | the same |

A group access token is used the same way, as `GITLAB_TOKEN`; it has not
been tried with workline yet. `CI_JOB_TOKEN` reaches no issue: never enough
to write. Store the token as a masked variable, not protected if merge
requests from unprotected branches must apply.

## Exit codes and outputs

- **Exit codes**: what each command's code means, and what a CI job
  does with it ([exit codes](exit-codes.md)).
- **A job that fails says why in its log**: the run's summary, as an
  error annotation on GitHub, an `ERROR:` line on GitLab.
- **Outputs**: stdout, reports, the job's summary, the runs, and what
  goes on the forge ([what a job gives back](job-outputs.md)).
