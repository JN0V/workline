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
- **Kept here**: the evidence and each persona's main points, grouped
  by lens. The personas' full answers stayed in the session.

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

## Each persona's main points

### Business analyst

- **A Definition of Ready per state**, not one at the end: what must be
  true to enter Tests, and to enter Ready.
- **"Fits one run" is part of ready**: a whole role or a person's try
  is not one issue's work.
- **"No test possible" is an answer**, with its reason, for the
  product owner to judge.

### Product manager

- **First, make the stops visible**: a person on the loop cannot steer
  what only a log says. Built before any new state.
- **Re-check the 25 ready issues once**, then run two small fresh ones
  through the whole line, timed: the first end-to-end result.
- **Blocked is a column**, read by a person, not a label asking them.

### Architect

- **One state table in the engine**, read by a pure function; the
  forges only carry events.
- **Events are not reliable**: a reconciler walks the issues and
  catches up; idempotent steps, one concurrency group per issue.
- **Start the next step explicitly**, not through the label's event:
  a label set by a bot's token may wake nothing on some forges. Cap
  the hops per issue.
- **Staleness from when the label was set**, read from the forge's
  label events, never from the last update.

### Test architect

- **Replace "always a new test" with a proof kind**: new tests, a flaky
  test passing N of N, the existing suite for a refactor, the docs'
  checks.
- **Check coverage of the scenarios mechanically**: each scenario a
  row in the `## Tests` table, the Draft linted. A product owner's
  re-read of every Draft adds tokens and a hop for little.
- **To validate is a required check**: the frozen tests green. A real
  try only for scenarios no test covers.

### Flow coach

- **Pull, with WIP limits per state**: 25 issues in Ready is a queue,
  not a flow.
- **A time limit per waiting state**, not only Ready; stale goes back
  to Refine with its reason.
- **The night is upkeep**: a line that waits for a cron loses a day a
  hop.

### Developer, devil's advocate

- **Six states is one too many hops**: each product owner pass costs
  tokens and a wait; keep it for exceptions, add it later if the
  developer's objections ask for it.
- **The red proof is not the line's fault**:
  [#699](https://github.com/JN0V/workline/pull/699) is red on ten
  conformance cases it never touched. Fix the proof environment apart.
- **A malformed answer should be asked again once**, as other roles
  do, not dropped
  ([#753](https://github.com/JN0V/workline/pull/753)'s try).
- **A claim with no pull request must expire**: a day, then released.

## What workline takes

- All of the above, as [ADR-0059](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md)
  decides, in its build order: visible stops first, then the state
  table and the reconciler, then a timed first result, then the board,
  WIP and staleness.
- **Left for later**: a product owner's review of the tests, only if
  objections show the need.
