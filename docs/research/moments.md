# Moments — what a developer waits for, and where a check is skipped (2026-10-19)

**Question.** workline runs checks and AI at a commit, a push, each push
of a merge request, its ready review, main, the night and the release.
How long does a developer wait at each moment today, what budgets does
the ecosystem keep, where can each check be skipped, and which of a
panel's changes remove the most wait
([#203](https://github.com/JN0V/workline/issues/203) asks the same of
SonarQube)?

**Verdict.**

- **The hooks block in the terminal**: a commit takes 0.9 s, nearly all
  of it two gitleaks scans; a push 6 s, 5 of them the documentalist's
  whole-repository checks, the same for 1 commit or 16. `--no-verify`
  skips both in 0.03 s.
- **On a merge request, the head's verdict comes in 101 s** (median, 70
  pushes; 154 s p90, 276 s max). About half is setup: the engine built
  in each job, about 20 s a job. The line itself, its roles and their AI, takes 22 s median.
- **Queues are not the wait**: 1 to 3 s a job on GitHub's runners,
  0.1 s on GitLab.com's.
- **The ecosystem's budget is 10 minutes** for a build's feedback (DORA,
  Fowler, Shopify's p95). Below a second for an analysis in the editor
  (Google). A day for a person's review (Google). Slow hooks "discourage
  developers from using them" (Thoughtworks).
- **No study counts `--no-verify`**; git, pre-commit and husky each
  document a way to skip hooks. A check that matters runs again where
  it cannot be skipped: the merge request.
- **AI reviewers stay out of the path**: they post comments; where they
  set a status, it fails on little or nothing by default.

Measured on 2026-10-19: hooks on a copy of workline at `05fb4b8`
(laptop, Intel i7-10510U, engine at `1d65cca`, no agent); GitHub runs
of the 20 latest pull requests by a person, #465 to #501 (2026-10-17 to
19); GitLab pipelines of the sandbox's
[!8](https://gitlab.com/JN0V/workline-sandbox/-/merge_requests/8). Web
pages read on 2026-10-19, each with its own date.

## The wait at each moment, measured

| Moment | Who waits | Wait today | Where it goes |
|---|---|---|---|
| Commit, `commit-msg` | the developer, terminal | 0.88–0.93 s, 11 commits (code, docs, refused alike); `--no-verify` 0.03 s | two gitleaks scans, 0.41–0.45 s each (the diff, then the message) |
| Commit, a refused message the agent rewrites | the developer, terminal | 12–20 s (evaluation, 2026-10-11, Haiku 5.5, 4.4–5.7k tokens in); 37–57 s on 2026-10-04; one real hook 78.5 s (2026-10-04, Haiku 4.5, 8.6k in, 8.7k out) | the agent's call |
| Push, `pre-push` (opt-in) | the developer, terminal | 5.9–6.2 s, 1 commit or 16; `--no-verify` 0.03 s | committer 0.9 s, documentalist 4.9–5.2 s |
| First result on a push | the author, on the forge | 6–12 s since the gates' job (8 pushes); 58 s median before (62 pushes), code scanning's check | here the gates' job only checks out: workline has no gate |
| The head's `workline/verdict` | the author, then the merge policy | 101 s median, 154 s p90, 276 s max | see the path below |
| The review posted | the author, a colleague | 115 s median, 173 s p90 | the apply job after the judge |
| Ready run, every lens (17 pull requests opened ready) | the colleague | the line 4–71 s; verdict 103 s median, 152 s max | #501: one lens call 42k in, six judges 40k |
| The heaviest ready run (#500, after a cancelled one) | the colleague | the line 133 s; verdict 276 s | two lens calls 93k, ten judges 268k, two of them 104k each |
| workline's own `go test` (not shipped) | the author | 174 s median, 192 s p90 | the last check on most pushes |
| After a merge (#501) | nobody | 15 runs in 6.3 min | main, the release and docs-release pull requests |
| Release pull request, each main move | the person releasing | 1.6–2.2 min (6 runs, 10-19) | documentalist, 6.4k tokens in a run |
| Gardening (2026-10-11 to 18) | nobody | 4.8 to 14.5 min | the documentalist's catch-up, the backlog |

## The verdict's path on GitHub

Median of 70 pushes. A newer push cancelled 7 runs.

1. **Push to the gates' job done**: 6–12 s, 2 s of queue. With a real
   linter (the sandbox's
   [#62](https://github.com/JN0V/workline-sandbox/pull/62)): 34 s,
   golangci-lint's 1 s of it, 26 s setting up Go and building the engine.
2. **The judge's job**: 2 s of queue, 62 s of run. Setting up Go and
   building the engine 21 s, Claude's install 4 s, the line 22 s (p90
   57 s), code scanning's upload 7 s.
3. **`workflow_run` starts the apply workflow**: 2 s, then 3 s of queue.
4. **The verdict's job**: about 30 s, 20 of them setting up Go and
   building the engine, 3 s of work. The apply job beside it: 38 s, 21 of them
   building.

So about 40 s of the 101 s builds the engine twice. This repository
builds it from the pull request's commit on purpose (the shipped
templates download a release); a cache shared by the jobs would keep
that.

On GitLab.com (!8, three pipelines, the engine built in each job): the
gates done at 51–62 s, golangci-lint 1.5 s of it; the judge at
109–119 s; the apply at 162–177 s. A job with the engine already in its
image (the sandbox's nightly) takes 24 s; a job's start, the image
and its preparation, about 12 s.

## Budgets elsewhere

| Source | Budget | Link, its date |
|---|---|---|
| DORA, continuous integration | tests "should not take more than a few minutes to run, with an upper limit of about 10 minutes" | [dora.dev](https://dora.dev/capabilities/continuous-integration/), undated |
| DORA, test automation | feedback "in less than ten minutes" | [dora.dev](https://dora.dev/capabilities/test-automation/), 2025-07-17 |
| Fowler | "the XP guideline of a ten minute build is perfectly within reason" | [article](https://martinfowler.com/articles/continuousIntegration.html), 2024-01-18 |
| Google, presubmit (TAP) | "the average wait time to submit a change is around 11 minutes, often run in the background" | [SWE book, ch. 23](https://abseil.io/resources/swe-book/html/ch23.html), 2020 |
| Google, Tricorder | in the editor "typically less than 1 second and ideally less than 100 ms"; under 10% effective false positives | [ch. 20](https://abseil.io/resources/swe-book/html/ch20.html), 2020 |
| Google, review | feedback "within 24 (working) hours"; first comment, median under an hour for small changes | [ch. 9](https://abseil.io/resources/swe-book/html/ch09.html); [Sadowski et al.](https://sback.it/publications/icse2018seip.pdf), 2018 |
| Shopify | CI "in under 10 minutes in the 95th percentile" | [blog](https://shopify.engineering/faster-shopify-ci), 2021-02-24 |
| GitHub, its own CI | a 45-minute wait for a green run brought to 15 | [blog](https://github.blog/engineering/infrastructure/making-github-ci-workflow-3x-faster/), 2020-10-29 |
| Three companies (Elazhary et al.) | builds of 10 minutes or less, "reducing developer context-switching, and reducing developer blocking" | [arXiv:2103.04251](https://arxiv.org/abs/2103.04251), 2021 |
| A literature review | "builds that take more than 10 minutes may lead the development team to give up on using CI" | [arXiv:2103.05451](https://arxiv.org/abs/2103.05451), 2021 |
| DevEx (Noda, Storey, Forsgren, Greiler) | slow feedback: developers "wait or decide to switch tasks"; no threshold | [ACM Queue, 2023](https://michaelagreiler.com/wp-content/uploads/2024/06/DevEx-WhatDrivesProductivity.pdf) |

## Where a check is skipped

| Check | Skipped by | Caught later by |
|---|---|---|
| `commit-msg`, `pre-commit` | `git commit --no-verify` ([githooks](https://git-scm.com/docs/githooks)); hooks never installed; a commit on the forge's web page; an agent's own commits | the committer on the merge request's whole range |
| `pre-push` | `git push --no-verify` ([git-push](https://git-scm.com/docs/git-push)); opt-in per project | the same, and the documentalist on the merge request |
| A hook of pre-commit or husky | `SKIP=<id>` ([pre-commit](https://pre-commit.com/)); `HUSKY=0`, `-n` ([husky](https://typicode.github.io/husky/how-to.html)) | CI |
| A red check | re-run until green: "at least 1.72%" of builds restarted, median merge time "from 16h to 48h" with a restart ([MSR 2020](https://arxiv.org/abs/2003.11772)); flaky tests: "rerunning tests is by far the most common reaction" ([ICST 2022](https://arxiv.org/abs/2203.00483)) | a required check and the merge policy ([ADR-0042](../adr/0042-the-engine-merges-by-the-projects-policy.md)) |
| A check not required | merged anyway | main after the merge |

- **Hooks**: "overloading hooks with slow-running checks can
  discourage developers from using them"; keep them "minimal"
  ([Thoughtworks Radar](https://www.thoughtworks.com/radar/techniques/pre-commit-hooks),
  Adopt, 2025-11-05). pre-commit keeps slow hooks for `pre-push` or
  `manual`, a stage "not … automatically triggered by any git hook".
- **No count of bypasses found**: no study or survey of `--no-verify`.

## AI reviewers and slow analyses, out of the path

| Tool | Time | Holds the merge? | Source |
|---|---|---|---|
| Copilot code review | "usually takes less than 30 seconds" | a "Comment" review, not counted toward required approvals | [docs](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/request-a-code-review/use-code-review) |
| CodeRabbit | not stated | a status for required checks; `fail_commit_status` false by default; reviews each push, pauses after 5 reviewed commits | [configuration](https://docs.coderabbit.ai/reference/configuration) |
| Greptile | "Posts findings in ~3 minutes as PR comments" | `statusCheck` on by default | [docs](https://www.greptile.com/docs) |
| Sourcery | not stated | one check, failing only on "blocking security findings"; re-reviews "capped at five per pull request" | [docs](https://docs.sourcery.ai/reviews/anatomy-of-a-review/) |
| GitHub merge queue | "average wait time to ship a change" cut by 33% at GitHub | needs the `merge_group` event | [blog](https://github.blog/engineering/engineering-principles/how-github-uses-merge-queue-to-ship-hundreds-of-changes-every-day/), 2024-03-06; [docs](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue) |
| GitLab merge trains | up to 20 pipelines in parallel; Premium and Ultimate | a failed one is not merged, those behind restart | [docs](https://docs.gitlab.com/ci/pipelines/merge_trains/) |

**SonarQube's analysis is asynchronous**: "not complete until the
relevant background task has been completed", tasks "processed
serially" ([background tasks](https://docs.sonarsource.com/sonarqube-server/analyzing-source-code/background-tasks/)).
`sonar.qualitygate.wait` polls, 300 s by default, "only … if it's
necessary" ([CI](https://docs.sonarsource.com/sonarqube-community-build/analyzing-source-code/ci-integration/overview)).
The Community Build analyses the main branch only
([editions](https://www.sonarsource.com/blog/sonarqube-compare-editions/),
2025-10-27). SonarQube Cloud analyses a pull request on each push; on
its Free plan "only when the pull request is merged into the main
branch" ([pull requests](https://docs.sonarsource.com/sonarqube-cloud/analyzing-source-code/pull-request-analysis)).
`api/qualitygates/project_status` takes `pullRequest` or `branch`, gives
OK, WARN, ERROR or NONE, 404 with no analysis; on SonarQube Cloud it is
deprecated since 16 September 2025, no replacement named
([web services](https://sonarcloud.io/api/webservices/list)).

## The panel's changes, by wait removed

The panel of 2026-10-19 ranked by tokens; the maintainer put the wait
first. Inferred from the measures above, not tried.

| Change | Wait removed | Tokens saved |
|---|---|---|
| The engine built or fetched once a run, not in every job | ~20 s a job, ~40 s of the verdict's 101 s | none |
| Lenses and triage that only warn, off the verdict's path | the AI's time on what does not block, within the line's 22 s median, 133 s max | none |
| Push: only the docs the range made suspect; whole-repository checks on main | ~5 s, every push, in the terminal | none |
| Commit: one gitleaks scan for the diff and the message | ~0.4 s, every commit | none |
| Drafts with no AI | the AI's time on each draft push | a lens a draft push |
| After ready, AI only on what is new; a finding judged once | the judges' time (#500: ten, 268k) | most of the line's |
| The release pull request's AI once labelled | none (nobody waits) | ~0.8M a week here: 129 main moves (10-12 to 18) × 6.4k |
| Gates in a merge queue or train, no AI | the queue's turn | none |

What workline decides from this:
[ADR-0050](../adr/0050-each-act-has-its-moment-and-a-wait-budget.md).
