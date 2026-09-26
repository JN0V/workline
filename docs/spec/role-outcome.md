---
type: reference
sources: [internal/engine, internal/intent, internal/role, internal/agent]
verified: agent:documentalist
checked: 0554e32
status: draft
---
# Role outcome — verdict and intentions

Part of the [role contract](role-contract.md): what a run ends with.

## Verdict

```yaml
status: pass | block | human | blocked-external
summary: One line a human reads first.
findings:
  - rule: internal-code
    where: commit a1b2c3d, subject
    message: "'AC-3' means nothing outside the project; say what changed."
```

Every run ends with a status; there is no run without one. `blocked-external`
means something outside the role failed — an expired token, a used-up quota, a
forge that did not answer. It is not the role's verdict on the work, and it must
never be read as one.

Findings map to SARIF, so the same verdict can be posted on GitHub or GitLab:
`--sarif` writes them for code scanning, `--code-quality` as GitLab's Code
Quality report, rule `<role>/<rule>`, on the file and line `where` names — at
the heading of its `#anchor`, else line 1. A finding that blocks is an
`error` (`blocker`), one lowered by `enforce` a `warning` (`minor`), one from
a run that passed a `note` (`info`). One whose `where` names no file — a
commit's subject, a folder, a gate's check — is left out: both formats need a
file, and `--json` still has it.

### Warn before block

A new rule is not trusted on day one. A project sets how hard each rule bites:

```yaml
roles:
  committer:
    enforce:
      internal-code: warn        # block (default) | warn | off
```

The engine recomputes the status from the findings: a `block` verdict whose
blocking findings are all set to `warn` becomes a `pass` that shows them. Rules
start in `warn`, and move to `block` once the false positives are gone.

A rule can also be adopted on existing code without fixing all past violations
first: the violations present today are frozen in a baseline, with a date. Only
new violations block; once the date passes, the frozen ones block too. *Not
built yet.*

## Intentions

The agent never acts. It states what it wants done, and the engine does it — or
refuses. The catalogue is closed and belongs to the engine:

| Intention | Effect | Applied |
|---|---|---|
| `commit-message` | replace the message being written | locally |
| `patch` | a unified diff, applied as its lines read (agents get hunk counts wrong), or `{file, content}` to replace one file; limited to `duties.writes` | locally or as a merge request |
| `comment` | a comment on the issue or merge request | forge |
| `label` | add or remove labels | forge |
| `issue` | report a problem found outside the task, without fixing it | forge, or `.workline/issues/` without one |
| `release` | tag a commit with a version and publish release notes | git and forge |
| `handoff` | name the next role and why | engine |
| `note` | a message for the human, nothing else | report |

A role that needs another intention asks for it to be added here; it does not
invent one. Intentions that reach the forge are applied by a separate step that
holds the write token and no AI key.

## Stay on the task

Working on one thing often reveals a bug somewhere else — in a module the same
people own, but with another responsibility. Fixing it on the spot mixes two
changes in one review, one commit and one release note, and the second fix gets
none of the attention it deserves. So a role reports it and moves on.

- A run can carry a **scope**: the paths or modules the task is about, taken
  from the issue, the spec or the command line (`--scope`; only the command
  line today).
- A `patch` touching anything outside the scope is refused at apply, like a
  write outside `duties.writes`.
- What the agent found there becomes an `issue` intention: what is wrong, where,
  how it was noticed, and the task it was found during. The applier searches
  open issues first, and comments on a matching one instead of opening a
  duplicate.
- The fix happens later, as its own task, through the normal line.

The committer applies the same rule to messages: several messages proposed for
one commit are refused, with a note to split the commit instead.

## No AI

When `--ai none`, or when the agent fails, the run still completes: `post` runs
without intentions and `without-ai` states what that means. *Not read yet:
each role's `post` decides on its own today. And when the agent could not be
reached and `post` does not pass, the run ends `blocked-external`, not `block`.*

- `block` — the check failed and only a human can fix it; the verdict says how.
- `report` — the work is left as a note or a TODO for a human.
- `pass` — the AI would only have improved something already acceptable.
