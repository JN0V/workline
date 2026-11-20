---
sources:
  - defaults/ci/github/workline-fork.yml
  - defaults/ci/github/workline-apply.yml#jobs.apply.if
  - defaults/ci/github/workline.yml#jobs.gates
  - internal/gate/gate.go#overThreshold
  - internal/gate/gate.go#first
checked: e2e165c2d
verified: agent:claude-code
---
# Gates on a fork's pull request

Part of [gates in CI](gates-in-ci.md): what a fork's run gets, and why.

On GitHub, a fork's run gets a read-only token and no secret
([GitHub's rule](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#workflows-in-forked-repositories)).
The gates still run; what they find is told, not shown beside the lines.

- **What it gets**:
  - the jobs `gates` and `propose` as checks, red when a gate counts what
    the change adds;
  - one comment, the propose job's report, from workline-fork.yml: each gate's
    verdict, its new findings counted, and the first five places of
    those over the gate's `max` (`src/auth/clean.go:10 errcheck`);
  - the run's file `workline-gates`, kept a day: every new finding, its
    file, line, rule and message (`gh run download <run> -n workline-gates`).
- **What it does not get**:
  - no check `workline gates`, nothing beside a line, no summary comment
    by rule: workline-apply.yml applies nothing from a fork's run;
  - no triage: the inspector needs the agent's token, which a fork's run
    never gets.
- **Why**: annotations need a token that writes checks. The job `gates`
  runs the fork's code, a linter's config, a plugin, a build: it never
  holds one. workline-fork.yml, which reads the fork's run from your
  side, only comments.
- **What you can do**:
  - require the check `gates` in your branch's rules: a change a gate
    blocks cannot merge, from a fork too. The run is the fork's copy of
    workline.yml: read a change to `.github/workflows/` before trusting
    its checks;
  - choose whose runs wait for your approval, first-time contributors'
    or every external contributor's (Settings → Actions → General,
    [approval for fork runs](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository#controlling-changes-from-forks-to-workflows-in-public-repositories));
  - never `pull_request_target`, nor a private repository's "Send write
    tokens to workflows from pull requests": both put a write token
    beside the fork's code
    ([why](https://securitylab.github.com/resources/github-actions-preventing-pwn-requests/)).
- **On GitLab**: a fork's merge request runs in the fork, with its
  variables, none of yours; its Code Quality report is the fork's
  pipeline's. Whether the merge request's widget shows it is not tried
  yet.
