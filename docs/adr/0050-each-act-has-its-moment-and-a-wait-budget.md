# ADR-0050: Each act has its moment, and each moment a wait budget

- **Status:** accepted; steps 1 and 2 built ([the build plan](#build-plan))
- **Date:** 2026-10-19
- **Decided by:** the maintainer, on 2026-10-19: a panel's direction,
  validated in broad lines, corrected: first the time a developer waits
  at each moment, as a slow hook or pipeline makes a developer skip
  everything (`--no-verify`, a red check ignored); tokens second. The
  same day, the two choices left open ([§7](#7-decided-by-the-maintainer-on-2026-10-19)),
  and later that day the second changed: the rewrite at a commit on by
  default, capped at 30 s ([§7](#7-decided-by-the-maintainer-on-2026-10-19)).
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
| Commit | the developer, in the terminal | 1 s; 30 s when the agent rewrites a refused message (§7) | 0.9 s; 12–78.5 s with the rewrite |
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
   which runs it anyway. A hook calls an agent only for a refused
   message, the rewrite at a commit, capped at 30 s (§7).

### 3. The moment table

**B**: its check fails, the merge waits. **W**: said, never holding the
verdict; *async*: posted by a job of its own. —: not run.

| Act | Draft | Ready, once | Each push after | Queue or train | Main after merge | Night | Release pull request | Release |
|---|---|---|---|---|---|---|---|---|
| Committer: rules, gitleaks, on the range | B | B | B | — | — | — | B | — |
| Gates: the project's tools, new findings | B | B | B | B, merged result | W, whole report, an issue; the next base | — | B | — |
| Documentalist, no AI: docs the change made suspect | W | B | B | — | W, whole-repository budgets, an issue | — | B | — |
| Documentalist's AI on the docs due | — | B | B, new ones | — | — | W, capped; the release pull request's too (§7) | — | — |
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
  release pull request's AI at each main move, now at night; the
  whole-repository findings at each push; a hook's agent with no cap
  on its wait.

### 7. Decided by the maintainer on 2026-10-19

- **The release pull request's documentalist AI runs at night only**,
  with the gardening. At each move of main, only the no-AI count of
  docs due stays, and holds the release. Refused: once a person accepts
  it, the panel's, since a doc fix would then change the head after the
  person's yes ([ADR-0042]).
- **The agent's rewrite at a commit is on by default, capped at 30 s**
  (changed the same day, below). The hook says so before it starts;
  past 30 s it refuses with the reasons, the message kept for the
  person (principle 5). `rewrite: false`, a project's or a person's,
  turns it off: the hook then refuses in about 1 s with the reasons.
  `ai: none` means no agent at all.
  - **First decided: opt-in** (2026-10-19, morning), off unless a
    setting turned it on, since the one real hook measured took 78.5 s.
  - **Changed** (2026-10-19, afternoon), on the measure of a firmware
    library's 300 latest commits ([#528](https://github.com/JN0V/workline/pull/528)):
    with no rewrite, 106 of them, 35 %, would be refused, each for a
    person to fix by hand. A bounded wait, only on a bad message, is
    preferred to that rate; the 30 s cap keeps it within the commit's
    budget for a rewrite (§1).

## Build plan

By the wait removed each time, then tokens:

1. The engine once a run, shared by the jobs: ~40 s of the verdict's 101 s, every push. Built, once a commit, in [workline's own workflows](../ci-github.md#worklines-own-workflows); the templates fetch a release.
2. Commit: the agent's rewrite capped at 30 s (§7; first planned opt-in); one gitleaks scan: 12–78.5 s down to at most 30 s on a refused message, about 1 s with the rewrite off; ~0.4 s every commit. Built: the committer's `rewrite`, on by default, and `ai-max-seconds`, 30, which the engine holds for every call of a run ([the role contract](../spec/role-contract.md)); a project or a person turns the rewrite off for themselves, and a project's `ai: none` keeps every agent away. The two scans run side by side rather than as one: gitleaks reads a message on its standard input, where its rules on file paths cannot apply. Measured: 0.52–0.58 s a commit, refused or not, from 0.92–1.03 s.
3. Lenses and triage that warn, off the verdict's path: their AI time, within the line's 22 s median, 133 s at most.
4. Push in 5 s: only the docs the range made suspect, whole-repository checks on main: ~5 s every push.
5. No AI on a draft: the AI's time on each draft push.
6. After ready, AI on what is new; a patch judged once; 5 commits a push: the judges' time (#500: ten).
7. The release pull request's AI at night only (§7): its part of 1.6–2.2 min; ~0.8M tokens a week here.
8. A merge queue's or train's run: gates only: the queue's turn.
9. Main after a merge: whole-repository findings once, an issue; gates kept as the base: no wait, less noise.
10. The night: a release dry run; a release job on GitLab: no wait.
11. Each step's seconds in the summary, a moment over its budget said: no wait, it measures the rest.
12. SonarQube's read at these moments ([#203]): adds a wait, capped at 5 minutes, beside the first result.
13. doctor's checks of §6: no wait.

## Consequences

- A developer waits seconds in the terminal, minutes on the merge
  request; what takes longer comes as a comment. The verdict
  ([ADR-0042]) holds what blocks, nothing that only warns.
- Hooks stay a convenience; the merge request stays the check ([ADR-0011]).

[#203]: https://github.com/JN0V/workline/issues/203
[ADR-0010]: 0010-docs-judged-on-the-merge-request-and-by-gardening.md
[ADR-0011]: 0011-the-review-is-on-the-merge-request-not-the-push.md
[ADR-0013]: 0013-what-was-judged-is-not-asked-again.md
[ADR-0017]: 0017-the-release-manager.md
[ADR-0020]: 0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md
[ADR-0042]: 0042-the-engine-merges-by-the-projects-policy.md
[ADR-0044]: 0044-the-apps-key-stays-with-the-default-branchs-runs.md
[ADR-0049]: 0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md
