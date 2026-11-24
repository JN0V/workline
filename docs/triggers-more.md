---
sources:
  - cmd/workline/main.go#runRole
  - cmd/workline/main.go#sampleCmd
  - cmd/workline/main.go#followCmd
  - cmd/workline/main.go#applyCmd
  - cmd/workline/main.go#importIssues
  - cmd/workline/main.go#gateCmd
checked: e81df7b34
verified: agent:claude-code
---
# More triggers

Part of [running each role from any trigger](triggers.md): the product
owner alone, a release, the weekly sample, an import and your own
scanners, each as a bare command. `$ai` is your choice of agent: `none`,
`claude`, `cmd:<command>` ([usage](usage.md)).

## Release

Before your release tool tags; a non-zero exit holds it.

- On a machine: `workline route release`.
- At a tag (GitLab's `workline:release`): `workline route release --ai
  none --forge none`, no token. The jobs that publish need it.
- On each schedule, a dry run: `workline route release --dry-run --forge
  none --json > dry-run.json` (no agent, no token, writes nothing), then
  `workline apply --release-dry-run dry-run.json --forge gitlab` (token,
  no agent) ([the release](release.md)).
- On a push to the default branch: `workline follow --base <branch>
  --forge gitlab` (token, no agent), `user.name` and `user.email` set
  first, as for `apply`. It closes the tester's Draft of a closed
  issue, and takes the states' labels off closed issues. With `merge:`, it runs
  the [merge policy](merge-policy.md): after each proposing job too, given the
  head's verdict `route --verdict` wrote (`--verdict <file>`).

## Weekly sample

`workline sample --forge gitlab --out sample.json` (agent, reads the
forge, writes nothing; `--week 2026-W40`): every role's acts drawn
([ADR-0046](adr/0046-the-auditor-re-checks-every-roles-acts.md)). Then `workline sample --apply
sample.json --forge gitlab` (token, no agent), which writes the week on
one issue, the level the product owner's acts suggest among it
([ADR-0033](adr/0033-the-weekly-sample-draws-the-product-owners-acts.md)).

## Import

- `workline issues import <file> --apply` has the agent and the write
  token in one process: run it by hand.
- In a pipeline, split it as any other role: `workline issues import
  <file> --json > line.json` (agent, a read token, writes nothing), then
  `workline apply --line line.json` (write token, no agent);
  [ci.md](ci.md#importing-a-file).

## Your own scanners

`--sarif <file>` beside `--code-quality` writes every step's findings
for GitHub's code scanning.
`gate:<name>` steps, or `workline gate <name>`, run your own scanners; on
a merge request given its `range`, or with `workline gate <name> --base
<rev>`, only the findings the change adds count
([new findings](spec/gates.md#on-a-merge-request-the-findings-the-change-adds)).

## The product owner alone

On its own trigger:

```sh
workline run-role product-owner --event schedule --ai "$ai" --no-apply --forge gitlab --json > po.json
workline apply "$(jq -r '."run-dir"' po.json)"
```

