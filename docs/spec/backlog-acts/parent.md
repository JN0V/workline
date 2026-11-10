---
type: reference
sources: [internal/backlog/parent.go, internal/builtin/productowner/productowner.go, internal/line/parent.go, internal/forge/forge.go, internal/forge/github.go, internal/forge/gitlab.go]
status: draft
checked: 8538d558
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
  when a part delivered quotes it, in its own Verification, in a scenario
  of its Validation, or in the text of what closed it, case, spaces, `` ` ``, `*`, `_` and the final
  punctuation aside, said with where; "not proved" otherwise ("not proved yet" while a part is open). A parent
  without a Verification is said to have none.
- **An item given to a part at the split**
  ([#1279](https://github.com/JN0V/workline/issues/1279)): quoted in a
  scenario of the part's Validation (any form `work.Scenarios` reads), it
  is proved by the tests the part's Verification declares for that
  scenario — its list item opening on the scenario's name in bold, then
  the test ids in code spans — each found in the code as below, by its id or else its last
  word (`tests.test_x.test_y` is `test_y`). The quote alone, a draft
  written before any test, proves nothing: "not proved" when the quote is
  in no scenario, when the Verification names no test for it, or when it
  says `no test can check it` (its reason said, a person judges).
- **A test named as proof** (ADR-0029, amended): an item that names a
  test in a code span — a test file by its path, as the project's one
  definition of a test file names it (the tester's `test-paths`, else the
  listed [packs'](../../packs.md), else the generic patterns,
  [ADR-0065](../../adr/0065-code-only-what-converges-language-knowledge-in-packs.md#3-one-definition-of-a-test-file)),
  one test in it as `path::name`, or any name right after the words
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
- **A name alone in a code span** (`` `TestWriteRows` passes``), by no
  language's rule (`backlog.BareNames`): looked up as a word in the test
  files, said where it is found, or "not found in a test file"; never a
  missing test, never a block
  ([ADR-0065, Rule A](../../adr/0065-code-only-what-converges-language-knowledge-in-packs.md#rule-a-a-guess-may-only-pass)).
- **All its parts closed**: the comment asks a person to accept the need
  by closing the parent or labelling it `workline:accepted`, or to reopen
  a part or open one for what is missing, naming first the parts not
  delivered and the items not proved; a finding `parent-to-accept`
  (info) says it in the job's summary.
- **The closing proposed** (ADR-0029, amended): the engine's `close`,
  reason `completed`, marked `parent`, with no quote — the comment on its
  parts is the evidence. Always proposed, at every level:
  `workline:proposed` on it; a person's `workline:accepted` closes it as
  completed, their yes named. The mark is the engine's: on an agent's
  intention it is dropped. It is checked at the proposal and at the
  yes: the issue split, every part closed now — else dropped,
  `parent-not-split` or `parent-part-open`. A yes dropped for a part
  open again is taken off, said in the role's comment: it never stands
  for a later act, nor moves the parent to ready.
- **Never closed by the role alone**: an agent's `close` on a parent, a
  duplicate or an announcement as obsolete, is dropped
  (`parent-accepted-by-a-person`) unless a person accepted it. The agent's
  task marks a parent's parts, open or closed.
- **The line tells it** where its state wakes nobody, a part closed or a
  yes put on since it last read it
  ([routing](../routing.md#a-parent-told-as-its-parts-close)).
