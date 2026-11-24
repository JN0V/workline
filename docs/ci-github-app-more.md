---
sources:
  - defaults/ci/github/workline-apply.yml#jobs.apply
  - defaults/ci/github/workline-merge.yml#on
  - internal/forge/own.go#github.own
checked: e81df7b34
verified: agent:claude-code
---
# The GitHub App: more ways

Part of [the GitHub App and its key](ci-github-app.md#set-it-up).

The same with `gh`, by an admin of the repository:

```sh
gh api -X PUT repos/OWNER/REPO/environments/workline-app \
  -F 'deployment_branch_policy[protected_branches]=false' \
  -F 'deployment_branch_policy[custom_branch_policies]=true'
gh api -X POST repos/OWNER/REPO/environments/workline-app/deployment-branch-policies \
  -f name=main -f type=branch
gh secret set WORKLINE_APP_PRIVATE_KEY --env workline-app -R OWNER/REPO < app.pem
gh secret delete WORKLINE_APP_PRIVATE_KEY -R OWNER/REPO    # the repository's own
```

- **Environments** need a public repository, or GitHub Pro, Team or
  Enterprise for a private one. Without, the key stays a repository
  secret, in reach of any branch's workflow, the merge policy's statuses
  with it.
- **The release tool's branch** (release-please's
  `release-please--branches--main`), optional, to keep others off it:
  Settings → Rules → New branch ruleset, target `release-please--*`,
  "Restrict creations", "Restrict updates" and "Restrict deletions", the
  App in the bypass list.
- **Without an App**, the jobs use their own token: the checks of a fix
  wait for someone to approve them, and gardening needs Settings →
  Actions → General → "Allow GitHub Actions to create and approve pull
  requests". The records are then read from `github-actions[bot]`'s
  comments.

