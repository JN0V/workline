---
sources: [cmd/workline, internal/engine/engine.go, internal/hooks]
checked: 9e31d4b
verified: agent:claude-code
---
# The commands, in detail

Each command of [usage](usage.md#commands), with every option and case.
What a role does is on its page ([roles](roles.md)).

## Running roles

### workline run-role

`workline run-role <role> --event <event>` runs one role: prepare,
propose, judge, apply, again if `pre` left work for later (`in/more`), up
to 5 rounds. Options: [usage](usage.md#options-of-run-role-and-route).

### workline route

`workline route <event>` runs the steps routing names for the event, in
order; the first that does not pass stops the line. On `schedule`, every
step runs and the worst verdict stands (`fail-fast`,
[ADR-0037](adr/0037-the-schedule-runs-every-step.md)).

### workline apply

`workline apply <run-dir>...` or `--line <file>`:

- applies runs judged with `--no-apply`, or resumes a run stopped while
  applying;
- `--line` takes the runs a `route --no-apply --json` or
  `issues import --json` result lists as `pending`;
- `--summary`, as for `route`.

### workline gate

`workline gate <name>` runs a gate declared in `.workline/config.yaml`
([gates](spec/gates.md)); it returns 0 or 1.

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
([backlog acts](spec/backlog-acts.md#importing-a-file)).

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

The weekly sample of the docs the documentalist confirmed (moved their
`checked`)
([ADR-0014](adr/0014-checked-is-earned-by-what-was-read.md), step 4;
[ADR-0015](adr/0015-the-weekly-sample-is-written-on-the-forge.md)).

- **Which docs**: those whose `checked` its commits moved
  (`Workline-Role: documentalist`, or `verified: agent:documentalist`
  newly set) in the commits reaching the branch in the last whole ISO week
  (`--week 2026-W40`; `--since <rev>` for the commits after one). One in
  ten, rounded up, drawn the same on a rerun of the week.
- **Read** whole against its sources at the commit its `checked` names, by
  the judge (`--judge`, else `WORKLINE_JUDGE`, else the documentalist's
  `sample.judge`), never the model the commit's `Workline-Model` names.
  Every quote the judge gives is checked; a comment is no evidence.
- **Writes nothing**; `--out <file>` keeps the result.
- **`workline sample --apply <file> --forge <forge>`** (default: the
  project's `forge`), with no agent:
  - comments the week on one tracking issue, labelled
    `documentalist-step-0`;
  - opens a merge request putting back a `checked` found false;
  - with no forge, writes nothing and says so.
- **The product owner's acts**: on a project with its report, `--apply`
  also draws one in ten of the acts it did alone that week, from its
  record, onto the issue "workline: the weekly sample of the product
  owner's acts", each with its day, level and whether a person undid it,
  and the level they suggest
  ([ADR-0033](adr/0033-the-weekly-sample-draws-the-product-owners-acts.md)).
- Exits 2 when a doc is left for a person; `--summary` as for `route`.

### workline follow

Run when the default branch moves (CI's `follow` job,
[ADR-0034](adr/0034-the-release-fix-follows-its-base.md)), with no agent.

- Each open merge request from `workline/<role>/release`
  ([ADR-0017](adr/0017-the-release-manager.md)) whose base moved under it
  is rebuilt on the base's new tip and force-pushed, the merge request
  kept (`rebuilt`):
  - its change redone file by file, a doc's front matter key by key, the
    base's value kept where both changed a key;
  - one on the tip already is left alone.
- **Never rebuilt** (`not-rebuilt`, a warning): one holding a commit
  without the role's `Workline-Role` trailer, or a merge, or that did not
  leave from the base's history (its merge request goes into another
  branch).
- **Left for a person**: one whose change no longer applies
  (`no-longer-applies`, exit 2).
- Options: `--base` (default: the branch checked out), `--forge`,
  `--repo`, `--json`.
