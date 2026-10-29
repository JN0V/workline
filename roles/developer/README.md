---
sources:
  - roles/developer/role.yaml
  - ci/github/workline-develop.yml
  - internal/developer
checked: 492192ca
verified: agent:claude-code
---
# Developer

```mermaid
flowchart LR
  issue["<b>Reads</b><br/>the ready issues,<br/>in order"]
  role(["Developer"])
  claim["<b>Claims</b><br/>the first<br/>nobody took"]
  never["<b>Never</b><br/>an assigned issue,<br/>a merge"]
  code["<b>Writes</b><br/>the code,<br/>within bounds"]
  out["<b>Opens</b><br/>ready if green,<br/>else a Draft"]
  issue --> role --> claim --> code --> out
  role --> never
```

For people: what the role does and how to set it. The AI never reads this
file. All roles: [docs/roles.md](../../docs/roles.md). Where it stands:
[status](docs/status.md); its design:
[ADR-0054](../../docs/adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md).

**Does**, today (steps 1 to 6 of the
[build plan](../../docs/adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md#build-plan)):

- **Picks** the first ready issue in the backlog's order that nobody
  took: [the pick](docs/pick.md).
- **Claims** it, with an agent set: a label and a comment on the issue,
  the record the [auditor](../auditor/role.yaml) can read.
- **Writes the code**, with an agent, held to its bounds: a file
  outside the Scope, a frozen acceptance test, a file that steers
  agents or a gate's are refused, no agent judging
  ([the bounds](docs/bounds.md)). A test that does not fit, it objects
  to on the tester's Draft.
- **Proves it** on the project's tests, the tester's `test-command`,
  then opens its pull request: ready when green with no cap reached,
  else a Draft ([the proof and the caps](docs/proof.md)).
- **Opens it on the tester's Draft**, the acceptance tests with it,
  its code and tests in commits apart, its record hidden on it; the
  agent holds no forge token ([the pull request](docs/pull-request.md)).
- **Runs when routed**: when an issue is labelled ready, and on the
  schedule ([the triggers](docs/triggers.md)).
- **Fixes the review** of its pull request, a fresh run a verdict or
  a person's comment; the five whys at round three, a person past
  round five ([the review loop](docs/review-loop.md)).
- **With no agent**: names the issue it would take; claims nothing.

**Does not, yet**: the merge by kind (step 7), the measure (step 8).

**Never**: merge, approve, put `workline:accepted`, unassign a person,
close or move an issue.

## Run it

```sh
workline develop --ai none           # names the issue it would take
workline develop --ai claude         # claims it, writes the code; any agent: claude, cmd:<command>
workline develop --issue 42 --ai none
workline develop --merge-request 7 --ai claude   # a review round on its pull request
```

- **`--forge`**: GitHub, GitLab, the local forge or a command's; the
  project's `forge` setting by default.
- **In CI**: route it on `issue` and `schedule`; the GitHub and GitLab
  templates run it ([the triggers](docs/triggers.md)).

## Settings

In `.workline/config.yaml`, under `roles: {developer: {settings: …}}`:

| Setting | Default | What it does |
|---|---|---|
| `claim-label` | `workline:developing` | the label that says the developer took an issue |
| `ai-max-tokens` | 300000 | what a run's agent calls may spend; reached, a Draft |
| `ai-max-seconds` | 1200 | what a run waits on its agent |
| `proof-seconds` | 600 | how long the project's tests may run on its code; past it, a Draft |
| `review-rounds` | 5 | the review rounds it runs on its pull request |
| `person` | `workline:human` | the label its pull request gets past the last round |

The tests it runs are the tester's `test-command` and `test-report`
([the tester](../tester/README.md)).

## Cost

- **The pick, the bounds and the proof**: no agent, no token; one listing of the
  open issues and one of the open merge requests.
- **The code**: one agent call a run, `ai-max-tokens` at most.
- **The proof**: one run of the project's tests, `proof-seconds` at most.
