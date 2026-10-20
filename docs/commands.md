---
sources: [cmd/workline, internal/engine/engine.go, internal/hooks]
checked: 1f6feed1
verified: agent:claude-code
judged: e343388
---
# The commands, in detail

Each command of [usage](usage.md#commands), with every option and case.
What a role does is on its page ([roles](roles.md)).

## Running roles

### workline run-role

`workline run-role <role> --event <event>` runs one role: prepare,
propose, judge, apply, again if `pre` left work for later (`in/more`), up
to 5 rounds, then a task it asks last (`in/then`). [Options](usage.md#options-of-run-role-and-route).

### workline route

`workline route <event>` runs the steps routing names for the event, in
order; the first that does not pass stops the line. On `schedule`, every
step runs and the worst verdict stands (`fail-fast`,
[ADR-0037](adr/0037-the-schedule-runs-every-step.md)); an agent's proposal
refused there is said as warnings, not failed
([amended](adr/0037-the-schedule-runs-every-step.md#amendment-2026-10-09-a-refused-proposal-is-said-not-failed)).
With `--gates-only`, the event's gates alone run, each one, for a CI job
of their own; `--gates-from <file>` reads their verdicts from its result
instead of running them ([gates in CI](gates-in-ci.md)).
With `--warn-apart`, a role that reads what only warns apart on the event
(role.yaml's `apart`: the reviewer's lenses that warn, on a merge request)
leaves it; `--warn-only` runs only that, never a gate nor another role,
and writes no verdict (`--verdict` refused): what only warns, apart from
the verdict ([apart](../roles/reviewer/docs/apart.md)). Both flags, or one
and an `--input warn=` naming the other run, are misuse.
A release tool's merge request: on `schedule` with a forge and
`--open-merge-request`, its docs are judged first, on its head;
`route release --target merge-request:<n>` does it now, its head checked
out for the run, and `--target merge-request:release` finds the one open,
for a nightly job ([more](../roles/documentalist/docs/push.md#a-release-tools-pull-request)).
With `--input queue=true`, a merge queue's or train's run: the gates
alone, on the merged result, the steps left named; `--verdict` and
`--warn-only` refused ([a merge queue or train](merge-queue.md)).
`route main`, after a merge: the merge request's gates, then the
whole repository's checks, every step run; `--gates-kept <folder>`
keeps the gates' findings there by commit, and on a merge request reads
its merge base's there ([main after a merge](main-after-merge.md)).
`route release --dry-run`: what would hold the release if it were cut
now, on the release merge request open, else the branch's tip; every
step, no agent, nothing applied, said as warnings
([the release](release.md)).

### workline apply

`workline apply <run-dir>...` or `--line <file>`:

- applies runs judged with `--no-apply`, or resumes a run stopped while
  applying;
- `--line` takes the runs a `route --no-apply --json` or
  `issues import --json` result lists as `pending`;
- stops at the first run that does not pass, and says the runs left after
  it (`runs-not-applied`), with the command that resumes them;
- `--summary`, as for `route`: what was done on each issue, what failed,
  what was left.
- `--from-merge-request <n> --head <sha>`: merge request `<n>`'s own CI
  run read as data — only its proposals; settings, roles, forge this
  job's, from the base (`--repo`); once the branch moved on, only the
  reviewer's review of the head judged, never a fix
  ([more](spec/routing.md#on-a-forge-judge-then-apply)).
- With both, on GitHub: the line's documentalist findings as annotations
  of the check `workline findings` on the head judged; `checks: write`
  ([#412](https://github.com/JN0V/workline/issues/412)).
- `--gates <file>`, with `--from-merge-request`: what the gates found the
  change adds, from the result `route --gates-only --json` printed in
  that run; on GitHub, the annotations of the check `workline gates` and
  one summary comment ([gates](spec/gates.md#shown-on-the-merge-request)).
- `--main <file>`, alone: what `route main --json` found, kept in one
  issue, edited only when what it lists changes; `--forge`, `--repo`
  ([main after a merge](main-after-merge.md)).
- `--release-dry-run <file>`, alone: what `route release --dry-run
  --json` found, in one comment on the release merge request it judged,
  else in a part of main's issue; edited only when what it lists
  changes ([the release](release.md)).

### workline gate

`workline gate <name>` runs a gate declared in `.workline/config.yaml`
([gates](spec/gates.md)); it returns 0 or 1. Its summary says how it
counted ([more](spec/gates.md#with-workline-gate)):

- **Every finding**, by default.
- **`--base <rev>`**: only those the change adds since the merge base of
  `<rev>` and `HEAD`; `--gates-kept <folder>`, the base's read as main
  kept them.

## workline review

The reviewer reviews this branch before it is pushed
([the reviewer](../roles/reviewer/README.md),
[ADR-0020](adr/0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md)).

- **The range**: `<base>..HEAD` (`--base`, default the role's `base`
  setting, `main`), every lens (`--lenses` names some).
- **What it does**: the rules on the comments the change adds, then each
  lens, each finding's quote found again, the change's findings told from
  those outside it, each important one judged.
- **What it prints**: the findings, the tokens each call used against the
  reviewer's `ai-max-tokens`, and the path of the run's `out/review.json`,
  for the author's agent to fix them before pushing.
- **Options**: `--json`, `--sarif`, `--code-quality`, `--ai`, `--forge`
  (where an issue for what lies outside the change goes).
- **A spec instead**: `--spec <file>` or `--issue <n>`
  ([#128](https://github.com/JN0V/workline/issues/128)), on the
  reviewer's `spec` event: the file as it reads in the working tree, or the
  issue's body from the forge, through the spec lenses (ambiguous,
  unverifiable, out of scope, contradicted by the code); each finding
  quoted from the spec, the important ones judged
  ([a spec](../roles/reviewer/docs/spec.md)).

## The backlog

### workline item ready

`workline item ready <id>` moves a work item to `ready`, once its Need,
Verification, Validation and Scope are written (`--forge` reads it from
the forge).

### workline issues

`workline issues`, `workline issues show <n>` or `show !<n>` reads the
local forge (`forge: local`,
[ADR-0016](adr/0016-writes-go-where-the-project-lives.md)): lists the
issues (`#<n>`) and merge requests (`!<n>`) kept in the clone, with their
state and labels, or shows one whole, its comments after its body. It
writes nothing.

### workline issues import

`workline issues import <file> [--apply]` moves a roadmap or backlog file,
whatever its form, to the forge's issues
([backlog acts](spec/backlog-acts/import.md)).

- The product owner reads it a share at a time; each item still to do is
  opened once, its text quoted from the file.
- **Without `--apply`**: only says what it would open, writes nothing, and
  lists its runs as pending (`to apply:`; `--json`: `pending`).
  `workline apply --line <result>` opens them with no agent, in the job
  that holds the write token.
- **The map**
  ([ADR-0030](adr/0030-an-import-maps-every-item-to-its-issue-or-why-not.md)):
  each item of the file, by its lines and first words, to its issue —
  opened, already open, closed — or why it has none: done, the words
  quoted; not an item, why; past the cap.
  - The lines no answer holds are listed under "Not covered", which exits
    2 (`--json`: the result's `coverage`).
  - The map stays the judge's: run the import again to see the issues
    open.
- **Options**: `--ai`, `--forge`, `--lines`; `--summary` as for `route`,
  the map with it.


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

### workline sample

[The weekly sample](weekly-sample.md) of every role's acts
([ADR-0014](adr/0014-checked-is-earned-by-what-was-read.md), step 4;
[ADR-0046](adr/0046-the-auditor-re-checks-every-roles-acts.md)): one in
ten of the acts of the last whole ISO week (`--week 2026-W40`; `--since
<rev>` for the commits after one) per role and kind, at least one when a
kind has any, drawn the same on a rerun.

- **The acts**: the docs whose `checked` the documentalist moved
  (`Workline-Role: documentalist`, or `verified: agent:documentalist`
  newly set); from the forge (`--forge`, else the project's), the product
  owner's acts done alone, the findings the reviewer's judge kept or
  dropped on every run, from the lines its summary comment hides
  ([more](../roles/reviewer/docs/judge.md#every-finding-judged-kept-for-the-weekly-sample)),
  the committer's rewrites. What cannot be drawn is said.
- **Each re-checked**, one call an act, by the judge (`--judge`, else
  `WORKLINE_JUDGE`, else `sample.judge`), never a model that acted;
  a finding blind. False only with a quote the engine finds again, a
  comment of the code no evidence; else for a person. A false one goes
  to three judges (`sample.panel`), who settle it. Without a judge, each
  is listed for a person.
- **A cap**: `sample.ai-max-tokens`, 400,000 by default; each role's first
  act read before any second, the rest listed unread.
- **A squash**: each merge request's head is fetched from the forge's ref
  for it, to read the commit a `checked` names and who moved it there.
- **Writes nothing**; `--out <file>` keeps the result.
- **`workline sample --apply <file> --forge <forge>`** (default: the
  project's `forge`), with no agent:
  - comments the week on one tracking issue, its acts numbered, labelled
    `documentalist-step-0` when a `checked` is false; rewrites the count
    there from every week and the maintainer's 👍 or 👎;
  - opens an issue for a true finding the reviewer's judge dropped, still
    in the code, three a week at most;
  - opens a merge request putting back a `checked` found false;
  - with no forge, writes nothing and says so.
- **The product owner's acts**: each read on the issue's text and code
  its record says it read; each a person undid said, with the level they
  suggest ([ADR-0033](adr/0033-the-weekly-sample-draws-the-product-owners-acts.md));
  its former issue of its own closed.
- Exits 2 when a doc or an act is left for a person; `--summary` as for
  `route`.

### workline follow

Run when the default branch moves (CI's `follow` job,
[ADR-0034](adr/0034-the-release-fix-follows-its-base.md)), with no agent.

- Each open merge request from `workline/<role>/release`
  ([ADR-0017](adr/0017-the-release-manager.md)) whose base moved under it
  is rebuilt on the base's new tip and force-pushed, the merge request
  kept (`rebuilt`):
  - its change redone file by file, a doc's header key by key, the
    base's value kept where both changed a key;
  - one on the tip already is left alone.
- **Never rebuilt** (`not-rebuilt`, a warning): one holding a commit
  without the role's `Workline-Role` trailer, or a merge, or that did not
  leave from the base's history (its merge request goes into another
  branch).
- **Left for a person**: one whose change no longer applies
  (`no-longer-applies`, exit 2).
- **`--gardening`**, run by gardening's applying job: each other merge
  request of a role (`workline/<role>/<task>`) is tended too
  ([ADR-0034, amended](adr/0034-the-release-fix-follows-its-base.md#amendment-2026-10-12-gardenings-merge-requests-follow-too-or-are-closed)):
  - rebuilt the same way (`rebuilt`);
  - closed, one comment saying why (`closed`), when nothing is left to
    merge: the base holds its change, changed the same lines, or changed
    the docs it only records as read (a record whose docs it did not
    change is rebuilt);
  - one a person committed to is left to them, named when it is in
    conflict with the base (`not-rebuilt`), or when git cannot tell
    (`conflict-unread`).
- **People's pull requests**, with the documentalist's `update-branches:
  conflicts`: one in conflict only on lines workline writes gets the base
  merged in (`merged`); any other is left, named (`not-merged`)
  ([when the base moves](../roles/documentalist/docs/follow.md)).
- **The [merge policy](merge-policy.md)**, with `merge:`: auto-merge on
  for what a person accepted — at `normal`, workline's own routine work
  too —, the first in line brought up to date.
- Options: `--base` (default: the branch checked out), `--forge`,
  `--repo`, `--json`, `--summary`; `--verdict <file>`, a head's verdict
  `route --verdict` wrote, kept first (`--verdict-only`: nothing else);
  `--merge-group <branch> --head <commit>`, with `merge:`, the yes of the
  head a merge queue's group ends with set on the group's commit, and
  nothing else ([a merge queue](merge-queue.md)).
