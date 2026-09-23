# Portability — one role, any coding agent

**Verdict.** Rules, skills and MCP are close to portable; subagents and hooks
are not. Write each role in a neutral folder, generate each agent's native
files, and keep the real checks in git hooks and CI, which need no AI.

## Standards

| Standard | State in 2026-09 | Use |
|---|---|---|
| [AGENTS.md](https://agents.md) | Read by Codex, Cursor, Copilot, OpenCode, Goose, Aider, Zed and more. Claude Code reads it only as a fallback when no CLAUDE.md exists (since 2026-09-18). Nearest file wins, so nested files scope context. | Short shared base, plus one file per area of the repo |
| [Agent Skills](https://agentskills.io) (`SKILL.md`) | Supported by about 40 agents. Three-step loading: ~100 tokens of name and description at start, body on match, extra files on demand. | Procedures of a role, with `scripts/` for the deterministic part |
| Subagent formats | No standard: Markdown for Claude, Gemini, OpenCode, Cursor, Copilot; TOML for Codex; JSON for Kiro CLI | Generated |
| MCP | Supported almost everywhere; each client's config file differs | Tools, when structured access helps |
| Hooks | No standard; event names and shapes differ per agent | Optional speed-up only |
| [ACP](https://github.com/agentclientprotocol/agent-client-protocol) | JSON-RPC to drive any agent; native in Gemini, OpenCode, Cursor, Copilot CLI, Goose; adapters for Claude and Codex | Candidate for a single runtime interface |

Skill folders differ: Codex, Cursor, VS Code and Amp read `.agents/skills/`;
Claude Code reads only `.claude/skills/`. A symlink or a generator is needed.

**Gemini CLI is being replaced by Antigravity CLI (`agy`)**, closed source, since
2026-05. Free, Pro and Ultra users lost Gemini CLI on 2026-06-18.

## Generators (one source, native files per agent)

| Tool | ★ | Notes |
|---|---|---|
| [rulesync](https://github.com/dyoshikawa/rulesync) | 1.5k | Rules, MCP, commands, subagents, skills, hooks, permissions for ~50 tools. Ships a single binary as well as npm. |
| [ruler](https://github.com/intellectronica/ruler) | 2.9k | Simpler; rules and MCP, nested folders |
| [microsoft/apm](https://github.com/microsoft/apm) | 3.9k | Package manager with lockfile for agent setups |
| [getsentry/dotagents](https://github.com/getsentry/dotagents) | 238 | `agents.toml` plus symlinks, lockfile, `doctor --fix` |
| [ai-rulez](https://github.com/Goldziher/ai-rulez) | 143 | Rules, skills, agents for 20 platforms |
| [rjmurillo/ai-agents](https://github.com/rjmurillo/ai-agents) | 47 | 22 roles written once, generated per harness |

## Running agent CLIs headless

| Tool | Notes |
|---|---|
| [headless-cli](https://github.com/RobertTLange/headless-cli) | One command for claude, codex, agy, cursor, opencode, pi; normalized effort; `--print-command` dry run. Best reference for our adapters. |
| [acpx](https://github.com/openclaw/acpx) | Headless client for any ACP agent; pre-1.0 |
| coder/agentapi | Archived 2026-09-13 — avoid |
| jenkinsci/ai-agent-plugin | One step over 8 agent CLIs, a handler per agent: the adapter pattern |

## Ideas to take

- `DUTIES.md` per role: what it may and may not touch (OpenGAP).
- Inject rule names only (~100 tokens), full rule on demand (tikalk/adlc-team-skills).
- Lockfile and `doctor --fix` for installed role packs (dotagents).
- Evals proving a role's rules actually help (microsoft/agentrc).
- `git diff | run reviewer`: each role as a Unix filter (jrswab/axe).
