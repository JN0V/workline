---
sources:
  - internal/hooks/hooks.go#dispatcher
  - internal/hooks/hooks.go#InstallGlobal
  - internal/hooks/hooks.go#UninstallGlobal
  - cmd/workline/main.go#hooksCmd
  - cmd/workline/main.go#hookCmd
  - cmd/workline/main.go#report
  - roles/committer/code/committer.go#Post
  - roles/committer/code/committer.go#rewriting
  - internal/kit/commitmsg/commitmsg.go#Check
  - internal/line/seconds.go#Result.Times
  - internal/line/seconds.go#Step.timed
  - roles/committer/role.yaml#settings
  - roles/committer/role.yaml#personal
checked: dd77bf922
verified: agent:claude-code
---
# Every commit checked

Part of [install](install.md).

`workline setup` does this when you say yes; by hand:

```sh
workline hooks install --global     # git's global core.hooksPath now goes through workline
workline hooks uninstall --global   # gives core.hooksPath back as it was
```

- Each global hook hands over to the one that held `core.hooksPath` before,
  or else to the repository's own (`.githooks/`, `.git/hooks/`), from a
  linked worktree (`git worktree add`), a submodule or a bare repository
  too: nothing that ran before stops running.
- Hooks written by another version of workline, or a repository that
  sets its own `core.hooksPath`: `workline doctor` says so and prints the
  fix ([troubleshooting](troubleshooting.md#a-hook-does-not-fire)).
- A repository opts out with an empty `.workline/off` file.

Try it in any repository:

```sh
git commit --allow-empty -m "AC-3 fix the thing"
# workline committer: block — rewrite the commit message
#   format subject: the subject must read `type(scope): summary`…
# workline: 0.6 s in all — committer 0.5 s
# commit aborted; your message is kept in .git/COMMIT_EDITMSG: to edit it and commit again, git commit -e -F .git/COMMIT_EDITMSG
```

- **With no agent**, or `rewrite: false` under
  `roles: {committer: {settings: …}}` (the project's
  `.workline/config.yaml`, or your own config), it refuses in about a
  second, as above.
- **With an agent**, the same commit goes through, rewritten, 30 s at most
  ([the committer](../roles/committer/docs/rewrite.md#rewrite-the-agent-30-s-at-most),
  [ADR-0050](adr/0050-each-act-has-its-moment-and-a-wait-budget.md)):

```text
workline committer: the message is refused; the agent (claude) rewrites it, 30 s at most…
workline committer: pass — message rewritten
workline: 14 s in all — committer 14 s (rewrite 13 s)
workline: the commit goes on with this message instead of yours:
  │ fix: fix the thing
  │
  │ Refs: AC-3
```

## Update in place

To update workline, install the new binary where the old one is: the
global hooks call the binary where it was when they were installed
(`command -v workline` says where).
