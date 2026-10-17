---
type: reference
sources: [internal/backlog/backlog.go, internal/backlog/ledger.go, internal/backlog/say.go, internal/report/summary.go, internal/engine/backlog.go]
status: draft
checked: 21e91932
verified: agent:claude-code
---
# On the issue

Part of [Acts on the backlog](../backlog-acts.md): what the role shows on each issue, and in the job's summary.

No report issue (ADR-0038). Each issue holds what the role holds of it,
in its state ([The issue's state](state.md)), and shows it in the same comment, the
state folded under it (`backlog.FormatState`):

- what the role did there on the last day it acted, a line an act — "Accepted by @x"
  when it was a person's yes; a refine says what it wrote in one line,
  "Added: Verification, Scope. Changed: —. See the issue's edit history."
  (`backlog.refined`): Changed, the role's own sections it rewrote — a draft, or a text as it wrote it — it never
  changes a person's sentence, so nothing of theirs is quoted, on GitLab's
  Free plan either, which keeps no text of a description's versions; the
  text before is in the forge's edit history, which no forge links to;
  a question to the reporter, or a text proposed to an outsider, says so
  in one line linking to that comment (`linkComments`), "in the comment
  below" on a forge with no address for it or when its comments do not
  read (an unreachable forge stops the run): written once, in the comment
  that names the reporter and so notifies them;
- each act it proposes there, in plain words (`backlog.offer`), and a
  refine's drafts, folded;
- while the issue waits on a person: how to answer, a short list, one
  gesture a line (`State.answers`) — `workline:accepted` (yes), a draft
  edited (shown only when there is a draft), a comment (revise), the
  label taken off (not now), a closing (it should not exist); a closing
  only advised: closing it, the label taken off, a comment;
- once set aside: that nothing more is proposed until the issue changes.

Written at the end of each run (`Plan.Persist`), edited in place, never a
new comment: one mail when it is created, none after. A comment with
nothing to say is the state alone, as before.

**The label.** `workline:proposed` (on a forge with scoped labels,
GitLab: `workline::proposed`), created once, is on an open issue while it
waits on a person: a proposal in its state, or a draft of the role's in
its body and not ready. Taken off when it no longer waits, when it is
closed, and when a person accepts. A label a bot took off, as the forge's
label events say, is put back.

**Its labels follow its state**, at every run that writes the issues'
states (`Plan.Persist`), with no agent: an open
issue left with no draft of the role's (`backlog.HasDraft`) loses
`workline:draft`; one set aside — this run or an earlier one — and left
with no draft, its body a person's, loses `workline:to-refine` too. One
still holding a draft keeps them.

**No quick action.** Every line of text the engine writes on a forge
that starts with `/`, outside fenced code — in a comment, a note's edit,
a section written in a body — is escaped (`backlog.Inert`): GitLab runs a
quick action at a line's start when a comment or a description is
written and when it is edited.

**The job's summary** (ADR-0035) says the run, one finding an issue:
`done` and `done-as-accepted`, `proposed`, `set-aside`,
`left-to-a-person`, `next-ready`, `stuck`, `demoted`, `report-closed`.
It groups them (`report.Summary`): one heading a group — done alone,
done as a person accepted, proposed, left to a person (`left-to-a-person`,
`asks-spent`, `spec-rounds-spent`), read and left incomplete
(`nothing-proposed`, `scope-without-files`, `sources-unknown`: each says
what a person may do), set aside, next, stuck, held back by the engine
(an act of the agent's not done or only proposed, and why: nothing for a
person to do) —, under it one line an issue, its
first three findings joined, each cut at a word to its share of 120
characters (a code span the cut leaves open closed), then "and N more";
the issue linked to its page
(`forge.IssuePages`); 10 issues a group, then "and N more"; the groups
waiting on a person link the filter on the label (`forge.LabelFilter`).
The other findings follow, a line each: on an issue, the issue linked and
its message, no rule's name; on the backlog (`proposals-waiting`,
`demoted`, `setting-gone`), "the backlog" and its message. The role's own
line counts what a person will see on the issues, by issue — "issues: 2
done alone, 1 proposed to a person, 5 set aside by a person" —, never the
writes the run takes (`engine.onIssues`).

**Judged, not applied** (`--no-apply`, CI's first job): the engine
decides the plan (`backlog.Decide`, reading the forge only), keeps it in
the run folder (`out/acts.yaml`) and says it, headed "to do alone, once
applied", "to propose, once applied"; `workline apply` does that plan,
not another. A forge that cannot be read then is said (`plan-unread`,
warn), and the apply decides.

**The first run after a report issue**: the record on an open issue
titled "Backlog — product owner" is read once — its proposals, acts done
and undone, closings found wrong, closings, acts with their day —, each
issue's part written to that issue's state, unless it holds one already;
then the report is closed, a comment linking to the filter on the label
(`forge.LabelFilter`). Its pause, ticks, measure and changes are not
carried. A record that does not read leaves the report open, said
(`record-broken`).
