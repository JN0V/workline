---
sources:
  - internal/agent/claude.go
  - internal/agent/command.go
  - internal/agent/prompt.go
  - internal/agent/seal.go
  - internal/developer/credential.go
  - internal/developer/developer.go
checked: 0d116a14
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
- **No credential in reach**: no forge token in its environment, and
  git, ssh, gh and glab sealed ([below](#no-credential-in-reach)).

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
  turns, tokens and the tools it used, one count a tool, go in the
  claim's record.

## No credential in reach

The same on a person's machine and in CI: the engine starts the agent
in a sealed environment, with git's and the tools' own switches
(checked against git 2.53, OpenSSH 10.2, gh 2.103, glab 1.53):

| What would hand a credential | Sealed by |
|---|---|
| a forge token in the environment | taken out (`GH_TOKEN`, `GITLAB_TOKEN`, `CI_JOB_TOKEN`…) |
| git's system and global config | `GIT_CONFIG_NOSYSTEM=1`, `GIT_CONFIG_GLOBAL` an empty file |
| a credential helper, any scope, the repository's too | `credential.helper` set empty in `GIT_CONFIG_COUNT`, read last |
| an unscoped `http.extraheader` | the same, set empty |
| a prompt | `GIT_TERMINAL_PROMPT=0`, `GIT_ASKPASS` and `SSH_ASKPASS` fail |
| an ssh key or agent | `GIT_SSH_COMMAND="ssh -F /dev/null -o IdentitiesOnly=yes -o IdentityFile=/dev/null -o BatchMode=yes"`, `SSH_AUTH_SOCK` dropped |
| gh's and glab's login | `GH_CONFIG_DIR`, `GLAB_CONFIG_DIR` an empty folder |
| gh's token in the keyring, which it reads with no config | `GH_TOKEN`, `GH_ENTERPRISE_TOKEN`, `GITLAB_TOKEN` set to a placeholder that opens nothing: an environment token comes first |

- **Kept**: `HOME`, so the agent's own login (`~/.claude`) works.
- **Checked, not trusted**: git, under the sealed environment, must
  find no helper left.
- **Refused**, what no environment hides: a token written in a file
  the agent can read
  ([the pull request](pull-request.md#who-opens-it)).
- **Its limit**: an agent that sets its own environment back, or reads
  the keyring or an ssh key file itself (`gh auth token --user <you>`
  still reads it), reaches what the person reaches. The seal stops git and the tools handing one; boxing the
  shell is [not yet](#not-yet) done.

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
