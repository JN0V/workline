---
sources: [internal/doctor, internal/hooks, cmd/workline, internal/builtin/documentalist/documentalist.go, internal/builtin/productowner/productowner.go, internal/forge/local.go]
checked: 08a4bfa
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
- **A repository sets its own `core.hooksPath`** (husky, lefthook…): git
  takes it over the global one. `workline doctor` in the repository says
  `hooks-bypassed`, naming the path, when its `commit-msg` (and `pre-push`,
  when routed) does not hand over to workline; `hooks-path` when it does.
  A hook hands over when it is executable and runs `workline hook <hook>`
  (or workline's global hook) as a command before a top-level `exit`: a
  comment or an `echo` does not count.
  The fix it prints:
  - the path is `.githooks` or `.git/hooks`: unset it (`git config --local
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

## Docs suspect

`docs suspect: N` at the push counts the docs whose `sources` the pushed
commits touched; it never asks and never blocks. They are judged on the
merge request, by gardening, or when you run `workline docs`. To clear one
by hand: read it against its sources, fix it, and set its `checked` to the
commit you read them at. A doc that names no `sources` is never suspect:
`workline init` proposes them.

On a release, a doc made suspect since the last release holds it (`due`,
`suspect`) until judged.

`shallow-clone`: the clone lacks history the documentalist needs — the
last release, or the commit a doc's `checked` names, so whether its
sources changed cannot be told. It holds the release, and any run whose
task is such a doc; `workline doctor` warns too. Fetch the whole history:
`git fetch --unshallow`; in CI, GitHub's `fetch-depth: 0`, GitLab's
`GIT_DEPTH: 0` (workline's templates set both).

## Gardening paused

`gardening-paused`: as many of the documentalist's merge requests wait for
review as `max-open-merge-requests` allows (5). Review or close them;
gardening proposes again when fewer wait.

## The product owner paused, or does nothing

- `proposals-waiting`: more issues wait on your answer than
  `proposals-max` (10): it reads only those you commented on. Answer
  some — `workline:accepted`, a comment, the label taken off.
- An issue you commented on is read first, set aside or not: set aside,
  your comment brings it back, completed as any other unless it says "not
  now". A comment written before you took `workline:proposed` off goes
  with that "not now", and does not bring it back.
- `setting-gone`: `ignored-runs-max` is no longer read; take it off.
- An issue gets nothing: the summary's "read, left incomplete" says why —
  no file found for its Scope (link or name its files, or write the Scope
  yourself), or nothing drafted (read again once it changes or someone
  comments).
- It runs only where routed: `schedule: [documentalist, product-owner]`.
- An issue refined never goes ready, the reviewer after it in the line
  ([#128](https://github.com/JN0V/workline/issues/128)): `spec-not-reviewed` — the reviewer reads one issue a run, none
  without an agent or with `forge-writes: false`; `spec-findings-open` —
  read the reviewer's comment on the issue; `spec-rounds-spent` — a
  person decides: `workline:accepted`, or `workline:ready` by hand.
- A kind of act stays proposed on an issue where a person undid one, and
  on every issue once undone `undone-max` times (`demoted`).
- A file the issues were imported from, archived since, flags them each
  time it is edited: name it in `archived`
  ([ADR-0032](adr/0032-a-changed-need-flags-the-issues-built-on-it.md)).
- `autonomy: cautious` notes and proposes: see the issues bearing
  `workline:proposed`.
- Nothing said stuck though issues wait: `stuck-unknown` says
  which issues the forge gives no day for — the local forge keeps none, a
  plugged forge may refuse `trail` or give comments without `created`.
- `lines-unread`: an issue imported from a file, whose lines cannot be
  followed — the commit it was opened at gone after a force-push, or
  beyond a shallow clone: fetch the whole history (`fetch-depth: 0`).

## Token caps and the agent

- `ai-max-tokens` (documentalist, reviewer): the run spent its cap; the rest
  waits for the next run. The reviewer leaves the commits unrecorded
  (`review-not-whole`, `lens-failed`) and says what each call used. `ai-max-calls`, `parts-max-per-run`, the product owner's
  `issues-per-run` and `code-lines-max`, the reviewer's `diff-lines-max`,
  `code-lines-max` and `tests-lines-max` size each run (each role's page).
- `blocked-external` (exit 3): the agent's quota, a timeout, the forge or a
  repository failed. Never a verdict: run it again later. The CI templates
  turn it into a warning, the docs listed for a person.
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
