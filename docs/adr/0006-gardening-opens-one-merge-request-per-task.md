# ADR-0006: Gardening opens one merge request per task, on a branch of its own

- **Status:** accepted
- **Date:** 2026-09-28

## Context

On `schedule`, the documentalist gardens: it reads stale docs again, merges
repeated passages and short cards, condenses docs over budget. Its patches are
written to the working tree. On a machine, a person reviews them; in CI, the
job's checkout is thrown away, and the work with it. The role's
`max-open-merge-requests` setting, meant to stop proposing when too many wait
for review, is not read.

Dependency bots solved the same problem: Renovate and Dependabot open one
branch and one pull request per update, update that pull request when they
run again rather than open another, and stop at a limit of open ones
(`open-pull-requests-limit`, 5 by default in Dependabot; `prConcurrentLimit`
in Renovate). peter-evans/create-pull-request commits a job's working tree to
a fixed branch and opens or updates its pull request, on GitHub only; GitLab
has no equivalent, only push options that open a merge request.

## Decision

- A run given `--open-merge-request` (with a forge) puts what its patches
  wrote on a branch `workline/<role>/<key>`, from the branch it ran on, in one
  commit, force-pushed, and opens a merge request for it — or updates the one
  already open for that branch. The working tree goes back to the branch it
  was on.
- The role names the key and the title, in `out/merge-request.yaml`
  (`{key, title, body}`); a key per task, so running a task again updates its
  merge request. Without it, the key is the role's name.
- Before the role runs, the engine counts the open merge requests on the
  role's branches and gives the number to the role (`WORKLINE_OPEN_MERGE_REQUESTS`);
  the role decides what it proposes. The documentalist proposes no gardening
  task at `max-open-merge-requests`.
- The engine does it itself — git, and the forge's API — on GitHub and
  GitLab alike, in the job that applies, which already holds the write token.

## Consequences

- Gardening in CI keeps its work, one reviewable merge request per task.
- A person's commits on a workline branch are overwritten by the next run of
  the same task, as a dependency bot does; a person who wants to keep them
  takes the branch over under another name.
- On GitHub, a pull request opened with the job's own token does not trigger
  workflows; a project that wants its checks there gives the apply job
  another token.
- The engine now writes to branches and opens merge requests: only the apply
  job, never the one holding the agent's key, is given that right.
