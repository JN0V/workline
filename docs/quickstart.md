---
sources: [cmd/workline, internal/setup, internal/doctor, internal/hooks]
checked: 8ccdfb74
verified: agent:claude-code
---
# Quickstart

From nothing to workline checking your commits, your docs and your
branches, on your machine, in ten minutes. No AI is needed for any step;
with one, refused messages are rewritten and docs fixed.

## 1. Install and set up the machine

Install the binary ([install](install.md#1-the-binary)), then:

```sh
workline setup     # the global hooks, your agent, the tools the roles use; asks first
workline doctor    # what is set up, what is missing, and the command for each
workline --help    # every command; `workline <command> --help`, its options
```

`workline setup --yes` takes the defaults; `--ai none` keeps every AI off.
The agent is your choice: none, Claude Code (`claude`, a login once), or
any command as `cmd:` ([the choice](install.md#3-the-agent-your-choice)).

## 2. Try the committer

In any repository:

```sh
git commit --allow-empty -m "AC-3 fix the thing"
```

- **With an agent**, it is rewritten, 30 s at most (`fix: fix the thing`,
  `Refs: AC-3`), and printed
  ([ADR-0050](adr/0050-each-act-has-its-moment-and-a-wait-budget.md)).
- **Without AI**, or with `rewrite: false` in the committer's settings,
  the project's or your own, it blocks in about a second and says why,
  your message kept.
- An empty `.workline/off` file turns workline off for a repository.

## 3. Adopt a repository

```sh
cd your-repo
workline init            # routes pre-push; proposes each doc's sources
git diff                 # review the `sources` headers it wrote, then commit them
```

With an agent, `init` writes the proposed headers in the working tree;
without, it lists them, for you to write. Commit `.workline/config.yaml`
too.

From now on, a push counts the docs its commits made suspect, in one line,
without an agent. Then, when you choose:

```sh
workline docs            # the agent judges them; you keep or drop each change
workline review          # the reviewer reads this branch before you push
```

`workline init --review` also puts the reviewer on each merge request;
`--human-po yes` says a person is the Product Owner.

### Try each role once, on a real repository

To check the documentalist, the reviewer and the product owner work on a
repository (at work, say), after `setup`, `doctor` and `init --review`:

```sh
workline doctor          # lists nothing missing; each item comes with its command
workline docs            # documentalist: judges the docs changed since their last commit
workline review          # reviewer: reads the branch against the default branch
workline run-role product-owner --event issue --target issue:<n> --ai claude --forge github
```

- The repository needs a doc changed since its last commit, a branch with a
  diff against the default branch, and one open issue `<n>`.
- `--forge gitlab` for GitLab; `--ai` the agent you chose in `setup`.
  Run without `--no-apply`, the product owner writes its comment or refined
  section on the issue itself.
- `workline docs` ends with the agent judging the suspect docs, and lists
  the changes to keep or drop.
- `workline review` prints the findings; when there are none it still
  prints a short output: copy it into your record.
- Note the issue number and the text the product owner wrote.
- Each gap met on the way — a step that needed help — goes in an issue that
  quotes the command and the output seen.

## 4. Add CI

On GitHub or GitLab, copy the templates ([ci.md](ci.md)): each merge
request judged, gardening at night, a fix committed for you to review. On
a pipeline the templates do not fit — another trigger, an internal tool, a
plain script: [triggers.md](triggers.md).

## Next

- [Install](install.md) — your machine and CI, in full.
- [Concepts](concepts.md) — the words used everywhere.
- [Roles](roles.md) — what each role does, its settings.
- [Configuration](config.md) — `.workline/config.yaml`, your config, variables.
- [Troubleshooting](troubleshooting.md) — when something does not run.
