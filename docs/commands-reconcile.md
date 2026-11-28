---
sources:
  - cmd/workline/main.go#routeCmd
  - internal/line/states.go#ReconcileEvent
checked: 3f432ab14
verified: agent:claude-code
---
# workline route reconcile

Part of [the commands that do the work](commands-work.md).

The line of states' safety net
([the reconciler](spec/routing.md#the-reconciler)): every open issue's
step due that no event started, run.

- **Due**: the role its state wakes is routed on `issue`, and the line's
  record on it does not hold that state, body and labelling yet.
- **`--ai`**, `--open-merge-request`, `--forge`: as for each step;
  on a forge, `--open-merge-request` is the default, as for `route issue`.
- **At most `reconcile-max`** steps a run (3); `--input reconcile=list`
  names them and starts none.
- **Priority 1 first, then the one waiting longest**, so a new issue never waits behind
  old ones run after run
  ([the reconciler](spec/routing.md#the-reconciler)).
- **`--input room=true`**, with `reconcile=list`: only the issues whose
  next state has room again, for the CI job that freed it.
- **Room first**: a step whose next state is at its limit waits, said
  once on its issue; one priority-1 issue at a time passes a full Ready
  ([room in the next state](spec/routing.md#room-in-the-next-state)).
- **Blocked, read again**: news on a blocked issue wakes the product
  owner, else every `revisit` days (7)
  ([blocked, read again](spec/routing.md#blocked-read-again)).
- **A parent**: a part closed, or a person's yes, wakes the product
  owner on it with no agent, even with no state label
  ([a parent, told as its parts close](spec/routing.md#a-parent-told-as-its-parts-close)).
- **Stale**: an issue past its state's age limit whose Scope names what
  the code no longer holds goes back to refine
  ([staleness](spec/routing.md#staleness)).
- **The developer's rounds**: a round due on its open pull requests that
  no event started, counted in `reconcile-max`.
- **The tester's answers**: a Draft an objection waits on, listed for
  its answer.
- **Checks GitHub lost**: a developer's pull request whose head has no
  check of the project's 15 minutes after its push gets its branch
  merged with its base, pushed as the App, so its checks run.
- **Run twice in a row**, the second does nothing more.
