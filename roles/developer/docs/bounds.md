---
sources:
  - internal/developer/bounds.go
  - internal/developer/developer.go
  - internal/developer/worktree.go
  - internal/developer/prbounds.go
  - internal/steer/steer.go
checked: 87d9ef80
verified: agent:claude-code
---
# The bounds: what its code may never touch

Part of [the developer](../README.md). Step 2 of
[ADR-0054's plan](../../../docs/adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md#what-bounds-it).

```mermaid
flowchart LR
  agent(["Agent"])
  patches["<b>Patches</b><br/>and objections"]
  post["<b>Post</b><br/>no agent"]
  kept["<b>Applied</b><br/>within bounds"]
  refused["<b>Refused</b><br/>said, recorded"]
  agent --> patches --> post --> kept
  post --> refused
```

## What it checks

The agent proposes patches, or edits a worktree
([below](#a-worktree-its-diff-is-the-code)). The engine's judge step,
with no agent, checks each file a patch touches, in this order. The
first rule that matches refuses the whole patch:

| Rule | The file is… | Refused even if the Scope names it |
|---|---|---|
| `gate-file` | under `.workline/`, `.github/workflows/`, `.github/actions/`, `.gitlab/ci/`, or `.gitlab-ci.yml`; a script a gate's check runs; a check's `config` | yes |
| `steering-file` | a file that steers agents: AGENTS.md, CLAUDE.md, a page they link, a prompt; a file in a role's folder (`roles/<name>/`, its `role.yaml` there) | yes |
| `acceptance-test-edit` | a test the tester froze on its open Draft ([the hand-over](../../tester/docs/hand-over.md)) | yes |
| `outside-scope` | none of the paths below | — |

**What it may touch:**

- the paths in backticks in the issue's Scope;
- the product owner's `sources` for the issue;
- the tests beside them: a name holding `test` or `spec`, in a Scope
  file's folder or under a Scope folder, or in a `test`/`tests` folder
  that sits there. A Scope `src` lets in no test at the root.

**Never in bounds:**

- a Scope or source naming the repository's root (`.`, `./`): it is
  dropped, else every file would be in bounds;
- anything, when the engine cannot read the gates' config or the files
  that steer agents: every patch is refused (`bounds-unread`), the
  error said, rather than one let past.

A patch naming no file, or a diff header whose two names cannot be told
apart with no other line naming them, is refused (`patch-unread`).
A rename or a deletion is checked on both sides, a name git quotes read
back. A gate's `config`
path is compared cleaned (`./x`, `x/`); a folder covers its files.

## A worktree: its diff is the code

Its agent works in a worktree
([ADR-0056](../../../docs/adr/0056-the-developers-agent-works-in-a-sandbox-with-tools.md);
what each agent gets: [the agent](agent.md)).

- **Where**: the commit its code starts from (the tester's Draft, or
  the commit read), checked out in the run's folder. The agent runs
  there, its path in `WORKLINE_WORKTREE`, and edits in place.
- **What is taken**: the diff against that commit, new files in,
  whatever the agent committed, a deleted file a deletion; one patch a
  file, held to the rules above. A file's mode is not read.
- **Refused before the rules**: a link out of the tree
  (`symlink-out`); another link, a binary, a submodule (`not-text`).
- **Reverted**: a refused file leaves nothing. The worktree stays
  while the agent may be asked again, then the engine removes it,
  whatever post did; only what passes the bounds goes on.

## On the pull request

The same rules hold on its pull request, as a check: whoever pushed
its commits, the agent, a person or the engine
([ADR-0058](../../../docs/adr/0058-the-forge-guards-the-project-not-the-agents-environment.md#4-the-developers-bounds-become-a-check-on-its-pull-request)).
The check above, before the push, stays a fast first pass.

- **Run**: `workline develop --bounds --merge-request <n>`, or the
  developer listed on the `merge-request` event's line
  (`routing.events.merge-request`). No agent.
- **Which pull requests**: a `workline/developer/<n>` branch, held to
  issue `n`'s bounds. Another branch is a person's: it passes, said
  (`not-a-developers-branch`), and is reviewed as theirs.
- **What it reads**: each file the head changed since its merge base,
  its rules read at the base's tip, never at the head: a pull request
  cannot loosen its own check. The head and base: `--input
  range=<base>..<head>`, else the branches as fetched.
- **Frozen tests**: from the tester's record, with each test's blob. A
  test as the tester wrote it passes; changed or deleted, it blocks,
  deleted even when the range shows no trace of it.
- **A refused file blocks**, named with its rule; a link out of the
  tree is `symlink-out`. Its next round or a person reverts it.
- **Kept for the auditor**: a comment on the pull request,
  `sticky=developer/bounds`, its hidden record `act: bounds` with each
  file refused, kept even when it blocks.
- **Not yet**: the check required on the default branch, run from its
  definition, in the CI templates (step 2 of
  [the ADR's plan](../../../docs/adr/0058-the-forge-guards-the-project-not-the-agents-environment.md#build-plan)).

## Why these rules

- **Gate files**: a gate checks the code; the code's author never
  changes what checks it. Only a person does.
- **Steering files**: they steer every agent, the developer's too.
  Whatever the Scope says, a person changes them.
- **Frozen tests**: the tester wrote them from the issue's scenarios,
  before the code ([ADR-0055](https://github.com/JN0V/workline/pull/648)).
  The developer objects; it never edits them.
- **Outside the scope**: anything found there belongs in an issue
  ([principle 8](../../../docs/PRINCIPLES.md)).

## What happens to a refusal

- **Left out**: the refused patch is not applied; the others are.
- **Said**: a `warn` finding for each file, named by its rule.
- **Recorded**: the claim's comment on the issue lists each file and
  why. "A person must do it" when the file is a gate's or steers
  agents. Its hidden record keeps `refused: [{file, rule}]`, for the
  [auditor](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md).

## What else it refuses

- **Labels and comments of the agent's** (`not-the-developers`): the
  claim is the engine's. The developer never accepts, merges, closes
  or moves an issue. The catalogue holds no merge to propose.
- **One comment is allowed: an objection** to a frozen test. The agent
  gives the test, its line, the scenario, the kind and why, as a map:
  its question shows the item whole, since a text comment is refused.
  The engine
  posts it on that line of the tester's Draft, opened by
  `workline:objection`, in [the tester's form](../../tester/README.md#frozen-tests-and-objections).
  The tester keeps or rewrites the test.
- **An objection to a test that is not frozen** for the issue, or
  missing a field, or when the Draft's commit cannot be read: refused,
  said.

## Not yet

- **A refused patch is not asked again**: the code is applied without
  it; the proof (step 3) shows what is missing.
- **Where the code goes**: a branch of its own, proven, opened ready
  or as a Draft ([the proof](proof.md)); a refused file is said in its
  `## Left out` ([the pull request](pull-request.md)).
