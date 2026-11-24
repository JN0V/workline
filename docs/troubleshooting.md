---
sources:
  - internal/doctor/doctor.go
  - internal/doctor/prompt_overrides.go#flatPrompts
  - internal/doctor/languages.go#languages
  - internal/hooks/hooks.go#HandsOver
  - internal/linters/files.go#notVendored
  - roles/committer/code/secrets.go#Secrets
  - internal/gate/gate.go#Load
  - cmd/workline/main.go#doctorCmd
  - cmd/workline/main.go#doctorPrint
  - cmd/workline/cli.go#misuse
checked: 542fc9cb6
verified: agent:claude-code
---
# Troubleshooting

Start with `workline doctor`: it says what is set up on this machine and in
this repository, each missing thing with the command that sets it up, and
changes nothing. It exits 1 only on an error (an agent that cannot be
called, a config that does not load). Each run is kept in
`.git/workline/runs/` — the agent's answer as it came, each call and its
tokens — and `--json` prints every finding.

## A hook does not fire

- **Not installed**: doctor says `hooks-not-installed`; run `workline hooks
  install --global`.
- **The binary moved**: the hooks call it where it was when installed.
  Install it to the same place, or run `workline hooks install --global`
  again.
- **Written by another version**: doctor says `hooks-outdated`: they
  may skip a repository's own hooks. Run `workline
  hooks install --global` again; `--repo` too, then commit
  `.githooks/commit-msg`.
- **A repository sets its own `core.hooksPath`** (husky, lefthook…): git
  takes it over the global one. `workline doctor` in the repository says
  `hooks-bypassed`, naming the path, when its `commit-msg` (and `pre-push`,
  when routed) does not hand over to workline; `hooks-path` when it does.
  A hook hands over when it is executable and runs `workline hook <hook>`
  (or workline's global hook) as a command before a top-level `exit`: a
  comment or an `echo` does not count.
  The fix it prints:
  - the path is `.githooks` or `.git/hooks` (the main one, from a linked
    worktree too): unset it (`git config --local
    --unset core.hooksPath`, or `--worktree` where it is set there); the global hooks run workline, then hand over
    to that folder;
  - another folder: add `workline hook <hook> "$@" || exit $?` at the top of
    each hook there. `pre-push` reads the pushed refs on its input: a hook
    that reads them too must copy them first.
- **The repository opted out**: an empty `.workline/off` (doctor says so).
- **`pre-push` does nothing**: it runs only when `.workline/config.yaml`
  routes `pre-push` (`workline init` does). `git push --no-verify` skips it.
- **A commit goes through unchecked for secrets**: gitleaks is missing; the
  verdict says `secrets-not-checked`. `workline setup` installs it.

## A command fails before it runs

| Sign | Cause |
|---|---|
| exit 64 and the usage | the command was misused, named on the first line: a missing `<role>` or `<event>`, an unknown option, a bad value; `workline <command> --help` shows its options |
| `.workline/config.yaml: line N: …` | a key the engine does not know, a retired role, or a value YAML reads otherwise than written: it blocks rather than be ignored |
| `no gate named "x"` | `workline gate` or `gate:x` names a gate not declared; the gates declared follow |

## A prompt override no longer read

- `prompt-override-moved` (an error): a role's prompt you overrode, in
  `.workline/roles/<role>/` or your own config folder, sits where the role
  no longer reads it: roles' prompts are under `prompts/`, and the shipped
  one runs in its place. Doctor prints the `mv` that puts it back.

## A language no tool reads

- `language-gaps`: no tool reads one kind of it (lint, types, security,
  dependencies); doctor names the tool and the gate check to add
  ([languages](languages.md)). A warning: nothing blocks on it.
- **A tool your CI runs is not seen**: a script of your own runs it.
  Name it in a gate check.
- **Vendored or generated code is counted**: mark its folder
  `linguist-vendored` or `linguist-generated` in `.gitattributes`.

## Docs suspect, gardening paused

`docs suspect: N` at the push, a record or a merge request not read, a
shallow clone, gardening paused: what each means and what to do, in
[the docs' troubleshooting](troubleshooting-docs.md).

## The product owner paused, or does nothing

Too many proposals waiting, a night's share cut, an issue that gets
nothing or never goes ready, a hidden record that does not read: in
[the product owner's troubleshooting](troubleshooting-product-owner.md).

## Token caps and the agent

A run that spent its tokens or calls, an input cut, a size or a
backstop crossed, an agent that cannot be called, and the other findings
about caps and the agent: in
[token caps and the agent](troubleshooting-limits.md).

## Forge writes refused

`forge: none`, `forge: cmd:`, `forge: local` in CI, GitLab tokens,
findings or gates the forge does not show: in
[forge writes refused](troubleshooting-forge.md).

## An issue does not move, a pull request blocked, nothing merges

A state that wakes no role or waits, a line that stops after its hops,
conversations to resolve, a merge waiting for a yes or not knowing the
App, and the rest: in [the line and the merge](troubleshooting-merge.md);
the states themselves: [the line of states](spec/routing.md#the-line-of-states).

## Where this page reads from

- **What doctor checks and says**: `internal/doctor/doctor.go`, the
  prompts moved in `prompt_overrides.go`, the languages in
  `languages.go`.
- **When a hook hands over**: `internal/hooks/hooks.go`.
- **Vendored and generated code**: `internal/linters/files.go`.
- **Secrets not checked**: `roles/committer/code/secrets.go`.
- **A gate not declared**: `internal/gate/gate.go`.
- **The commands and their exit codes**: `cmd/workline/main.go` and
  `cmd/workline/cli.go`.
