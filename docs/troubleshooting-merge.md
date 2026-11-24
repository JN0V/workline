---
sources:
  - internal/engine/merge.go#mergePolicy
  - internal/engine/merge.go#heldByConversations
  - internal/line/states.go#roundsDue
checked: e81df7b34
verified: agent:claude-code
---
# Troubleshooting: the line and the merge

Part of [troubleshooting](troubleshooting.md).

## An issue does not move

The line of states says why on the issue and in the run's summary
([the line of states](spec/routing.md#the-line-of-states)):

- `state-wakes-none`: no state label, or its state's role is not routed
  on `issue`; the reason follows.
- `state-waits`: the next state is full (`routing.wip`); it starts when
  room appears.
- `hops-capped`: past `max-hops` steps, it went Blocked; put a state's
  label back to start again.
- `reconcile-left`: past `reconcile-max`, it waits for the next run.
- Blocked: read again on news, else every `revisit` days
  ([blocked, read again](spec/routing.md#blocked-read-again)).

## A pull request blocked, its checks green

- **Conversations to resolve**: GitHub's rule "require conversation
  resolution" holds the merge while one is open. `workline doctor` says
  whether the default branch has it.
- **Code scanning's**: an older workline sent every finding
  there, one conversation each. Copy `defaults/ci/github/workline.yml` and
  `workline-apply.yml` of this release: the documentalist's findings
  become a check's annotations, which open none.
- **The reviewer's comments** are conversations by design. The reviewer
  resolves its own once the code answers them
  ([how](../roles/reviewer/docs/lines-again.md#not-repeated-found-again)); a
  person resolves the rest. `workline follow` lists the open ones that
  hold an auto-merge.
- **`gh pr merge --admin` refused**, "Repository rule violations": by
  design, nobody bypasses `main`. Put `workline:accepted` on it; in an
  emergency, the written way round
  ([nobody bypasses `main`](merge-policy-no-bypass.md#the-emergency)).

## Nothing merges

The [merge policy](merge-policy.md) acts only with `merge:` in
`.workline/config.yaml`; `workline follow` says why in the job's summary
([each word](merge-policy.md#what-the-summary-says)):

- `label-ignored`: the label was put by someone who may not push nor
  merge; one who may puts it again.
- `verdict-missing`: no verdict of the line on that head yet; it waits
  for the next propose job.
- `waits-for-a-yes`, at `normal`: workline's pull request with a push, a
  commit or a page not its own, or of a kind a revert sent back to a
  person's yes — someone who may push and merge closes the issue it names
  to restore it —; the developer's feature or task, or a fix whose bug no
  test reproduced; the release tool's always.
- `merge-app-unknown`: at `normal`, the token is a person's, or the forge
  does not say its App (GitHub: `WORKLINE_APP`): nothing merges alone.
- `merge-record-lost`: at `normal`, no mark of the App's on the base's last
  100 commits — the first run too —: someone who may push and merge closes
  the issue it names, and workline's own work merges alone again.
- `merge-unsupported`, `merge-token`, `merge-unguarded`: nothing is
  turned on — a forge with no auto-merge, a GitHub job's own token, a
  base that does not require `workline/accepted`
  ([set up](merge-policy.md#set-up)). The local forge merges itself
  ([the merge](local-line-merge.md)).
- `auto-merge-refused` on the local forge: `main` checked out with
  uncommitted changes, a conflict, or a branch moved since it was
  weighed; commit or stash, rebase, or run the line again.
- `verdict-not-kept`: `follow --verdict` with no `merge:` set, or on a
  forge without auto-merge; on the local forge, `route merge-request`
  could not keep its verdict in the clone: the request waits, run the
  line again.
