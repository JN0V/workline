---
sources: [cmd/workline, internal/role/config.go, internal/engine/engine.go, internal/hooks]
checked: 0efb09c
verified: agent:claude-code
---
# Using workline

The commands, their options and exit codes, as the engine reads them today.
What a role does is on its page ([roles.md](roles.md)); the files, the
settings and the variables, in [config.md](config.md).

## Commands

| Command | Does |
|---|---|
| `workline run-role <role> --event <event>` | runs one role: prepare, propose, judge, apply, again if `pre` left work for later (`in/more`), up to 5 rounds |
| `workline route <event>` | runs the steps routing names for the event, in order; the first that does not pass stops the line |
| `workline apply <run-dir>...` or `--line <file>` | applies runs judged with `--no-apply`, or resumes a run stopped while applying; `--line` takes the runs a `route --no-apply --json` or `issues import --json` result lists as `pending`; `--summary` as below |
| `workline review` | the reviewer reviews this branch before it is pushed (roles/reviewer, ADR-0020): `<base>..HEAD` (`--base`, default the role's `base` setting, `main`), every lens (`--lenses` names some); the rules on the comments the change adds, then each lens, each finding's quote found again, the change's findings told from those outside it, each important one judged. Prints the findings, and the path of the run's `out/review.json`, for the author's agent to fix them before pushing; `--json`, `--sarif`, `--code-quality`, `--ai`, `--forge` (where an issue for what lies outside the change goes) |
| `workline gate <name>` | runs a gate declared in `.workline/config.yaml` |
| `workline item ready <id>` | moves a work item to `ready`, once its Need, Verification, Validation and Scope are written (`--forge` reads it from the forge) |
| `workline issues` / `workline issues show <n>` / `show !<n>` | reads the local forge (`forge: local`, ADR-0016): lists the issues (`#<n>`) and merge requests (`!<n>`) kept in the clone, with their state and labels, or shows one whole, its comments after its body; writes nothing |
| `workline issues import <file> [--apply]` | moves a roadmap or backlog file, whatever its form, to the forge's issues (docs/spec/backlog-acts.md, "Importing a file"): the product owner reads it a share at a time, each item still to do opened once, its text quoted from the file; without `--apply`, only says what it would open, writes nothing, and lists its runs as pending (`to apply:`; `--json`: `pending`), which `workline apply --line <result>` opens with no agent, in the job that holds the write token (ADR-0030's map stays the judge's: run the import again to see them open). Then a map: each item of the file, by its lines and first words, to its issue — opened, already open, closed — or why it has none — done, the words quoted; not an item, why; past the cap —, and the lines no answer holds under "Not covered", which exits 2 (`--json`: the result's `coverage`). `--ai`, `--forge`, `--lines`; `--summary` as below, the map with it |
| `workline hooks install --global` / `uninstall --global` | takes `core.hooksPath` for every repository, and gives it back as it was; the hooks run the `commit-msg` and `pre-push` lines, then hand over to the hooks that were there |
| `workline hooks install --repo` | writes `.githooks/commit-msg` in this repository; remove that file to uninstall |
| `workline setup` | sets up this machine, asking: the global hooks, your agent (`ai:` in your config), the tools the roles use, each installed with the command it shows; then prints `workline doctor`. Run again, it offers what is set up as the default. `--hooks yes\|no`, `--ai <agent>` and `--install <tool,...>\|all\|none` answer a question; `--yes` takes the defaults; without a terminal, every question must be answered so |
| `workline init` | adopts this repository: routes `pre-push` to the committer and the documentalist in `.workline/config.yaml`, unless the project routes it already, and has the agent propose the `sources` of each doc that says nothing of them, in the working tree for you to review and commit (roles/documentalist/docs/push.md); without an agent, lists them, proposing the sources a doc's name matches (a code folder or file of that name), for you to review, never written. Run it again to do what is left. `--review` also adds the reviewer to the project's merge-request line (ADR-0020). It asks whether a person is the project's Product Owner — `--human-po yes\|no` answers with no terminal —: yes sets the product owner to `autonomy: cautious` (ADR-0026), unless the project set a level |
| `workline follow` | run when the default branch moves (CI's `follow` job, ADR-0034): each open merge request from `workline/<role>/release` (ADR-0017) whose base moved under it is rebuilt on the base's new tip — its change redone file by file, a doc's front matter key by key, the base's value kept where both changed a key — and force-pushed, the merge request kept (`rebuilt`); one on the tip already is left alone. One holding a commit without the role's `Workline-Role` trailer, or a merge, or that did not leave from the base's history (its merge request goes into another branch), is never rebuilt (`not-rebuilt`, a warning); one whose change no longer applies is left for a person (`no-longer-applies`, exit 2). `--base` (default: the branch checked out), `--forge`, `--repo`, `--json`; no agent |
| `workline version` | the engine running: the release it was built as, else the module version Go recorded, else `(devel)` |
| `workline docs` | has the docs made suspect since they were last judged — `refs/workline/docs-judged`, else the last release (the highest version tag merged, `release.tags`), else every commit; `--since <rev>` to choose — judged by your agent, pushed or not (ADR-0010), then asks you, doc by doc, to keep or drop each change (`v` shows it); those kept go in one `docs:` commit, alone (ADR-0007). Without a terminal, the changes stay in the working tree for a person, who reviews them the same way with `workline docs --review`. Docs with changes not committed are refused, so the agent's are reviewed alone. Once the run passed with nothing left to review, the ref moves to HEAD |
| `workline sample` | the weekly sample of the docs the documentalist vouched for (ADR-0014, step 4; ADR-0015): of the docs whose `checked` its commits moved (`Workline-Role: documentalist`, or `verified: agent:documentalist` newly set) in the commits reaching the branch in the last whole ISO week (`--week 2026-W40`; `--since <rev>` for the commits after one), one in ten, rounded up, drawn the same on a rerun of the week; each read whole against its sources at the commit its `checked` names, by the judge (`--judge`, else `WORKLINE_JUDGE`, else the documentalist's `sample.judge`), never the model the commit's `Workline-Model` names. Every quote the judge gives is checked; a comment is no evidence. Writes nothing; `--out <file>` keeps the result, and `workline sample --apply <file> --forge <forge>` (default: the project's `forge`), with no agent, comments the week on one tracking issue, labels it `documentalist-step-0` and opens a merge request putting back a `checked` found false; with no forge it writes nothing and says so. On a project with the product owner's report, `--apply` also draws one in ten of the acts it did alone that week, from its record, onto the issue "workline: the weekly sample of the product owner's acts", each with its day, level and whether a person undid it, and the level they suggest (ADR-0033). Exits 2 when a doc is left for a person; `--summary` as below |
| `workline doctor` | says what is set up on this machine (git, the global hooks, the agent, the tools the roles use) and in this repository (which hooks git runs there, whether the documentalist runs before a push, how many docs declare their sources, which files read as docs it does not read), with the command that sets up each thing missing; changes nothing. It exits 1 only on an error — the agent named cannot be called, the config does not load — never for a tool left out; `--json` prints every check |

Options of `run-role` and `route`:

| Option | |
|---|---|
| `--repo <dir>` | the repository (default: here) |
| `--ai <agent>` | `none`, `claude`, or `claude:<model>@<effort>` to force a model (an alias or an exact id), an effort, or both, whatever the role's tier asks: `claude:opus`, `claude:@high`; or `cmd:<command>`, any other agent (below); default: `WORKLINE_AI` for the hook, else the project's `ai`, else yours, else none |
| `--input name=value` | an input for the role (`route`: for every step), e.g. `range=<base>..<head>` for the committer on a merge request |
| `--input-file name=path` | `run-role` only: an input read from a file, written back by the intention that targets it (the hook's message file) |
| `--forge <forge>` | `github` (needs `gh`), `gitlab` (its API, with `GITLAB_TOKEN`, else glab's token if glab is set up, else CI's job token; the instance and project from CI, else `GITLAB_HOST` and the remote), `local` (kept in the clone, never pushed: below), `cmd:<command>` (another forge, plugged by a command: docs/spec/forge-command.md), `none`; default: the project's `forge` |
| `--target issue:<n>` / `merge-request:<n>` | where comments and labels go |
| `--branch <name>` | the branch the merge request `--target` names comes from, for a job that does not ask the forge (CI's judging job, which holds no token); default: the forge says. A release tool's branch (`release.branches`) has the merge request held as the release (ADR-0017) |
| `--scope <glob>` | a path the task is about, repeatable; a patch outside it is refused |
| `--no-apply` | judge, then stop; apply later, in a job that holds the write token |
| `--push-to-merge-request` | with a forge and `--target merge-request:<n>`: what the patches write is committed to that merge request's branch and pushed, without force; from a fork, or when the branch moved on, the diff goes in one comment instead. With `local`, committed on the local branch, nothing pushed. On a release tool's merge request, which the tool rewrites, it goes instead to a merge request of its own, `workline/<role>/release`, into the release's base, and the release waits until it is merged (ADR-0017) |
| `--open-merge-request` | with a forge: what the patches write goes on a branch `workline/<role>/<task>`, pushed, with a merge request opened or updated for it; the role is told how many of its merge requests are open (ADR-0006). With `local`, the branch stays in the clone |
| `--roles <dir>` | a folder of roles used instead of the shipped ones |
| `--sarif <file>` / `--code-quality <file>` | also write the findings as SARIF (GitHub code scanning) or a GitLab Code Quality report; `route`: every step's (docs/spec/role-outcome.md) |
| `--summary <file>` | also add to the file what a CI job's page shows: the verdict, each step's verdict and findings, the agent's calls and notes, what was applied and what waits to be (ADR-0035). Markdown (GitHub's `$GITHUB_STEP_SUMMARY`), or HTML for a file named `.html`; added to, never overwritten, so a judging then an applying job write one page. Repeatable; `apply`, `sample` and `issues import` take it too |
| `--json` | print the result as JSON: status, summary, findings, the agent's notes, each agent call (agent, tier, effort, the exact model that answered, tokens in, cached and out, cost, seconds), and for a line its steps and pending runs |

## Another agent

`cmd:<command>` makes any command the agent: another provider's CLI, or a
wrapper around your own. It runs with `sh -c`, outside the repository:

- its input is the prompt, the role's persona first;
- `WORKLINE_TIER` and `WORKLINE_EFFORT` say what the role asks (`light`,
  `standard`, `frontier`; `none` to `max`), for it to pick its model;
- its output is the answer: the YAML list of proposals, a code fence tolerated;
- it may write what answered, in YAML, to the file `WORKLINE_CALL` names:
  `model`, `tokens-in`, `tokens-cached`, `tokens-out`, `cost-usd`;
- a non-zero exit, or no answer within the role's `model.timeout`, ends the
  run as `blocked-external`, as a used-up quota does.

Claude Code, run this way (the command the judge's trial used):

```sh
--ai 'cmd:echo "model: haiku" > "$WORKLINE_CALL"; claude -p --model haiku --tools ""'
```

## Exit codes

| Code | Status |
|---|---|
| 0 | `pass` (findings may still be shown) |
| 1 | `block`, or an error |
| 2 | `human`: a person must decide |
| 3 | `blocked-external`: something outside failed (agent quota, forge, a repository) — never a verdict on the work |
| 64 | the command was misused |

`workline gate` returns 0 or 1. An unknown option exits 2, from Go's option
parser, which a script cannot tell from `human` (#101).

## Before a push

**The review is on the merge request, not the push** (ADR-0011). A push
asks nothing, unless you set `approve-push: true` in your own config: then
the hook lists the commits leaving, and asks `Push? [y]es / [N]o / [v]iew`
on the terminal; `v` writes a page showing each commit, its message and
its changes (`.git/workline/push.html`) and opens it. Each question waits ten
minutes; none answered is a no. Without a terminal — an agent's shell, an
editor's button — the question goes to the editor window the push came from
(VS Code, VSCodium), found among the hook's parent processes, never from
what the pushing process sets: an input box at the top of the window, with a
desktop notification; type `y` then Enter to push, `v` to see the commits;
Enter alone stops. With no editor, a desktop dialog (zenity); with neither,
the push is refused. `approve-push-via: [terminal, dialog, editor]` changes
the order, or leaves channels out (ADR-0008). `git push --no-verify` skips
it.

The `pre-push` hook runs the project's `pre-push` line on the commits being
pushed, with no agent — a push never waits on one — only when
`.workline/config.yaml` routes it, since the global hook reaches every
repository:

```yaml
routing:
  events: {pre-push: [committer, documentalist]}
```

A step that blocks stops the push. The docs suspect are only counted, in one
line — those these commits made so, those before — never listed nor asked
about: they are judged on the merge request, by gardening, or when you run
`workline docs` (ADR-0010). A derived block behind the code is left as it
is and only counted: gardening regenerates it on the default branch after the
merge (ADR-0027), so never edit one by hand. Sizes and links the line reports
on every run are only counted. `git push
--no-verify` skips the hook.
