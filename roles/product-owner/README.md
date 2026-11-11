---
sources: [roles/product-owner/role.yaml, internal/builtin/productowner, internal/backlog, internal/sample/acts.go, routing.default.yaml]
checked: 8538d558
verified: agent:claude-code
---
# Product owner

```mermaid
flowchart LR
  when["<b>When</b><br/>gardening (opt-in)<br/>an issue to refine<br/>importing a file"]
  role(["Product owner"])
  alone["<b>Alone</b><br/>names the code,<br/>completes, readies<br/>an evident issue"]
  proposed["<b>Proposed</b><br/>on the issue,<br/>label workline:proposed"]
  person["<b>You answer there</b><br/>accept, comment,<br/>take the label off, close"]
  when --> role
  role --> alone
  role --> proposed --> person
```

For people: what the role does and how to set it. The AI never reads this
file. All roles: [docs/roles.md](../../docs/roles.md). A run, step by
step: [docs/run.md](docs/run.md).

**Does**: keeps a project's open issues true to the code and ready to
build, between the need a person states and the result they accept
([ADR-0018](../../docs/adr/0018-the-product-owner.md)). What it proposes
is on each issue; you answer there
([ADR-0038](../../docs/adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)).

**Does not**:

- write code or docs (`writes: []`: the forge's issues only);
- write the Need or Validation of a person's issue as final;
- rewrite a Need, Example or Validation a person wrote or edited, or
  any section they deleted; their Scope it rewrites to answer the
  reviewer
  ([ADR-0060](../../docs/adr/0060-the-person-is-on-the-loop.md));
- write an issue's Verification: the tester writes it from its tests
  ([ADR-0066](../../docs/adr/0066-a-role-mirrors-a-human-role-ai-first-where-code-is-rigid.md));
- move an outsider's issue to `ready`, nor one with a draft left;
- close as "not planned", or reopen an issue;
- keep a report issue, or any memory outside the issues.

## When

| Event | Fired by | Does |
|---|---|---|
| `schedule` | CI gardening, with `product-owner` in the `schedule` line | reads `issues-per-run` open issues, completes and proposes |
| `issue` | its state, `workline:to-refine`, where the project routes the role on `issue` ([the line of states](../../docs/spec/routing.md#the-line-of-states)) | reads that issue alone, at once: completes it, readies it to its tests |
| `issue` | a blocked issue with news, or a week on ([blocked, read again](../../docs/spec/routing.md#blocked-read-again)) | sends it back, or says why it stays blocked |
| `import` | `workline issues import <file>` | opens each item still to do in a file as an issue ([the command](../../docs/commands-work.md#workline-issues-import)) |

## On each issue

- **Its one comment**, edited in place: what it did, what it proposes,
  what it wants from you. What it knows of the issue is in the comment,
  out of sight. Posted anew when it proposes something: one mail per
  proposal, holding only what it tells you.
- **The label `workline:proposed`** (GitLab: `workline::proposed`) while
  the issue waits on you: its drafts, or a proposal.
- **What waits on you** is a saved filter on that label.
- **A board**: an issue set ready moves to its ready column
  ([boards](../../docs/boards.md)).
- **The night's summary** is in the CI job's summary: done, proposed,
  left to a person, read and left incomplete, next, stuck, one linked
  line an issue, in plain words
  ([what it writes](docs/outputs.md#in-the-ci-jobs-summary)).

## How you answer

| You want | You do | Next run |
|---|---|---|
| yes | label `workline:accepted` (bulk from the list) | does what it proposed, makes the drafts yours, moves it to `ready`; takes its label off |
| revise | a plain comment | 👀 on it, rewrites its own drafts in place, replies in one line |
| correct it yourself | edit the body | what you edited is yours, never rewritten; an edited draft counts as accepted |
| not now | take `workline:proposed` off | proposes nothing more on that issue until you edit it or comment |
| it should not exist | close it | never reopens it |

- **Silence is never a yes** to close, merge or release. Past
  `proposals-max` issues waiting, the role reads only those you
  commented on, and labels no more: the next wait for room.
- **A question you leave unanswered** does not stop the issue: past a
  delay, 24 h at `normal`, the role writes its assumption on the issue
  for you to correct ([how long it waits](docs/settings.md#how-long-a-question-waits)).
- **Three rounds at most** (the draft, two revisions), then one line
  "left to a person" in the summary, and said in its own comment on the
  issue: who wrote, why it stops, what comes next.
- **Who counts**: the reporter and the project's people; their comment
  on any issue is read first; on one in refine, an answer to the
  reviewer's question too, its next run woken for it, what it settles
  written into the Scope. A `ready` that writes nothing of it is held
  once, the answer read again. A bot's, a stranger's or a "+1" comment
  never brings an issue back.
- Accepting many issues from the list is quick: read what you accept.
- **The same label on a pull request** is a yes to merge it, where the
  project sets a [merge policy](../../docs/merge-policy.md).

## Alone or proposed

At `autonomy: normal`, the default:

- **Alone**: names an issue's code; completes it — its Scope from the
  code, Need and Validation as drafts; asks its reporter what is
  missing; closes an evident duplicate; moves an evident issue to
  `ready` (Need, Validation, Scope, none a draft); labels an issue with no kind
  bug, feature or task, once ([the kind](docs/kind.md)).
- **Out of Blocked, at every level**: a blocked issue the line wakes it
  on, sent back to the state it came from or to refine; or kept
  blocked, its reason said afresh; put `workline:blocked` back to undo
  ([how](../../docs/spec/backlog-acts/refine.md#a-blocked-issue-read-again)).
  The developer's ask for a file outside the Scope, from before the
  Scope was a hint, wakes it at once: it widens the Scope, or says why
  not. So does a run too big for its role:
  it narrows the Scope, splits it, grants it `workline:size/large` when
  its need cannot be cut (proposed at `cautious`), or leaves it to a
  person. At ready, it predicts the size first
  ([split on size](../../docs/spec/backlog-acts/split.md#split-on-size)).
- **Back to refine, at every level**: a ready issue that no longer fits,
  its reason in its comment; put `workline:ready` back to undo
  ([why](../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-01-back-to-refine-needs-no-yes)).
- **Proposed** on the issue: past a cap a run; an outsider's issue to
  `ready`; an act whose only evidence is a stranger's comment.
- **Never invented**: an issue too thin, mixing topics or resting on a
  closed issue gets one plain question; one the code already does is
  said on it for you to close.
- **Written for a reader**: the person's part on top — a one-line Need,
  a real example, Given / When / Then scenarios —, the builder's below a
  line; plain words, a link for each decision cited, a closed issue said
  closed ([writing an issue](docs/writing.md),
  [what it writes](docs/outputs.md#on-each-issue)).
- **Each thing said once**: what an import or another role opened an
  issue with becomes its Need draft; your text on top is its Need; a
  draft repeating what is above it is not written
  ([how](docs/writing.md#each-thing-said-once)).
- **Undone by you** (`workline:ready` taken off or put back, a closing
  reopened…):
  that kind is proposed on that issue from then on; undone `undone-max`
  times across the issues, on every issue.

## Settings

Under `roles: {product-owner: {settings: …}}` in `.workline/config.yaml`
([config reference](../../docs/config.md)); every key, with the off-by-default
acts: [docs/settings.md](docs/settings.md).

| Key | Default | Says |
|---|---|---|
| `autonomy` | the project's `autonomy`, else `normal` | `cautious` notes and proposes; `enterprising` does more a run |
| `issues-per-run` | `8` | issues read a run |
| `proposals-max` | `10` | issues waiting on you before the role slows down |
| `undone-max` | `3` | acts of a kind undone before it is proposed everywhere |

```yaml
roles:
  product-owner:
    settings: {autonomy: cautious, issues-per-run: 5}
```

## Cost

- **One call a run** (standard tier, context budget 120k tokens,
  estimated): 45k to 62k tokens in for four issues on a real backlog, up
  to 97.6k with their code; `issues-per-run`, `code-lines-max` and
  `code-bytes-max` size it,
  and a run reads only the issues that fit the budget, the rest first at
  the next run.
- A person's answer, an accepted proposal, an undo: read with no agent.

## Without AI

Nothing is judged, but each issue gets its state, accepted proposals and
an `agreed` reply are still done, labels follow your answers, and the
summary says what is next and stuck.

**Status**: [beta](../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md), nightly on workline's own issues and in the CI of
another project of its author's; [docs/status.md](docs/status.md); each try in its pull request ([before 2026-10-13](docs/tried.md)).
