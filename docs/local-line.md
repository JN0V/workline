---
sources:
  - internal/forge/local.go#Local
  - cmd/workline/main.go#issuesCmd
  - cmd/workline/main.go#initCmd
  - cmd/workline/main.go#initPacks
  - internal/packs/builtin/python-pytest.yaml#detect
  - internal/role/role.go#ProjectConfig
  - defaults/routing.yaml#states
  - internal/doctor/tests.go#testCommand
  - internal/doctor/tests.go#testCommandRuns
  - internal/doctor/tests.go#testsFix
  - internal/engine/engine.go#RunsDir
checked: ca636665
verified: agent:claude-code
---
# The whole line with no forge

From a need to a merged change on your machine, with no GitHub nor
GitLab: the product owner refines, the tester writes the tests, the
developer the code, the reviewer reads it; the merge policy merges what
it may alone, you say yes to the rest.

```
need ──► product owner ──► tester ──► developer ──► review ──► merge
 you      refines it        tests      code          the line    the policy, or you
```

- **Issues and merge requests stay in the clone** (`.git/workline/`),
  never committed nor pushed
  ([ADR-0016](adr/0016-writes-go-where-the-project-lives.md)).
- **You act by command**: `workline issues` opens, labels, comments and
  edits a section, as you would on a forge's page.
- **An agent is needed** for the roles that write: `claude`, or any
  command as `cmd:` ([the agent](install.md#3-the-agent-your-choice)).
  With `none`, each role says what it skipped
  ([principle 5](PRINCIPLES.md)).

## 1. Install and adopt

Install the binary and set the machine up ([install](install.md)), then
in the repository:

```sh
workline init --review   # routes pre-push and the reviewer; proposes each doc's sources
```

## 2. Route the line

Edit `.workline/config.yaml`: keep the `packs:` line `init` wrote (for a Python repository, `packs: [python-pytest]`), and set the rest as below:

```yaml
packs: [python-pytest]   # keep the line init wrote, whatever it is
forge: local
routing:
  events:
    pre-push: [committer, documentalist]
    merge-request: [committer, documentalist, tester, reviewer]
    issue: [product-owner, reviewer, tester, developer]
autonomy: normal
merge: {autonomy: normal}
roles:
  tester:
    settings:
      test-command: python -m pytest -q --junitxml={report}   # yours: runs the tests, writes JUnit XML to {report}
      test-paths: ["tests/**"]                                # where your tests live
```

- **`issue`**: an issue's state label wakes its role
  ([the line of states](spec/routing.md#the-line-of-states)).
- **`test-command`**: any runner that writes a JUnit XML report
  ([the tester's settings](../roles/tester/README.md#settings)).
- **`autonomy`**: how far the roles go alone ([autonomy](autonomy.md)).
- **`merge`**: what merges with no yes of yours
  ([the merge policy](merge-policy.md)). Left out, you merge.

```sh
workline doctor          # says a missing test-command, or one that cannot run here
git add .workline/config.yaml && git commit -m "chore: adopt workline"
```

Each run is kept in `.git/workline/runs`.

## 3. Write the need

```sh
workline issues open "sc-late-fee takes NaN for an amount" \
  --body-file need.md --label workline:to-refine --label bug
```

- Write it as a person would: what you saw, what you want
  ([writing an issue](../roles/product-owner/docs/writing.md)).
- `--body "text"` for a short one; `--body-file -` reads stdin.
- `workline issues` lists them; `workline issues show 1` shows one whole.

## 4. Run each step

Each step wakes the role the issue's state names, and its result moves
the issue on: [run each step](local-line-steps.md) gives the command,
how to answer the product owner and what to do when a step stalls.

## 5. Review the merge request

The `merge-request` line routed above reads the developer's change:
[review the merge request](local-line-review.md).

## 6. The merge

What the policy says yes to merges into `main` in this clone, the rest
waits for your yes: [the merge](local-line-merge.md).

## Where this page reads from

The local forge is `internal/forge/local.go`; the commands `workline issues`
and `workline init` are `cmd/workline/main.go`; the pack a Python
repository gets is `internal/packs/builtin/python-pytest.yaml`; the
project's settings are `internal/role/role.go`; the line's states are
`defaults/routing.yaml`; what the doctor says of a `test-command` is
`internal/doctor/tests.go`; the runs' folder is `internal/engine/engine.go`.
The line itself: [routing](spec/routing.md#the-line-of-states).
