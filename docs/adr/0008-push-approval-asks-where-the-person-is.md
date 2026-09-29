# ADR-0008: The push approval asks the person where they are: terminal, editor, or desktop

- **Status:** proposed
- **Date:** 2026-09-29
- **Amends:** ADR-0007, whose approval refused any push without a terminal

## Context

ADR-0007 has a person approve every push, on the terminal the hook opens.
Without one the push is refused. But the person works in VSCodium as much as
in a terminal: they push from Source Control, and they ask Claude Code, in
the editor, to push. Both are refused today, so the approval pushes them back
to a terminal, or to turning it off, which leaves agents free again.

What exists (docs/research/push-approval.md): no hook manager asks anything
outside a terminal. But git and ssh already ask a person through askpass,
yes/no questions included; VS Code's git extension answers askpass requests
with an input box in the editor, on a socket its git processes inherit; and
a desktop dialog is one command away. The agents' own confirmation rules
belong to each agent, match only the command's text, and are reported not to
prompt in the VS Code extension: the guard stays in git.

The agent's shell has no askpass variable, but it descends from the editor's
extension host, which listens on the same kind of socket. Found through the
hook's ancestors, the socket reaches the window the person is in, and the
agent cannot substitute one of its own.

## Decision

- **The hook asks through the first channel that reaches a person**, in this
  order:
  1. **the terminal** (`/dev/tty`), as ADR-0007 has it, `[d]ocs` included;
  2. **the editor**: the hook walks up its ancestor processes to a VS Code
     family editor (its executable: `code`, `codium`, `code-oss`, and
     others as they are tried), takes the `vscode-git-*.sock` that process
     listens on, and asks on it, as the editor's askpass would. The input box
     says what leaves, where, and what to type: `y` pushes, `v` opens the
     page of the push and asks again; anything else, empty included, stops
     it;
  3. **a desktop dialog**, when there is a display: zenity or kdialog run by
     absolute path, osascript on macOS, with the same summary and a button
     to view;
  4. otherwise the push is refused, the first line of the message saying so
     — an editor shows that line in its error dialog — naming no way around.
- **What the agent sets is never trusted**: `GIT_ASKPASS`, `SSH_ASKPASS`,
  `VSCODE_GIT_*`, `PATH`. The socket comes from the ancestors, the dialog
  from a fixed path. The agent may make the question appear; only the person
  answers it.
- **The same timeout** (10 minutes): a question left unanswered stops the
  push; an answer given later is ignored.
- **Judging the docs stays on the terminal** (`d`, `workline docs`): the
  review goes doc by doc. Outside a terminal the question names the docs the
  commits made suspect, and says to run `workline docs`.
- **Linux first.** Ancestors and sockets are read from `/proc`; on macOS and
  Windows the editor channel is skipped, saying so, until it is tried there.
- `approve-push: false` in the person's own config still turns it all off.

## Consequences

- A person pushes from Source Control, or has an agent push, and approves in
  the editor they are looking at. An agent's push is no longer refused: it
  waits for a person.
- The editor channel rests on an internal protocol of VS Code, which can
  change without notice. A conformance case pins what workline sends; a
  change of the editor is caught when the push is tried again, and the hook
  falls back to the dialog, then to refusing, never to pushing.
- An agent set on pushing still can (`--no-verify`, xdotool under X11): the
  approval stops habit, as ADR-0007 said; protected branches stay the
  barrier.
- Two editor windows: the question goes to the one the push came from, not
  the one in focus.

## To try before accepting

On this machine, VSCodium on Wayland: a push from Source Control; Claude
Code asked to push; a push while the terminal is also open (the terminal
wins); no answer for ten minutes; Escape; and the dialog, with the editor
channel off.
