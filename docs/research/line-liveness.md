# Keeping every item moving or saying why (2026-11-04)

**Question.** On 2026-11-04, a first trial outside workline found 30
defects in the line in one day. Most were an issue left with nothing
due and nothing said, a loop guard keyed on the wrong thing, or an
engine failure routed like a role's verdict. How do controllers,
workflow engines and supervisors keep every item either moving or
saying why it waits, retry their own failures without counting them as
decisions, and stop real loops? What do they test it with?

**Verdict.**

- **Decide on the level, not the edge**: compare what is wanted with
  what is, on every pass, whatever event came or did not
  (Kubernetes controllers). Today's `due` is an edge: "the state, the
  body or the label changed".
- **Every object carries its condition, with a reason**: a status
  that names why it is not ready, kept current by the controller
  (Kubernetes `status.conditions`, Argo CD's health, a Kanban card's
  blocked reason). "Acted, nothing to run" is no condition.
- **One history, replayed**: each step appended once, the decision a
  pure function of it (Temporal's event history, Step Functions'
  execution history). Tests replay real histories.
- **Two kinds of failure**: a retryable error of the machine, retried
  under its own policy, and a non-retryable one, handled as an outcome
  (Temporal's retry policy and non-retryable errors, Step Functions'
  `Retry` and `Catch`). A machine error never becomes a decision.
- **Restart intensity, per child**: so many restarts in so long, then
  escalate (Erlang/OTP supervisors); a poison message goes to a dead
  letter queue after its retries, for a person.
- **Loops told by repetition, not by count**: the same action on the
  same observation, again (OpenHands' stuck detector), the same
  generation already observed (Kubernetes' `observedGeneration`).
- **The property first, then every path**: state liveness ("each item
  eventually done or waiting with a reason") and check it on generated
  and recorded histories (TLA+ liveness, deterministic simulation
  testing, stateful property-based tests).
- **Read the forge's own merge state**: GitHub's `mergeStateStatus`,
  GitLab's `detailed_merge_status` say why a pull request does not
  merge; do not model their rules again.
- **What was proposed**: [ADR-0063](../adr/0063-every-open-issue-is-due-waiting-or-done.md).

## Method

- **The evidence**: the root-cause analysis of 2026-11-04 over the
  trial's 30 defects, its round reports on
  [#757](https://github.com/JN0V/workline/issues/757), and
  `internal/line` at `40cf77cd`.
- **The ecosystem's words**: *level-triggered*, *reconcile loop*,
  *status conditions*, *observedGeneration*, *event history*,
  *deterministic replay*, *retry policy*, *non-retryable error*,
  *dead letter queue*, *restart intensity*, *stuck detection*,
  *liveness property*, *deterministic simulation testing*, *stateful
  property-based testing*.
- **Sources**: each system's public documentation, as known on
  2026-11-04; the links were not fetched again for this card. A
  reviewer checks a claim against its link before it decides anything.

## Level-triggered reconciliation

- **Kubernetes controllers** run a reconcile loop: read the desired
  state and the observed one, act to close the gap, requeue. A missed
  watch event costs latency, never correctness: the periodic resync
  sees the level again
  ([controller pattern](https://kubernetes.io/docs/concepts/architecture/controller/)).
- **The work queue dedups by key**: an object due twice is reconciled
  once; a failed reconcile is requeued with backoff, the object's state
  untouched.
- **For workline**: the reconciler already exists
  ([ADR-0059 §8](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#8-event-driven-the-night-for-upkeep)),
  but it asks `due`, an edge test. A level test asks "what does this
  issue need now?" and always has an answer.

## Status conditions and a blocked reason

- **Kubernetes `status.conditions[]`**: `type`, `status`, `reason`
  (a CamelCase code), `message` (for a person), `lastTransitionTime`,
  `observedGeneration`
  ([API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties)).
  A reader always knows whether the object is ready and, if not, why.
- **`observedGeneration`**: the spec's generation the controller last
  acted on. Equal to the current one means "seen this input"; it keys
  repetition on the input, not on time or on a count.
- **Argo CD** reports a sync status and a health status per
  application (`Healthy`, `Progressing`, `Degraded`, `Suspended`,
  `Missing`), each with a message
  ([health](https://argo-cd.readthedocs.io/en/stable/operator-manual/health/)).
- **Kanban**: a blocked card carries its blocker and who removes it;
  policies are explicit. Workline borrowed the Definition of Ready
  already ([ADR-0059 §9](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#9-pull-and-room-in-the-next-state)).

## A durable history, replayed

- **Temporal** keeps each workflow's event history, append-only; the
  workflow code is replayed over it and must be deterministic.
  Replaying recorded histories in tests (`WorkflowReplayer`) catches a
  change that would decide differently
  ([event history](https://docs.temporal.io/workflow-execution/event),
  [replay testing](https://docs.temporal.io/develop/go/testing-suite#replay)).
- **AWS Step Functions** keeps an execution history of each state's
  entry, exit and failure.
- **For workline**: one log per issue, appended by each step, read by
  a pure function, is the same shape. The trial's issues, frozen with
  their comments, are recorded histories to replay.

## Retryable and non-retryable failures

- **Temporal's retry policy**: initial interval, backoff, maximum
  attempts, and `NonRetryableErrorTypes`; an `ApplicationError` marked
  non-retryable goes to the workflow code as an outcome
  ([retry policies](https://docs.temporal.io/encyclopedia/retry-policies)).
- **Step Functions**: `Retry` (by `ErrorEquals`, `MaxAttempts`,
  `BackoffRate`) for errors of the machine, `Catch` routing to another
  state for the rest; `States.Timeout`, `States.TaskFailed` are named
  ([error handling](https://docs.aws.amazon.com/step-functions/latest/dg/concepts-error-handling.html)).
- **Queues** (Amazon SQS, RabbitMQ, Sidekiq): after its retries, a
  message goes to a *dead letter queue*, kept for a person, never
  dropped nor looped.
- **For workline**: over budget, code not given, a test that does not
  build, a patch that does not apply, a run lost are the machine's
  errors. Each retries with changed input under its kind's budget, then
  waits for a person. A role's verdict ("no test is possible for this
  need") is the `Catch`: it routes.

## Restart intensity

- **Erlang/OTP supervisors**: at most `intensity` restarts within
  `period` seconds per supervisor, then the supervisor itself stops and
  escalates; restart strategies per child (`permanent`, `transient`,
  `temporary`)
  ([supervisor principles](https://www.erlang.org/doc/system/sup_princ.html)).
- **For workline**: `max-hops` is one global counter fed by every
  move, decisions and failures alike. A budget per failure kind, per
  issue and state, escalating to a person, is the supervisor's shape.

## Loops told by repetition

- **OpenHands' stuck detector** stops an agent repeating the same
  action with the same observation, or alternating between two
  ([source](https://github.com/All-Hands-AI/OpenHands/blob/main/openhands/controller/stuck.py)).
- **Kubernetes**: a controller that sees `observedGeneration` equal to
  the generation does not act again for that input.
- **For workline**: "the same role, on the same input, gave the same
  outcome" is the loop, whatever the count. A ring of roles on unchanged
  inputs is the alternating case.

## The property first

- **TLA+** states safety and liveness ("eventually done") and checks
  every behaviour of a model; fairness says a step always enabled is
  eventually taken ([Lamport](https://lamport.azurewebsites.net/tla/tla.html)).
- **Deterministic simulation testing**: FoundationDB's simulator and
  TigerBeetle's VOPR run the real code against a fake world, seeded,
  with faults injected, and check invariants on each run.
- **Stateful property-based testing**: QuickCheck's state machine
  models, Hypothesis' `RuleBasedStateMachine`, Go's `rapid` state
  machines: random sequences of commands, invariants checked after each.
  Go's own `testing/quick` and a seeded `math/rand` need no new
  dependency (principle 11).
- **For workline**: no test today runs the line over many steps; each
  conformance case runs one command on one snapshot. A simulator with
  fake agents of each outcome class, over the trial's real issues,
  checks "every open issue is due, waiting with a reason, or done"
  after each tick.

## The forge's own merge state

- **GitHub**: the GraphQL `mergeStateStatus` (`CLEAN`, `BLOCKED`,
  `BEHIND`, `DIRTY`, `UNSTABLE`, `HAS_HOOKS`, `DRAFT`, `UNKNOWN`) and
  `reviewDecision`
  ([MergeStateStatus](https://docs.github.com/en/graphql/reference/enums#mergestatestatus)).
- **GitLab**: `detailed_merge_status` on a merge request
  (`mergeable`, `not_approved`, `discussions_not_resolved`,
  `ci_still_running`, `ci_must_pass`, `need_rebase`, `conflict`,
  `draft_status`, `blocked_status`, …)
  ([merge requests API](https://docs.gitlab.com/api/merge_requests/#merge-status)).
- **For workline**: `follow` built readiness from its own reads and
  missed unresolved conversations
  ([#1050](https://github.com/JN0V/workline/issues/1050)). The forge's
  answer is the wait's reason; workline adds what is its own (a
  verdict, a person's yes).

## What workline takes, and what not

| Taken | Not taken |
|---|---|
| A reconcile on the level, events for speed | A controller framework, a workflow server (principle 11) |
| A condition with a reason on every open issue, said on it | A new label per condition: the six states stay |
| An append-only log, a pure decision, replayed in tests | Temporal's determinism rules for the roles: agents are not deterministic, only the decision is |
| Retryable machine errors, a budget per kind, then a person | A global hop counter as the loop guard |
| Repetition on the same input as the loop | TLA+ itself: the property is checked by a simulator in Go |
| The forge's merge state as the reason | Workline's own model of the forge's merge rules |
