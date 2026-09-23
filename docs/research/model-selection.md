# Model selection — per role, without naming a model

**Verdict.** Per-prompt routers (LiteLLM, RouteLLM, OpenRouter auto, Plano) are
heavy proxies we do not need. A static grid per role, refreshed from open data
and reviewed by a human, is enough. Spec: [../spec/model-grid.md](../spec/model-grid.md).

## Data sources

| Source | Format | Licence | Gives |
|---|---|---|---|
| [models.dev](https://github.com/anomalyco/models.dev) | `api.json`, daily | MIT | models per provider, cost, status, accepted effort values |
| [Epoch AI hub](https://epoch.ai/data/benchmark_data.zip) | 87 CSVs, daily | CC BY 4.0 | SWE-bench, Terminal-Bench, GPQA, FrontierMath, arenas |
| [LMArena dataset](https://huggingface.co/datasets/lmarena-ai/leaderboard-dataset) | parquet | CC BY 4.0 | writing preference, webdev, agent |
| OpenRouter `/api/v1/models` | JSON, public | — | effort values, expiry dates |
| Artificial Analysis | API, free key | attribution | intelligence and coding indices |

Stale: Aider polyglot (2025-10), SWE-bench JSON (2026-02). LiveBench licence unclear.

## Effort and model flags per agent CLI

| CLI | Model | Effort |
|---|---|---|
| `claude -p` | `--model` alias or id | `--effort low…max` |
| `codex exec` | `-m` | `-c model_reasoning_effort=minimal…xhigh` |
| `agy -p` | `--model` (fails loudly on unknown) | `--effort low\|medium\|high` |
| `opencode run` | `-m provider/model` | `--variant` |
| `cursor-agent -p` | `--model` | in the model name (`-high`) |

## Prior art

- wshobson/agents: static tiers per agent (Claude only).
- TAKT: profiles per persona or step, promotion to a stronger model after N failures.
- microsoft/conductor: `reasoning.effort` per agent, translated per provider.
- warden: role → vendor profiles, reviewer from an independent vendor.
- busybees: model per role with a fallback profile on usage limits.
- doordash agentic-orchestrator: model recommendations from locally available capabilities.
- headless-cli: the effort mapping tables to copy.

## Pitfalls

Benchmark scores mix model and harness; arenas measure chat preference, not
repository work; effort level is part of a score; subscriptions and API keys
reach different models; models get retired — check status before a run.
