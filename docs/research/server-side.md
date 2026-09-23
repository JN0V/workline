# Server-side — roles in CI or on a server

**Verdict.** Nothing is both forge-agnostic and AI-agnostic except PR-Agent,
which only reviews. The pieces exist; the portable layer is ours to build: a
role runs through one command, and thin CI templates call it on GitHub and GitLab.

## Security model to copy

- **gh-aw safe outputs**: the agent job is read-only and emits JSONL requests
  (add label, comment, open PR…), each with a `max`. A separate job with scoped
  write permissions validates and applies them.
- **Linux Foundation triage**: Prepare / Propose / Apply; Apply checks evidence
  digests before minting credentials, caps batches, can be re-run alone.
- **Split credentials** (atulg4/agentic-sdlc): the AI job has no write token,
  the publisher has no AI key.
- **claude-code-action defaults**: only users with write access trigger it,
  bots are ignored, hidden markdown and HTML are stripped from issue content.
- **codex-action**: the API key goes through a proxy, never into the job's env;
  `output-schema` forces structured output.

## GitLab specifics

- GitLab CI has **no native trigger on labels or comments** without Duo
  Premium/Ultimate.
- Workaround without a server: a pipeline trigger token used as a project
  webhook; the event arrives in `TRIGGER_PAYLOAD`. One pipeline per event.
- Or a thin relay that turns `@mention` webhooks into pipeline triggers
  ([Schickli/agent-for-gitlab](https://github.com/Schickli/agent-for-gitlab)).
- Claude Code documents a GitLab CI integration (beta): `claude -p` in a job.
- **Shared roles for a team**: the [renovate-runner](https://gitlab.com/renovate-bot/renovate-runner)
  pattern — one central project, one scheduled pipeline, a group token, and it
  discovers every repo that has a config file.

## Webhook servers and bots

| Project | ★ | Notes |
|---|---|---|
| [PR-Agent](https://github.com/the-pr-agent/pr-agent) | 13k | Forge- and AI-agnostic; one LLM call per command; community-maintained |
| [tmseidel/ai-git-bot](https://github.com/tmseidel/ai-git-bot) | 170 | Reviewer, docs sync, QA, coder; four forges; heavy server |
| [claude-hub](https://github.com/claude-did-this/claude-hub) | 487 | One container per task, HMAC-checked webhooks |
| [owainlewis/machinist](https://github.com/owainlewis/machinist) | 455 | Go worker exposing named commands only |
| multica | — | Heavy platform over ~26 agent CLIs |

## Risks to handle

- Prompt injection from issues and merge requests: allowlist triggerers, strip
  hidden content, agent read-only, writes only through validated proposals.
- Never let an agent edit CI or role files; protect them with CODEOWNERS.
- Loops between bots: ignore events the bot caused.
- Cost: turn limits, timeouts, capped fix rounds, one run per issue at a time.
- Fork merge requests must never see secrets.
