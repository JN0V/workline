---
sources: [cmd/workline, internal/agent/agent.go]
checked: beb53f8
verified: agent:claude-code
---
# Take only a part

workline is one binary with its roles inside, and needs nothing but git (and
`gh` to reach GitHub; GitLab is reached through its API). Any existing pipeline can call one role, and
leave the rest:

```sh
workline run-role committer --event merge-request --ai none \
  --input range=origin/main..HEAD --json    # exit code: 0 pass, 1 block, 2 human, 3 external
workline run-role documentalist --event schedule --ai none --json
workline gate release --json   # a gate declared in .workline/config.yaml: your scanners' SARIF, with thresholds
```

- The exit code carries the verdict; `--json` gives the findings to whatever
  reads them.
- The global hook hands over to the hooks that were there before; nothing
  stops running.
- `--no-apply` and `workline apply` fit a pipeline that keeps tokens apart.
- A role is a folder (`role.yaml`, facets, `pre` and `post` in any language):
  a team can replace one facet (`.workline/roles/<role>/`), or run its own
  roles with `--roles <dir>` ([role contract](spec/role-contract.md)).

The command for each event, from any trigger: [triggers.md](triggers.md).

Not there yet: an agent other than Claude Code built in (`--ai cmd:<command>`
runs any).
