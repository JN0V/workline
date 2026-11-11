# ADR-0067: A role request behind main merges main in

- **Status:** accepted (2026-11-11), built by
  [#1353](https://github.com/JN0V/workline/issues/1353)
- **Date:** 2026-11-11
- **Decided by:** a panel of three independent agents, in consensus
  (the maintainer's rule: consensus acts)
- **Extends:** [ADR-0041] (`follow` merged the base into a branch, never
  a rebase; removed by [ADR-0051] but for the recognition of its merges)
  and [ADR-0042] (the engine merges by the project's policy, which left
  "up to date" to a later decision)
- **Builds on:** [ADR-0013] (what was judged is not asked again),
  [ADR-0054] (the developer's review loop), [ADR-0055] (frozen tests and
  objections); principles 1, 6, 12
- **Research:** [merge-conflicts.md](../research/merge-conflicts.md)

## Context

- **Two role requests change the same file; one merges; the other
  conflicts with main, and nothing brings it up to date.** The V1
  acceptance round 2 ([#1266]) lost TG-2 (Kotlin) and DC-3 (C++) to it.
- **What happened instead:**
  - `follow` said `rebase … on it`, in its own output only.
  - A round asked by a person copied main's code in a one-parent commit:
    `git merge-tree` still saw the conflict, and each next round listed
    more of main's files beyond the Scope, 7 by round 3.
  - Another round wrote no fix: the agent may not commit.
- **The engine already has the pieces**: git's merge in memory
  (`git merge-tree`), its own commits on a role's branch, a review round
  starting from a commit it prepares, the reviewer diffing from the
  merged-in tree ([ADR-0013]'s amendment).

## Decision

**When the merge of a role request is refused because it conflicts with
its base, the engine merges the base in.** A merge, never a rebase: no
history rewritten, no force-push; the tester's and the reviewed commits
keep their ids.

### When: lazy, for V1

- **Only when the policy would merge it** and git's merge of the base
  meets a conflict, or, clean, when the forge holds it behind its base.
  The same on every forge (local, GitHub, GitLab): plain git.
- **Not chosen for V1: eager.** Updating every request behind main at
  each push to main runs every request's checks for nothing in conflict.
  Deferred after V1.

### Clean: the engine alone

- **Merged clean**: a two-parent merge, the branch's head first, the base
  second; committed by the engine, no agent, 0 tokens; pushed as a
  fast-forward, never forced.

### In conflict: one developer round

- **Said on the request and its issue** by `follow`, in the refused
  merge's one comment ([#1354]): the files named, and a record the
  round reads, the head and the base it was read at; edited when the
  round fails.
- **A round is due** (the reconciler's): it starts from the branch's head
  with the base merged in, git's conflict markers in its files.
- **Its task: settle the markers only**, keeping both intents. An edit
  to another file is refused; a frozen test is never touched: a conflict
  in one goes to the tester's objection path.
- **The agent never commits.** The engine checks no marker is left, then
  makes the commit from the tree, two parents (`commit-tree -p head -p
  base`, no merge state in a checkout), and pushes it with no force.
- **The proof runs on the merge**: the whole test command, both requests'
  frozen tests in it. Red, it is not pushed.
- **One failed round stops**: markers left, edits that do not apply, or
  the tests red: nothing pushed, the issue blocked for a person, said on
  the request and the issue. No second round at that head.

### Then: as any new head

- **The checks run again and the reviewer judges the settlement**: its
  diff starts from the merged-in tree, so it reads the branch's own
  change and how the conflict was settled, none of main's code; earlier
  verdicts stand ([ADR-0013]).
- **The Scope is read from the merge base** (`main...head`): main's own
  files never count as beyond the Scope.

## Consequences

- **TG-2's and DC-3's case merges with no person's act**, when the round
  settles it; a person settles the rest, told where.
- **A person's yes covers the head it was given on** ([ADR-0042]): the
  engine's clean merge keeps it (its tree is git's own merge); a round's
  settlement is a new head, and the yes is asked again.
- **Code that read the round's start as one commit** reads the merge
  instead: the round's proof, its edits' base and the clean-checkout run
  (#1349) work on the merge's tree; a conformance case guards a merge
  that would drop main's side.
- Conformance: merge/local-conflict-merged-in-by-a-round,
  merge/behind-held-merged-in-clean,
  developer/merge-round-dropping-main-not-pushed,
  developer/merge-round-markers-left-stops.

## Not chosen

- **A rebase**: rewrites the reviewed and the tester's commits, and
  needs a force-push.
- **The agent merges by itself**: it may not commit, and its copies of
  main became one-parent commits git still saw in conflict.
- **A loop of rounds**: one failed settlement goes to a person.

[ADR-0013]: 0013-what-was-judged-is-not-asked-again.md
[ADR-0041]: 0041-workline-settles-its-own-lines-when-the-base-moves.md
[ADR-0042]: 0042-the-engine-merges-by-the-projects-policy.md
[ADR-0051]: 0051-a-check-sits-where-its-catch-is-worth-its-cost.md
[ADR-0054]: 0054-the-developer-takes-a-ready-issue-to-a-pull-request.md
[ADR-0055]: 0055-the-tester-writes-the-acceptance-tests-before-the-code.md
[#1266]: https://github.com/JN0V/workline/issues/1266
[#1354]: https://github.com/JN0V/workline/issues/1354
