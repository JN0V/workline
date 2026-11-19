---
sources:
  - internal/gate/kept.go#keptPath
  - internal/gate/kept.go#readKept
  - internal/gate/kept.go#keptBase
  - internal/gate/kept.go#clean
  - internal/gate/gate.go#RunWith
  - internal/gate/gate.go#runner.check
checked: c46a93d6c
verified: agent:claude-code
---
# The next base

Part of [main after a merge](main-after-merge.md).

## Kept, then read

- **Kept by commit**: every finding of each SARIF check of main's run,
  in the folder `--gates-kept` names (`<commit>.json`).
- **Read as the merge base's**: a merge request based on that commit
  runs its tools once, at its head, rather than again at the merge base
  ([ADR-0049](adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md#6-guards)).
- **Only when they stand for a run there**: the same command, `output`
  and `config`; no file the tool reads its config from changed by the
  merge request; the same tool, by the version its SARIF names and the
  hash of the binary its command runs. Else the tools run again there,
  and the summary says why.
- **A tool naming no version, not found on the PATH**: run again.
- **None kept for the merge base** (missing, expired, another commit's):
  run again, said.
- **Said in the summary**: `lint 1 new, 3 before it (6.5s; the merge
  base's kept on main, not run again)`.

## Written by main alone

- **Main's run alone writes it**: a branch's pipeline can never write
  the base another merge request reads, so no finding a change adds can
  be made to look old. Each forge's store holds to that
  ([GitHub](main-after-merge.md#github), [GitLab](gitlab/main-after-merge.md)).
- **Kept only** from a tree that is HEAD's commit, every tracked file as
  committed; else said, nothing kept.
