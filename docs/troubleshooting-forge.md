---
sources:
  - internal/forge/local.go#InCI
  - internal/doctor/variables.go#tokensScoped
  - defaults/ci/github/workline-apply.yml#jobs.apply.permissions
checked: e81df7b34
verified: agent:claude-code
---
# Troubleshooting: forge writes refused

Part of [troubleshooting](troubleshooting.md).

- `forge: none` (the default) refuses every write that needs a forge, the
  issue a role opens included, and says so.
- `forge: cmd:<command>`: `workline doctor` runs it once, writing
  nothing, and warns when it is not found, fails or does not answer
  ([what the doctor checks](spec/forge-command.md#what-the-doctor-checks)).
- `forge: local` refuses its writes in CI (`CI`, `GITHUB_ACTIONS` or
  `GITLAB_CI` set): pass `--forge github` or `--forge gitlab` to jobs that
  write.
- GitLab: `CI_JOB_TOKEN` reaches no issue; give `GITLAB_TOKEN`
  ([ci-gitlab.md](ci-gitlab.md#gitlabcom)). A **protected** variable never reaches a merge
  request's pipeline on an unprotected branch.
- GitLab, `tokens-not-scoped`: a token left to every environment reaches
  the jobs running your linters; scope it to `workline`.
  `tokens-scope-unread`: listing the variables needs a Maintainer's
  token. A job saying `WORKLINE_GITLAB_TOKEN is not set, or not scoped`:
  the variable is scoped elsewhere, or the template's copy is older than
  its environment ([the environment](gitlab/environment.md)).
- `findings-not-annotated`, on GitHub: the check `workline findings` was
  refused. Give `workline-apply.yml`'s `apply` job `checks: write`, and a
  GitHub App the permission **Checks**, read and write
  ([the App](ci-github-app.md)). The findings stay in the propose job
  summary.
- `gates-not-shown`, on GitHub: the check `workline gates` or the gates'
  summary comment was refused, the same rights; the findings stay in the
  gates' job summary. `moved-on`: the branch moved on before the apply,
  and its next run shows its own ([gates in CI](gates-in-ci.md)).
- A gate's `check-error` in `propose`, "the gates' job left no result": the
  job `gates` did not run it, or failed first; read its log. Copy
  `workline.yml` of this release again if it has no job `gates`.
