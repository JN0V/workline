---
type: reference
sources: [internal/builtin/productowner/productowner.go, internal/backlog/ledger.go, internal/backlog/opening.go]
status: draft
checked: 9054d164
verified: agent:claude-code
---
# Reading

Part of [Acts on the backlog](../backlog-acts.md): which issues a run reads, in what order, with what code.

A run reads at most `issues-per-run` issues, with at most `code-lines-max`
lines of the code they name (ADR-0018, amended by ADR-0038), and no more
than its prompt holds within the role's context budget: pre measures the
prompt as the engine does (`agent.Prompt`, `agent.Tokens`, room kept for
an answer asked again) and leaves out the last issue to read while it is
over, said (`issues-cut`); those left out are recorded as nothing, so
read first at the next run. One issue alone over it goes to the engine,
which does not ask the agent (`prompt-over-budget`): then no issue is
taken as read (`unasked` in the run's state) — no "read and left", no
person's comment answered, the summary saying the agent was not asked
(#427). An issue workline keeps for its own work (`backlog.Tracking`:
the old report, the weekly sample's issue under its titles, a role's
issue kept in place — its body's last line "Kept up to date by the …
role." —, a demotion of the merge policy) is never read nor listed, and
no act is done nor proposed on it (`workline-issue`, #429); a role's
finding and an import are work. The order: an issue a person who counts commented on since the role read it
— waiting on them ([A person's answers](answers.md)), set aside, or neither: their
word, read first, the agent told so —; one read again for a change to
what it was built on ([A changed need](changed-need.md)); one an act was proposed on only because a run's cap was
reached — the act decided again, the proposal settled once its issue is
read —; a person's issue never read, newest first (`backlog.ByPerson`: no
import or finding key on a line of its own, no bot author), one opened
since the role's last run read in the run that first sees it, its first
state written before any act; then those with something new since they
were read — a commit touched their sources, a person changed its body
(`body`), a person reopened what the role closed (`closed`), the
reviewer's spec findings —; then one left with nothing under an older
way of reading (`backlog.LeftBehind`: its `rules` below
`backlog.Reading`, a section lacking, no code named, no act ever done,
nothing waiting on anyone), read again once, newest first — one past
`issues-per-run` at a later run —, so a fix of the engine reaches the
issues it failed on; last, the catch-up of an
import's, a finding's or a bot's issues never read, oldest first. From
`proposals-max` issues waiting on a person, only those a person
commented on are read. One
set aside by a person is not read until its body changes or a person who
counts comments — or, once, when such a comment brought it back under an
older way of reading and it stayed aside (`backlog.BroughtBack`: a
counted comment written after the label came off, by the forge's label
events, heard since; read with those left with nothing). An issue with nothing new is not read again, however old
(ADR-0018). `backlog.Reading` is raised by a change that may give such an
issue an act: a fix of what the engine dropped, of what the task tells
the agent. An issue on the same code as one read is given with
its body beside it, up to six, as the original a duplicate would be closed
against; the others are listed by title only. The code an issue names: its
sources (a folder given as the list of its files), the paths its title,
body and people's comments name — a link to a file on the forge,
`…/blob/<ref>/<path>` or GitLab's `…/-/blob/…`, is that file —, a file
named alone when one file of the commit has that name, a symbol quoted
as code; at most three an issue. When an issue read names no file the
commit holds, the task lists the repository's folders, nearest the root
first (150 at most), for the agent to name a Scope's files from. Each issue read gets `judged`
moved to the run's commit; without an agent, or when its answer does not
read, none is.
