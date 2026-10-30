---
sources: [tests/conformance/runner_test.go]
checked: 0bbdcbd9
verified: agent:claude-code
---
# Conformance — what a case expects

What a case can check after its run, beyond the example in
[Conformance](../conformance.md#a-case); what it gives and runs:
[a case in full](cases.md).

## Expect

`expect` lists only what the case is about; anything not listed is not checked.

- `findings` — each listed finding must be present (matched by `rule`, and by
  `level`, `where` and a part of its `message` and `fix` if given). `level:
  block` is a finding that blocks: of level `block`, or of none in a run
  that did not pass, as the [role outcome](../role-outcome.md) reads it.
- `checks` — the doctor's checks, ok ones included: each listed must be
  present, by `rule`, and by `level`, `where`, a part of `message` and of
  `fix` if given.
- `no-checks` — the doctor's checks that must not be there, by `rule`.
- `no-findings` — findings that must not be there (by `rule` and `where`,
  and `level`, a part of the message and of `fix` when given).
- `agent-calls` — how many times the agent was called.
- `calls` — each call, in order: the fields listed must match (`agent`,
  `task`, `tier`, `effort`, `asked`, `model`).
- `applied` / `refused` — intentions applied or refused, by kind.
- `steps` — for a line, the steps that ran, in order.
- `forge` — fields the simulated forge must hold afterwards: per item, by
  `id`, `comments` (a count), `posted` (the comments posted on it, each
  one mailing who follows it; an edit posts none), `edits` (the times its
  body was rewritten, as the forge's edit history counts them), `labels` (the exact
  set), `comment-contains`
  (a text some comment holds, or a list of them) and `comment-lacks` (a
  text none does, or a list of them), `closed`
  and the `reason` it was closed for, its `milestone`, `branch`, `base`,
  `title` and the `parent` it is a sub-issue of (0 for none), `blocked-by`
  the issues it waits on in the forge's relation, `body-contains` and
  `body-lacks` (a text its body holds, or does not, or a list of them),
  `reactions` (the reactions on its comments, by the comment's id),
  `fields` (its values in the issue fields, exactly these, by name),
  `weight` (its GitLab weight, -1 for none),
  `threads` (a merge request's comments on lines, exactly these, in the
  order posted: each `{path, line, contains, lacks, resolved}`, and
  `replies`, `replies-contain`: how many replies under it, texts they
  hold), `auto-merge` (a merge request's on), `draft` (opened as a
  draft) and `merged` (the forge merged it);
  `absent: true` — no item with that id. Items are listed by kind:
  `issues`, `merge-requests`, and `labels`, the labels the forge defines,
  their order as id, by `title` and `rank` (a GitLab prioritized label's,
  0 for none); `statuses`, the last status of a name on
  a commit (a revision), by `state`, `by` and a part of its
  `description`, or `absent: true`; `check-runs`, the last check of a
  name on a commit, its `title` and `count` of annotations when given,
  each annotation listed held by `path`, `line`, `level` and a part of
  its message (`contains`), or `absent: true`.

## Expect, after the run

- `pushed` / `pushed-message` — a text a file holds on a branch of the
  case's `origin`, or the message of that branch's tip; `pushed-log` —
  texts its commits past `main` hold, oldest first, each subject then
  its files, in the order given.
- `pushed-lacks` — a text a file of that branch does not hold (or no
  such file); `pushed-log-lacks` — texts no commit message past `main`
  holds.
- `not-pushed` — branches of the case's `origin` the run left where they
  were; `on-top` — a branch of `origin` whose tip holds another's
  (`{workline/documentalist/suspect: main}`); `merged` — a branch of
  `origin` whose tip is a merge of another's tip into its tip before the
  run, its first parent (`{feat/sessions: main}`); `gone` — branches of
  `origin` absent after the run.
- `signed` — a branch of `origin` whose tip carries a signature `git
  verify-commit` accepts with the repository's config (`true`), or none
  (`false`).
- `branches` — a text a file holds on a local branch of the repository.
- `issues-listed` — texts `workline issues list` prints afterwards.
- `summary` — a text the result's summary holds.
- `coverage` / `not-covered` — with `issues-import`, entries the import's
  map must hold, each matched on the fields given (`lines`, `state`,
  `issue`; `words` and `why` by a part of them); with `then: resume`,
  the judge's map, `workline apply` printing none.
- `notes` — texts the agent's notes must hold, all rounds together.
- `sarif` / `code-quality` / `left-out` — with `reports`, results each report
  must hold, and the places neither may name; `sarif-lacks`, rules no
  SARIF result may have; `code-quality-kept`, with `again-after`, each
  fingerprint of the first run's Code Quality report in the second's.
- `summary-file` / `summary-html` — with `summary`, texts the Markdown and
  the HTML summary must hold, in this order; with `cli`, the Markdown in
  `workline-summary.md`; `summary-lacks`, texts the Markdown must not hold.
- `moment` — the line's moment
  ([ADR-0050](../../adr/0050-each-act-has-its-moment-and-a-wait-budget.md)),
  each field given checked: `name`, `budget` in seconds, `over`, and the
  `step` named; every step and the line say their seconds.
- `refused-kept` — how many refused answers the run folders keep.
- `calls-kept` — how many agent calls the run folders record.
- `run-files` — files of the run folder (`out/claims.yaml`), each holding a
  text (`contains`), or not (`not-contains`), each a text or a list.
- `files` — paths that must exist (`exists`, or not), or contain a text or
  each of a list of texts (`contains`), or lack them (`lacks`), afterwards.
- `exit` / `stdout` / `stderr` — with `cli` or `sh`: the exit code, and texts
  printed on each stream; `stderr-lacks`, texts not printed there;
  `stdout-lacks`, with any run, texts not printed on stdout.
