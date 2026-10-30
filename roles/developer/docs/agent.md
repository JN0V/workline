---
sources:
  - internal/agent/claude.go
  - internal/agent/command.go
  - internal/agent/prompt.go
  - internal/developer/developer.go
checked: 62a62929
verified: agent:claude-code
---
# The agent: where it works, what it gets

Part of [the developer](../README.md). Its design:
[ADR-0056](../../../docs/adr/0056-the-developers-agent-works-in-a-sandbox-with-tools.md).

The agent is a choice: none, `claude`, or `cmd:<any command>`.

## Where it works

- **A worktree** of the commit its code starts from: the tester's
  Draft, or the commit the run read; in a review round, its pull
  request's head ([the review loop](review-loop.md#a-round)). It reads,
  edits and runs commands there.
- **Its diff is its code**, held to the bounds once it is done
  ([the bounds](bounds.md#a-worktree-its-diff-is-the-code)), then
  proven on a fresh tree ([the proof](proof.md#a-fresh-tree)).
- **The rights of whoever runs it**: on a machine, the person's; in
  CI, the job's. Nothing refused, said once in the
  run's output and in the pull request; the forge's guards decide what
  merges ([the pull request](pull-request.md#who-opens-it)).

## What each agent gets

| Agent | How it runs |
|---|---|
| none | not run: the issue is named, nothing claimed |
| `claude` | `claude -p` in the worktree, with tools ([below](#claude)) |
| `cmd:<any>` | run in the worktree, the question on its input; `WORKLINE_WORKTREE`, `WORKLINE_MAX_TURNS`, `WORKLINE_MAX_SECONDS` set |

## claude

Checked against Claude Code 2.1.314; each flag in its `--help`:

- **Tools**: `--tools Read,Edit,Write,Glob,Grep,Bash`: read, edit,
  search, and a shell to run the tests. No web tool, no MCP server
  (`--strict-mcp-config`).
- **No person asked**: `--permission-mode dontAsk` with the same tools
  in `--allowedTools`: those run, anything else is denied.
- **Bounded**: `--max-turns` from `ai-max-turns`; killed at
  `ai-max-seconds` with the commands it started
  ([the caps](proof.md#the-caps)).
- **No session kept**: `--no-session-persistence`.
- **Its prompt**: Claude Code's own, for its tools, then the role's
  (`--append-system-prompt`).
- **The repository does not configure it**: `--setting-sources user`,
  the person's settings only. The worktree's `.claude/` settings
  (hooks, permissions) and `CLAUDE.md` are not loaded: a hook there
  would run a command, a permission widen its tools. It may read the
  project's conventions as any file.
- **Its shell without its login**: `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`;
  Claude Code takes its key or token out of the commands' environment.
- **What it used**: `--output-format stream-json --verbose`; its
  turns, fresh tokens, cache reads and the tools it used, one count a
  tool, go in the claim's record.

## The question

It names the code, never pastes it; the agent reads it in its worktree:

- the issue, whole;
- the paths it may change: the Scope, the tests beside them;
- the tester's frozen tests, by path, and how to object to one;
- the test command, the tester's `test-command`, or why its setting
  could not be read;
- its bounds: turns, seconds, no commit nor push.

It answers with its edits, then a YAML list of objections, `[]` if
none. Closing words that are no list lose nothing of its code:

- **They name no objection**: read as none, and said (`answer-mended`).
- **They name one**: its agent failed (`agent-invalid-output`); its
  code is still proven and pushed, as a Draft that says why.

## Not yet

- **Its shell boxed in its worktree**: today it runs as the job's
  user, the network as the runner allows it; Claude Code's sandbox
  needs bubblewrap and user namespaces the runners do not all give
  ([#719](https://github.com/JN0V/workline/issues/719),
  [ADR-0056](../../../docs/adr/0056-the-developers-agent-works-in-a-sandbox-with-tools.md#amendment-the-agents-settings-and-its-shell)).

- **Tokens and time a run, measured** with a real agent: step 7 of
  [ADR-0056](../../../docs/adr/0056-the-developers-agent-works-in-a-sandbox-with-tools.md).
