# A line of states — a six-persona review (2026-10-31)

**Question.** Since the build path went live, no issue has gone from
need to merged code. The maintainer proposed a line of six states, each
label waking the next role at once, the night kept for upkeep. Is it
the right line, and what should change before it is built?

**Verdict.**

- **Make every stop visible first**: a refusal, a claim, a wait or a
  block written on the issue, whatever line is chosen.
- **Keep the states, fewer of them**: Refine, Tests, Ready, Doing,
  Blocked; Done is the forge's close at the merge.
- **No agent where a check will do**: the engine checks the tests
  cover each scenario; the product owner is called on an exception
  only. To validate is a required check, not an agent pass.
- **Events for speed, a reconciler for truth**: the line must not
  depend on an event arriving, nor on a nightly cron.
- **Pull, not push**: a role starts only when the next state has room.
- **What was decided**: [ADR-0059](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md).

## Method

- **A multi-persona review**: one proposal read by several reviewers,
  each given one role's lens and asked to critique it, then their
  points compared. A common practice in agile teams (three amigos, a
  design review with a devil's advocate), here with an agent per
  persona. Written for workline; no framework's prompts used.
- **Six personas**: business analyst, product manager, architect, test
  architect, flow coach (scrum master), and a developer as devil's
  advocate.
- **Each read**: [PRINCIPLES.md](../PRINCIPLES.md), the roles' READMEs,
  [routing.md](../spec/routing.md), [boards.md](../boards.md),
  [ADR-0054](../adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md),
  [ADR-0055](../adr/0055-the-tester-writes-the-acceptance-tests-before-the-code.md),
  [ADR-0057](../adr/0057-workline-finds-the-forges-board-and-moves-its-cards.md),
  [#752](https://github.com/JN0V/workline/issues/752), and the evidence
  below.
- **Kept here**: the evidence, and each persona's verdict, objections
  and additions, condensed from its answer.

## The evidence, 2026-10-31

- **Nothing end to end** since about 2026-10-29: one developer pull
  request, red and idle 25 h
  ([#699](https://github.com/JN0V/workline/pull/699)); two tester
  Drafts ([#677](https://github.com/JN0V/workline/pull/677),
  [#713](https://github.com/JN0V/workline/pull/713)); all open.
- **25 issues labelled `workline:ready`**, median wait about 18 days.
  Some stale: [#695](https://github.com/JN0V/workline/issues/695) named
  a function deleted weeks before. Some not buildable by one run: a
  whole role, a try only a person can make.
- **Refusals in CI logs only**: one issue claimed by the developer
  since 2026-10-30, no pull request, no word on the issue.
- **The night drives the line**: the gardening cron starts hours late,
  and skipped 2026-10-22 to 10-26. The tester takes one issue a night.
- **Ready wakes the developer before its tests exist**: it refuses
  ([#752](https://github.com/JN0V/workline/issues/752)).
- **No test possible, rightly**: a flaky-test fix
  ([#619](https://github.com/JN0V/workline/issues/619)) never reaches
  the developer.

## The proposal reviewed

| State | Set by | Next, at once |
|---|---|---|
| to-refine | a new or stale issue | the product owner refines it |
| needs-tests | the product owner | the tester writes the tests |
| tests-ready | the tester | the product owner re-checks issue and tests |
| ready-to-dev | the product owner | the developer takes it |
| to-validate | the developer, its pull request ready | the tester validates again |
| done | the tester | the issue closed |

## Each persona's answer

### Business analyst

- **Verdict**: the right direction, ready should mean buildable; but
  mostly a relabelling. It misses three root causes: the tester's one
  issue a night, silent refusals, builds broken by their environment.
- **Objections**:
  - GitLab starts no pipeline on a label: it needs a webhook and a
    trigger token, or a schedule. "At once" holds on GitHub only.
  - [routing.md](../spec/routing.md#the-line) forbids the engine's own
    writes from waking the line, and caps handoffs at 3; the proposal
    has 5 hops.
  - About 13 of the 25 ready issues date from 2026-10-09 to 10-18,
    before the developer existed: ready then meant refined, not
    buildable. A "fits one developer run" check is needed.
  - [#699](https://github.com/JN0V/workline/pull/699) is red on 10
    cases it never touched: the environment.
  - Silence, as on [#79](https://github.com/JN0V/workline/issues/79),
    needs a duty to speak and a claim timeout.
  - Exclusive labels cost writes and races, except GitLab Premium's
    scoped labels.
- **Adds**: the tester takes several issues a run, on its own label; a
  Blocked state.
- **On the open points**: validate before the merge, as a required
  check; the product owner settles, three exchanges, then Blocked; the
  product owner may send a no-test issue to ready with another
  verification; a limit per state, in hours for a claim or a pull
  request waiting.

### Product manager

- **Verdict**: keep the events and the staleness; cut to four visible
  states: Refine, Tests, Ready, Doing; Done is the forge's close.
- **Objections**:
  - to-validate repeats CI and the reviewer.
  - A product owner's re-check of the tests is new judgement with no
    evidence of need: the tester proves them red, the developer can
    object.
  - Six labels are too many for a newcomer.
- **Adds**: trust needs every act written on the issue, a comment an
  act; a claim released after about 24 h, by the engine's timer; a
  Blocked column; the board mapped per forge (GitHub's Status, one
  value; GitLab Premium's scoped labels; Free's plain labels swapped);
  features accepted by a person, fixes merged by policy.
- **Build order**, taken whole by ADR-0059: what is written on the
  issue first; then the tester woken by its label, the developer by
  ready; the 25 ready issues re-checked once and two fresh ones timed;
  the board's columns; a product owner's test review last, only if
  objections show the need.

### Architect

- **Verdict**: sound if the engine holds the state machine and a
  periodic reconciler guarantees progress; fragile if the workflows
  carry it.
- **Risks**:
  - The state spread over 11 workflows: the developer's hard-codes its
    label in an `if:`; GitLab's templates drift.
  - Lost events: the cron late, skipped 2026-10-22 to 10-26; GitLab
    needs a webhook and an outside tool
    ([gitlab-trigger.md](../gitlab-trigger.md)); a person's machine has
    no events.
  - A token falling back to `GITHUB_TOKEN` (`steps.app.outputs.token
    || github.token`): its labels wake nothing, silently.
  - Races: a concurrency group on the developer's workflow only; a
    label removed and added is two events; the forge holds no lock.
  - Two sources of truth: labels and the board
    ([ADR-0057](../adr/0057-workline-finds-the-forges-board-and-moves-its-cards.md)
    keeps labels the truth; GitHub's project item events are an
    organisation's).
- **Recommends**: one state table in `routing.default.yaml`, a pure
  function; `workline route issue --target issue:<n>`; thin forge
  adapters; explicit chaining (`workflow_dispatch`, GitLab's pipelines
  API), hops capped; a reconciler hourly in CI and as a command on a
  machine; idempotent steps (re-read, a record per state and body
  hash); exclusive labels, scoped on GitLab; one concurrency group per
  issue; limits per state counted from the label's event (GitHub's
  timeline, GitLab's `resource_label_events`), never `updated_at`.

### Test architect

- **Verdict**: keep tests first and frozen; cut the product owner's
  re-check of each issue; make to-validate a check with no agent.
- **Objections**:
  - Frozen tests can impose a design:
    [#677](https://github.com/JN0V/workline/pull/677) invents
    `WORKLINE_LYCHEE_RELEASE_URL` and takes the checksum from the fake
    server, against "pinned". A product owner would not see it; the
    first objection will. Make objecting cheap.
  - [#677](https://github.com/JN0V/workline/pull/677) adds four
    `errcheck` findings: tests must pass lint before ready.
  - Matching scenarios to tests is mechanical: the `## Tests` table.
  - A tester's agent pass at to-validate repeats the frozen guard, the
    three-of-three green proof and the reviewer. Worth it only for a
    scenario marked not tested: a real try under `## Tried`.
  - `objection-person` still defaults to `workline:human`.
- **Adds**: proof kinds: new tests; a flaky test passing N of N
  ([#619](https://github.com/JN0V/workline/issues/619)); for a
  refactor, the suite with its coverage and killed mutants kept; the
  documentalist's checks for docs.
- **Not measured yet**: a tester run is capped at 300k tokens and 20
  min; no write run on a real issue measured.

### Flow coach

- **Verdict**: a sound Kanban skeleton, but drawn as a staged push
  line with no WIP limit, no pull, no blocked slot: a waterfall in
  disguise.
- **Data**: 25 ready, median age 24 days by its count; 2 to refine, 2
  developing. The last 40 closed issues had a median lead time of 0
  days: work done in sessions, not through the line.
- **Objections**:
  - Push floods the slowest step: start only when downstream has room.
  - No WIP limits. Suggested: refine 5, needs-tests 3, tests-ready 3,
    ready 3, developing 1 or 2, to-validate 2.
  - Two product owner and two tester passes are hand-off waste; three
    amigos fold them into one.
  - Blocked is a flag, not a state.
  - A fixed two weeks is wrong: age per state against its 85th
    percentile, and a re-check when the issue is pulled.
- **Adds**: a Definition of Ready and Done per state; an expedite lane
  for a broken CI; metrics: throughput a week, lead and cycle time at
  p85, aging work in progress, blocked count, cumulative flow.

### Developer, devil's advocate

- **Verdict**: against the proposal as drawn. The line is stuck on
  broken hand-offs and a red proof, not on missing states.
- **Objections**:
  - [#699](https://github.com/JN0V/workline/pull/699) is red on its
    test, gates, judge and commit format: 11 red tests, 10 of them
    conformance cases (secrets, forbidden terms, leaks) that
    `internal/tools/tools.go` likely never touches. The environment.
  - Agent calls an issue go from about 4 to 6, two more per loop; 6
    event hops instead of 3.
  - First to break: the product owner and the tester sending an issue
    back and forth.
- **Simpler**: make [#699](https://github.com/JN0V/workline/pull/699)
  green with CI's tools in the proof environment; write refusals on
  the issue; the tester on ready, its Draft waking the developer; the
  reviewer checks the tests against the scenarios on the Draft.
- **Keeps**: events, the night for upkeep, staleness in every state
  (age and Scope checked with no agent, the product owner for what is
  flagged), "no test possible" with a reason, a visible state for each
  wait.

## Where they agreed

- **Every stop written on the issue**, first: the business analyst,
  product manager and developer.
- **No product owner re-check of each Draft**: the product manager,
  test architect, flow coach and developer.
- **to-validate is CI, not an agent pass**: all who raised it.
- **The engine holds the state, a reconciler guarantees progress**:
  the architect, echoed by the business analyst and developer on lost
  events.
- **Fix [#699](https://github.com/JN0V/workline/pull/699)'s
  environment apart**: the business analyst and developer.

## What workline takes

- All of the above, as [ADR-0059](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md)
  decides, in its build order: visible stops first, then the state
  table and the reconciler, then a timed first result, then the board,
  WIP and staleness.
- **Left for later**: a product owner's review of the tests, only if
  objections show the need.
