---
type: reference
sources: [internal/engine, internal/intent, internal/role, internal/agent, internal/verdict, internal/report]
verified: agent:claude-code
checked: 13d7bca
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
final: false   # true: the block is not the agent's answer's, so it is not asked again
```

Every run ends with a status; there is no run without one. `blocked-external`
means something outside the role failed — an expired token, a used-up quota, a
forge that did not answer. It is not the role's verdict on the work, and it must
never be read as one.

A `block` from `post` after the agent answered has the agent asked again,
with the findings as the reasons (docs/spec/model-grid.md), unless `final`:
`post` says the block does not come from the answer — a release held by
docs the answer was not about, a check that could not run — and asking
again would change nothing, but grow the question with every finding.

Findings map to SARIF, so the same verdict can be posted on GitHub or GitLab:
`--sarif` writes them for code scanning, `--code-quality` as GitLab's Code
Quality report, rule `<role>/<rule>`, on the file and line `where` names — at
the heading of its `#anchor`, else line 1. A finding that blocks is an
`error` (`blocker`), one lowered by `enforce` a `warning` (`minor`), one from
a run that passed, or of level `question` (a reviewer's question for a
person), a `note` (`info`). One whose `where` names no file — a
commit's subject, a folder, a gate's check — is left out: both formats need a
file, and `--json` still has it, as does `--summary`, the summary a CI job's
page shows (ADR-0035).

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
| `patch` | a unified diff, applied as its lines read (agents get hunk counts wrong), which may create a file (`--- /dev/null`) or delete one (`+++ /dev/null`), or `{file, content}` to replace one file; limited to `duties.writes` | locally, as a merge request of its own (`--open-merge-request`), or as a commit on the merge request run on (`--push-to-merge-request`) |
| `comment` | a comment on the issue or merge request; `{body, sticky: key}` keeps one comment, edited on each run (with `update-only: true`, never opened); `issue: n` puts it on that issue instead | forge |
| `label` | add or remove labels | forge |
| `issue` | report a problem without fixing it: found outside the task, or one only people can settle (code disagreeing with a spec); `{title, body, sticky: true}` keeps one issue on the forge, its body rewritten on each run (with `update-only: true`, never opened); `{title, body, at: {path, text}}` opens one once, through the one way every role shares (docs/spec/backlog-acts.md, "Opening issues"): the engine keys it from the line `at` quotes (else its title), looks in the issues open and closed, opens none for a subject held, at most `issues-max` a run, labelled `needs-triage`, the role named, with the product owner's state comment naming the file and the commit (ADR-0018) | forge (`forge: local` keeps it in the clone); refused without one |
| `close` | close an issue as a duplicate (`duplicate-of`) or obsolete, its evidence quoted from a file or an issue; never as not planned; done, proposed or dropped by the engine — obsolete announced on the issue first, closed by the engine days later on silence and a second judge's yes (docs/spec/backlog-acts.md, ADR-0024) | forge |
| `keep` | keep an issue announced obsolete open: its label off, its evidence recorded, never announced again on it (docs/spec/backlog-acts.md) | forge |
| `open` | open an issue from a file's item, its text quoted, once (docs/spec/backlog-acts.md, "Importing a file") | forge |
| `skip` | when importing, an item of the file not opened, by its lines, and why: `done` (the words that say so quoted), `held` (the issue that holds it) or `not-item` (why); the engine checks the reason and puts it in the import's map (docs/spec/backlog-acts.md, "The map"; ADR-0030) | never: kept in `out/skips.yaml`, read by the import |
| `milestone` | put an issue in a release's milestone, created if none is open (docs/spec/backlog-acts.md) | forge |
| `order` | set an issue's priority, one label of `workline:priority/1` (the most pressing) to `/4`; a priority a person set is kept (docs/spec/backlog-acts.md, "Ordering") | forge |
| `sources` | name the code an issue is about, a line of it quoted; the issue is read again with it (docs/spec/backlog-acts.md) | forge |
| `refine` | add the sections an issue lacks — Scope and its files, Verification, Need and Validation as drafts — never rewriting one there; on an outsider's issue, proposed to its reporter in a comment until a person agrees (docs/spec/backlog-acts.md, "Refining to ready") | forge |
| `ready` | move an issue to `ready`, once the engine finds its four sections there and none a draft — or the drafts accepted with the label `workline:accepted`; an outsider's is proposed, unless accepted | forge |
| `unready` | move a ready issue back to refine (`workline:ready` off, `workline:to-refine` on), the issue told why; always proposed, done once a person ticks it (docs/spec/backlog-acts.md, "Refining to ready"; ADR-0032) | forge |
| `ask` | ask an issue's reporter what is missing; again only after an answer, never the same question, three rounds then the report (ADR-0021) | forge |
| `split` | break an issue too big to be one need into 2 to 6 issues, each with its four sections, opened through the one way and linked to it — a sub-issue, a GitLab task, or a task list in its body; never split twice (docs/spec/backlog-acts.md, "Splitting") | forge |
| `rename` | set an issue's title; a title a person set after the role's is kept (docs/spec/backlog-acts.md, "Renaming") | forge |
| `depend` | name the open issues an issue waits on (`blocked-by`): the forge's own relation, or a line in its body; never a cycle, a person's link kept (docs/spec/backlog-acts.md, "What an issue waits on"; ADR-0028) | forge |
| `undepend` | take off a link the role set (`blocked-by`), its blocker still open: always proposed to a person; a closed blocker's link the engine takes off itself, a person's never (ADR-0028) | forge |
| `handoff` | name the next role and why | engine |
| `note` | a message for the human, nothing else | report |
| `finding` | a lens's answer (role contract, "In parts"): `{lens, severity, title, why, cause: {path, quote}, symptom, fix, claim}`, `severity` `important` or `nit`, `lens` the one that found it, read when the lenses are asked together; `claim` the author's words a lens that `cites: claim` quotes; a cause's `path` `#4` for the issue the change closes; the engine finds each quote again, and tells the change's from the rest by where the cause lies (ADR-0020) | never: read by the reviewer's `pre` |
| `claim` | a part's answer about a passage of the question (role contract, "In parts"): `{lines, status, quote, source: {path, lines, quote}, why}`, `status` one of `contradicted`, `partial`, `supported`; or, beside a `patch`, why it takes words out | never: read by `pre`, which puts the parts' claims together, or by `post` |

A role that needs another intention asks for it to be added here; it does not
invent one. Intentions that reach the forge can be applied by a separate step
(`--no-apply`, then `workline apply`) that holds the write token and no AI key.

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

When `--ai none`, or when the agent fails, the run still completes: `post`
runs on the role's own proposals alone, and `without-ai` states what that
means. *Not read yet: each role's `post` decides on its own today. And when
the agent could not be reached and `post` does not pass, the run ends
`blocked-external`, not `block`.*

- `block` — the check failed and only a human can fix it; the verdict says how.
- `report` — the work is left as a note or a TODO for a human.
- `pass` — the AI would only have improved something already acceptable.
