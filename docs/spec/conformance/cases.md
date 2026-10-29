---
sources: [tests/conformance/runner_test.go]
checked: 21ce8c2c
verified: agent:claude-code
---
# Conformance — a case in full

What a case can give, run and expect beyond the example in
[Conformance](../conformance.md#a-case), in full.

## Read strictly

A case is read strictly, by the YAML library's own check: a key the runner
does not read, at any level — a misspelled `body-contain`, a `body` it has
no check for — fails the case, naming the key and its file, whatever
`pending.txt` says. A case never passes while checking nothing. The forge
`given` holds, and the edits in `between`, are checked against the
simulated forge's own state, by the JSON library's.

## Given

`given` can also carry `models-seen`: the models this machine saw answer before
the run (`{"fake:light": {model: …}}`), in the file `WORKLINE_MODELS_SEEN` names.
It can also carry `repos`: other repositories to build first, each
exported by name as an environment variable holding its path, for
multi-repository cases. `env` sets variables for the run (`$VAR` expands):
a `PATH` without a tool, or with a fake one first. A case runs without
the machine's `WORKLINE_*` and CI variables: it sets those it needs.
`engine-tags` runs the case on an engine built from this source with those Go build tags, a
change only the tests make: `workline_alias` gives how a person accepts a
need a new heading, written first, the shipped one still read
([ADR-0039](../../adr/0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md));
`workline_quick` gives a GitHub or GitLab call 2 s, not 60, and tries it
again 10 ms after, not 2 s, so a case on a call that stalls lasts seconds
(a call tried again: [GitHub](../../github/calls.md),
[GitLab](../../gitlab/calls.md)).
Such a case does not run on an engine `WORKLINE_ENGINE` names: it fails,
saying so.

## Run, instead of a role

`run` can also carry:

- `route: <event>` instead of `role` — run the event's whole line;
- `gate: <name>` instead of `role` — run one gate;
- `test: [<argument>...]` instead of `role` — run `workline test` on the
  repository (`[check, --base, main]`), the tester's check with no agent;
- `develop: [<option>...]` instead of `role` — run `workline develop`
  on the repository, on the simulated forge, with the case's `ai`;
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
  sample (`workline sample`), reading the simulated forge, when the case
  gives one, and never writing to it; `then: apply` then writes what it
  found with `workline sample --apply`, on the simulated forge;
- `follow: [<option>...]` instead of `role` — `workline follow` on the
  repository (`[--base, main]`), on the simulated forge, or GitLab's with
  `forge: gitlab`; with `then-line: {event, branch, range, ai}`, the line
  then runs on that branch of `origin`, checked out, as CI runs it on its
  push: status, steps and calls are the line's, the findings both runs';
- `cli: [<argument>...]` instead of `role` — `workline` with these
  arguments, as typed, in the repository, with `given.env`: help, misuse, a
  hook (`[hook, commit-msg, .git/COMMIT_EDITMSG]`); checked by `exit`,
  `stdout`, `stderr`, `stderr-lacks`, `files`, and `summary-file` for the
  `workline-summary.md` its arguments name;
- `sh: <line>` instead of `role` — a shell line in the repository, the
  engine first on `PATH` as `workline`: what a person types around it
  (`workline hooks install --global && git push`), checked as `cli` is;

## Run, around the role

- `route: ready` with `item` — ask routing to move a work item;
- `target` — the issue or merge request comments and labels go on
  (`{merge-request: 1}`); `branch` — the branch that merge request comes from;
  `args` — more options, as typed, after the run's own (`[--dry-run]`);
- `open-merge-request: true` — put what the patches write in a merge
  request of its own; with a merge request as `target`, it goes in a
  comment there;
- `reports: true` — also write the findings as SARIF and Code Quality;
- `gates-only: true` — the route runs the event's gates alone
  (`--gates-only`); `gates-job: {env, input, after}` — a run of the gates
  alone first, with these variables and inputs, then the commands `after`
  gives, whose result the route reads (`--gates-from`), as
  CI's judge reads the gates' job, the run apart and the second run too;
  that job takes no `--warn-*` of the route's, and runs again after
  `again-after`, as on each push; `gates-from: <file>` — read from that
  file instead, by the case's folder;
- `warn-apart: true` — the route leaves what only warns to a run apart
  (`--warn-apart`); `warn-only: true` — the route is that run
  (`--warn-only`); `then: apart` — after a run with `warn-apart`, the run
  apart, the same line with `--warn-only`, on the forge the first left:
  status and summary stay the first run's, the verdict; agent calls,
  calls and findings are both runs'; `apart-status` in `expect`, the run
  apart's status, `pass` when not given;
- `summary: true` — also write `--summary`, Markdown and HTML, and give the
  same files to the `apply` or `sample --apply` that follows;
- `main-issue: true` — after the run, and after the second with `then:
  again`, what it printed kept in main's issue (`apply --main`), its
  findings added; `kept-on-main: <rev>` — first, main's run after a merge
  on that revision, checked out then left, its gates' findings kept in a
  folder the run reads (`--gates-kept`); `dry-run-posted: true` — after
  the run, and after the second with `then: again`, what a release's
  dry run printed posted (`apply --release-dry-run`), its findings added;
- `scope` — the run's scope, as a ready work item would give it;
- `no-apply: true` — judge, and stop before applying;
- `forge` — a forge spec given as `--forge` instead of the simulated forge
  (`local`, `none`, `cmd:<command>`, `gitlab`), the sample's `then: apply`
  included;
- `then: resume` — after the run, resume it with `workline apply`: the run, or
  every run a line judged and did not apply, after the edits to the forge
  given in `between`, as `then: again` takes them; `then: resume-elsewhere` — the
  same with another cache and the roles built into the engine, as CI's
  second job; `then: again` — after the run, the same run once more, on
  the forge and the repository as the first left them, after a person's
  edits to the forge given in `between` (`{issues: [{id: 4, body: …}]}`:
  each item, by kind and id, takes the fields given): what one run
  records, the next reads back. Both runs' agent calls, calls and findings
  are counted together, the rest is the second's; `again-after`, commands
  run in the repository before the second run (a second push), the first
  run's Code Quality report kept for `code-quality-kept`;
- `then: judged-twice`, `applied-between` and `applied-out-of-order` — a push landing before
  the run is applied ([#375](https://github.com/JN0V/workline/issues/375)):
  the commands in `pushed` run, the same run judges the new head, and both
  are applied, the first first. `judged-twice` judges both before either
  is applied; `applied-between` applies the first before the second
  judges, as the lock the judging and applying jobs share makes it. With
  `apply-from`, each is applied with the head its run judged as `--head`;
  with `gates-job`, that job runs again on the new head before the second
  judges, as CI runs it on each push. `applied-out-of-order` judges both,
  the second with `CI_PIPELINE_ID=102`, a later pipeline, and applies the
  second first ([#598](https://github.com/JN0V/workline/issues/598)); the
  rest is the first's apply's, the last.
  Both judges' agent calls and calls are counted, all four's findings; the
  rest is the last apply's;
- `apply-from` — with `then: resume`, `judged-twice` or
  `applied-between`, apply as a job of the base's
  applies a merge request's own CI run (`apply --from-merge-request`):
  `merge-request`, the one it judged; `base`, the branch checked out
  first; `head`, a branch of `origin` whose tip is `--head`; what that
  run's code could have done to its folders: `run` (keys of
  `out/run.yaml` written over), `intents` (proposals added), `listed` (the
  run listed at that path instead), `setup` (commands run first: what the
  branches became since); `env`, the applying job's variables
  (`WORKLINE_APP`, `$VAR` expanded); `line: true`, the line's result
  applied as CI's job does (`apply --line`), its runs, none when it
  proposed nothing; `gates: true`, the run's result shown as the gates'
  (`apply --gates`); `again`, commands after the apply (a second push),
  then the run and the apply once more, on the head they leave;
- `tamper: in/` — change the prepared input between prepare and apply.

What a case expects after its run: [Conformance — what a case expects](cases-expect.md).
