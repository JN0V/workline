---
sources: [cmd/workline, internal/hooks, internal/role/config.go, internal/doctor/variables.go]
checked: 8ccdfb74
verified: agent:claude-code
---
# The commands that set up

The commands that set up this machine and this repository, from
[usage](usage.md#commands), and what the hooks do [before a push](#before-a-push).
Step by step: [install](install.md).

## workline hooks

- `workline hooks install --global` / `uninstall --global`: takes
  `core.hooksPath` for every repository, and gives it back as it was. The
  hooks run the `commit-msg` and `pre-push` lines, then hand over to the
  hooks that were there, from any worktree of a repository.
- `workline hooks install --repo`: writes `.githooks/commit-msg` in this
  repository; remove that file to uninstall.

## workline setup

Sets up this machine, asking: the global hooks, your agent (`ai:` in your
config), the tools the roles use, each installed with the command it
shows; then prints `workline doctor`.

- Run again, it offers what is set up as the default.
- `--hooks yes|no`, `--ai <agent>` and `--install <tool,...>|all|none`
  answer a question; `--yes` takes the defaults.
- Without a terminal, every question must be answered so.

## workline init

Adopts this repository; run it again to do what is left.

- Routes `pre-push` to the committer and the documentalist in
  `.workline/config.yaml`, unless the project routes it already.
- Has the agent propose the `sources` of each doc that says nothing of
  them, in the working tree for you to review and commit
  ([at the adoption](../roles/documentalist/docs/push.md)). Without an
  agent, lists them, proposing the sources a doc's name matches (a code
  folder or file of that name), for you to review, never written.
- `--review` also adds the reviewer to the project's merge-request line
  ([ADR-0020](adr/0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md)).
- Asks whether a person is the project's Product Owner (`--human-po
  yes|no` answers with no terminal): yes sets the product owner to
  `autonomy: cautious`
  ([ADR-0038](adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)),
  unless the project set a level.

## workline doctor

Says what is set up, with the command that sets up each thing missing;
changes nothing.

- On this machine: git, the global hooks (and whether this version wrote
  them), the agent, the tools the roles use.
- In this repository: which hooks git runs there, whether the
  documentalist runs before a push, how many docs declare their sources,
  which files read as docs it does not read.
- For each language the repository holds: which tools read it (lint,
  types, security, dependencies), which it lacks, and the gate check to
  add; a warning, never an error ([languages](languages.md)).
- On GitHub: whether the default branch requires every conversation
  resolved before a merge, and how workline's findings reach pull
  requests ([#412](https://github.com/JN0V/workline/issues/412)). A rule
  it cannot read — classic branch protection, without an admin's token —
  is said. `--forge` names the forge; by default the project's `forge`,
  else GitHub when its workflows run workline.
- On GitLab (`--forge gitlab`, the project's `forge`, or a
  `.gitlab-ci.yml` naming workline): whether workline's tokens are
  scoped to the environment `workline`, read from the project's
  variables, their values never said; a token that may not list them
  (only a Maintainer's may) is said
  ([the environment](gitlab/environment.md#workline-doctor)).
- Exits 1 only on an error — the agent named cannot be called, the config
  does not load — never for a tool left out; `--json` prints every check.

## workline version

The engine running: the release it was built as, else the module version
Go recorded, else `(devel)`.

## Before a push

**The review is on the merge request, not the push**
([ADR-0011](adr/0011-the-review-is-on-the-merge-request-not-the-push.md)).
A push asks nothing, unless you set `approve-push: true` in your own
config ([ADR-0008](adr/0008-push-approval-asks-where-the-person-is.md)):

- **On a terminal**: the hook lists the commits leaving and asks
  `Push? [y]es / [N]o / [v]iew`; `v` writes a page showing each commit,
  its message and its changes (`.git/workline/push.html`) and opens it.
- Each question waits ten minutes; none answered is a no.
- **Without a terminal** (an agent's shell, an editor's button): the
  question goes to the editor window the push came from (VS Code,
  VSCodium), found among the hook's parent processes, never from what the
  pushing process sets. An input box at the top of the window, with a
  desktop notification: type `y` then Enter to push, `v` to see the
  commits; Enter alone stops.
- **With no editor**, a desktop dialog (zenity); with neither, the push is
  refused.
- `approve-push-via: [terminal, dialog, editor]` changes the order, or
  leaves channels out. `git push --no-verify` skips it.

The `pre-push` hook runs the project's `pre-push` line on the commits being
pushed, with no agent — a push never waits on one — only when
`.workline/config.yaml` routes it, since the global hook reaches every
repository:

```yaml
routing:
  events: {pre-push: [committer, documentalist]}
```

- A step that blocks stops the push.
- The docs these commits made suspect are only counted, in one line,
  never listed nor asked about: they are judged on the merge request, by
  gardening, or when you run `workline docs`
  ([ADR-0010](adr/0010-docs-judged-on-the-merge-request-and-by-gardening.md)).
- Only the docs the commits touched are read. The others, suspect from
  before, and passages written twice are said on the merge request and by
  gardening
  ([ADR-0050](adr/0050-each-act-has-its-moment-and-a-wait-budget.md)).
- A derived block behind the code is left as it is and only counted:
  gardening regenerates it on the default branch after the merge
  ([ADR-0027](adr/0027-derived-blocks-are-regenerated-on-the-default-branch.md)),
  so never edit one by hand.
- The sizes and links of the docs the commits touched are only counted.
- Its seconds, in one line: `1.2 s in all — committer 0.5 s,
  documentalist 0.7 s`; over the push's 5 s, one line more names the
  step that took the most, and the push goes on
  ([the budgets](spec/routing.md#each-steps-seconds-a-moments-budget)).
- `git push --no-verify` skips the hook.
