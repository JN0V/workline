---
type: reference
sources: [internal/backlog/parent.go, internal/builtin/productowner/productowner.go, internal/forge/forge.go, internal/forge/github.go, internal/forge/gitlab.go]
status: draft
checked: 21055408
verified: agent:claude-code
---
# A parent and its parts

Part of [Acts on the backlog](../backlog-acts.md): a parent issue reported on as its parts close.

A parent is accepted by a person, from what its parts delivered
(ADR-0029). Its parts are the forge's own relation (`Issue.Children`:
GitHub's sub-issues of the same repository, listed for the parents its
listing counts; GitLab's tasks, one GraphQL query), the task list under
`## Sub-issues` in its body (`- [ ] #13`, a person's included) — one rule
for the comment, the summary and the order (`backlog.Parts`); a part a
person unlinked is no longer one. An issue whose state lists a split
(`backlog.SplitInto`) is still never closed by the role.

- **Ready** means what it means on any issue; a parent is refined as any.
  It is never the first ready issue offered (`next-ready`): its parts are
  what is built.
- **Its comment**, at every run, with or without an agent,
  on every open parent whose state reads: one comment marked
  `<!-- workline:sticky=<role>/parts -->`, edited in place, and not edited
  at all when its text is the same (`Sticky` compares first). A table of
  its parts — open (ready or not), closed as completed with what closed it
  (`Closers`: GitHub's `ClosedEvent.closer`, a pull request or a commit;
  GitLab's state events, the last closing's `source_commit` or merge
  request, found among `closed_by`'s; none: "by hand"; a forge that
  refuses to say: "not read", and a finding `closers-unread` (warn) —
  one that does not answer stops the run), closed as not planned or as a duplicate, or
  gone from the forge: those three not delivered. GitLab and the local
  forge keep no reason: a closed part there is taken as done. Then each
  item of its Verification — each list item, or the section whole — proved
  when a part delivered quotes it, in its own Verification or in the text
  of what closed it, case, spaces, `` ` ``, `*`, `_` and the final
  punctuation aside, said with where; "not proved" otherwise ("not proved yet" while a part is open). A parent
  without a Verification is said to have none.
- **A test named as proof** (ADR-0029, amended): an item that names a
  test in a code span — a test file by its path (`tests/…`, `…_test.go`,
  `….spec.ts`, `test_….py`), one test in it as `path::name`, a test's own
  name (`TestX`, `test_x`, `testX`), or any name right after the words
  "test", "test case" or "conformance case" — never "case" alone, prose's
  "in that case" (`backlog.TestNames`) — is proved only when the code
  at the run's commit holds it too: the file there, a file and never a
  folder, its path taken literally; the name a word in a test file
  (`backlog.CodeTests`, `git grep` at `HEAD`) — found, not proved
  declared as a test there; the comment says where. A test the code does not hold leaves the item not proved, its
  quote said, and a finding `proof-test-missing` (warn) names it: a name
  is no proof until it is there. A git that failed is never read as a
  test missing: the item is not proved, "could not be looked for", and
  `proof-test-unread` (warn) says why. Whether the test passes is CI's,
  not read here.
- **All its parts closed**: the comment asks a person to accept the need
  by closing the parent, or to reopen a part or open one for what is
  missing, naming first the parts not delivered and the items not proved;
  a finding `parent-to-accept` (info) says it in the job's summary. The
  role never closes a parent nor announces it obsolete: a `close` on one, a duplicate
  or an announcement as obsolete, is dropped
  (`parent-accepted-by-a-person`) unless a person accepted it. The agent's
  task marks a parent's parts, open or closed.
