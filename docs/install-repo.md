---
sources:
  - cmd/workline/main.go#initCmd
  - cmd/workline/main.go#prePush
  - cmd/workline/main.go#prePushLine
  - cmd/workline/main.go#reviewCmd
  - cmd/workline/main.go#runRole
checked: e2e165c2d
verified: agent:claude-code
---
# Adopt a repository

Part of [install](install.md).

```sh
cd your-repo
workline init      # routes pre-push; proposes each doc's sources, for you to review and commit
```

- From then on, each push runs the committer and the documentalist on the
  commits leaving, with no agent: one line counts the docs they made
  suspect.
- Until a doc names its `sources`, nothing tells when it goes wrong.
- `--review` puts the reviewer on each merge request too; `workline review`
  reads a branch before you push it.
- `--human-po yes` says a person is the project's Product Owner: the
  product owner role then proposes what sets direction rather than doing it
  ([ADR-0038](adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)).

To start the product owner on one open issue:
`workline run-role product-owner --event issue --target issue:<n> --forge github`
([try each role once](quickstart.md#try-each-role-once-on-a-real-repository)).

Step by step, in ten minutes: [the quickstart](quickstart.md). With no
forge, the whole line from a need to a merge:
[the line with no forge](local-line.md).

