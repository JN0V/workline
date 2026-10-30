# The agent's sandbox — focused research (2026-10-30)

**Question.** When a coding agent takes an issue to a pull request, how
does it work on the code — tools, rights, sandbox — and what does that
cost? For the developer's agent
([ADR-0056](../adr/0056-the-developers-agent-works-in-a-sandbox-with-tools.md)),
after [issue-to-pull-request.md](issue-to-pull-request.md).

**Verdict.**

- **Every tool studied gives its agent tools in a sandbox**; none
  pastes the code into one call.
- **The cost is in tokens re-read each turn**: about a million tokens
  an issue-sized task, tens of millions a long one, most of them cache
  reads; a turn, time or cost cap holds it.
- **The agent's rights while it works are kept small**: no push, or a
  short-lived repository token; a firewall or an egress proxy.

## Findings

- **Each tool of [issue-to-pull-request.md](issue-to-pull-request.md#the-products)
  gives its agent tools in a sandbox**: it reads the repository, edits
  files, runs the tests and tries again. Copilot's environment is set up
  by `copilot-setup-steps.yml` ([about][cop-about]); GitLab's needs
  `agent-config.yml` to "install dependencies or run tests" ([developer][gl-dev]).
- **Edits in place, from a script**:
  - aider: `--message` sends "a single message … process reply then
    exit", `--yes` answers every prompt, the files named are edited in
    place, auto-commits on unless `--no-auto-commits` ([aider][aider]);
  - Codex: `codex exec` "runs in a read-only sandbox" by default;
    `--sandbox workspace-write` lets it edit ([codex][codex]);
  - Claude Code: `--tools` restricts the tools, `--max-turns` "limit[s]
    the number of agentic turns", no limit by default,
    `--max-budget-usd` stops at an estimated spend ([CLI][cc-cli]).
- **Rights while it works**: Copilot's agent "can only perform simple
  push operations" and its internet access is restricted by a firewall
  ([risks][cop-risks]); gh-aw's agent job is read-only, behind an egress
  proxy ([architecture][aw-arch]). Claude Code's action holds a
  short-lived repository token, Contents read and write ([security][cc-sec]).
- **Prompt injection**: Copilot filters hidden characters and HTML
  comments ([risks][cop-risks]); Claude Code's action strips them too,
  which "reduces but does not eliminate prompt injection risk — keep
  workflow permissions minimal and validate all outputs" ([security][cc-sec]).
- **Caps in use**: SWE-agent's `per_instance_cost_limit` 3.0
  ([model config][swe-model]); OpenHands 50 iterations, gh-aw 20 min
  and `max-turns`, Copilot 59 min
  ([issue to pull request](issue-to-pull-request.md#the-products)).

## What it costs

Per task, an agent with tools, as published. "Fresh" is new input and
output; cache reads are the history read again each turn.

| Task | Agent, model | Rounds | Tokens | Cost |
|---|---|---|---|---|
| SWE-bench Verified, 500 issues | mini-swe-agent (bash only), Claude Sonnet 4.5 | 51 | 0.911M, no cache split | 0.504 USD ([pruner][pruner], table 1) |
| SWE-bench Verified, 500 issues | OpenHands, eight models | — | Sonnet 4.5 "over 1.5 million more tokens than GPT-5"; no mean in the text | — ([token study][tokens]) |
| SWE-bench, 2,294 issues | SWE-agent, GPT-4 Turbo | — | — | 1.59 USD a resolved issue, 4 USD budget ([SWE-agent][swe-paper]) |
| SWE-Serve, long tasks, 48 min a task | Claude Code, Claude Opus 5 at max | — | 327.4k fresh input, 111.3k output, 32.7M cached input | 21.08 USD ([SWE-Serve][serve], figures 16–17) |
| the same, 58 min a task | mini-swe-agent, Claude Opus 5 at max | a 350 limit | 266.6k fresh input, 122.0k output, 25.4M cached input | 17.40 USD |

- **The "1000x"** of the token study's abstract ("agentic tasks are
  uniquely expensive, consuming 1000x more tokens than code reasoning
  and code chat") compares OpenHands' SWE-bench Verified runs with
  CRUXEval questions ("3500x more tokens than a typical single-round
  reasoning task") and multi-turn coding chats ("1200x more tokens than
  a multi-round chatting task"), cache reads not told apart in it. Not
  a measure against one call holding the code, as workline's today.
- **Variance**: "runs on the same task can differ by up to 30x in total
  tokens"; "the most expensive problem, on average, costs ∼7 million
  more tokens than the cheapest" ([token study][tokens]).
- **Cache reads dominate**: for Sonnet 4.5, "cache reads dominate both
  raw token volume and dollar cost" ([token study][tokens]); they are
  priced a tenth of fresh input (GPT-5: 0.125 against 1.25 USD a
  million, the study's appendix).
- **Not found**: fresh and cached tokens apart for an issue-sized task.

## Not found

A tool that bounds the agent's diff by files after it
worked, as workline's bounds do; gh-aw's `allowed-files` is the nearest,
on the pull request it writes.

[aider]: https://aider.chat/docs/scripting.html
[codex]: https://learn.chatgpt.com/docs/non-interactive-mode
[cc-cli]: https://code.claude.com/docs/en/cli-reference
[cop-risks]: https://docs.github.com/en/copilot/concepts/agents/cloud-agent/risks-and-mitigations
[aw-arch]: https://github.github.com/gh-aw/introduction/architecture/
[cc-sec]: https://github.com/anthropics/claude-code-action/blob/main/docs/security.md
[tokens]: https://arxiv.org/abs/2604.22750
[pruner]: https://arxiv.org/abs/2601.16746
[swe-paper]: https://arxiv.org/abs/2405.15793
[serve]: https://arxiv.org/abs/2609.26777
[swe-model]: https://swe-agent.com/latest/reference/model_config/
[cop-about]: https://docs.github.com/en/copilot/concepts/agents/cloud-agent/about-cloud-agent
[gl-dev]: https://docs.gitlab.com/user/project/merge_requests/developer/
