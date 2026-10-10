<!-- workline
sources: [tests/conformance/runner_test.go]
-->
# A conformance case: its fields

The fields a case of [conformance](../spec/conformance.md) can carry beyond the example there.

## Given

`given` can also carry `models-seen`: the models this machine saw answer before
the run (`{"fake:light": {model: …}}`), in the file `WORKLINE_MODELS_SEEN` names.
It can also carry `repos`: other repositories to build first, each
exported by name as an environment variable holding its path, for
multi-repository cases. `env` sets variables for the run (`$VAR` expands):
a `PATH` without a tool, or with a fake one first.

## Run: what to run

`run` can also carry:

- `route: <event>` instead of `role` — run the event's whole line;
- `gate: <name>` instead of `role` — run one gate;
- `doctor: true` instead of `role` — run `workline doctor` on the repository;
- `init: true` instead of `role` — run `workline init` on the repository,
  with `init-options: [<option>...]` if given (`--human-po yes`);
- `setup: [<option>...]` instead of `role` — run `workline setup` with these
  options, on a git config of the case's own;
- `issues-import: [<argument>...]` instead of `role` — `workline issues
  import` on the repository, on the simulated forge;
- `review: [<option>...]` instead of `role` — `workline review` on the
  repository, with these options;
- `sample: [<option>...]` instead of `role` — draw and read the weekly
  sample (`workline sample`), with no forge; `then: apply` then writes what
  it found with `workline sample --apply`, on the simulated forge;
- `follow: [<option>...]` instead of `role` — `workline follow` on the
  repository (`[--base, main]`), on the simulated forge, or GitLab's with
  `forge: gitlab`;
- `cli: [<argument>...]` instead of `role` — `workline` with these
  arguments, as typed, in the repository: help, misuse; checked by `exit`,
  `stdout`, `stderr`, and `summary-file` for the `workline-summary.md`
  its arguments name;

## Run: items, targets, resuming

- `route: ready` with `item` — ask routing to move a work item;
- `target` — the issue or merge request comments and labels go on
  (`{merge-request: 1}`); `branch` — the branch that merge request comes from;
- `open-merge-request: true` or `push-to-merge-request: true` — put what the
  patches write on a merge request;
- `reports: true` — also write the findings as SARIF and Code Quality;
- `summary: true` — also write `--summary`, Markdown and HTML, and give the
  same files to the `apply` or `sample --apply` that follows;
- `scope` — the run's scope, as a ready work item would give it;
- `no-apply: true` — judge, and stop before applying;
- `forge` — a forge spec given as `--forge` instead of the simulated forge
  (`local`, `none`, `cmd:<command>`, `gitlab`), the sample's `then: apply`
  included;
- `then: resume` — after the run, resume it with `workline apply`: the run, or
  every run a line judged and did not apply; `then: resume-elsewhere` — the
  same with another cache and the roles built into the engine, as CI's
  second job;
- `tamper: in/` — change the prepared input between prepare and apply.

## Expect: findings and calls

`expect` lists only what the case is about; anything not listed is not checked.

- `findings` — each listed finding must be present (matched by `rule`, and by
  `where` and a part of its `message` and `fix` if given).
- `checks` — the doctor's checks, ok ones included: each listed must be
  present, by `rule`, and by `level`, `where`, a part of `message` and of
  `fix` if given.
- `no-findings` — findings that must not be there (by `rule` and `where`,
  and a part of the message when given).
- `agent-calls` — how many times the agent was called.
- `calls` — each call, in order: the fields listed must match (`agent`,
  `task`, `tier`, `effort`, `asked`, `model`).
- `applied` / `refused` — intentions applied or refused, by kind.
- `steps` — for a line, the steps that ran, in order.

## Expect: the forge and branches

- `forge` — fields the simulated forge must hold afterwards: per item, by
  `id`, `comments` (a count), `labels` (the exact set), `comment-contains`
  (a text some comment holds, or a list of them) and `comment-lacks` (a
  text none does), `closed`
  and the `reason` it was closed for, its `milestone`, `branch`, `base`,
  `title` and the `parent` it is a sub-issue of (0 for none), `blocked-by`
  the issues it waits on in the forge's relation, `body-contains` and `body-lacks` (a text its body holds, or does
  not), `reactions` (the reactions on its comments, by the comment's id);
  `absent: true` — no item with that id. `labels` there lists the
  labels the forge defines, their order as id, by `title`.
- `pushed` / `pushed-message` — a text a file holds on a branch of the
  case's `origin`, or the message of that branch's tip.
- `not-pushed` — branches of the case's `origin` the run left where they
  were; `on-top` — a branch of `origin` whose tip holds another's
  (`{workline/documentalist/release: main}`).
- `branches` — a text a file holds on a local branch of the repository.
- `issues-listed` — texts `workline issues list` prints afterwards.

## Expect: outputs and files

- `summary` — a text the result's summary holds.
- `coverage` / `not-covered` — with `issues-import`, entries the import's
  map must hold, each matched on the fields given (`lines`, `state`,
  `issue`; `words` and `why` by a part of them); with `then: resume`,
  the judge's map, `workline apply` printing none.
- `notes` — texts the agent's notes must hold, all rounds together.
- `sarif` / `code-quality` / `left-out` — with `reports`, results each report
  must hold, and the places neither may name.
- `summary-file` / `summary-html` — with `summary`, texts the Markdown and
  the HTML summary must hold, in this order; with `cli`, the Markdown in
  `workline-summary.md`.
- `refused-kept` — how many refused answers the run folders keep.
- `calls-kept` — how many agent calls the run folders record.
- `run-files` — files of the run folder (`out/claims.yaml`), each holding a
  text (`contains`), or not (`not-contains`), each a text or a list.
- `files` — paths that must exist, or contain a text or each of a list of
  texts, or lack one (`lacks`), afterwards.
- `exit` / `stdout` / `stderr` — with `cli`: the exit code, and texts
  printed on each stream.
