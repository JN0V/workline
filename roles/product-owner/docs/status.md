# Product owner — where it stands (2026-10-18)

**In one line:** refocused by
[ADR-0038](../../../docs/adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md):
it completes issues (sources, drafts, a question to the reporter), closes
an evident duplicate and readies an evident issue alone, proposes the
rest on each issue (`workline:proposed`, one comment edited in place),
and reads a person's answer there. Milestones, priorities, splits,
renames, links, obsolete issues and changed needs are off by default,
their code kept; its acts done alone are drawn by the weekly sample of
every role's. Tried on both sandboxes; a trial on
workline and another project of its author's is running. Beta.

Each try: up to 2026-10-13 in [tried.md](tried.md), an archive (the first
nights of the trial, 2026-10-08 to 10-12, among them); since, in its pull
request's `## Tried`
([ADR-0043](../../../docs/adr/0043-a-try-is-told-in-its-pull-request.md)).

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: met once
   [ADR-0039](../../../docs/adr/0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md)
   is built. Missing holds only that decision and
   [ADR-0040](../../../docs/adr/0040-priority-and-effort-are-the-forges-the-product-owner-estimates-them.md),
   next to build; the rest are tries on real use, or parked as issues.
2. **The main issue validated on real use**: open. The trial of
   [ADR-0038](../../../docs/adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)
   measures the proposals on each issue — accepted unchanged, edited,
   revised, set aside, ignored —; the maintainer validates it from them.
3. **AI verdicts measured on the acts on by default**: in part. An evident
   duplicate: `keeps-a-planted-backlog`, 5 in 5 on Sonnet. Left: refine
   (drafts accepted unchanged in the trial; rated blind, the new drafts
   won 6 of 6 but were accepted as is once,
   [tried](tried.md#2026-10-09--issues-a-person-reads-old-and-new-blind-sonnet)),
   ready (moved alone and not undone), and the kind once
   [ADR-0039](../../../docs/adr/0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md)
   builds it (9 of 10 right, labels hidden).

## Built

Cases in
[tests/conformance/cases/product-owner](../../../tests/conformance/cases/product-owner)
and [backlog](../../../tests/conformance/cases/backlog); its evaluation
in [tests/evaluation/cases/product-owner](../../../tests/evaluation/cases/product-owner).

- **Proposals on each issue**, a person's answer read there: `proposal-*`,
  `accepted-*`, `revise-a-persons-comment`, `who-counts-and-rounds`,
  `proposed-taken-off-sets-aside`, `report-moves-to-the-issues`,
  `no-quick-action-in-the-roles-text`, `scoped-labels-on-gitlab`,
  `undone-proposed-on-that-issue-only`, `outsider-evidence-only-proposed`,
  `deleted-*`.
- **What the trial's first nights showed**: `scope-files-taken-from-its-text`,
  `linked-file-is-code-named`, `folders-listed-when-no-code-named`,
  `edited-draft-is-the-persons`, `labels-follow-the-state`,
  `obsolete-said-when-off`, `draft-links-decisions-and-says-closed`,
  `summary-in-plain-words`.
- **What the trial's first week showed**: a proposal mails once,
  `proposal-on-a-*-posted-*`, `proposal-update-posts-nothing`; a run
  reads what fits its budget, `reads-what-fits-the-budget`,
  `over-budget-*`; no more labelled than `proposals-max`,
  `proposals-labelled-within-the-cap`, `proposals-held-labelled-later`;
  a question past the night's limit asked the night after,
  `capped-ask-*`; workline's own issues and bots left unasked,
  `workline-issues-left-alone`, `bot-reporter-asked-nothing`.
- **A person's word first**: `set-aside-*`, `brought-back-*`,
  `left-with-nothing-read-again-once`, `task-says-whose-sections`.
- **Issues a person reads**, the person's part on top:
  `refine-writes-the-persons-part-first`,
  `draft-above-the-line-stays-a-draft`, routing's
  `item-ready-heading-twice`.
- **What no person wrote, in plain words**: `description-*`,
  `level-cautious-proposes-example-and-steps`, `question-written-once`,
  `closings-capped`.
- **Refining, the reporter asked**: `refine-*`, `answer-read-*`,
  `ask-the-reporter-once`, `same-question-*`, `asked-before-*`,
  `rounds-capped-then-person`, `ready-checked-by-the-engine`; an
  outsider's issue proposed to its reporter: `outsider-*`,
  `*-reply-agrees`, `stranger-reply-not-agreement`, `agreed-*`.
- **A duplicate closed**, its original quoted and given whole; a closing a
  person undid found: `close-a-duplicate`, `original-given-whole`,
  `wrong-closing-*`.
- **Obsolete, announced then closed**: `obsolete-*`.
- **A backlog read a share a run**, read again when its code or a person
  moved: `reads-a-*`, `read-again-when-*`, `first-run-reads-nothing-new`.
- **The code an issue is about named**: `code-named-*`, `sources-named`,
  `sources-a-folder`.
- **Ordering and milestones**, by due date: `order-*`, `milestone-set`,
  `slipped-*`, `summary-next-milestones-by-due-date`.
- **Where priority and effort are kept**, off by default
  ([ADR-0040](../../../docs/adr/0040-priority-and-effort-are-the-forges-the-product-owner-estimates-them.md),
  step 2): `priority-*`.
- **A backlog file imported**, mapped item by item, judged then applied
  apart: backlog's `import-*`.
- **One way for every role to open an issue**: backlog's `issue-*`,
  `role-finding-read-as-draft`.
- **Splitting and renaming**: `split-*`, `rename-*`.
- **GitLab as GitHub**: forge's `gitlab-*`, `comment-author-read`.
- **Autonomy levels**, an act undone demoting its kind: `level-*`,
  `undo-*`, `undone-max-demotes-the-kind`,
  `demoted-kind-survives-level-change`,
  `explicit-act-setting-overrides-level`.
- **A parent and its parts**, a test named as proof read from the code:
  `parent-*`, forge's `gitlab-parent-closers`.
- **What an issue waits on**, a link taken off once its reason is gone:
  `depend-*`, `undepend-*`, `persons-link-kept`,
  `split-child-waits-on-sibling`, `task-lists-blocked-after-blocker`,
  backlog's `*blocker*`, `cycle-reported-not-looped`.
- **Next and stuck**, in the job's summary: `summary-*`, `report-stuck-*`.
- **A changed need**: `changed-*`, `unchanged-need-flags-nothing`,
  `unready-accepted-moves-back`, `archived-source-flags-nothing`.
- **The weekly sample of its acts**, folded into every role's, each act
  keeping what its run read — `at`, `model`, `body`
  ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md)):
  `acts-recorded-with-day-and-level`, `acts-record-what-was-read`,
  `sample-*`, `weekly-sample-said-gone`.
- **A spec read before ready**: `ready-held-while-a-spec-finding-is-open`,
  `ready-released-once-the-spec-is-answered`,
  `refine-answers-the-spec-findings`.
- **A hidden record that does not read is said**:
  `broken-state-writes-nothing`,
  `proposal-unread-agreement-writes-nothing`,
  `ready-held-while-the-spec-record-does-not-read`.
- **An evident duplicate left right**: the evaluation's
  `keeps-a-planted-backlog`.

## Missing, in the order to build it

1. **Issue sections and kinds, proposed**
   ([ADR-0039](../../../docs/adr/0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md),
   [examples](proposed-headings.md)): established section names, each
   thing said once, a bug readied only with its steps, the kind label set
   by the role. Built after the trial of
   [ADR-0038](../../../docs/adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md).
   No issue yet.
2. **Priority and effort where the forge keeps them, proposed**
   ([ADR-0040](../../../docs/adr/0040-priority-and-effort-are-the-forges-the-product-owner-estimates-them.md),
   [examples](priority-and-effort.md),
   [#475](https://github.com/JN0V/workline/issues/475)): six steps,
   conformance first, then a live try on a new jn0v-lab repository.
   - **Step 1 built**: the levels read by name through one table
     (`internal/scale`), no behaviour change.
   - **Step 2 built, off by default**: where each axis is kept, found when
     the settings ask (`priority`, `effort`); the forge's words created,
     ranked on GitLab; a value in two places writes nothing. Tried on the
     GitLab sandbox, judged and not applied (its pull request's Tried).
     The reviewer still orders the specs it reads by workline's own
     labels: it does not read the role's settings.
   - **Next**: step 3, the effort and the grid.
3. **Tries on real use**, nothing to build (no issue yet):
   - **answers on the issue**: the trial, measured (To leave beta, 2); a
     real agent's revision after a comment; an outsider's and a bot's
     comment, live; the night's summary, its plain wording and a body's
     footer kept out, live; the prompt's writing rules read by the
     maintainer on a real night;
   - **a person's word first**: on GitLab; the `ready` wording with a real
     agent;
   - **issues a person reads**: an Example written by a real agent; on
     GitLab;
   - **what no person wrote**, rewritten by a real agent, live;
   - **refining**: a real outsider's issue, a reporter's own `agreed`, the
     gardening template on GitLab with a release holding this;
   - **autonomy**: a real agent's rename undone, `ready` taken off and a
     split's child closed, live; `workline init`'s question on a terminal;
   - **sources and imports**: a file the agent was not shown named as a
     source; a real agent's answer to a closed issue in an import's plan;
     a real agent's import with `skip`s — DomoticsCore's roadmap, its map
     read against its 56 issues — and a `skip`'s tokens; an import in
     GitHub Actions;
   - **ordering**: a person's priority kept, live; milestones live, on
     GitHub and GitLab, and by due date on GitLab;
   - **a parent**: a test named as proof read live on GitLab;
   - **what an issue waits on**: a GitLab Premium link, a link taken off
     on GitLab, a real agent's `undepend` accepted, all live;
   - **next and stuck**: an issue stuck past 14 days, live; the night's
     summary read for
     [#164](https://github.com/JN0V/workline/issues/164)'s Validation;
   - **off by default, once turned on**: closing as obsolete (a real delay
     of days; the label taken off, an exempt label, a judge's no, live; 7
     days and 3 a run measured); a
     changed need (a real agent proposing `unready` only when it should,
     an imported file's lines changed live, GitLab,
     [#165](https://github.com/JN0V/workline/issues/165)'s Validation).
4. **Parked**, closed as not planned with `workline:parked`, reopened when
   real use asks: a yes in a reply or by a bot
   ([#290](https://github.com/JN0V/workline/issues/290)); splits on GitLab
   ([#291](https://github.com/JN0V/workline/issues/291)); the forge's own
   rank ([#292](https://github.com/JN0V/workline/issues/292)); a part or a
   blocker in another repository
   ([#293](https://github.com/JN0V/workline/issues/293)); a Verification
   item proved in other words
   ([#294](https://github.com/JN0V/workline/issues/294)); a changed
   Verification or Validation
   ([#295](https://github.com/JN0V/workline/issues/295)); a person's
   verdict on a sampled act
   ([#296](https://github.com/JN0V/workline/issues/296)); a pull request
   closed unmerged no longer a start
   ([#297](https://github.com/JN0V/workline/issues/297)).
5. **Open elsewhere**: the Forgejo sample's backlog operations
   ([#65](https://github.com/JN0V/workline/issues/65)); the developer role
   taking the `next-ready` issue
   ([#117](https://github.com/JN0V/workline/issues/117)).
