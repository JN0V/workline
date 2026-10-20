---
sources: [internal/doctor/doctor.go, internal/doctor/conversations.go, internal/hooks/hooks.go, cmd/workline/main.go, cmd/workline/cli.go, internal/forge/local.go]
checked: a88dc872
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

## A language no tool reads

- `language-gaps`: no tool reads one kind of it (lint, types, security,
  dependencies); doctor names the tool and the gate check to add
  ([languages](languages.md)). A warning: nothing blocks on it.
- **A tool your CI runs is not seen**: a script of your own runs it.
  Name it in a gate check.
- **Vendored or generated code is counted**: mark its folder
  `linguist-vendored` or `linguist-generated` in `.gitattributes`.

## Docs suspect, gardening paused

`docs suspect: N` at the push, a release held by docs, `record-unread`,
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
- `ai-max-seconds` (committer): the rewrite took more than the 30 s it
  may; the message is refused with its reasons, as with no agent.
- `personal-setting-ignored`: your own config sets what a role does not
  let a person set; the project's stands
  ([your own config](config.md#your-own-config)).
- `review-not-whole`, "past parts-max" (reviewer): the change is too
  large to read in the parts a run allows; split it, or raise
  `parts-max` ([in parts](../roles/reviewer/docs/parts.md)).
- `blocked-external` (exit 3): the agent's quota, a timeout, the forge or a
  repository failed. Never a verdict: run it again later. The CI templates
  turn it into a warning, the docs listed for a person. A GitLab call is
  tried again first ([which](gitlab/calls.md)).
- `agent-unavailable`: the agent named cannot be called; doctor says why.
- `model-changed`: another model answered an alias than last time on this
  machine; said once, never blocking.

## Forge writes refused

- `forge: none` (the default) refuses every write that needs a forge, the
  issue a role opens included, and says so.
- `forge: local` refuses its writes in CI (`CI`, `GITHUB_ACTIONS` or
  `GITLAB_CI` set): pass `--forge github` or `--forge gitlab` to jobs that
  write.
- GitLab: `CI_JOB_TOKEN` reaches no issue; give `GITLAB_TOKEN`
  ([ci-gitlab.md](ci-gitlab.md#gitlabcom)). A **protected** variable never reaches a merge
  request's pipeline on an unprotected branch.
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

## A pull request blocked, its checks green

- **Conversations to resolve**: GitHub's rule "require conversation
  resolution" holds the merge while one is open. `workline doctor` says
  whether the default branch has it.
- **Code scanning's**: workline before
  [#412](https://github.com/JN0V/workline/issues/412) sent every finding
  there, one conversation each. Copy `ci/github/workline.yml` and
  `workline-apply.yml` of this release: the documentalist's findings
  become a check's annotations, which open none.
- **The reviewer's comments** are conversations by design: answer or
  resolve each.

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
  to restore it —; the release tool's always.
- `merge-app-unknown`: at `normal`, the token is a person's, or the forge
  does not say its App (GitHub: `WORKLINE_APP`): nothing merges alone.
- `merge-record-lost`: at `normal`, no mark of the App's on the base's last
  100 commits — the first run too —: someone who may push and merge closes
  the issue it names, and workline's own work merges alone again.
- `merge-unsupported`, `merge-token`, `merge-unguarded`: nothing is
  turned on — the local forge has no auto-merge, a GitHub job's own
  token, a base that does not require `workline/accepted`
  ([set up](merge-policy.md#set-up)).
- `verdict-not-kept`: `follow --verdict` with no `merge:` set, or on a
  forge without auto-merge.
