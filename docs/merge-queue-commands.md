---
sources:
  - cmd/workline/main.go#routeCmd
  - cmd/workline/main.go#followCmd
  - internal/line/line.go#QueueInput
checked: 3f432ab14
verified: agent:claude-code
---
# The commands, for another CI

Part of [a merge queue or a merge train](merge-queue.md).

```sh
workline route merge-request --gates-only --forge none --input queue=true \
  --input range=<built-on>..<merged> --json > gates.json
workline route merge-request --ai none --forge none --input queue=true \
  --input range=<built-on>..<merged> --gates-from gates.json
```

- **`--input queue=true`**: the line runs its gates alone, and names
  the steps it left. Only `true` or `false`.
- **Refused with it**: `--verdict` and `--warn-only`.
- **One job is enough** when nothing else needs the gates apart: the
  second command without `--gates-from` runs them.
- **The yes on a group** (GitHub): `workline follow --forge github
  --merge-group <the group's branch> --head <its commit>`, with the
  App's token.

## Where it is read

`main.go` reads the flags of both commands; `line.go` builds the input of
`queue`.
