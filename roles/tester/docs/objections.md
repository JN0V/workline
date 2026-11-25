---
sources:
  - roles/tester/role.yaml
  - roles/tester/code
  - internal/kit/testmark
  - defaults/ci/github/workline-objection.yml
  - defaults/ci/github/workline-answer.yml
  - defaults/ci/github/workline-gardening.yml
checked: e81df7b34
verified: agent:claude-code
---
# Frozen tests and objections

Part of [the tester](../README.md). The tests it froze stay as it wrote
them; a builder who disagrees objects, never edits.

## The record and the guard

- **The record**: on the Draft, workline's own comment lists each test
  file, its git blob, the scenarios read, the tests declared for each. Only the tester's runs write it.
  What it freezes is the tester's own lines: a whole new file, or the
  lines it added to one already there.
- **The guard**, no agent: route the tester on `merge-request`. On its
  Draft, and on a pull request closing the Draft's issue (the
  developer's; each Draft of the issues it closes), a frozen line changed
  or removed, or its file deleted, by anyone else blocks,
  `tester-test-changed`, naming the test's first line. Lines added
  beside them pass. No label lifts it, and no commit trailer is
  trusted. With no record to read, `tester-record-missing` blocks.

## An objection, and a test missing

- **An objection**: a comment on the test's line in the Draft, opened by
  `workline:objection`, one field a line: `test:`, `scenario:`,
  `kind:` (wrong interface, over-specified, contradicts another
  scenario, untestable as written, the need itself, a test missing),
  `why:`.
- **A test missing** ([#1265](https://github.com/JN0V/workline/issues/1265)):
  the review found a behaviour with no test, on code; the developer
  sends it, naming no scenario. The agent reads every scenario and
  the finding's file as the pull request has it
  ([#1416](https://github.com/JN0V/workline/issues/1416)), and
  answers:
  - *added*: the test in one of its tests of the issue, the file whole
    or a diff, proven as a rewrite;
  - *none needed*: its reason;
  - *lacks code* (`needs:`): the files it lacks, never read as *none
    needed*.
  - Its answer is said on the pull request whose review found it too,
    naming the finding's line and title: the developer reads it there,
    and the reviewer's judge weighs a *none needed*
    ([weighed](../../reviewer/docs/tester-answers.md)).
- **A person's review comment** on a test's line in the Draft, in no
  objection's form: read as a test missing, and answered the same way
  ([#1358](https://github.com/JN0V/workline/issues/1358)).

## The answer

- **The answer**: `workline test answer --merge-request <n> --ai claude`
  takes the first objection waiting, checked out at the branch's tip.
  The agent reads the scenario first, then answers in the thread:
  - *kept*, with its reason;
  - *rewritten*: its own commit on the branch, `Refs` the thread. Its subject keeps within the
    committer's subject limit and starts with a type the committer allows
    (`test` unless the project's settings forbid it). It
    must be red on the base, and keep its scenarios' declared tests,
    or name the new ones in a `covers` entry. The record
    and the reply follow only once the commit is pushed.
  - Either keeps a hidden line for the [weekly sample](../../../docs/weekly-sample.md).
- **The need itself** goes to the product owner, with no agent: the
  issue back to `workline:to-refine`, said there and in the thread.
- **Bounded**: three answers a thread (`objection-exchanges`).
  - The third first runs the
    [five whys](../../../skills/five-whys/SKILL.md) on the thread and
    posts them with its answer: the chain, the root cause, one action.
    Missing or malformed, the answer is refused and asked again.
  - After it, the product owner settles it: the issue back to refine,
    said there and in the thread.
  - Objected to again after that, the issue goes `workline:blocked`, its
    reason on it
    ([ADR-0059](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#6-disagreements-end-at-the-product-owner-then-blocked)).
  - No person is asked; a comment of one of the project's there is
    followed. Meanwhile the other objections are answered.
- **The agent is a choice** here too: with none, `no-agent`, it waits.
