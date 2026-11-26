---
sources:
  - internal/engine/engine.go#scriptEnv
  - internal/engine/engine.go#AIWaitsReady
  - internal/engine/engine.go#DraftInput
  - internal/forge/retry.go#LineEnv
  - internal/forge/retry.go#ProjectEnv
  - internal/forge/retry.go#RateWaitEnv
  - internal/forge/retry.go#RunsDirEnv
  - internal/engine/engine.go#writeWaiting
  - internal/engine/engine.go#writeHeld
  - internal/engine/engine.go#releaseRequest
  - internal/engine/engine.go#Result
  - internal/engine/engine.go#Run
  - internal/line/line.go#runRole
checked: e81df7b34
verified: agent:claude-code
---
# A role's variables

Part of [the environment](environment.md).

A role's `pre` and `post` receive:

- always: `WORKLINE_RUN_DIR`, `WORKLINE_EVENT`, `WORKLINE_AI`,
  `WORKLINE_ROLE`, `WORKLINE_BIN`, `WORKLINE_ROLES_DIR` (the folder of
  roles) and `WORKLINE_REPO` (the repository the run works in);
- `WORKLINE_RANGE` when the run is given a commit range
  (`--input range=...`);
- `WORKLINE_FORGE` when a forge is given, `WORKLINE_TARGET`
  (`merge-request:12`) with a target too;
- `WORKLINE_AI_WAITS=ready` on a draft (`--input draft=true`): the
  agent set aside, `WORKLINE_AI` is `none`
  ([ADR-0050](adr/0050-each-act-has-its-moment-and-a-wait-budget.md#4-what-the-table-holds-to));
- `WORKLINE_OPEN_MERGE_REQUESTS`, `WORKLINE_OPEN_MERGE_REQUEST_TASKS` and
  `WORKLINE_PROPOSED_TASKS` with `--open-merge-request`, and
  `in/waiting.yaml`, the files each waiting task changes, and
  `in/held.yaml`, those each other open merge request changes;
- on a release tool's merge request: `WORKLINE_RELEASE_BRANCH`, the
  branch, to every role ([ADR-0017](adr/0017-the-release-manager.md)); the reviewer
  leaves the versions the tool bumps
  ([#384](https://github.com/JN0V/workline/issues/384));
- run by a handoff: the inputs `handoff-from` and `handoff-reason`.
- in a round a role asks once the others are done (`in/then`): `WORKLINE_THEN`, its task, each once a run.

## The line's own

The project's commands never see the line's own variables: its tests,
as the tester's or the developer's proof, its checks, and the agent
([#953](https://github.com/JN0V/workline/issues/953)).

- **Which**: `WORKLINE_RUNS_DIR`, `WORKLINE_RATE_WAIT` (the wait for a
  forge's rate a run of the line sets), and every variable `pre` and
  `post` receive above, `WORKLINE_AI` among them: a project's test
  never calls the run's agent.
- **Why**: a project whose tests start workline's own engines would
  take them for theirs: its runs written in the job's folder, the
  oldest pruned, the live run with them.
- **Kept**: what a person sets for their own use, such as
  `WORKLINE_JUDGE`, `WORKLINE_ROLES` or `WORKLINE_MODELS_SEEN`.
