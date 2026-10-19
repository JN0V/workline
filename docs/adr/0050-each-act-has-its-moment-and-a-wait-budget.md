# ADR-0050: Each act has its moment, and each moment a wait budget

- **Status:** accepted; nothing built yet ([the build plan](#build-plan))
- **Date:** 2026-10-19
- **Decided by:** the maintainer, on 2026-10-19: a panel's direction,
  validated in broad lines, with one correction. The first criterion is
  the time a developer waits at each moment: a slow hook or pipeline
  makes a developer skip everything (`--no-verify`, a red check
  ignored). Tokens come second.
- **Builds on:** [ADR-0010], [ADR-0011] (the review is on the merge
  request), [ADR-0013] (what was judged is not asked again),
  [ADR-0017], [ADR-0020], [ADR-0042] (the engine merges on the head's
  verdict), [ADR-0044], [ADR-0049]; principles 4, 5, 6, 7, 9, 12, 13, 14
- **To amend once built:** [ADR-0020]: no lens on a draft; the lenses
  that warn post off the verdict's path
- **Research:** [moments.md](../research/moments.md)
- **Settles:** where SonarQube's read runs ([#203])

## Context

Measured on 2026-10-19 ([moments.md](../research/moments.md)):

- **Hooks block the terminal**: a commit 0.9 s, nearly all of it two
  gitleaks scans; a push 6 s, 5 of them the documentalist's
  whole-repository checks. `--no-verify` skips both in 0.03 s. An agent
  rewriting a refused message: 12 to 20 s, once 78.5 s.
- **The head's verdict on GitHub**: 101 s median, 154 s p90, over 70
  pushes. About 40 s builds the engine, twice; the line, its roles and
  their AI, 22 s median, 133 s at most. Queues: 1 to 3 s a job.
- **Others**: about 10 minutes for a build's feedback (DORA, Fowler);
  under a second for an analysis in the editor (Google); a day for a
  person's review. AI reviewers post comments; a status they set fails
  on little or nothing by default.
- **Every hook can be skipped**, and is not installed everywhere.

## Decision

### 1. A wait budget per moment

| Moment | Who waits | Budget, p90 | Today |
|---|---|---|---|
| Commit | the developer, in the terminal | 1 s; 30 s when the agent rewrites a refused message | 0.9 s; 12–78 s |
| Push (opt-in) | the developer, in the terminal | 5 s, no agent | 6 s |
| First result on a merge request's push | the author | 1 min | 6–12 s; 34 s with a real linter |
| The head's verdict, `workline/verdict` | the author, the merge policy | 3 min | 154 s |
| The ready review, every lens | a colleague | 10 min, posted when done | 158 s; 338 s the heaviest |
| A merge queue's or train's run | the queue | workline adds 1 min at most | not used |
| The release pull request's verdict | the person releasing | 3 min | 1.6–2.2 min |
| Main after a merge, the night, the release | nobody | none; tokens capped as today | 5–15 min |

- **Why**: the hooks near an editor's; the verdict well inside the
  ecosystem's 10 minutes, workline being one part of a project's CI;
  the ready review a build's 10 minutes, a colleague's taking hours.
- **A budget is a check, never a block** (principle 7): each step's
  seconds go in the run's summary; a moment over its budget says so,
  naming the step (principles 12, 14). A project may set its own.

### 2. An act over its budget

Never silently dropped. In this order:

1. **Cut its setup**: the engine fetched or built once a run, shared by
   the jobs; tools cached, pinned.
2. **What only warns runs apart**, posted when done, never on the
   verdict's path: the lenses that warn, the inspector's triage. They
   judge with no write token; a job holding it and no AI key posts
   (principle 6, [ADR-0044]).
3. **What blocks stays on the path, in parallel**: blocking acts in
   jobs side by side. Still over: the verdict stays pending, saying
   what it waits for; the act's cap ends it as a `check-error`, never a
   pass (principle 12).
4. **What is about the repository, not the change, moves** to main
   after the merge, or to the night: a warning, then an issue.
5. **What a hook cannot do in its budget moves to the merge request**,
   which runs it anyway. The agent's rewrite at a commit says so before
   it starts; past 30 s the message is refused with its reasons, kept
   for the person (principle 5).

### 3. The moment table

**B**: its check fails, the merge waits. **W**: said, never holding the
verdict; *async*: posted by a job of its own. —: not run.

| Act | Draft | Ready, once | Each push after | Queue or train | Main after merge | Night | Release pull request | Release |
|---|---|---|---|---|---|---|---|---|
| Committer: rules, gitleaks, on the range | B | B | B | — | — | — | B | — |
| Gates: the project's tools, new findings | B | B | B | B, merged result | W, whole report, an issue; the next base | — | B | — |
| Documentalist, no AI: docs the change made suspect | W | B | B | — | W, whole-repository budgets, an issue | — | B | — |
| Documentalist's AI on the docs due | — | B | B, new ones | — | — | W, capped | B, once accepted (§7) | — |
| Reviewer, a lens the project blocks on | — | B | B, new commits | — | — | — | — | — |
| Reviewer, lenses that warn | — | W async, every lens | W async, new commits, 5 a push | — | — | W, specs | — | — |
| Inspector's triage | — | W async | W async, new findings | — | — | — | — | — |
| SonarQube read (§5) | — | B | B | — | W, an issue | — | B | — |
| Product owner, release dry run, auditor | — | — | — | — | — | W | — | — |

### 4. What the table holds to

- **The verdict's path holds only what blocks**: committer, gates, the
  documentalist, a blocking lens, SonarQube where it blocks.
- **A draft gets no AI**: rules and gates only.
- **Ready runs the one full review**; after it, AI reads only what is
  new, and a finding is judged once ([ADR-0013], by patch id).
- **A queue's or train's run**: gates, never AI; it posts the statuses
  the forge requires for the group's commit. A check skipped says so.
- **The release** re-reads the verdicts given; no role runs at the tag.
  With no release pull request, `release` runs before the tag
  ([ADR-0017]).
- **Gates at the release only are refused**: old debt would block it,
  findings come in a batch, who made them is lost, the last tag is a
  fragile base.

### 5. SonarQube's read ([#203])

workline reads SonarQube's quality gate and issues through its Web API;
the team's own pipeline scans. No tokens.

| Edition | Merge request | Main after merge | Release pull request |
|---|---|---|---|
| Server Developer and up; Cloud's paid plans | B, from ready on: ERROR, or no analysis of the head | W, an issue | B |
| Community Build; Cloud's Free plan | —: no merge request analysed | W, an issue | B, on its base |

- **The exact commit**: an analysis of the head, or of the release pull
  request's base; never the last one found. Else the read waits.
- **The wait's own budget**: polled, 5 minutes at most, SonarQube's own
  `sonar.qualitygate.timeout` default, in a job beside the others, so
  the first result is never held. Past it, a `check-error`; the next
  run reads again. Where the forge says Sonar is done (its own check),
  that event replaces polling, a later step.
- **New code**: the "previous version" period, `sonar.projectVersion`
  set by the release tool; doctor checks it.
- **Not said twice**: Sonar's comments are not posted again; the
  reviewer and the inspector are told what Sonar covers.

### 6. Defaults, a project's moments, doctor, the safety nets

- **The tables are the shipped default** (`routing.default.yaml`, the
  templates). A project moves an act in `routing:`, as today; the line
  gains the events it lacks: a draft, a queue's group, main after a
  merge.
- **doctor flags a choice that defeats a guard**: the committer in
  hooks only; gates at the release only; an agent in a hook's line; a
  merge queue with no `merge_group` trigger; no new-code version
  bumped for SonarQube; a budget above 10 minutes.
- **Kept despite the duplication**: the committer on the merge
  request's range (hooks are opt-in, `--no-verify` skips them, web
  edits and agents never run them); the suspect-docs count at the push
  and on the merge request; gates on each main commit, the next base
  and a check on two green merge requests breaking together; the
  release pull request's count of docs due at each main move.
- **Removed**: a patch judged twice; the engine built in every job; the
  documentalist's AI at each main move on the release pull request;
  whole-repository findings at each push.

### 7. Left to the maintainer; the default holds meanwhile

- **The release pull request's AI**: once a person accepts it (the
  panel's), or at night only. Once accepted, a fix moves the head after
  the yes, and the release waits a run (1.6 to 2.2 min).
- **The agent's rewrite at a commit**: on, capped at 30 s (the
  default), or opt-in, the hook refusing in 0.9 s with the reasons.

## Build plan

By wait removed, then tokens:

1. The engine once a run, shared by the jobs: ~40 s of the verdict's 101 s.
2. Lenses and triage that warn, off the verdict's path: up to the line's AI time.
3. Each step's seconds in the summary; a moment over its budget said.
4. Push in 5 s: only the docs the range made suspect; whole-repository checks on main.
5. Commit in 1 s: one gitleaks scan; the agent's rewrite capped at 30 s.
6. No AI on a draft.
7. After ready, AI on what is new; a patch judged once; 5 commits a push.
8. The release pull request's AI as §7 settles: ~0.8M tokens a week here.
9. Main after a merge: whole-repository findings once, an issue; gates kept as the base.
10. A merge queue's or train's run: gates only.
11. The night: a release dry run; a release job on GitLab.
12. SonarQube's read at these moments ([#203]).
13. doctor's checks of §6.

## Consequences

- A developer waits seconds in the terminal, minutes on the merge
  request; what takes longer comes as a comment.
- The verdict ([ADR-0042]) holds what blocks, nothing that only warns.
- Hooks stay a convenience; the merge request stays the check
  ([ADR-0011]).

[#203]: https://github.com/JN0V/workline/issues/203
[ADR-0010]: 0010-docs-judged-on-the-merge-request-and-by-gardening.md
[ADR-0011]: 0011-the-review-is-on-the-merge-request-not-the-push.md
[ADR-0013]: 0013-what-was-judged-is-not-asked-again.md
[ADR-0017]: 0017-the-release-manager.md
[ADR-0020]: 0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md
[ADR-0042]: 0042-the-engine-merges-by-the-projects-policy.md
[ADR-0044]: 0044-the-apps-key-stays-with-the-default-branchs-runs.md
[ADR-0049]: 0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md
