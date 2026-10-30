# ADR-0056: The developer's agent works in a sandbox, with tools

- **Status:** accepted (2026-10-30)
- **Date:** 2026-10-30
- **Decided by:** the maintainer agreed to write it, after the role's
  first live run; the choice is theirs, [below](#for-the-person)
- **Amends:** [ADR-0054] (how its agent writes the code); the pick, the
  bounds, the proof, the pull request and the review loop stay
- **Amended by:** [ADR-0058](0058-the-forge-guards-the-project-not-the-agents-environment.md):
  step 2's credential refusal and the amendment's boxed
  shell give way to the forge's guards
- **Builds on:** [ADR-0042], [ADR-0044], [ADR-0050], [ADR-0055];
  principles 5, 6, 7, 10, 12, 13
- **Research:** [agent-sandbox.md](../research/agent-sandbox.md),
  [issue-to-pull-request.md](../research/issue-to-pull-request.md)

## Context

- **Today the agent is one call with no tools**: `claude -p --tools ""`
  (`internal/agent/claude.go`); the engine pastes the Scope's code in
  the question, up to 100 KB (`scopeCode` in
  `internal/developer/developer.go`); the agent answers YAML patches.
- **What the first live run and the measures showed**
  ([status](../../roles/developer/docs/status.md#measured)):
  - it cannot read a header, a caller or a build file outside the Scope;
  - its diff is malformed one run in two on one real repository
    ([#692](https://github.com/JN0V/workline/issues/692));
  - it cannot run the tests, so a red proof ends the run
    ([#704](https://github.com/JN0V/workline/issues/704));
  - a `cmd:` agent that edits files itself (aider, codex) cannot plug
    in: it must answer patches on its output (principle 10).
- **Every CI coding agent studied** lets its agent read, edit and run
  the tests in a sandbox, and iterate: Copilot's cloud agent, gh-aw,
  OpenHands, SWE-agent, Claude Code's action, GitLab's Developer flow
  ([research](../research/agent-sandbox.md)).

## Decision

### The agent works in a worktree, in the agent's job

- **Where**: the job that runs the agent today ([ADR-0054]'s first job,
  `judge` in the GitHub template), on a git worktree of the branch the
  code starts from (the tester's Draft, or the default branch).
- **Rights**: read and edit any file in the worktree, run commands in
  it, the project's test command first; network as the project's CI
  allows it (workline adds no firewall: the runner's or the forge's,
  as Copilot's and gh-aw's, are used where a project has one).
- **No write token, as today** (principle 6, [ADR-0044]): no forge
  token, no App key, no signing key in its environment
  ([ADR-0054] step 4's scrub stays); its own AI key only.
- **Checked, not trusted** (principle 7): the engine refuses to start
  the agent when the worktree's git config holds a credential (a
  checkout's persisted token): said, a block before any claim (no
  code, so no Draft to open), never a silent run.

### The engine bounds the result, after

- **The code is the worktree's diff** against the commit it started
  from, untracked files included, whatever the agent committed: the
  `patch` intention goes.
- **[ADR-0054]'s bounds apply to the diff, file by file**: Scope, the
  tester's frozen tests, the files that steer agents, `.workline/`,
  CI workflows, a gate's definition; a symlink out of the tree too.
  A refused file is reverted and said in the pull request
  ([the bounds](../../roles/developer/docs/bounds.md)).
- **The bounds stay the engine's**: the agent may touch any file while
  it works; only what passes the bounds leaves the job.
- **The other intentions stay**: an objection to a frozen test
  ([#700](https://github.com/JN0V/workline/issues/700)), what it left
  out, on its output as today. Amended by
  [#746](https://github.com/JN0V/workline/issues/746): an objection is
  written in the worktree's `.workline/objections.yaml`, read by the
  engine and taken out of the diff; closing words that do not read never
  cost the code, and are posted as they came on its pull request.

### The proof stays the engine's

- **On the bounded result, on a fresh tree** of the start commit: not
  the agent's worktree, where an ignored file or a cache it changed
  could make tests pass ([ADR-0054] step 3).
- **The agent runs the tests to iterate; its word proves nothing**: a
  green it reports is not read.

### Caps

- **Time**: `ai-max-seconds` (3600 s) on the agent, then the proof's
  `proof-seconds` (900 s); past it the agent is killed, its diff bounded
  and proven, a Draft (as gh-aw's 20 min, Copilot's 59). Raised from
  1200 and 600 at step 7: builder agents take 13 to 32 min a step.
- **Turns**: a new `ai-max-turns`, passed to the agent: 300. An
  issue-sized task takes 51 rounds on average
  ([the card](../research/agent-sandbox.md#what-it-costs)), so
  OpenHands' 50 would stop many; builder agents use 60 to 200 tool
  calls a step. Reached: a Draft.
- **Tokens**: `ai-max-tokens` (500000), read from the agent's report
  after the run: an agent has no live token cap, so turns and time hold
  it; a run over it is a Draft, said (principle 12). It counts fresh
  tokens: the input not read from cache, cache writes included, and the
  output. Cache reads, most of a tool loop's total, are recorded apart,
  never counted; counted, a cap would hold the cache, not the work
  (step 7, [status](../../roles/developer/docs/status.md#measured)).

### What each agent means

- **`none`**: unchanged, the pick alone; nothing claimed, nothing coded.
- **`claude`**: `claude -p` in the worktree, its tools read, edit,
  search and shell, `--max-turns`, no MCP server, no session kept.
- **`cmd:<any>`**: run in the worktree, the task on its input, the
  worktree's path in `WORKLINE_WORKTREE`, its caps in
  `WORKLINE_MAX_TURNS` and `WORKLINE_MAX_SECONDS`; it edits in place
  (aider `--message --yes`, `codex exec --sandbox workspace-write`) and
  answers the other intentions on its output, if any.
- **The question shrinks**: the issue, its bounds, the frozen tests'
  paths, the test command; no pasted code.

### What stays from ADR-0054

- The pick and the claim, the bounds' rules, the proof, the pull
  request and its sections, the review loop (a round edits the same
  way), the merge, the two jobs: `apply` alone holds the App's key.

## For the person

- **One choice**: the agent in a worktree with tools (this ADR), or
  keep the one-shot call and fix its defects one by one
  ([#692](https://github.com/JN0V/workline/issues/692),
  [#704](https://github.com/JN0V/workline/issues/704)).
- **What it costs**: from thousands of tokens a run to about a million,
  mostly cache reads ([consequences](#consequences)); a turn cap
  holds them.

## Build plan

Steps listed here; issues opened once the maintainer has read it. One
pull request a step, its cases first in
`tests/conformance/cases/developer/`, `pending.txt` listing what does
not pass.

1. **The worktree and its diff**, a stand-in `cmd:` agent that edits
   files: `developer-worktree-diff-is-the-code`,
   `developer-worktree-reverts-out-of-scope`,
   `developer-worktree-reverts-frozen-test`,
   `developer-worktree-reverts-steering-file`,
   `developer-worktree-refuses-symlink-out`,
   `developer-worktree-ignores-agent-commits`.
2. **No credential in reach**: `developer-worktree-env-has-no-token`,
   `developer-worktree-refuses-persisted-credential`; the templates'
   checkout without persisted credentials. The refusal is proposed for
   removal by [ADR-0058](0058-the-forge-guards-the-project-not-the-agents-environment.md#6-what-goes-what-stays).
3. **The proof on a fresh tree**: `developer-proof-ignores-worktree-cache`,
   `developer-proof-red-after-agent-green`.
4. **The caps**: `developer-turn-cap-makes-draft`,
   `developer-time-cap-kills-agent`, `developer-token-cap-makes-draft`.
5. **`claude` with tools** and the question without code:
   `developer-claude-runs-in-worktree`; `scopeCode` deleted.
6. **The review loop** in the same shape:
   `developer-round-edits-in-worktree`.
7. **Measured**: [ADR-0054] step 8's planted issues and copies of real
   repositories again, tokens and time a run beside today's.

## Consequences

- **Tokens a run rise, from thousands to millions**: about 2k a
  planted case and 12k on the first live run today. Published, per
  task, with tools ([the card](../research/agent-sandbox.md#what-it-costs)):
  - an issue-sized task (SWE-bench Verified, Claude Sonnet 4.5, a
    bash-only agent): 0.91M tokens, 51 rounds, 0.50 USD on average, no
    cache split published;
  - a long task (48 min, Claude Code, Claude Opus 5 at max):
    327k fresh input, 111k output, and 32.7M cached input, 21 USD;
  - one task runs "up to 30x" another run of itself.
  - **So**: fresh tokens (new input and output) likely in the hundreds
    of thousands a run, cache reads in the millions, which cost a tenth
    of fresh input. Issues bounded by a Scope may sit lower: step 7
    measures it, both counted apart.
- **Latency**: minutes of turns instead of one answer, under a
  60 min cap; off a person's path, the developer has a cap, no wait
  budget ([ADR-0050]).
- **Safety**: a prompt injection in the issue or the repository steers
  an agent with no write token whose result is bounded after it; it
  can still spend its caps and reach the network with its AI key: the
  project's CI egress rules and the caps are the guard.
- **Defects that go**: a malformed diff
  ([#692](https://github.com/JN0V/workline/issues/692)), a file the
  agent was not shown; a red proof can be fixed in the run
  ([#704](https://github.com/JN0V/workline/issues/704)). The agent's
  job needs the project's toolchain, as its proof already does
  ([#703](https://github.com/JN0V/workline/issues/703)).
- **Unchanged**: proving on today's default branch
  ([#701](https://github.com/JN0V/workline/issues/701)), a test of its
  own where the tester is not routed
  ([#693](https://github.com/JN0V/workline/issues/693)), easier to
  write once it can run it.
- **The tester later, not now**: it would gain the same (a caller read,
  its test run red); moved once the developer's step 7 has measured the
  cost, by an amendment of [ADR-0055].

## Not chosen

- **Keep the one-shot call with pasted code**: the context stays the
  Scope, diffs stay fragile, a `cmd:` agent that edits cannot plug in.
- **The engine as the agent's tool host** (its own read, edit, run
  tools, bounds checked live): an agent harness to build and keep, and
  a `cmd:` agent with its own tools still out (principle 13).
- **Give the agent the forge's token**, as Claude Code's action does:
  the agent would push past the bounds (principle 6).
- **Run the agent on a person's machine**: the same command runs there
  (principle 10), but the line's default stays CI; a person is not a
  runner.

## Amendment: the agent's settings and its shell

Added 2026-10-30, at step 5's review.

- **The repository does not configure `claude`**: `--setting-sources
  user`. Its `.claude/` hooks would run commands, its permissions widen
  the tools; `CLAUDE.md` is not loaded either, the role's facets stay
  its context. The agent may read the project's conventions as files.
- **Its shell never sees its login**: `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`,
  Claude Code's own scrub, on by default in GitHub Actions.
- **Not yet, its shell boxed**: Claude Code's sandbox (`sandbox.enabled`,
  `network.allowedDomains`, `filesystem.allowWrite`) needs bubblewrap
  and unprivileged user namespaces, which Ubuntu 24.04+ restricts by
  AppArmor and a Docker runner may not give. Until each template's
  runner is tried, a same-user process may still read the login, write
  outside the worktree, reach the network
  ([#719](https://github.com/JN0V/workline/issues/719)).

[ADR-0042]: 0042-the-engine-merges-by-the-projects-policy.md
[ADR-0044]: 0044-the-apps-key-stays-with-the-default-branchs-runs.md
[ADR-0050]: 0050-each-act-has-its-moment-and-a-wait-budget.md
[ADR-0054]: 0054-the-developer-takes-a-ready-issue-to-a-pull-request.md
[ADR-0055]: 0055-the-tester-writes-the-acceptance-tests-before-the-code.md
