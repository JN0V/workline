---
sources:
  - internal/forge/local.go#Local
  - internal/forge/local.go#InCI
  - internal/forge/local.go#Local.writable
  - internal/forge/local.go#Local.SettleMerges
  - internal/engine/done.go#settleMerges
  - internal/forge/forge.go#Open
  - internal/forge/forge.go#Missing
  - cmd/workline/main.go#localRange
  - cmd/workline/main.go#followCmd
  - cmd/workline/main.go#sampleCmd
  - cmd/workline/main.go#docsJudgedRef
  - cmd/workline/main.go#docsJudgedFrom
  - defaults/ci/github/workline-gardening.yml#jobs.propose
checked: e2e165c2d
verified: agent:claude-code
---
# workline with no forge

Part of [workline on another forge, or none](ci-other-forges.md).

A project with no forge — pushed to `main`, no merge request — keeps what
workline writes in its own clone:

```yaml
forge: local
```

- Issues and merge requests are files under `.git/workline/`, never
  committed; gardening's merge requests are local branches, nothing
  pushed.
- `workline issues` reads them, and writes a person's acts
  ([write the need](local-line.md#3-write-the-need)); you merge a
  branch with git.
- After that merge, `workline follow` closes the issues the merge request
  says it closes (`Closes #<n>`), their state's label taken off, as a
  forge does.
- Each role's run, its Draft or merge request:
  [run each step](local-line.md#4-run-each-step).
- `workline route merge-request --target merge-request:<n>`, its branch
  checked out, judges its base to its branch: no `range` to give.
- No CI to set up: gardening and the [weekly sample](ci.md#the-weekly-sample)
  run on your machine — `workline route schedule --open-merge-request`,
  then `workline follow --gardening` to rebuild or close its earlier
  branches, `workline sample --out sample.json` then `workline sample
  --apply sample.json` — and `workline docs` judges before a release.
- In CI — `CI`, `GITHUB_ACTIONS` or `GITLAB_CI` set — the local forge
  refuses its writes, loud: the job's clone is thrown away, and what it
  would hold with it. What writes nothing still runs, so a project whose
  config says `local` for its laptops passes `--forge` to its CI jobs that
  write.
- With `forge: none` (the default), nothing is written to a forge: a write
  that needs one — the issue a role opens included — is refused and says
  so, naming `forge: local`.
