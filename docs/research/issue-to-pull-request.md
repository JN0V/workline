# From an issue to a pull request — focused research (2026-10-27)

**Question.** How do existing coding agents take an issue to a pull
request: what starts them, where they run and with which rights, what
bounds their scope and their cost, how they prove the change, how they
answer a review, and what goes wrong? For the developer role
([#117](https://github.com/JN0V/workline/issues/117)).

**Verdict.**

- **A person starts every one of them**: assign the issue, put a label,
  mention the agent. Only gh-aw also runs on a schedule. Only users with
  write access can start them.
- **The agent never merges, nor approves its own work**: Copilot cannot
  mark its pull request ready, approve or merge it, and the person who
  asked cannot approve it either. Claude Code's action does not even open
  the pull request: a person does.
- **The agent holds no write right in the best designs**: gh-aw runs it
  read-only; a separate job, after a threat check, writes ("safe
  outputs"). That is principle 6.
- **One issue, one branch, one pull request**, the branch under the
  agent's prefix (`copilot/`, `claude/`); gh-aw caps pull requests at one
  a run and holds a list of files the agent may touch, and of protected
  files it may not.
- **Each caps a session, none caps review rounds**: time (Copilot
  59 min), turns (OpenHands 50), cost (SWE-agent 3 USD an issue).
- **Proof is the agent running the project's tests in its sandbox**; no
  tool ties it to the issue's own test plan. A failure keeps the work:
  OpenHands pushes the branch and comments on the issue; gh-aw opens an
  issue when the pull request is refused; GitLab opens a draft.
- **The main failure is not a wrong change but one nobody reads**: of
  600 rejected agent pull requests, 38 % were abandoned or never
  reviewed, 23 % duplicates, 17 % red CI; only 5 % wrong or incomplete.
  Each extra failed CI check cuts the odds of a merge by about 15 %.

## The products

### GitHub Copilot coding agent (now "Copilot cloud agent")

- **Trigger**: assign it an issue, start it "from the agents panel, a
  conversation, or an issue or pull request", or mention `@copilot` on a
  pull request. "Only users with write access to the repository can
  trigger" it; comments from others never reach it ([risks][cop-risks]).
- **Sandbox**: an "ephemeral cloud development environment" on GitHub
  Actions ([about][cop-about]); a firewall with a "recommended
  allowlist", on the agent's shell only ([firewall][cop-fw]).
- **Rights**: pushes only to a new `copilot/` branch, or to the branch of
  the pull request it was mentioned on. Branch protections and required
  checks apply. CI on its pull requests waits for a person's "Approve and
  run workflows", unless an admin allows it ([risks][cop-risks]).
- **Never**: marks its pull request "Ready for review", approves or
  merges it. "The user who asked Copilot to create a pull request cannot
  approve it." Commits are authored by Copilot, the person co-author,
  signed, linked to the session's logs ([risks][cop-risks]).
- **Scope**: "one pull request per task"; "a maximum execution time of
  59 minutes" ([about][cop-about]).
- **Proof**: it may "run tests and linters" in an environment set up by
  `copilot-setup-steps.yml`; it also runs CodeQL, secret scanning and a
  check of new dependencies before it finishes ([risks][cop-risks]).
- **Review**: `@copilot` in a comment makes it push again; "Add to
  batch" gathers review comments for one session; no cap on rounds
  ([existing PR][cop-pr]).
- **Cost**: "one premium request per session", plus Actions minutes
  ([changelog][cop-cost]).

### GitHub Agentic Workflows (gh-aw)

- **Trigger**: `slash_command`, `label_command`, `schedule`; `roles:`
  `[admin, maintainer, write]` by default; `stop-after:` a deadline "to
  control costs" ([triggers][aw-trig]).
- **Rights**: the agent's job runs "with minimal read-only permissions";
  writes happen in separate "safe output" jobs, after a threat-detection
  job with no write access gives a verdict. Egress goes through a proxy
  limited to `network.allowed` ([architecture][aw-arch]).
- **Scope**: `create-pull-request` at `max: 1` a run; `draft:` "a policy,
  not a default", the agent cannot change it; `allowed-files` "an
  exclusive allowlist"; `protected-files` by default `request-review`,
  else `blocked` or `fallback-to-issue`; `fallback-as-issue` opens an
  issue when the pull request is refused ([safe outputs][aw-pr]).
- **CI**: "PRs do not trigger CI by default": the job's own token starts
  no workflow ([safe outputs][aw-pr]).
- **Caps**: the agent's step 20 min; `max-turns` 500; `max-ai-credits`
  1000 ([frontmatter][aw-front]). The agent is a choice: Copilot, Claude,
  Codex, Gemini and others.

### Claude Code GitHub Actions

- **Trigger**: `@claude` (`trigger_phrase`), a label, an assignee, or a
  fixed `prompt`; only users with write access; bots refused unless
  listed in `allowed_bots` ([action.yml][cc-action], [security][cc-sec]).
- **Rights**: the project's runner; a short-lived token for one
  repository ([security][cc-sec]).
- **Never**: approves ("for security reasons"), merges. On an issue it
  pushes a `claude/` branch and "links back to a prefilled PR creation
  page": a person opens the pull request ([capabilities][cc-cap]).
- **Cost**: `--max-turns` alone ([configuration][cc-conf]).

### OpenHands resolver

- **Trigger**: the `fix-me` label, removed once read, or a comment
  mentioning `@openhands-agent`; the example workflow also listens to
  review comments ([package][oh-pypi], [workflow][oh-wf]).
- **Outcome**: "Create a draft PR if successful"; else the branch pushed
  and a comment on the issue ([package][oh-pypi]). **Cap**:
  `max_iterations`, 50 by default ([issue][oh-iter]); once ignored by a
  bug ([fix][oh-fix]).

### SWE-agent and mini-swe-agent

- **SWE-agent** takes an issue's URL; opening the pull request is off
  by default (`open_pr: False`) ([run config][swe-run]);
  `per_instance_cost_limit` 3.0 USD ([model config][swe-model]). No loop
  on review comments.
- **mini-swe-agent**: `cost_limit` 3.0; at the limit, an empty
  submission, the trajectory kept ([agent][mini]).

### Devin

- **Trigger**: its web app, a chat tool, `/devin <prompt>` on a pull
  request ([GitHub][devin-gh]).
- **Review**: "Devin automatically responds to comments on PRs its
  sessions are working on"; no cap on rounds written.
- **Rights**: its App writes contents, pull requests and workflows; the
  docs advise branch protection so checks pass "before Devin can merge
  changes". **Cost**: compute-unit caps an organisation
  ([usage][devin-cost]).

### GitLab Duo Developer flow

- **Trigger**: assign the issue to its service account, the "Implement"
  button, a mention, or a work item's status; Developer role at least,
  "Allow foundational flows" on for the group ([developer][gl-dev]).
- **Proof**: it opens a draft merge request, on the project's runners. Without `agent-config.yml`
  it "cannot install dependencies or run tests"; "large, open-ended tasks
  are more likely to hit iteration limits", no number given.
- **Review**: a mention on a line, or a review then a mention for every
  open thread.

## What goes wrong, measured

- **33,596 agent pull requests** of the AIDev dataset ([AIDev][aidev]),
  from projects over 100 stars ([failed agentic PRs][fail]):
  - 71 % merged; by agent from 43 % to 83 %; by task, docs 84 %, CI 79 %,
    fixes 64 %, performance 55 %.
  - Each extra failed CI check cuts the odds of a merge by about 15 %;
    the number of review comments and revisions does not matter.
  - Of 600 rejected: 38 % abandoned or never reviewed, 23 % duplicates,
    17 % CI or tests red, 4 % a feature not wanted, 3 % wrong, 2 %
    incomplete.
- **Another sample of 353 rejected**: about a third technical, a third
  the workflow, a third no reason; rejections "substantially overstate
  agent error" ([rejected][rej]; read from its summary only).
- **Self-reported**: 67 % of Devin's pull requests merged, up from 34 %
  ([Cognition][devin-rev]).

## What workline takes

| Seen elsewhere | For the developer |
|---|---|
| Agent read-only, a separate job writes (gh-aw) | Principle 6 already: the agent proposes a patch; `apply`, holding the App's key and no AI key, opens the pull request |
| Never ready, approve, merge (Copilot) | No merge intention in its catalogue; merging stays `follow`'s, by the project's policy ([ADR-0042](../adr/0042-the-engine-merges-by-the-projects-policy.md)) |
| One pull request a run, a file allowlist, protected files (gh-aw) | One issue a run; its writes bounded by the issue's Scope; the files that steer agents never touched |
| Duplicates and unread pull requests are most rejections | Never a second pull request for an issue; several open, never two touching the same files |
| Red CI cuts the merge odds (MSR 2026) | The project's tests and the Verification run before the pull request leaves Draft |
| Failure keeps the work (OpenHands, gh-aw) | Over budget or red: a Draft, saying what failed |
| Caps on a session, not on rounds | A token and time cap a run; rounds capped by the reviewer, five, the five whys at three, then a person |
| A pull request opened with the job's token runs no CI (gh-aw) | Opened by workline's App, so the project's checks run |
| A person starts each one, by assigning it the issue | The developer assigns itself the first issue the product owner set `ready`; a person assigning themselves keeps it |

**Not found**: a tool tying the proof to the issue's own test plan; a
written cap on review rounds; a measure of agents' cost per merged pull
request.

[cop-risks]: https://docs.github.com/en/copilot/concepts/agents/cloud-agent/risks-and-mitigations
[cop-about]: https://docs.github.com/en/copilot/concepts/agents/cloud-agent/about-cloud-agent
[cop-fw]: https://docs.github.com/en/copilot/how-tos/use-copilot-agents/cloud-agent/customize-the-agent-firewall
[cop-pr]: https://docs.github.com/en/copilot/how-tos/use-copilot-agents/cloud-agent/make-changes-to-an-existing-pr
[cop-cost]: https://github.blog/changelog/2025-07-10-github-copilot-coding-agent-now-uses-one-premium-request-per-session/
[aw-trig]: https://github.github.com/gh-aw/reference/triggers/
[aw-arch]: https://github.github.com/gh-aw/introduction/architecture/
[aw-pr]: https://github.github.com/gh-aw/reference/safe-outputs-pull-requests/
[aw-front]: https://github.github.com/gh-aw/reference/frontmatter/
[cc-action]: https://github.com/anthropics/claude-code-action/blob/main/action.yml
[cc-sec]: https://github.com/anthropics/claude-code-action/blob/main/docs/security.md
[cc-cap]: https://github.com/anthropics/claude-code-action/blob/main/docs/capabilities-and-limitations.md
[cc-conf]: https://github.com/anthropics/claude-code-action/blob/main/docs/configuration.md
[oh-pypi]: https://pypi.org/project/openhands-resolver/0.3.1/
[oh-wf]: https://github.com/OpenHands/OpenHands/blob/main/openhands/resolver/examples/openhands-resolver.yml
[oh-iter]: https://github.com/OpenHands/OpenHands/issues/5263
[oh-fix]: https://github.com/OpenHands/OpenHands/pull/6865
[swe-run]: https://swe-agent.com/latest/reference/run_single_config/
[swe-model]: https://swe-agent.com/latest/reference/model_config/
[mini]: https://mini-swe-agent.com/latest/reference/agents/default/
[devin-gh]: https://docs.devin.ai/integrations/gh
[devin-cost]: https://docs.devin.ai/enterprise/features/usage-policies
[devin-rev]: https://cognition.ai/blog/devin-annual-performance-review-2025
[gl-dev]: https://docs.gitlab.com/user/project/merge_requests/developer/
[aidev]: https://arxiv.org/abs/2507.15003
[fail]: https://arxiv.org/html/2601.15195v1
[rej]: https://arxiv.org/pdf/2605.22534
