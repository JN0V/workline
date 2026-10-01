# The documentalist in CI, on GitHub

**Verdict.** Judging docs belongs on the merge request and on a schedule,
where the forge already asks people to review; the push only counts. GitHub
no longer lends a model of its own for free; a Claude subscription can run
the official `claude` in Actions on the owner's repositories. A bot's commit
needs a token of its own for the checks to run again. Checked 2026-09-30.

## Models GitHub provides

- **GitHub Models is gone**: closed to new customers in June 2026, retired
  on 2026-07-30 (github.blog changelog). `actions/ai-inference` paths broke.
- **Copilot CLI runs in Actions** with the job's `GITHUB_TOKEN`
  (`permissions: copilot-requests: write`, `copilot -p … -s`), since
  2026-07-02; on a personal repository it bills the owner's Copilot seat, in
  AI credits (1 credit = $0.01, per token). Free: an unpublished allowance,
  automatic model choice only; Pro ($10): 1,500 credits a month; Pro+ ($39):
  7,000. Claude models (Haiku 4.5, Sonnet 5.5, Opus 5.5…) are in the list
  for paid plans. A non-interactive run denies any tool not pre-approved.
- **A Claude subscription**: `claude setup-token` gives a one-year token
  for `anthropics/claude-code-action` or the `claude` binary
  (`CLAUDE_CODE_OAUTH_TOKEN`), documented by Anthropic. It shares the
  owner's usage limits, and the terms assume ordinary, individual use: the
  owner's own repositories, the unmodified binary, modest volume.

## A protected main, one maintainer

GitHub lets nobody approve their own pull request; one approval required
makes a solo maintainer force every merge. Rulesets accept 0 approvals for
"repositories that use pull requests as a record of changes rather than to
gate on approvals": a pull request and the checks still required, then
auto-merge (`gh pr merge --auto`) merges when they pass. A bot's approval
counting as the review is what GitHub's "Actions cannot approve" setting was
made to stop (2022).

## A bot's commit on the pull request

- A push or a pull request made with `GITHUB_TOKEN` starts no workflow;
  since 2026-06-11, its `pull_request` runs wait for someone to click
  "Approve workflows to run". The required checks of the new head wait with
  them. A GitHub App's installation token
  (`actions/create-github-app-token`) runs them unattended, short-lived;
  Anthropic's action recommends the same.
- Forks: `pull_request` gives a read-only token and no secrets; running the
  fork's code under `pull_request_target` is the known "pwn request"; since
  2026-07-20 `actions/checkout` refuses it by default. On a fork: comment,
  never push, and mind prompt injection in what the agent reads.

## What a team adds (review of workline's templates, 2026-09-30)

- GitLab keeps protected variables from unprotected branches: a token
  stored protected never reaches a merge request's branch.
- A bot's commit resets approvals (GitLab "reset approvals on push", GitHub
  "dismiss stale reviews") and makes the author's next push non-fast-forward;
  merge trains and merge queues drop a branch pushed during them.
- After a squash or rebase merge, a `checked` naming a commit of the branch
  names a commit main never holds.
- Every push of the merge request judges again a doc not moved on: one the agent
  left a note on, or a fork's, without a cache of verdicts;
  on DomoticsCore, about 100k tokens a judgement, through wide `sources`.

## Caps and caches (2026-10-01)

- **Caps count what was spent, never estimate ahead.** Claude Code's
  `--max-turns` and `--max-budget-usd` stop one call once crossed, the
  answer that crossed it paid; the result carries `modelUsage`, tokens per
  model (code.claude.com, cli-reference). gh-aw caps turns and AI credits
  a run, warns the agent at 80 to 99%, then cuts it; a run ending on its
  budget passes since gh-aw PR #49614; `max-daily-ai-credits` sums a day's
  runs from their artifacts and skips the agent job before it starts.
  claude-code-action, Codex's action and aider have no cap of their own.
- **Work left over waits for the next run, listed.** Renovate caps PRs an
  hour and open at once, before acting; what is over is shown
  "Rate-Limited" on its dashboard. CodeRabbit counts reviews an hour per
  developer and posts a passing "Review rate limited" check.
- **Caches of verdicts**: CodeRabbit reviews the commits since the last one
  it reviewed (`full review` starts over). agentics' wiki writer keeps, on
  a git branch, each page's sources with their hashes — computed by the
  LLM itself, model and prompt left out of the key, cleared by hand.
  promptfoo keys on provider, prompt, config and variables, and never
  caches an error. The key belongs to the engine: content ids, the exact
  model, the role's version.
