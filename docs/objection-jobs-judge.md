---
sources:
  - defaults/ci/github/workline-answer.yml#jobs.judge
  - defaults/ci/github/workline-answer.yml#jobs.apply
  - "defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:objection"
  - "defaults/ci/gitlab/workline.gitlab-ci.yml#workline:objection:head"
  - "defaults/ci/gitlab/workline.gitlab-ci.yml#workline:objection:judge"
  - "defaults/ci/gitlab/workline.gitlab-ci.yml#workline:objection:apply"
checked: c46a93d6c
verified: agent:claude-code
---
# The judge and apply jobs

Part of [An objection's jobs](objection-jobs.md): what each job holds,
on GitHub and on GitLab.

## head (GitLab only)

- Reads the Draft's head with `git ls-remote`, before any of the
  Draft's code runs, holding no agent's token.
- The judge checks that head out; apply applies at that head only.

## judge

- The agent's token, read rights, no write token: it proposes
  (`--no-apply`).
- It checks out the Draft's head, and runs your test command there: a
  rewrite must be red on the base.
- On GitHub, that head is the Draft's open pull request's tip when the
  run starts, never the objection run's `head_sha`: a re-run or a push
  since would leave it behind.
- So it runs the Draft's code: what it leaves is not trusted.
- On GitLab, it reads the forge with a separate read-only token, not
  the bot's: that one is unset before the Draft's code runs.

## apply

- The write token (GitHub: the App's, in the environment
  `workline-app`), no AI key, from the default branch's checkout
  ([ADR-0044](adr/0044-the-apps-key-stays-with-the-default-branchs-runs.md)).
- The judge's run is data
  ([what apply takes from it](objection-jobs-apply.md)):
  - it names a run's folder, checked by the job (`<date>T<time>.<ns>-<role>`,
    under the runs folder);
  - on GitHub, downloaded outside the checkout.
- The head: the one judge read on GitHub; on GitLab, the one
  `workline:objection:head` reads before the judge, holding no agent. A
  judge reporting another head applies nothing.
