---
sources: [internal/line/backstop.go#backstop, internal/line/backstop.go#backstop.reason, internal/line/backstop.go#backstop.said, internal/line/backstop.go#passed, internal/line/budget.go#minutes, internal/line/crossed.go#crossed, internal/line/crossed.go#unanswered, internal/line/freed.go#freed, internal/line/freed.go#woken, internal/line/line.go#stopped, internal/line/line.go#failed, internal/line/outcome.go#ErrorBudget, internal/line/revisit.go#revisit, internal/line/seconds.go#round, internal/line/size.go#sized, internal/line/stale.go#clock, internal/line/states.go#never, internal/line/unrefined.go#RefineRetries, internal/line/unrefined.go#unrefined, internal/line/wip.go#waits, internal/line/wip.go#waiting, internal/line/wip.go#retest, internal/work/headings.go#Scope, internal/work/work.go#parts, defaults/routing.yaml]
checked: a19e8cb0
verified: agent:claude-code
---
# Routing — the record of a state

Part of [Routing](routing.md): what the line keeps of each state it acted on, in [the line of states](routing-states.md#the-line-of-states).

### Once a state: the record

- **Idempotent**: the line keeps one comment of its own on each issue
  it ran a step for (`workline:sticky=line/state`), holding the state
  acted on, the body's hash, when the state's label was put on (the
  forge's label events), and the hops so far. The same three again run
  nothing (`state-acted`); labelled anew, or its body changed, the step
  is due again.
- **A person's labels are input, the role's own writes are not**
  ([ADR-0063, §3](../adr/0063-every-open-issue-is-due-waiting-or-done.md#3-the-input-of-a-step)):
  the record also keeps `workline:accepted` and the size tiers
  (`labels`). After a step's run, the body and the size tiers are
  recorded as the run left them, `workline:accepted` as the step
  started on it: the role's own edit does not wake it again; a label a
  person puts on afterwards does. A record from before takes its size
  tiers as seen, and `workline:accepted` as new only when put on after
  its last run (`ran`).
- **A clean review takes no agent**: refine's step woken by the
  reviewer's review alone, nothing important open, every section and
  scenario there, runs the product owner with no agent: its move to
  ready is a rule (`state-ready-clean`). Left undone, the line says so
  (`state-ready-held`) and the agent reads the review at the next run.
- **A refusal is not acted**: an issue in the tester's state whose last
  tester run wrote no tests, said in the tester's comment, is never
  left waiting there: the line moves it to `workline:blocked`, the
  tester's reason said (`tests-refused`;
  [#794](https://github.com/JN0V/workline/issues/794)). Not for a
  machine error: see below.
- **A machine error moves nothing**
  ([ADR-0063, §6](../adr/0063-every-open-issue-is-due-waiting-or-done.md#6-typed-outcomes)):
  each run's outcome is typed from its findings (`verdict.OutcomeOf`).
  A run blocked by one of a few codes is a machine error, by kind:
  `engine-error` (run: the run failed, any shape, a new one too),
  `agent-invalid-output` and `no-test-written` (answer),
  `patch-unreadable` (patch), `build-broken` (build: the tester's test
  or the developer's code), `no-new-test` and `report-missing` (report), `proof-over-time`
  (time: a test command stopped at its deadline, [#1197](https://github.com/JN0V/workline/issues/1197)). No label, no state, no hop: the record
  keeps the count by kind (`errors`), and its comment says the error and
  when the role runs again. The step is due again on the same input
  after 10 minutes (`state-retry`; before, `state-waits` with the time),
  given what failed at the end of its task; 2 tries per kind within
  24 h of that kind's first error (`line.ErrorBudget`). The next error
  of the kind, or its 24 h passed, moves the issue to
  `workline:blocked`, the error said (`machine-error-spent`). A new
  input — the state entered anew, the body or labelling changed —
  starts the count again. The developer's code that does not build is
  asked again once in the run, with the error, as the tester's test is
  ([#1083](https://github.com/JN0V/workline/issues/1083)); not when its
  agent objects to a frozen test beside it: the tester answers that.
  The tester's comment then says the line runs it again, never that it
  blocks it.
- **A run too big for its role asks the product owner first**
  ([ADR-0064, step 3](../adr/0064-size-is-a-target-not-a-wall.md#2-two-checkpoints);
  [#1168](https://github.com/JN0V/workline/issues/1168)): its outcome is
  `needs-input(product-owner: size)`, never a machine error, never run
  again on the same input, which crosses the same
  ([#1160](https://github.com/JN0V/workline/issues/1160)).
  - **When**: its agent not asked for its prompt's size
    (`prompt-over-budget`, passed or not); blocked at its token cap
    (`ai-max-tokens`). A cut of what it read (`input-cut`) is said, never
    a crossing
    ([amended](../adr/0064-size-is-a-target-not-a-wall.md#2-two-checkpoints)).
  - **Then**: the step moves the issue to `workline:blocked`, the size
    said, and that the product owner reads it at once (`size-crossed`).
    The record keeps the role (`size-of`) and the state (`sized`).
  - **The product owner reads it** at the next run, no news needed
    (`state-revisit`), told the facts: what crossed, with the run's
    numbers, and the issue's fresh tokens so far against its normal and
    its backstop
    ([#1193](https://github.com/JN0V/workline/issues/1193)). It narrows
    the Scope and sends it back; grants it `workline:size/large` and sends it back; splits it,
    `split` on for it, and keeps it blocked for a person to accept from
    its parts
    ([ADR-0064, step 4](../adr/0064-size-is-a-target-not-a-wall.md#3-the-product-owner-decides-once-a-state-an-issue);
    [#1171](https://github.com/JN0V/workline/issues/1171),
    [#1172](https://github.com/JN0V/workline/issues/1172)); or keeps it
    blocked, `person: true`. Its run failing, or its answer not read,
    keeps the question, asked again within the machine-error budget.
  - **Once a state, per issue**: a second crossing from a state in
    `sized`, the product owner's own crossing, or a project whose
    product owner reads no blocked issue, waits for a person at once
    (`size-crossed-again`, `size-crossed-own`); so does a crossing at
    `workline:size/x-large`, the top tier (`size-crossed-top`).
  - **The work is kept**: only the state's label moves; a run whose role
    moved the issue keeps that move.
  - A record from before, its machine error of kind `size` or `budget`,
    is spent at once, for a person (`machine-error-spent`).
  - **A bug predicted past its size**, readied for its tests first
    (`split-after-tests` in the product owner's state): due to its
    developer, it is blocked the same way, no run crossed
    (`size-predicted`, `size-of: prediction`), so the product owner
    splits or grants it once its tests are written
    ([split on size](backlog-acts/split.md#split-on-size);
    [#1212](https://github.com/JN0V/workline/issues/1212)).
- **An issue's size tier** ([ADR-0064, §3](../adr/0064-size-is-a-target-not-a-wall.md#3-the-product-owner-decides-once-a-state-an-issue)):
  `workline:size/large` (×2), the product owner's grant, or
  `workline:size/x-large` (×4), a person's label only. Every step on the
  issue is given it (input `size-tier`, `size-tier` said), and every size
  target of its run multiplied: `context.budget`, `ai-max-tokens`,
  `code-lines-max`, `code-bytes-max`, `diff-lines-max`; `workline size`
  measures the issue against its normal times the tier.
- **The per-issue backstop**
  ([ADR-0064, §5](../adr/0064-size-is-a-target-not-a-wall.md#5-the-hard-backstop);
  [#1174](https://github.com/JN0V/workline/issues/1174)): the fresh
  tokens of every run on the issue (`spent`), all roles, its merge
  requests' runs included, against 4× its normal, the tier left out.
  - **Its normal**: its kind's, kept by `workline size`, else the seed
    (400k, 800k at effort High): a backstop of 1.6M, or 3.2M.
  - **`large` never raises it**; a person's `workline:size/x-large`
    lifts it to 16× the normal (6.4M at the seed). Nothing above it.
  - **Before a step**: reached, the step is not run; the issue moves to
    `workline:blocked`, the total and the backstop said
    (`backstop-crossed`; `backstop` in the record).
  - **After a run**: one that takes it there blocks it in the same step,
    wherever its role moved it on the line, over a size question to the
    product owner; its work, a pull request, is kept.
  - **A person's, with no time-out**: no product owner reads it again,
    news or a week (`backstop-waits`). A person puts `x-large` and moves
    it back by its label, or splits it: each new issue counts its own.
  - **A review round stops there too** (the panel of 2026-11-08): due
    on an issue at its backstop, the round is not run, by the line nor
    by `workline develop --merge-request`; the issue is blocked, the
    pull request kept open (`backstop-crossed` on the pull request).
  - **The liveness caps stay**: `max-hops`, `ai-max-turns`,
    `ai-max-seconds`, the machine-error budget, the rounds' `rounds`.
  - **At `x-large`'s 16×, nothing lifts it** (the same panel): a
    person splits the issue or finishes it by hand.
- **The code a role lacks is given**: the product owner naming the
  code an issue is about (`sources`) that its task did not give
  ([the code picker](code-picker.md)), its Scope left for that code, or
  the tester answering `needs: <paths> — why` is a needs-input of the
  engine (`needs-code`), never "no test possible". Due at once: the
  role runs again in the same step, given it (`state-given`; its task
  says so, the tester's and the product owner's get those files). Given
  them, it may name other files, none given yet (a folder's files led
  it to one): it runs once more, given those too
  ([#1410](https://github.com/JN0V/workline/issues/1410)). Said
  lacking again, the step itself moves the issue to
  `workline:blocked` for a person to name its files (`needs-spent`).
- **No code on the same input, a person's**: the developer's claim
  whose agent wrote no code, past `claim-hours` with the issue's body
  unchanged, is not let go to be taken again on that input: the line
  moves the issue to `workline:blocked`, the claim's label off, the
  reason written (`no-code-spent`,
  [#1120](https://github.com/JN0V/workline/issues/1120)). A changed
  body still lets the claim go at once.
- **Left free by the developer, due again**
  ([ADR-0059, amended](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-02-a-ready-issue-left-free-wakes-the-developer-again)):
  a wake of the developer records when (`woke`) and the issues it held
  then (`held`): its claims an open pull request closes, a claim with no
  code holding nothing ([#952](https://github.com/JN0V/workline/issues/952)).
  A ready issue, assigned to nobody, is due again
  (`state-freed`) once its claim was put on since and let go, a hop
  more, so a take-and-let-go cycle ends at `max-hops`; or once an issue
  in `held` is held no more; or, passed over for an overlap, once its
  Scope names no file a claim held changes, once for the comment that said it
  (`unheld`)
  ([#998](https://github.com/JN0V/workline/issues/998)). A record from before `woke` is due once
  where the developer wrote its comment. Never on a clock: no loop.
- **A developer run lost is a machine error**: a run the line woke that
  ends with no verdict read from its post (`engine-error`, as when
  post dies; the run's result says `no-verdict`) reached nothing
  ([#953](https://github.com/JN0V/workline/issues/953)): of kind run,
  as above. A record from before typed outcomes keeps when it was lost
  (`lost`): read as one such error, run again, never written again.
- **Refine's record says the reviewer was asked** on the body (`asked`),
  where the reviewer is routed on `issue`. Acted on with no such mark,
  its spec still waiting on a review, the step is due again, once: a
  record from before the reviewer read it in refine's own run
  ([#840](https://github.com/JN0V/workline/issues/840)). Marked, no
  review come, the step is due once more, for the reviewer: it may have
  skipped the spec then
  ([#1070](https://github.com/JN0V/workline/issues/1070),
  `state-review-missing`; `review-again` in the record). After that, it
  stays acted: no loop. A spec waits on a review by the product owner's
  rule: its sections read with its state, text on top its Need.
- **A review answered by asking for a person's Scope** under the
  rule before
  [ADR-0060](../adr/0060-the-person-is-on-the-loop.md): the product
  owner's state shows a section it may rewrite now and was not given
  then (`backlog.Unrevised`). The step is due again, once a review
  (`revise`, the review's hash): the product owner rewrites it, the
  person no longer waited on.
- **A question past its delay**
  ([ADR-0060](../adr/0060-the-person-is-on-the-loop.md#1-questions-time-out-as-lazy-consensus)):
  in refine, acted on, the product owner's question to the reporter
  unanswered past `acts.ask.wait` — counted from the question's comment
  as the forge dates it (`created_at`), never the issue's last update —,
  or its drafts, all it lacks, unaccepted past it (from the forge's time
  of `workline:draft`). The step is due again, once
  (`state-timeout`; `timeout` in the record, `ask=N` or
  `drafts=<body>`), no agent until then: the product owner writes its
  assumption, or takes the drafts as written. An answer before the
  delay cancels it; `wait: never` (cautious) wakes none
  ([settings](../../roles/product-owner/docs/settings.md#how-long-a-question-waits)).
  Until then the line says it waits (`state-waits`): on its reporter,
  or a person's yes on its drafts, and the day the product owner takes
  its assumption, never silent
  ([#1082](https://github.com/JN0V/workline/issues/1082)). Its comment
  on the issue, written after the product owner's run, says the same:
  what waits, why, and what of a person ends it — a yes on its drafts
  or on what it proposes, or the reporter's answer
  ([#1195](https://github.com/JN0V/workline/issues/1195)).
  Its drafts' own body — the one its state's digest names —, or the
  reviewer's review of it with nothing important open, make the step
  due no sooner: the product owner could write nothing. A person's edit
  since (even one a night review read), a comment it has not read, a
  label put on again, or an important finding does
  ([#1127](https://github.com/JN0V/workline/issues/1127)).
  Its drafts with no review yet on that body, the reviewer on the line:
  the step is the reviewer's alone, the product owner not called
  (`state-review-pending`; `alone` in the record). In CI, the reviewer
  of the judge job read the body before the apply job wrote the drafts;
  its review lands, then the rule above holds it. A retry of that step
  after a machine error, and the once more of a review that did not
  come (`state-review-missing`), run the reviewer alone too; its errors
  and its block name the reviewer
  ([#1150](https://github.com/JN0V/workline/issues/1150)).
- **A person's comment, unread, due again**
  ([#1331](https://github.com/JN0V/workline/issues/1331)): in refine,
  acted on, a person's comment (`backlog.Counted`) the product owner's
  state says it has not read — an answer to the reviewer's question,
  or to its own. The step is due again, once for those comments
  (`state-comment`; `heard` in the record, their count when the line
  woke it): a run that leaves them unread is no loop. On the local
  forge, which names no author, a comment counts as a person of the
  project's. Its run's `ready` held for them, nothing written into the
  spec (`answer-held` in its state), the step is due once more
  (`state-comment-held`; `answer-held` in the record), never again on
  the same comments.
- **Left unrefined, due again**
  ([ADR-0059, amended](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-03-an-issue-left-unrefined-is-woken-again)):
  in refine, acted on, the product owner's state shows it read the body
  as it is and left a required section missing, nothing waiting on a
  person — no question unanswered, no drafts, no proposal, not split,
  left nor set aside. The step is due again on the same body
  (`state-unrefined`; `retry` in the record), twice at most
  (`line.RefineRetries`); the third time the issue moves to
  `workline:blocked`, what it lacks said (`refine-capped`). A body
  changed starts the count again. Never on the same code
  ([#980](https://github.com/JN0V/workline/issues/980)): the record
  keeps the sources each wake gave (`given`); the state naming the same
  when its read left it unrefined, the issue is blocked at once, the
  code it was given said (`refine-capped`).
- **Never read, due again**
  ([#1041](https://github.com/JN0V/workline/issues/1041)): in refine,
  woken on this body, the product owner's state shows it never read the
  issue (no state, or nothing judged). The run read nothing, so the step
  is due again once a body (`state-unread-body`; `unread` in the record).
- **No test possible, unsettled**
  ([#1053](https://github.com/JN0V/workline/issues/1053)): in refine,
  acted on, the tester's "no test possible" still stands on the body and
  the product owner's state says it picked no proof, sent no retest, nor
  rewrote the text. The issue moves to `workline:blocked`, the tester's reason said
  (`no-test-unsettled`): never left in refine with nothing due.
- **Not for an outside failure**: a step that ended
  `blocked-external`, its agent out of reach, keeps no record: the next
  event or reconciler runs it again.
- **Each run's fresh tokens are counted**
  ([ADR-0064, step 2](../adr/0064-size-is-a-target-not-a-wall.md#1-normal-size-is-measured-per-project)):
  the record keeps, by role, the fresh tokens of each run its steps made
  that called an agent (`spent`), through every state; its comment says
  the issue's total so far. A step ended `blocked-external` keeps none.
  A run on a merge request (the reviewer, the inspector, the tester's
  guard and answers, the developer's review rounds) is counted on the
  issue it closes, when applied: once a run, its id kept (`counted`); an
  issue with no record counts none. `workline size` derives the project's normals from them
  ([commands](../commands.md#workline-size)). The issue's total is held
  to its backstop (below).
- **Judged with no write token** (`--no-apply`), the records are left in
  the line's result (`states`) and written by `workline apply --line`,
  after its runs applied.
- **`max-hops`** (8): each state step on an issue is a hop, counted in
  its record. The one past the cap is not run: the issue goes
  `workline:blocked`, the other states' labels off, the count said in
  that comment (`hops-capped`). A person moving it back to a state by
  its label starts its count again. A hop is a new run, not a handoff
  (`max-handoffs` caps those inside one run).
- **Progress starts the count again**
  ([amended](../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-04-progress-starts-the-hop-count-again)):
  the record keeps the furthest state the issue reached (`reached`,
  along each state's `next`). A step into a state further still counts
  as hop 1: the cap stops a loop, never the step that ends it. Back in
  a state reached before, the count goes on.
