# Conformance — v1 (draft)

The specs say what must happen; these tests prove it does. They are written
before the engine, and they belong to the contract, not to one engine: any
implementation that passes them is a valid engine (ADR-0001).

## Two levels

| Level | Agent | Runs | Answers |
|---|---|---|---|
| **Conformance** | a fake agent replaying recorded proposals | every change, in CI, free | Does the engine obey the contract? |
| **Evaluation** | a real agent | on demand, costs tokens | Does a role do its job well, with this model? |

Conformance never calls a model, so it is deterministic and costs nothing.
Evaluation grades a role on its own, on the same fixtures, with a real agent;
its score is tracked over time rather than passed or failed.

## Layout

```
tests/conformance/
  fixtures/repos/<name>.sh    builds a git repository from scratch, deterministically
  fixtures/agents/<name>.yaml proposals the fake agent returns
  cases/<area>/<case>.yaml    one behaviour each
```

Repositories are built by scripts rather than stored, because a git repository
cannot live inside another one, and because a script shows exactly what the
history contains.

## A case

```yaml
case: committer/internal-code-in-subject
about: A ticket code in the subject blocks the commit, even without AI.

given:
  repo: basic                       # fixtures/repos/basic.sh
  setup:                            # shell lines run in the repo after it is built
    - echo "change" >> main.go && git add main.go
  forge:                            # starting state of the simulated forge, if any
    issues: []
  config: {}                        # .workline/config.yaml for this case

run:
  role: committer                   # or: route: <event>, or: gate: <name>
  event: commit-msg
  input: {message: "fix: AC-3"}
  ai: none                          # none | fake:<fixture> | unavailable:<reason>

expect:
  status: block
  findings: [{rule: internal-code}]
  agent-calls: 0
  applied: []
```

`run` can also carry:

- `scope` — the run's scope, as a ready work item would give it;
- `given.repos` — other repositories to build first, each exported by name
  as an environment variable holding its path, for multi-repository cases;
- `then: resume` — after the run, resume it with `run-role apply`;
- `tamper: in/` — change the prepared input between prepare and apply;
- `route: <transition>` with `item` — ask routing to move a work item.

`expect` lists only what the case is about; anything not listed is not checked.

- `findings` — each listed finding must be present (matched by `rule`, and by
  `where` if given).
- `agent-calls` — how many times the agent was called.
- `applied` / `refused` — intentions applied or refused, by kind.
- `forge` — fields the simulated forge must hold afterwards.
- `files` — paths that must exist, or contain a text, afterwards.

## The fakes

- **Fake agent.** `fake:<fixture>` returns the proposals in
  `fixtures/agents/<fixture>.yaml`, without reading the prompt. It checks the
  engine's handling of proposals, not their quality.
- **Unavailable agent.** `unavailable:quota` (or `auth`, `network`) fails the
  way a real agent does when a quota runs out.
- **Simulated forge.** An in-memory forge holding issues, labels, comments,
  merge requests and releases. It can be told to fail on the N-th write, to
  test recovery after a partial apply.

## Evaluation

An evaluation case uses the same `given` and `run`, with a real agent, and adds
`grade`: what a good answer contains ("the message names the changed
behaviour", "the product doc keeps every MUST"). Grading is done by checks where
possible, and by a judge from another provider where not (`independent-of` in
the model grid).
