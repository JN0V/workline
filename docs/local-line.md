---
sources: [cmd/workline/main.go, internal/forge/local.go, internal/forge/local_merge.go, internal/forge/local_threads.go, internal/doctor/tests.go]
checked: 796746533
verified: agent:claude-code
---
# The whole line with no forge

From a need to a merged change on your machine, with no GitHub nor
GitLab: the product owner refines, the tester writes the tests, the
developer the code, the reviewer reads it; the merge policy merges a
fix, you say yes to a feature.

```
need ──► product owner ──► tester ──► developer ──► review ──► merge
 you      refines it        tests      code          the line    the policy, or you
```

- **Issues and merge requests stay in the clone** (`.git/workline/`),
  never committed nor pushed
  ([ADR-0016](adr/0016-writes-go-where-the-project-lives.md)).
- **You act by command**: `workline issues` opens, labels and comments,
  as you would on a forge's page.
- **An agent is needed** for the roles that write: `claude`, or any
  command as `cmd:` ([the agent](install.md#3-the-agent-your-choice)).
  With `none`, each role says what it skipped.

## 1. Install and adopt

Install the binary and set the machine up ([install](install.md)), then
in the repository:

```sh
workline init --review   # routes pre-push and the reviewer; proposes each doc's sources
```

## 2. Route the line

Write `.workline/config.yaml` (`init` wrote its routing; replace it):

```yaml
forge: local
routing:
  events:
    pre-push: [committer, documentalist]
    merge-request: [committer, documentalist, tester, reviewer]
    issue: [product-owner, reviewer, tester, developer]
autonomy: normal
merge: {autonomy: normal}
roles:
  tester:
    settings:
      test-command: python -m pytest -q --junitxml={report}   # yours: runs the tests, writes JUnit XML to {report}
      test-paths: ["tests/**"]                                # where your tests live
```

- **`issue`**: an issue's state label wakes its role
  ([the line of states](spec/routing.md#the-line-of-states)).
- **`test-command`**: any runner that writes a JUnit XML report
  ([the tester's settings](../roles/tester/README.md#settings)).
- **`autonomy`**: how far the roles go alone ([autonomy](autonomy.md)).
- **`merge`**: what merges with no yes of yours: at `normal`, a bug's
  fix ([the merge policy](merge-policy.md)). Left out, you merge.

```sh
workline doctor          # says a missing test-command, with the setting to add
git add .workline/config.yaml && git commit -m "chore: adopt workline"
export WORKLINE_RUNS_DIR=~/.cache/workline-runs   # outside the clone: see below
```

`WORKLINE_RUNS_DIR` keeps the runs out of `.git/`: Claude Code refuses
to edit a file under `.git/`, where the developer's worktree would be
([#1304](https://github.com/JN0V/workline/issues/1304)).

## 3. Write the need

```sh
workline issues open "sc-late-fee takes NaN for an amount" \
  --body-file need.md --label workline:to-refine --label bug
```

- Write it as a person would: what you saw, what you want
  ([writing an issue](../roles/product-owner/docs/writing.md)).
- `--body "text"` for a short one; `--body-file -` reads stdin.
- `workline issues` lists them; `workline issues show 1` shows one whole.

## 4. Run each step

Each run wakes the role the issue's state names, then moves the issue on:

```sh
workline route issue --target issue:1 --ai claude
workline issues show 1   # what it wrote, and what it asks of you
```

| State label | Wakes | What you do |
|---|---|---|
| `workline:to-refine` | the product owner, then the reviewer on the spec | answer it (below) |
| `workline:needs-tests` | the tester: tests on `workline/tester/issue-1`, a Draft `!1` | nothing |
| `workline:ready` | the developer: code on `workline/developer/1`, `!2` | nothing |
| `workline:developing` | nobody: its merge request waits for the review | review it (step 5) |

Run the same command after each step until the issue is
`workline:developing` with its merge request open.

- **Your checkout stays as it is**: the tester's tests and the
  developer's code are written and committed on their branch in a
  worktree apart; your branch, and your changes not committed, are
  never touched ([#1372](https://github.com/JN0V/workline/issues/1372)).

### Answer the product owner

At `normal`, it proposes its drafts and waits for your yes:

```sh
workline issues label 1 --add workline:accepted   # yes: it moves the issue on at its next run
workline issues comment 1 --body "Refuse inf too."  # it revises what it proposes
```

A comment answers the reviewer's questions on the spec the same way:
the product owner reads it at its next run and writes what it settles
into the Scope.

Its comment on the issue lists every answer
([its run](../roles/product-owner/docs/run.md)).

## 5. Review the merge request

No forge runs a job on a push here: the reconciler does, from `main`.

```sh
workline route reconcile --ai claude  # reviews !2's new head, runs a round due
workline issues show !2               # what the line found on it
```

- **The merge-request line** runs on each developer's request, not a
  draft, whose head has no verdict yet: the committer, the tester's
  check and the reviewer. Their findings go on `!2`.
- **The line's verdict** on that head is kept in the clone, for the
  merge (step 6). A new head is reviewed at the next reconcile.
- **A block is said on `!2`**: which step blocked, its findings, what
  you can do, in one comment edited at each run
  ([#1327](https://github.com/JN0V/workline/issues/1327)).
- **A review round**: a finding, or your comment
  (`workline issues comment '!2' --body "…"`), wakes the developer at
  the next reconcile. Its fix is committed on its branch, unless you
  have that branch checked out: then it says so and waits.
- **A test the review finds missing** goes to the tester, on its Draft
  `!1`: the next reconcile runs its answer, which adds the test or says
  why none is needed
  ([#1265](https://github.com/JN0V/workline/issues/1265)).
- **Each runs in a worktree of the head**, apart: your checkout is
  never read nor written.
- **By hand**:
  - the line: `git checkout workline/developer/1`, then
    `workline route merge-request --target merge-request:2 --ai claude`.
    A doc fix it proposes goes in a comment on `!2`, never in your tree.
  - a round, from any branch: `workline develop --merge-request 2 --ai claude`.
    It runs in a worktree of `!2`'s head.

## 6. The merge

```sh
git checkout main
workline follow          # merges what the policy says yes to; closes its issue
workline issues          # #1 closed, !1 and !2 merged
```

`workline follow` does what a forge's auto-merge would
([ADR-0042](adr/0042-the-engine-merges-by-the-projects-policy.md#amendment-2026-11-10-1322-the-local-forge-merges-itself)):

- **A bug's fix merges alone**: its tests red on `main` first, every
  gate passed, the reviewer's verdict on its head, no Important finding.
- **A feature or a task waits for your yes**, said on `!2`:
  `workline issues label !2 --add workline:accepted`, then `follow`.
- **Only the head that passed**: a commit since waits for the next
  reconcile's review.
- **Into `main` in this clone**: a merge commit naming `!2`; nothing is
  pushed. `main` checked out with uncommitted changes, or a conflict:
  nothing merges, the reason said on `!2` and on its issue.
- **Behind `main`, which changed one of its files**: no run checks
  `main` here, so `follow` merges `main` into its branch and runs your
  `test-command` on that merge. Green: the branch moves there, the next
  reconcile reviews it, the next `follow` merges it. Red: nothing moves,
  the test named on `!2` and its issue
  ([#1370](https://github.com/JN0V/workline/issues/1370)).
- **A conflict with `main`** names its files: the next reconcile's
  developer round merges `main` in, never a rebase
  ([behind main](../roles/developer/docs/conflict.md)). One that fails
  blocks the issue: you merge `main` into its branch, or close `!2`
  ([#1354](https://github.com/JN0V/workline/issues/1354)).
- **No `merge:` setting**: you merge with git, then `follow` closes the
  issue:

```sh
git checkout main && git merge --ff-only workline/developer/1
workline follow
```

## When a step stalls

- **A state that wakes nobody**: `workline issues show 1` says why on
  the issue; `workline route reconcile --ai claude` runs what is due.
- **A run that changed nothing**: run again after the body or labels
  change, or after you answer with `workline issues comment`; the line
  does not ask twice on the same input.
- **More**: [troubleshooting](troubleshooting.md),
  [the product owner's](troubleshooting-product-owner.md).
