# A call from a Claude Code session, measured (2026-10-17)

**Question.** Why does a role's call cost twice the tokens when it starts
from inside a Claude Code session? Same engine, same case
(`documentalist/drifted-version-bump`, tests/evaluation/results.tsv):
44,090 tokens in from an agent's shell, 21,701 from a clean one
(systemd-run).

**Verdict.**

- **One variable: `CLAUDE_CODE_ENTRYPOINT=claude-vscode`.** The editor's
  extension sets it; every process the session starts inherits it, and
  workline's `claude` adapter passed it on.
- **With it, Claude Code (2.1.302) adds 234 to 237 tokens to the prompt,
  and on a long question has Haiku read the whole question again**
  (6,179 tokens in, 15 out, beside Sonnet's 5,790), whatever model was
  asked: twice the tokens in.
- **The values `cli` and `sdk-cli` add nothing.** A session started in a
  terminal, or from the SDK, does not double the call.
- **`CLAUDE_EFFORT` does not override `--effort`**: 88 tokens out without
  it, 87 with `CLAUDE_EFFORT=max`, on `--effort low`.
- **Fix**: the adapter drops the variables a session exports
  (internal/agent/claude.go, `claudeSession`). The login a CI job needs
  (`CLAUDE_CODE_OAUTH_TOKEN`, `ANTHROPIC_API_KEY`, `ANTHROPIC_BASE_URL`…)
  and a person's own settings pass. A `cmd:` agent gets the environment as
  it is.

## How it was measured

One call shaped like the adapter's, run from `/tmp`, in an agent's shell
of a session started from the editor:

```sh
echo "Say OK." | env -i HOME="$HOME" PATH="$PATH" \
  claude -p --output-format json --tools "" --strict-mcp-config \
  --no-session-persistence --system-prompt "Answer in one word." \
  --model sonnet | jq .modelUsage
```

- **Tokens in**: input + cache read + cache creation, per model in
  `modelUsage`.
- **Environments**: the session's as it is (`env`); `env -i` with `HOME`
  and `PATH` only; the session's minus one group (`env -u NAME…`);
  `HOME`, `PATH` and one variable.
- **Long question**: `Here is a list to ignore: item0 … item1499` then
  `Say OK.`, 5,553 tokens in all from a clean shell; medium: 300 items,
  1,453 in all.

## Numbers

Sonnet asked, unless said.

| Environment | Question | Sonnet in | Haiku in (side call) | Total in |
|---|---|---|---|---|
| session as it is (Haiku asked) | short | — | 776 | 776 |
| `HOME`, `PATH` (Haiku asked) | short | — | 541 | 541 |
| session minus every `CLAUDE*`, `MCP_*`, `AI_AGENT` | short | 542 | — | 542 |
| session minus `CLAUDECODE` | short | 781 | — | 781 |
| session minus `CLAUDE_CODE_SESSION_ID`, `_CHILD_SESSION`, `CLAUDE_PID`, `_EXECPATH` | short | 776 | — | 776 |
| session minus `CLAUDE_CODE_MESSAGING_*` | short | 777 | — | 777 |
| session minus `CLAUDE_CODE_ENTRYPOINT` | short | 542 | — | 542 |
| session as it is | long | 5,790 | 6,179 | 11,969 |
| `HOME`, `PATH` | long | 5,553 | — | 5,553 |
| session minus `CLAUDE_CODE_ENTRYPOINT` | long | 5,552 | — | 5,552 |
| `HOME`, `PATH`, `CLAUDE_CODE_ENTRYPOINT=claude-vscode` | long | 5,787 | 6,179 | 11,966 |
| `HOME`, `PATH`, `CLAUDE_CODE_ENTRYPOINT=cli` | long | 5,553 | — | 5,553 |
| `HOME`, `PATH`, `CLAUDE_CODE_ENTRYPOINT=claude-vscode` | medium | 1,690 | — | 1,690 |
| `HOME`, `PATH`, `CLAUDE_CODE_ENTRYPOINT=cli` or `sdk-cli` | medium | 1,453 | — | 1,453 |

- **The side call** came on the long question, not on the medium one: its
  threshold lies between them, not narrowed.
- **The evaluation's doubling** (44,090 against 21,701) is the side call
  reading the documentalist's question once more.

## With the engine

The committer rewriting a refused message (`fix: AC-3`) over a staged
change of 10,027 characters of Go, from the same session, Haiku asked:

```sh
workline run-role committer --event commit-msg --input message="fix: AC-3" \
  --ai claude --forge none --json
```

| Engine | Tokens in | Tokens out |
|---|---|---|
| v0.32.0, the session's variables passed | 12,263 | 928 |
| with `claudeSession` dropped | 5,711 | 711 |

## Not tried

- Other editors' entrypoint values; Claude Code versions other than
  2.1.302.
- `CLAUDE_CODE_SSE_PORT`, set in an editor's own terminal: absent from an
  agent's shell, not measured, not dropped.
- A new variable a later Claude Code exports: the list is explicit; run
  the probe above again when a call from a session costs more than one
  from a clean shell.
