---
sources:
  - defaults/ci/github/workline.yml#jobs.propose.permissions
  - defaults/ci/github/workline.yml#jobs.propose.env
  - defaults/ci/github/workline-apply.yml#jobs.apply.permissions
  - defaults/ci/github/workline-gardening.yml#jobs.propose.permissions
  - defaults/ci/github/workline-gardening.yml#jobs.propose.env
  - defaults/ci/github/workline-gardening.yml#jobs.apply.permissions
  - defaults/ci/github/workline-sample.yml#jobs.read.permissions
  - defaults/ci/github/workline-sample.yml#jobs.read.env
  - defaults/ci/github/workline-sample.yml#jobs.write.permissions
  - defaults/ci/github/workline-apply.yml#on
  - defaults/ci/github/workline-apply.yml#jobs.follow.if
  - docs/github/import-jobs.md
  - internal/forge/local.go#Local
checked: b2a36983f
judged: cd97f28
verified: agent:claude-code
---
# workline on another forge, or none

```mermaid
%%{init: {"flowchart": {"wrappingWidth": 400}}}%%
flowchart LR
  subgraph forgejo["Forgejo or Gitea: the GitHub workflows"]
    reads["<b>The agent, read rights</b><br/>propose · gardening: propose<br/>sample: read · import: propose"]
    writes["<b>FORGEJO_TOKEN, no agent</b><br/>apply · gardening: apply<br/>sample: write · import: apply<br/>follow, on a push to main"]
    reads --> writes
  end
  subgraph local["No forge: no CI"]
    machine["<b>Your machine</b><br/>gardening, the sample"] --> files[".git/workline/<br/>local branches"]
  end
```

What the jobs are, on every forge: [ci.md](ci.md). GitHub and GitLab are
spoken natively: [ci-github.md](ci-github.md), [ci-gitlab.md](ci-gitlab.md).

## Gitea, Forgejo, Codeberg…

- **Plugged by a command** that answers a small contract
  ([forge-command.md](spec/forge-command.md)); Forgejo and Gitea have
  one ready: [workline on Forgejo](ci-forgejo.md).

## No forge

- **`forge: local`** keeps issues and merge requests in the clone:
  [workline with no forge](no-forge.md).
