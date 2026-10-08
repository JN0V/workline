---
sources: [cmd/workline, internal/setup, internal/doctor, internal/hooks]
checked: 2b82821
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

Without AI it blocks and says why; with an agent, the message is rewritten
(`fix: fix the thing`, `Refs: AC-3`) and printed. `WORKLINE_AI=none git
commit …` turns the agent off for one commit; an empty `.workline/off`
file turns workline off for a repository.

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
