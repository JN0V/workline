# ADR-0001: Write the engine in Go, configure it in YAML, leave Windows out

- **Status:** accepted
- **Date:** 2026-09-23
- **Decided by:** Sébastien, after a BMAD party-mode round table (architect, developer,
  security engineer, open-source maintainer, devil's advocate)

## Context

The framework runs roles such as the committer, the release manager and the documentalist.
Each role is a folder holding a contract, a deterministic `pre` script, a deterministic
`post` script and its skills. A small engine does four jobs:

- runs a role (`run-role <role> --ai <agent>|none`);
- routes work to the next role;
- applies the intentions an agent wrote;
- evaluates gates.

The engine must behave the same on a laptop, in GitHub Actions, in GitLab CI and on a
homelab or VPS. In CI it holds forge tokens and AI keys while reading untrusted issue and
merge-request content. It must keep working when no AI is available, and it must not pull
in a large tree of transitive dependencies.

## Decision

1. **The engine is a single static Go binary**, built from day one. It stays a plain
   conductor: no role logic lives in it.
2. **Configuration files (`routing`, `gates`, role contracts) are YAML**, parsed with
   `go.yaml.in/yaml/v3`. The old `gopkg.in/yaml.v3` has been archived since April 2025.
3. **Windows is not supported natively.** Windows developers run roles in WSL or on a
   remote machine. Both are ordinary targets already.

## Conditions this decision depends on

- **The contract comes before the code.** The role contract, the intentions file, and the
  routing and gates schemas are written and versioned first, together with conformance
  tests. They are the durable asset; the engine can be rewritten against them.
- **Dependencies stay few.** `go.mod` keeps a short list of direct dependencies, and CI
  checks it. External tools such as conftest, reviewdog, `gh` and `glab` are called as
  binaries, never embedded as libraries.
- **Core roles need no other runtime.** A role that needs Python or anything else declares
  it in its contract, so the cost is visible before anyone installs the role.
- **The engine does not trust its input.** The component that applies intentions checks
  them against a strict schema and an allowlist of actions. The job that runs the agent
  holds no write token, and the job that applies intentions holds no AI key.

## Alternatives considered

- **Start on an existing runner** (Taskfile, mise tasks, just) and write Go later. This was
  rejected because the intentions applier handles write tokens and has to be Go from the
  start anyway. Adding a runner beside it would mean two things to install, not one.
- **Python with the standard library only.** Rejected because YAML parsing is not in the
  standard library, and Python versions differ between machines and CI images.
- **TypeScript compiled to a binary.** Rejected because it ships `node_modules` inside the
  binary instead of removing them, and the npm registry has suffered repeated
  supply-chain compromises in 2026.
- **POSIX shell.** Rejected because YAML needs `yq`, quoting is fragile, testing is weak,
  and only the author can debug it.
- **Rust.** It is viable, but cross-compiling is harder and fewer DevOps contributors know
  it. The surrounding tools (`gh`, `glab`, reviewdog, conftest, lefthook) are Go.
- **Dagger.** Rejected because it needs a container engine to route a label.

## Consequences

- Installing is a single download, checked against its checksum. We can sign it
  (cosign) and publish build provenance.
- Role authors write `pre` and `post` scripts in any language. The engine only runs
  commands.
- Contributors to the engine itself need Go.
- If Windows support is ever needed natively, we reopen this ADR. The Go binary already
  compiles for Windows; the work would be in the role scripts.
