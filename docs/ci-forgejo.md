---
sources:
  - defaults/ci/forgejo/workline-forge.sh
  - defaults/ci/forgejo/workline_forge_test.go#mock
  - docs/spec/forge-command.md#what-stays-with-the-engine
  - defaults/ci/github/workline.yml#concurrency
  - internal/sample/draw.go#actsOfTheWindow
  - internal/forge/command.go#command.MergeState
  - internal/forge/command.go#command.AutoMerge
  - internal/forge/command.go#command.Guarded
  - internal/forge/command.go#command.Accept
  - internal/forge/command.go#command.KeepVerdict
  - internal/forge/command.go#command.Verdict
  - internal/forge/merge.go#App
  - internal/forge/merge.go#Closing
  - internal/forge/merge.go#command.mark
  - internal/forge/merge.go#command.marked
checked: b2a36983f
verified: agent:claude-code
---
# workline on Gitea, Forgejo, Codeberg…

Part of [workline on another forge, or none](ci-other-forges.md).

Another forge is plugged by a command that answers its small contract, one
JSON request per operation ([forge-command.md](spec/forge-command.md)).

- **Forgejo and Gitea**, whose API has GitHub's shape: copy
  [defaults/ci/forgejo/workline-forge.sh](../defaults/ci/forgejo/workline-forge.sh) into the
  repository and name it in `.workline/config.yaml`:

  ```yaml
  forge: 'cmd:sh defaults/ci/forgejo/workline-forge.sh'
  ```
- It needs `curl` and `jq`, and:
  - `FORGEJO_URL`;
  - `FORGEJO_TOKEN`: issues and pull requests, write; user, read: its
    user is workline's own account, whose comments keep the records a
    role reads back
    ([ADR-0051](adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md));
    its user an administrator of the repository: reading the comments
    asks who of their authors may write;
  - `FORGEJO_REPO` (`owner/name`), unless `origin` names it.
- The engine pushes the branches itself, with the job's git credentials;
  the script does the rest.
- **The jobs**: Forgejo Actions reads, largely, GitHub's workflow syntax.
  The [GitHub templates](ci-github.md) are a start: `--forge github`
  replaced by the command, the GitHub App's steps by `FORGEJO_TOKEN`.
  Check that your version honours `concurrency`, the workflow's and the
  jobs' ([what they hold](github/jobs.md)).
- **Not tried yet** on a live instance — the script was run against a mock
  of the API only: tell us what differs.
- **The [weekly sample](ci.md#the-weekly-sample)** draws the product
  owner's acts there, not the reviewer's, the tester's nor the developer's: the contract lists no
  merge request merged. It says so each week.
- **The [merge policy](merge-policy.md)** needs `merge-state`,
  `auto-merge`, `guarded` and the verdict's operations — at `normal`,
  `app`, `closing`, `mark` and `marked` too —, which the sample does not answer yet: said,
  nothing turned on. The local forge merges itself
  ([the merge](local-line-merge.md)).
- A forge of another shape: write the command the contract asks.
