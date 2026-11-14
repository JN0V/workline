---
sources: [internal/doctor/doctor.go, internal/doctor/conversations.go, internal/doctor/variables.go, internal/hooks/hooks.go, cmd/workline/main.go, cmd/workline/cli.go, internal/forge/local.go]
checked: adf500ab
verified: agent:claude-code
---
# Troubleshooting

Start with `workline doctor`: it says what is set up on this machine and in
this repository, each missing thing with the command that sets it up, and
changes nothing. It exits 1 only on an error (an agent that cannot be
called, a config that does not load). Each run is kept in
`.git/workline/runs/` — the agent's answer as it came, each call and its
tokens — and `--json` prints every finding.

## A hook does not fire

- **Not installed**: doctor says `hooks-not-installed`; run `workline hooks
  install --global`.
- **The binary moved**: the hooks call it where it was when installed.
  Install it to the same place, or run `workline hooks install --global`
  again.
- **Written by another version**: doctor says `hooks-outdated`. Older
  hooks skipped a repository's own `.git/hooks/` from a linked worktree
  ([#551](https://github.com/JN0V/workline/issues/551)). Run `workline
  hooks install --global` again; `--repo` too, then commit
  `.githooks/commit-msg`.
- **A repository sets its own `core.hooksPath`** (husky, lefthook…): git
  takes it over the global one. `workline doctor` in the repository says
  `hooks-bypassed`, naming the path, when its `commit-msg` (and `pre-push`,
  when routed) does not hand over to workline; `hooks-path` when it does.
  A hook hands over when it is executable and runs `workline hook <hook>`
  (or workline's global hook) as a command before a top-level `exit`: a
  comment or an `echo` does not count.
  The fix it prints:
  - the path is `.githooks` or `.git/hooks` (the main one, from a linked
    worktree too): unset it (`git config --local
    --unset core.hooksPath`); the global hooks run workline, then hand over
    to that folder;
  - another folder: add `workline hook <hook> "$@" || exit $?` at the top of
    each hook there. `pre-push` reads the pushed refs on its input: a hook
    that reads them too must copy them first.
- **The repository opted out**: an empty `.workline/off` (doctor says so).
- **`pre-push` does nothing**: it runs only when `.workline/config.yaml`
  routes `pre-push` (`workline init` does). `git push --no-verify` skips it.
- **A commit goes through unchecked for secrets**: gitleaks is missing; the
  verdict says `secrets-not-checked`. `workline setup` installs it.

## A command fails before it runs

| Sign | Cause |
|---|---|
| exit 64 and the usage | the command was misused, named on the first line: a missing `<role>` or `<event>`, an unknown option, a bad value; `workline <command> --help` shows its options |
| `.workline/config.yaml: line N: …` | a key the engine does not know, a retired role, or a value YAML reads otherwise than written: it blocks rather than be ignored |
| `no gate named "x"` | `workline gate` or `gate:x` names a gate not declared; the gates declared follow |

## A language no tool reads

- `language-gaps`: no tool reads one kind of it (lint, types, security,
  dependencies); doctor names the tool and the gate check to add
  ([languages](languages.md)). A warning: nothing blocks on it.
- **A tool your CI runs is not seen**: a script of your own runs it.
  Name it in a gate check.
- **Vendored or generated code is counted**: mark its folder
  `linguist-vendored` or `linguist-generated` in `.gitattributes`.

## Docs suspect, gardening paused

`docs suspect: N` at the push, `record-unread`,
`shallow-clone`, `gardening-paused`, `merge-request-unread`: what each
means and what to do, in [the docs' troubleshooting](troubleshooting-docs.md).

## The product owner paused, or does nothing

`proposals-waiting`, `issues-cut`, an issue that gets nothing or never
goes ready, a hidden record that does not read: in
[the product owner's troubleshooting](troubleshooting-product-owner.md).

## Token caps and the agent

- `ai-max-tokens` (documentalist, reviewer): the run spent its cap; the rest
  waits for the next run. The reviewer leaves the commits unrecorded
  (`review-not-whole`, `lens-failed`) and says what each call used. `ai-max-calls`, `parts-max-per-run`, the product owner's
  `issues-per-run` and `code-lines-max`, the reviewer's `diff-lines-max`,
  `code-lines-max` and `tests-lines-max` size each run (each role's page).
- `input-cut` (a warning): the agent read part of what the run gives it;
  the line says what, how much of it, and the setting that raises it.
- `size-crossed`: a role's run on an issue was too big for it: its task
  over the context budget, or its token cap reached; a cut alone is not
  one. Never run again on the same input: the issue goes to
  Blocked, and the product owner reads it at once, to narrow its Scope,
  grant it `workline:size/large`, or leave it to you
  ([routing](spec/routing.md#the-line-of-states)).
- `size-crossed-again`, `size-crossed-own`, `size-crossed-top`: the
  same, the product owner already asked from that state, its own run too
  big, or the issue at `workline:size/x-large` already. Narrow the issue
  or split it, put `workline:size/x-large` on it (every size target ×4),
  or raise the setting the line names, then move it back by its label.
- `backstop-crossed`: the issue's runs, all roles, spent 4× its normal
  (1.6M fresh tokens at the seed); `large` never raises it. It waits for
  you, no product owner asked (`backstop-waits` on news), no review
  round run on its pull request. Put
  `workline:size/x-large` on it (16× its normal) and move it back by its
  label, or split it into new issues
  ([routing](spec/routing.md#the-line-of-states)).
- `size-tier` (info): every size target of the run multiplied by its
  issue's tier label; take it off to undo. `size-person`, `size-kept`,
  `size-once`, `split-depth`, `split-off`, `too-big-unasked`: a grant or
  a split on size dropped
  ([why](spec/backlog-acts/refine.md#granting-a-size)).
- `ai-max-seconds` (committer): the rewrite took more than the 30 s it
  may; the message is refused with its reasons, as with no agent.
- `personal-setting-ignored`: your own config sets what a role does not
  let a person set; the project's stands
  ([your own config](config.md#your-own-config)).
- `review-not-whole`, "past parts-max" (reviewer): the change is too
  large to read in the parts a run allows; split it, or raise
  `parts-max` ([in parts](../roles/reviewer/docs/parts.md)). Read whole
  by a person, `workline:accepted-whole` lets it merge
  ([merge policy](merge-policy.md#what-it-does)).
- `blocked-external` (exit 3): the agent's quota, a timeout, the forge or a
  repository failed. Never a verdict: run it again later. The CI templates
  turn it into a warning, the docs listed for a person. A GitHub or
  GitLab call is tried again first (which: [GitHub](github/calls.md),
  [GitLab](gitlab/calls.md)).
- `agent-unavailable`: the agent named cannot be called; doctor says why.
  The developer's `claude` on Linux: bubblewrap or socat missing, or a
  runner that refuses its sandbox; the finding names which
  ([its agent](../roles/developer/docs/agent.md#claude)).
- `model-changed`: another model answered an alias than last time on this
  machine; said once, never blocking.

## Forge writes refused

- `forge: none` (the default) refuses every write that needs a forge, the
  issue a role opens included, and says so.
- `forge: cmd:<command>`: `workline doctor` runs it once, writing
  nothing, and warns when it is not found, fails or does not answer
  ([what the doctor checks](spec/forge-command.md#what-the-doctor-checks)).
- `forge: local` refuses its writes in CI (`CI`, `GITHUB_ACTIONS` or
  `GITLAB_CI` set): pass `--forge github` or `--forge gitlab` to jobs that
  write.
- GitLab: `CI_JOB_TOKEN` reaches no issue; give `GITLAB_TOKEN`
  ([ci-gitlab.md](ci-gitlab.md#gitlabcom)). A **protected** variable never reaches a merge
  request's pipeline on an unprotected branch.
- GitLab, `tokens-not-scoped`: a token left to every environment reaches
  the jobs running your linters; scope it to `workline`.
  `tokens-scope-unread`: listing the variables needs a Maintainer's
  token. A job saying `WORKLINE_GITLAB_TOKEN is not set, or not scoped`:
  the variable is scoped elsewhere, or the template's copy is older than
  its environment ([the environment](gitlab/environment.md)).
- `findings-not-annotated`, on GitHub: the check `workline findings` was
  refused. Give `workline-apply.yml`'s `apply` job `checks: write`, and a
  GitHub App the permission **Checks**, read and write
  ([the App](ci-github-app.md)). The findings stay in the judge's job
  summary.
- `gates-not-shown`, on GitHub: the check `workline gates` or the gates'
  summary comment was refused, the same rights; the findings stay in the
  gates' job summary. `moved-on`: the branch moved on before the apply,
  and its next run shows its own ([gates in CI](gates-in-ci.md)).
- A gate's `check-error` in `judge`, "the gates' job left no result": the
  job `gates` did not run it, or failed first; read its log. Copy
  `workline.yml` of this release again if it has no job `gates`.

## An issue does not move

The line of states says why on the issue and in the run's summary
([the line of states](spec/routing.md#the-line-of-states)):

- `state-wakes-none`: no state label, or its state's role is not routed
  on `issue`; the reason follows.
- `state-waits`: the next state is full (`routing.wip`); it starts when
  room appears.
- `hops-capped`: past `max-hops` steps, it went Blocked; put a state's
  label back to start again.
- `reconcile-left`: past `reconcile-max`, it waits for the next run.
- Blocked: read again on news, else every `revisit` days
  ([blocked, read again](spec/routing.md#blocked-read-again)).

## A pull request blocked, its checks green

- **Conversations to resolve**: GitHub's rule "require conversation
  resolution" holds the merge while one is open. `workline doctor` says
  whether the default branch has it.
- **Code scanning's**: workline before
  [#412](https://github.com/JN0V/workline/issues/412) sent every finding
  there, one conversation each. Copy `ci/github/workline.yml` and
  `workline-apply.yml` of this release: the documentalist's findings
  become a check's annotations, which open none.
- **The reviewer's comments** are conversations by design. The reviewer
  resolves its own once the code answers them
  ([how](../roles/reviewer/docs/lines.md#not-repeated-found-again)); a
  person resolves the rest. `workline follow` lists the open ones that
  hold an auto-merge
  ([#1039](https://github.com/JN0V/workline/issues/1039)).
- **`gh pr merge --admin` refused**, "Repository rule violations": by
  design, nobody bypasses `main`. Put `workline:accepted` on it; in an
  emergency, the written way round
  ([nobody bypasses `main`](merge-policy-no-bypass.md#the-emergency)).

## Nothing merges

The [merge policy](merge-policy.md) acts only with `merge:` in
`.workline/config.yaml`; `workline follow` says why in the job's summary
([each word](merge-policy.md#what-the-summary-says)):

- `label-ignored`: the label was put by someone who may not push nor
  merge; one who may puts it again.
- `verdict-missing`: no verdict of the line on that head yet; it waits
  for the next judge.
- `waits-for-a-yes`, at `normal`: workline's pull request with a push, a
  commit or a page not its own, or of a kind a revert sent back to a
  person's yes — someone who may push and merge closes the issue it names
  to restore it —; the developer's feature or task, or a fix whose bug no
  test reproduced; the release tool's always.
- `merge-app-unknown`: at `normal`, the token is a person's, or the forge
  does not say its App (GitHub: `WORKLINE_APP`): nothing merges alone.
- `merge-record-lost`: at `normal`, no mark of the App's on the base's last
  100 commits — the first run too —: someone who may push and merge closes
  the issue it names, and workline's own work merges alone again.
- `merge-unsupported`, `merge-token`, `merge-unguarded`: nothing is
  turned on — a forge with no auto-merge, a GitHub job's own token, a
  base that does not require `workline/accepted`
  ([set up](merge-policy.md#set-up)). The local forge merges itself
  ([the merge](local-line.md#6-the-merge)).
- `auto-merge-refused` on the local forge: `main` checked out with
  uncommitted changes, a conflict, or a branch moved since it was
  weighed; commit or stash, rebase, or run the line again.
- `verdict-not-kept`: `follow --verdict` with no `merge:` set, or on a
  forge without auto-merge; on the local forge, `route merge-request`
  could not keep its verdict in the clone: the request waits, run the
  line again.
