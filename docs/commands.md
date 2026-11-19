---
sources:
  - cmd/workline/main.go#runRole
  - internal/engine/engine.go#maxRounds
  - cmd/workline/main.go#routeCmd
  - internal/line/line.go#Run
  - internal/line/line.go#MainEvent
  - internal/line/line.go#SkipInput
  - internal/line/line.go#stepsOf
  - internal/line/states.go#IssueEvent
  - internal/line/states.go#ReconcileEvent
  - internal/role/role.go#Role.ApartOn
  - roles/reviewer/role.yaml#apart
  - roles/inspector/role.yaml#apart
  - cmd/workline/main.go#applyCmd
  - internal/engine/engine.go#Resume
  - internal/engine/fromrun.go#RunName
  - cmd/workline/main.go#keepMain
  - cmd/workline/main.go#keepDryRun
  - internal/forge/annotate.go#FindingsCheck
  - internal/forge/annotate.go#GatesCheck
  - internal/engine/annotate.go#annotateFix
  - cmd/workline/main.go#gateCmd
  - cmd/workline/main.go#docsCmd
  - cmd/workline/main.go#docsJudgedRef
  - cmd/workline/main.go#docsJudgedFrom
  - cmd/workline/main.go#judgeDocs
  - internal/review/review.go#Docs
  - internal/release/release.go#Last
  - cmd/workline/main.go#sampleCmd
  - cmd/workline/main.go#exitFor
  - internal/sample/sample.go#OneIn
  - internal/sample/apply.go#StepZeroLabel
  - internal/sample/apply.go#issueIntro
  - cmd/workline/main.go#sizeCmd
  - internal/line/size.go#SizeTitle
  - internal/line/size.go#issueSeed
  - internal/line/size.go#sizeIntro
  - internal/line/size.go#Size
  - internal/line/size.go#doneRecords
  - internal/line/size.go#doneByMR
  - cmd/workline/main.go#followCmd
  - internal/engine/follow.go#Follow
  - internal/engine/follow.go#followMay
  - internal/engine/follow.go#maxDraftRebuilds
  - internal/engine/done.go#settleDone
  - internal/forge/local.go#Local.SettleMerges
checked: e4e4287bc
verified: agent:claude-code
---
# The commands, in detail

Each command of [usage](usage.md#commands), with every option and case.
What a role does is on its page ([roles](roles.md)).

## Running roles

### workline run-role

`workline run-role <role> --event <event>` runs one role
([the role contract](spec/role-contract.md)). A role with more to do
than one round leaves the rest for the next: 5 rounds at most.
[Options](usage.md#options-of-run-role-and-route).

### workline route

`workline route <event>` runs the steps routing names for the event, in
order; the first that does not pass stops the line.

- **On `schedule`**, every step runs and the worst verdict stands
  (`fail-fast: false`, [ADR-0037](adr/0037-the-schedule-runs-every-step.md)).
- **`--input skip=<step>,…`** leaves those steps out
  ([steps left out](spec/routing.md#steps-left-out)).
- **`--gates-only`**: the event's gates alone, each one, for a CI job of
  their own; `--gates-from <file>` reads their verdicts from its result
  ([gates in CI](gates-in-ci.md)).
- **`--warn-apart`**: a role that reads what only warns apart (role.yaml's
  `apart`: the reviewer's lenses that warn, the inspector's triage) leaves
  the event; `--warn-only` runs only that, writes no verdict, and given
  `--gates-from` hands the gates' result to the inspector
  ([apart](../roles/reviewer/docs/apart.md),
  [the inspector](../roles/inspector/README.md#set-it-up)). Both flags,
  or one and an `--input warn=` naming the other run, are misuse.
- **`--input queue=true`**: a [merge queue's or train's](merge-queue.md)
  run, the gates alone on the merged result; `--verdict`, `--warn-only` refused.
- **`route main`**, after a merge: the merge request's gates, then the
  event's own steps; `--gates-kept <folder>` keeps the gates'
  findings by commit, read back at a merge request's merge base
  ([main after a merge](main-after-merge.md)).
- **`route release --dry-run`**: what would hold the release cut now,
  every step, no agent, nothing applied, as warnings
  ([the release](release.md)); `--release-wait 30m` waits for the checks
  still running, then reads them again
  ([what holds a release](release-holds.md#read-again)).
- **`route merge-request --target merge-request:<n>`**, with no
  `--input range=`: the commits judged are read from its branches here;
  else give `--input range=<base>..<head>` and `--branch`.
- **`route issue --target issue:<n>`** runs the step the issue's state
  starts; **`route reconcile`** walks every open issue and starts the
  steps due ([the line of states](spec/routing.md#the-line-of-states)).
  With a forge, both hand each role's work over in a merge request
  (`--open-merge-request=false` to turn it off).

### workline apply

`workline apply <run-dir>...` or `--line <file>`:

- applies runs judged with `--no-apply`, or resumes a run stopped while
  applying, the agent not asked again;
- `--line` takes the runs a `route --no-apply --json` result lists as
  `pending`;
- stops at the first run that does not pass, and says the runs left after
  it (`runs-not-applied`), with the command that resumes them;
- `--summary`, as for `route`: what was done on each issue, what failed,
  what was left.
- Each `<run-dir>` given is a folder named as the engine names one
  (`<date>T<time>.<ns>-<role>`); another name, `..` among them, is
  misuse (64), nothing applied.
- `--from-merge-request <n> --head <sha>`: merge request `<n>`'s own CI
  run read as data — only its proposals; settings, roles, forge this
  job's, from the base (`--repo`); only what that merge request's line
  may do is applied, to it
  ([more](spec/routing.md#on-a-forge-judge-then-apply),
  [the tester's answers](objection-jobs.md)).
- With `--line` too, on GitHub: the line's documentalist findings as
  annotations of the check `workline findings` on the head judged; `checks: write`
  ([#412](https://github.com/JN0V/workline/issues/412)).
- `--gates <file>`, with `--from-merge-request`: what the gates found the
  change adds, from the result `route --gates-only --json` printed in
  that run; on GitHub, the annotations of the check `workline gates` and
  one summary comment ([gates](spec/gates.md#shown-on-the-merge-request)).
- `--main <file>`, alone: what `route main --json` found, kept in one
  issue, edited only when what it lists changes; `--forge`, `--repo`
  ([main after a merge](main-after-merge.md)).
- `--release-dry-run <file>`, alone: what `route release --dry-run
  --json` found, in a part of main's issue; edited only when what it lists
  changes; no issue opened when nothing would hold it
  ([the release](release.md)).

### workline gate

`workline gate <name>` runs a gate declared in `.workline/config.yaml`
([gates](spec/gates.md)); it returns 0 or 1. Its summary says how it
counted ([more](spec/gates-told.md#with-workline-gate)):

- **Every finding**, by default.
- **`--base <rev>`**: only those the change adds since the merge base of
  `<rev>` and `HEAD`; `--gates-kept <folder>`, the base's read as main
  kept them. An empty `--base` (an unset variable) is misuse.

The tester's, the developer's, the reviewer's and the backlog's
commands: [the commands that do the work](commands-work.md).

## The docs and main

### workline docs

Your agent judges the docs made suspect since they were last judged,
pushed or not
([ADR-0010](adr/0010-docs-judged-on-the-merge-request-and-by-gardening.md));
then you keep or drop each change, doc by doc (`v` shows it).

- **Since when**: `refs/workline/docs-judged`, else the last release (the
  highest version tag merged, `release.tags`), else every commit;
  `--since <rev>` to choose.
- **Kept** changes go in one `docs:` commit, alone
  ([ADR-0007](adr/0007-docs-reviewed-after-the-code-before-the-push.md)).
- **Without a terminal**, the changes stay in the working tree for a
  person, who reviews them the same way with `workline docs --review`.
- Docs with changes not committed are refused, so the agent's are reviewed
  alone.
- Once the run passed with nothing left to review, the ref moves to HEAD.
- **`--json`** prints the result as JSON on stdout; a refusal prints one
  too, its finding's rule naming the cause (`config-invalid`,
  `not-in-repo`, `since-unknown`, `no-terminal`, `review-failed`), and the
  reason stays on stderr.

### workline sample

[The weekly sample](weekly-sample.md) of every role's acts
([ADR-0014](adr/0014-checked-is-earned-by-what-was-read.md), step 4;
[ADR-0046](adr/0046-the-auditor-re-checks-every-roles-acts.md)): one in
ten per role and kind of act, of the last whole ISO week (`--week
2026-W40`; `--since <rev>`: what was vouched for after that commit).

- **The acts**: the docs whose `checked` the documentalist moved; from
  the forge (`--forge`, else the project's), the product owner's acts
  done alone, the findings the reviewer's judge kept or dropped, the
  committer's rewrites, and more ([every kind](weekly-sample.md)).
- **Each re-checked** by the judge (`--judge`, else `WORKLINE_JUDGE`,
  else `sample.judge`), never a model that acted. False only with the
  material's words, which the engine finds again; else for a person.
  Three judges settle a dispute.
- **Writes nothing**; `--out <file>` keeps the result for `--apply`.
- **`workline sample --tally <file>`** writes only what the runs folder's
  runs spent on agent calls, one line a run that called one: a CI job's
  artifact, read by the week's tokens
  ([in CI](weekly-sample.md#what-it-writes)). No file when none called
  an agent.
- **`workline sample --apply <file> --forge <forge>`** (default: the
  project's `forge`), with no agent:
  - comments the week on one tracking issue, its acts numbered, labelled
    `documentalist-step-0` when a `checked` is false; rewrites the count
    there from every week and the maintainer's 👍 or 👎;
  - opens an issue for a true finding the reviewer's judge dropped, still
    in the code, three a week at most;
  - opens a merge request putting back a `checked` found false.
- Exits 2 when a doc or an act is left for a person; `--summary` as for
  `route`.

### workline size

The normal size of an issue in this project
([ADR-0064](adr/0064-size-is-a-target-not-a-wall.md#1-normal-size-is-measured-per-project), step 2),
with no agent; written again each night by gardening, or run by hand.

- **Read**: the line's record on each issue, the fresh tokens of each
  run by role ([routing](spec/routing.md#once-a-state-the-record)).
  - The closed issues, the last updated first, until 100 the line
    touched (those with its record), 10 pages at most.
  - Done: closed by a merge request, as the forge links it; on the
    local forge, closed as completed.
  - The open issues in a state.
- **The normals**, per kind (bug, feature, task): an issue's total and
  a run of each role, the 85th percentile of the issues done.
- **The seed**, until 10 issues of a kind are done (a run's, until its
  role has 10 runs too): 400k an issue, a run its role's
  `ai-max-tokens`; both doubled for an issue of the highest effort.
- **Kept** in one comment on the issue "workline: the normal size of an
  issue", written again in place, with the open issues past their
  normal, their total or a run. `--no-apply` writes nothing.
- **Nothing written** with no line of states (`routing.states`), or no
  issue holding the line's count yet.
- **An issue's normal sets its backstop**, 4× it: past it, the line
  blocks the issue for a person
  ([routing](spec/routing.md#the-line-of-states)). A run's normal is
  reported only.
- `--forge`, `--json`, `--summary` as for `follow`.

### workline follow

Run when the default branch moves (CI's `follow` job), with no agent.

- **`--gardening`**, run by gardening's applying job: each merge
  request of a role (`workline/<role>/<task>`) whose base moved under it
  is tended
  ([ADR-0034, amended](adr/0034-the-release-fix-follows-its-base.md#amendment-2026-10-12-gardenings-merge-requests-follow-too-or-are-closed);
  [more](../roles/documentalist/docs/gardening-merge-requests.md#when-the-base-moves)):
  - rebuilt on the base's new tip and force-pushed, the merge request
    kept (`rebuilt`); one on the tip already is left alone; the
    tester's Drafts, 5 a run at most, the oldest first;
  - never rebuilt (`not-rebuilt`, a warning) when it did not leave from
    the base's history (its merge request goes into another branch);
  - closed, one comment saying why (`closed`), when nothing is left to
    merge: the base holds its change, changed the same lines, or changed
    the docs it only records as read (a record whose docs it did not
    change is rebuilt);
  - one a person committed to is left to them, named when it is in
    conflict with the base (`not-rebuilt`), or when git cannot tell
    (`conflict-unread`).
- **People's pull requests** are never written to: a conflict with the
  base, on a doc's `checked` too, is their author's
  ([ADR-0051](adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-3-no-docs-obligation-per-pr)).
- **The [merge policy](merge-policy.md)**, with `merge:`: auto-merge on
  for each merge request a person accepted, each merging once its own
  head is green.
- **What a merge left open**: the tester's Draft of a closed issue is
  closed, and a closed issue loses the states' labels
  ([the hand-over](../roles/tester/docs/hand-over.md#what-the-builder-does)).
- **A local merge request merged with git** (`forge: local`): each open
  issue its body says it closes (`Closes #<n>`) is closed, with a comment
  saying why, and the request is written `merged`, once; the summary
  counts both ([no forge](no-forge.md)).
- Options: `--base` (default: the branch checked out), `--forge`,
  `--repo`, `--json`, `--summary`; `--verdict <file>`, a head's verdict
  `route --verdict` wrote, kept first (`--verdict-only`: nothing else);
  `--merge-group <branch> --head <commit>`, with `merge:`, the yes of the
  head a merge queue's group ends with set on the group's commit, and
  nothing else ([a merge queue](merge-queue.md)).
