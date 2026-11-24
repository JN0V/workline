---
sources:
  - defaults/ci/github/workline-warn.yml#jobs.warn
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:warn:child
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:warn
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:warn:apply
  - defaults/ci/github/workline-apply.yml#on
  - defaults/ci/github/workline-apply.yml#jobs.apply.if
  - roles/inspector/code/inspector.go#prepare
checked: e81df7b34
verified: agent:claude-code
---
# The inspector's triage in CI

Part of [gates in CI](gates-in-ci.md): what the gates found, triaged
apart from the verdict.

With the inspector after the gates in the routing
([the inspector](../roles/inspector/README.md)):

- **Apart from the verdict**: the run of what only warns triages what the
  gates found the change adds, and posts it in a comment of its own once
  it answered; the verdict never waits for it.
- **The gates' own result**, never the gates run again beside the agent's
  token. On GitHub, workline-warn.yml waits for workline.yml's `gates`
  job on the same head, 15 minutes at most, and reads the result it left
  (`actions: read`); not on a draft, which asks no agent. On GitLab,
  `workline:warn` runs in a child pipeline started after
  `workline:gates`, whatever its verdict, and reads its `gates.json`
  with the bot's token.
- **No result found**: said in the job's log; the inspector triages
  nothing, and says so.
- **The lenses that warn wait too**: in the same job, they start once
  the gates' job ends.
