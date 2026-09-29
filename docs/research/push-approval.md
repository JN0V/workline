# Push approval without a terminal

**Verdict.** No hook manager asks a person anything outside a terminal; they
refuse or hang in an editor. But two channels already exist on a desktop, and
a hook can reuse them: the editor's own git prompt (VS Code, VSCodium) and a
desktop dialog. Agents' own confirmation rules are per agent, bypassed by how
a command is written, and reported broken in the VS Code extension: the guard
stays in git. Checked 2026-09-29, on VSCodium 1.117 and its git extension.

## Git's convention: askpass

Git asks for credentials through `GIT_ASKPASS`, `core.askPass`, then
`SSH_ASKPASS`, before the terminal (gitcredentials): the prompt as argument,
the answer on standard output. OpenSSH already asks yes/no this way
(`ssh-add -c` sets `SSH_ASKPASS_PROMPT=confirm`; the host-key question), as
`sudo -A` does. A yes/no question through askpass is established practice.
But the variables are the agent's to set: `GIT_ASKPASS=/bin/echo git push`
answers for the person. Never trusted as they come.

## The editor's prompt: VS Code's git socket

A push from Source Control runs git with `GIT_ASKPASS` pointing to the
extension's `askpass.sh`, and `VSCODE_GIT_IPC_HANDLE` naming a Unix socket in
`$XDG_RUNTIME_DIR` (`vscode-git-<hash>.sock`). Hooks inherit both.

- The protocol is HTTP on that socket: `POST /askpass`, body
  `{"askpassType":"https","argv":[…]}`, answer a JSON string. No token or
  nonce; the socket's folder is the user's alone.
- An `https` request whose text does not say "password" shows
  `showInputBox` in the editor window: the text as placeholder, `argv[4]` as
  title, kept open when focus leaves. Escape answers `""`. With fewer than
  five arguments it fails (500).
- `askpass.sh` passes `$*` unquoted: a question with spaces is split. A hook
  talks to the socket itself.
- It is internal and undocumented (extensions/git/src/askpass.ts, changed
  2026-05-30): the argument layout can move without notice.
- **An agent's shell inherits none of it.** Claude Code's Bash tool, in the
  VSCodium extension, has no `GIT_ASKPASS` nor socket variable. But its
  process descends from the editor's extension host, which listens on the
  same kind of socket: walking up the hook's ancestors to the editor's
  process, and taking the `vscode-git-*.sock` that process listens on
  (`/proc/net/unix` inodes against `/proc/<pid>/fd`), finds the right window.
  Globbing the folder does not: 39 sockets there, 5 alive.
- Found this way, and not from a variable, the agent cannot point the hook
  at a socket of its own; and writing to the real one only makes the
  question appear: the answer is typed in the editor.

When a push fails, VS Code shows a modal `Git: <first line of stderr>`, and
the rest in the Git output channel: the refusal's first line is what the
person reads.

JetBrains IDEs set `GIT_ASKPASS` to a helper talking XML-RPC
(`INTELLIJ_GIT_ASKPASS_*`); how a non-credential question renders is not
verified. Tower, Fork, GitKraken, Sourcetree: no such feature found.

## Desktop dialogs

`zenity --question` (0 yes, 1 no, 5 timeout; fails without a display),
kdialog, yad; `osascript display dialog` on macOS; a MessageBox on Windows.
[ncruces/zenity](https://github.com/ncruces/zenity) (923★, v0.10.15
2026-08) wraps them in Go without cgo, but finds the Linux tool on `PATH`,
where an agent can put a fake one: run it by absolute path. Over SSH or in
CI there is no display, and no dialog. A native Wayland dialog cannot be
clicked by synthetic input; under X11, `xdotool` can — a deliberate bypass.

## Hook managers and agents

- pre-commit declines interaction (issue #1127); husky-interactive gives up
  GUI clients; lefthook's `interactive: true` opens `/dev/tty` and hangs in
  editors (discussion #709).
- [agent-git-guard](https://github.com/martinfrancois/agent-git-guard)
  refuses a push when the environment names an agent; it never asks.
  [agent-guard](https://github.com/XuebinMa/agent-guard) refuses and prints
  a command the person runs in a terminal.
- Claude Code: an `ask` rule `Bash(git push *)` prompts in every mode, but
  `git -C . push` escapes it, and issue anthropics/claude-code#86754 (open,
  2026-08) reports that `ask` runs without prompting in the VS Code
  extension. Codex (`decision="prompt"`, issue #25312), Cursor
  (`beforeShellExecution`), Copilot (PreToolUse): each its own, each on the
  command's text.

## What to take

In order, the first that can reach a person: the terminal; the editor's
prompt, found through the hook's ancestors; a desktop dialog, by absolute
path, when there is a display; else refuse, in one first line, naming no way
around. The agent can make the question appear, never answer it. The
answer is typed, and an empty one — Enter, Escape — is no.
