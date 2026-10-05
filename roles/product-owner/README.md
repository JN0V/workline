---
sources: [roles/product-owner/role.yaml, internal/builtin/productowner, internal/backlog]
checked: 810dd7d
verified: agent:claude-code
---
# Product owner

For people: what the role does and how to set it. The AI never reads this
file. All roles: [docs/roles.md](../../docs/roles.md). What a run does,
step by step: [run.md](run.md).

**Does**: keeps a project's backlog — its open issues — true to the code
and in order, between the need a person states and the result they accept
(ADR-0018). A run reads a share of the open issues against the code, and:
closes a duplicate, its original quoted; announces what the code made
obsolete and closes it a week later on silence and a second judge's yes;
names an issue's code; sets milestones and priorities; refines an issue to
`ready` (Need and Validation as drafts for a person) and asks its reporter
what is missing; splits a need too big for one issue and, as its parts
close, says on it what each delivered and which items of its
Verification are proved, for a person to accept (ADR-0029); renames a
vague title; says what an issue waits on. It also imports a roadmap file as
issues, and keeps the one way every role opens an issue.

**Does not**: write code or docs (`writes: []`: the forge's issues only),
write the Need or Validation of a person's issue as final, close as "not
planned", close a split need — a person accepts it —, undo a person's
priority, title or link, or move an
issue to `ready` while a section is missing or a draft. What sets direction
it proposes rather than does at `autonomy: cautious`.

| Event | Fired by | Does |
|---|---|---|
| `schedule` | CI gardening, with `product-owner` in the `schedule` line | reads `issues-per-run` open issues, acts and proposes |
| `import` | `workline issues import <file>` (`--apply` to open) | opens each item still to do in a file as an issue |

## Settings

Under `roles: {product-owner: {settings: …}}` in `.workline/config.yaml`
([config reference](../../docs/config.md)). Each act's `mode` is `act`
(done, up to `max` a run), `propose` (a box in the report, for a person) or
`off`. The defaults are the `normal` level; `autonomy` lays another level
under the project's own settings, which win field by field:

| Key | `normal` (default) | `cautious` | `enterprising` |
|---|---|---|---|
| `issues-per-run` | `8` | | |
| `code-lines-max` | `1500` | | |
| `ignored-runs-max` | `3` (1 to 20; 0 never pauses) | | |
| `moved-percent-max` | `20` | `10` | `30` |
| `acts.open` | act, max 30 | | |
| `acts.sources` | act, max 10 | | |
| `acts.milestone` | act, max 10 | propose | |
| `acts.order` | act, max 10 | propose | max 15 |
| `acts.close-duplicate` | act, max 3 | propose | max 5 |
| `acts.close-obsolete` | act, max 3, `days: 7`, `exempt: [pinned, security]` | `days: 14` | max 5 |
| `acts.refine` | act, max 5 | `drafts: propose` | max 10 |
| `acts.ready` | act, max 5 | | max 10 |
| `acts.ask` | act, max 3, `rounds: 3` | max 2, `rounds: 2` | max 5 |
| `acts.split` | act, max 2 | propose | max 4 |
| `acts.rename` | act, max 5 | propose | max 10 |
| `acts.depend` | act, max 5 | propose | max 10 |

An empty cell is the `normal` value. `cautious` suits a project whose
Product Owner is a person (`workline init` asks): it keeps the acts that
check facts and proposes those that set direction. A kind a person undid
goes back to `propose`, whatever the level.

```yaml
roles:
  product-owner:
    settings:
      autonomy: cautious
      acts: {close-duplicate: {mode: propose}}
```

## Outputs

- One report issue, "Backlog — product owner": what was done, what is
  proposed (a box a person of the project ticks; done at the next run),
  what waits, the first ready issue that waits on nothing — never a split
  need —, and the split needs whose parts are all closed, to accept.
- On each issue it reads: a state comment; labels (`workline:priority/N`,
  `workline:draft`, `workline:obsolete`, `workline:ready`); milestones;
  sections written; a comment to an outsider reporter; sub-issues or tasks;
  GitHub's dependencies or GitLab's `is_blocked_by` links (Premium), else a
  line `Blocked by #12.` in the body.
- On a split need: one comment, edited in place, listing its parts — open,
  closed as completed with the pull request or commit that closed it, or
  not delivered — and each item of its Verification, proved where a part
  delivered quotes it, or not proved. Accept the need by closing it.
- `--json`, `--sarif`, `--code-quality` like any role.

A person accepts drafts with the label `workline:accepted`, on one issue or
many: the next run moves them to `ready`, with no agent.

## Cost

One call a run (standard tier, context budget 40k tokens): 45k to 62k tokens
in for four issues on a real backlog, most of it the agent's own prompt;
`issues-per-run` and `code-lines-max` size it. A closing as obsolete asks a
second judge of another model. A split need's report costs no tokens:
one or two forge calls a part closed. Three runs in a row with nobody answering
(`ignored-runs-max`) pause the role: no agent is asked until a person ticks
a box, writes on the report or undoes an act.

## Without AI

Nothing is judged: each issue taken gets its state comment; accepted drafts
(`workline:accepted`), ticked boxes and an `agreed` reply are still done;
an issue whose milestone was released moves to the next; the backlog's
order and what waits are still said, and a split need's parts reported.

**Status**: beta, nightly in DomoticsCore's CI and on workline's own
issues; [status.md](status.md), each try in [tried.md](tried.md).
