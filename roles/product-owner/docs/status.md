# Product owner — where it stands (2026-11-12)

**In one line:** refocused by
[ADR-0038](../../../docs/adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md):
it completes issues (sources, drafts, a question to the reporter), closes
an evident duplicate and readies an evident issue alone, proposes the
rest on each issue (`workline:proposed`, one comment edited in place),
and reads a person's answer there. Milestones, priorities, splits,
renames, links, obsolete issues and changed needs are off by default,
their code kept; its acts done alone are drawn by the weekly sample of
every role's. Tried on both sandboxes; its second trial
on real use (2026-10-17 to 10-24) did not meet its aim. In the V1
acceptance, it refined the 9 issues of each round on three repository
copies; round 3 merged all 9
([round 3](https://github.com/JN0V/workline/issues/1266#issuecomment-6803875764)).
Beta.

Each try: up to 2026-10-13 in [tried.md](tried.md), an archive (the first
nights of the trial, 2026-10-08 to 10-12, among them); since, in its pull
request's `## Tried`
([ADR-0043](../../../docs/adr/0043-a-try-is-told-in-its-pull-request.md)).

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: **open**. The two decisions it
   waited on are parked until after V1:
   [ADR-0039](../../../docs/adr/0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md)'s
   rest ([#435](https://github.com/JN0V/workline/issues/435)) and
   [ADR-0040](../../../docs/adr/0040-priority-and-effort-are-the-forges-the-product-owner-estimates-them.md)
   ([#475](https://github.com/JN0V/workline/issues/475)). Left: the
   role's bugs the V1 rounds and the trials found, kept for 1.0
   ([Missing](#missing-in-the-order-to-build-it), 1).
2. **The main issue validated on real use**: **awaits the
   maintainer**. No open issue holds its Validation: the trial of
   [ADR-0038](../../../docs/adr/0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)
   has none.
   - **What the maintainer would sign**: that the V1 rounds validate
     the role on real use. Refine to ready alone, the reporter asked
     when needed, on Kotlin, C++ and Python copies: 9 of 9 merged in
     round 3
     ([round 1](https://github.com/JN0V/workline/issues/1266#issuecomment-6779607472),
     [round 2](https://github.com/JN0V/workline/issues/1266#issuecomment-6796923928),
     [round 3](https://github.com/JN0V/workline/issues/1266#issuecomment-6803875764);
     V1 accepted in
     [ADR-0052](../../../docs/adr/0052-v1-0-when-the-core-roles-are-out-of-beta.md#amendment-2026-11-11-1266-v1-accepted)).
   - **Against it**: the second trial on workline, 2026-10-17 to 10-24,
     not met (the maintainer, 2026-10-26): none of its 11 proposals
     answered; 3 of 5 acts drawn by the weekly sample judged false
     ([#155](https://github.com/JN0V/workline/issues/155), week 42); too
     many emails ([#548](https://github.com/JN0V/workline/issues/548),
     fixed).
   - **On workline's line since**: the 26 ready issues re-checked, 11
     sent back to refine alone
     ([tried](https://github.com/JN0V/workline/pull/795#tried),
     [tried](https://github.com/JN0V/workline/pull/799#tried)); a
     person's Verification rewritten to answer the reviewer
     ([tried](https://github.com/JN0V/workline/pull/870#tried)).
3. **AI verdicts measured on the acts on by default**: **open**.
   - Measured: an evident duplicate, `keeps-a-planted-backlog`, 5 in 5
     on Sonnet; the kind, 9 of 10 right with labels hidden; split or
     grant on its size, the evaluation's `product-owner/size-*`
     ([#1224](https://github.com/JN0V/workline/pull/1224#tried)).
   - **Against it**: the weekly sample, W42 to W45, judged false 12 of
     13 `ready` and 7 of 8 `refine` drawn; `kind` 4 of 5 stand. No row
     answered by the maintainer
     ([the count](https://github.com/JN0V/workline/issues/155#issuecomment-6265992275));
     the sample is parked until after V1
     ([#155](https://github.com/JN0V/workline/issues/155)).
   - Left: refine and ready measured on known answers, or the sample's
     false rows read again by the maintainer.

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
- **What no person wrote, in plain words**:
  `level-cautious-proposes-example-and-steps`, `question-written-once`,
  `closings-capped`.
- **Each thing said once**
  ([ADR-0039](../../../docs/adr/0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md),
  amended): a role's text on top taken into the Need draft,
  `description-*`; a draft repeating what is above it left out,
  `each-thing-said-once`, `repeated-percent-max-is-a-setting`; a
  person's text on top read as the Need,
  `ready-reads-a-persons-text-on-top`.
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
  `sources-a-folder`; an imported issue read with its lines in the file
  it came from: `imported-issue-*`.
- **Ordering and milestones**, by due date: `order-*`, `milestone-set`,
  `slipped-*`, `summary-next-milestones-by-due-date`.
- **Where priority and effort are kept**, off by default
  ([ADR-0040](../../../docs/adr/0040-priority-and-effort-are-the-forges-the-product-owner-estimates-them.md),
  step 2): `priority-*`; the effort and the grid (step 3): `effort-*`,
  `grid-next-says-each-cell`, `estimates-stuck-said`, the reviewer's
  `spec-read-in-the-owners-order`; GitHub's fields (step 4): `field-*`;
  GitLab's weight (step 5): `weight-*`.
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
- **A bug's parent closed alone** on its reproduction test, green on the
  default branch; a bug split after its tests
  ([#1212](https://github.com/JN0V/workline/issues/1212)): routing's
  `parent-bug-*`, `parent-feature-not-proven-proposes`,
  `size-split-bug-*`, `size-ready-bug-too-big-goes-to-tests`,
  `size-split-asked-once-tests-written`;
  [tried](https://github.com/JN0V/workline/pull/1318) on the Python
  trial copy.
- **What an issue waits on**, a link taken off once its reason is gone:
  `depend-*`, `undepend-*`, `persons-link-kept`,
  `split-child-waits-on-sibling`, `task-lists-blocked-after-blocker`,
  backlog's `*blocker*`, `cycle-reported-not-looped`.
- **Next and stuck**, in the job's summary: `summary-*`, `report-stuck-*`.
- **A changed need**: `changed-*`, `unchanged-need-flags-nothing`,
  `unready-accepted-moves-back`, `unready-waiting-done-alone`,
  `unready-undone-proposed-there`, `unready-undone-waiting-stays-proposed`,
  `archived-source-flags-nothing`.
- **The weekly sample of its acts**, folded into every role's, each act
  keeping what its run read — `at`, `model`, `body`
  ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md)):
  `acts-recorded-with-day-and-level`, `acts-record-what-was-read`,
  `sample-*`, `weekly-sample-said-gone`.
- **A spec read before ready**: `ready-held-while-a-spec-finding-is-open`,
  `ready-released-once-the-spec-is-answered`,
  `refine-answers-the-spec-findings`.
- **A blocked issue read again**, on news or a week on, sent back or
  kept blocked ([#833](https://github.com/JN0V/workline/issues/833)):
  routing's `blocked-*`, `unblock-only-to-where-it-came-from`.
- **A question times out into an assumption**, by level — cautious
  waits, normal 24 h, enterprising at once —; a draft unanswered taken
  as written; a late answer reworked
  ([ADR-0060](../../../docs/adr/0060-the-person-is-on-the-loop.md#1-questions-time-out-as-lazy-consensus),
  [#858](https://github.com/JN0V/workline/issues/858)): `question-*`,
  `assumption-*`, `draft-unanswered-taken-as-written`,
  `draft-enterprising-taken-at-once`, routing's `question-*`.
- **A hidden record that does not read is said**:
  `broken-state-writes-nothing`,
  `proposal-unread-agreement-writes-nothing`,
  `ready-held-while-the-spec-record-does-not-read`.
- **An evident duplicate left right**: the evaluation's
  `keeps-a-planted-backlog`.
- **The kind it sets** ([the kind](kind.md)): bug, feature or task on
  an issue no label tells, once, a person's change kept. Cases
  `kind-set-where-none-is`, `kind-a-persons-change-kept`. Tried on
  [#876](https://github.com/JN0V/workline/pull/876#tried). Not built:
  GitHub's issue types read.
- **The kind of issues read before**, by a light read: title and start
  of body, many in one call, within the act's cap
  ([#878](https://github.com/JN0V/workline/issues/878)). Cases
  `kind-light-read-*`. Tried on
  [#885](https://github.com/JN0V/workline/pull/885#tried).
- **An issue past its size, split or granted**
  ([ADR-0064, step 4](../../../docs/adr/0064-size-is-a-target-not-a-wall.md#build-plan)):
  asked on a run too big for its role, or predicted at ready; a split
  `too-big: true`, on where `split` is off by default (not where the
  project set it off), a part never split again
  ([split on size](../../../docs/spec/backlog-acts/split.md#split-on-size));
  or `workline:size/large`, ×2 every size target of its runs
  ([granting a size](../../../docs/spec/backlog-acts/refine.md#granting-a-size)),
  proposed at `cautious`, `x-large` a person's. Cases `routing/size-*`.
  Tried on [#1173](https://github.com/JN0V/workline/pull/1173#tried):
  the grant, with a stub agent; not tried: a real agent's prediction.
- **Split or grant on facts**
  ([#1193](https://github.com/JN0V/workline/issues/1193)): asked on its
  size, told what crossed with the run's numbers, and the issue's fresh
  tokens against its normal and backstop; an atomic need granted, never
  split. Case `routing/size-task-names-what-crossed`; evaluation cases
  `product-owner/size-*` with a real agent. Tried on [#1224](https://github.com/JN0V/workline/pull/1224#tried).
- **The code an issue names, by the shared picker**
  ([the code picker](../../../docs/spec/code-picker.md)): each
  definition of a name, the lines named, a Scope written beside sources
  it was given. Cases `product-owner/picker-*`. Tried on
  [#1106](https://github.com/JN0V/workline/pull/1106#tried).
- **A feature in the next milestone** at `merge: {autonomy:
  enterprising}`, when it makes it ready
  ([the release candidate](../../../docs/release-candidate.md)). Case
  `ready-puts-a-feature-in-the-next-milestone`; not tried for real
  ([#883](https://github.com/JN0V/workline/pull/883#tried)).

- **A person's answer reaches the spec**, to the reporter or to the
  spec reviewer, kept across later rewrites
  ([#1345](https://github.com/JN0V/workline/pull/1345),
  [#1363](https://github.com/JN0V/workline/pull/1363)); held in V1
  round 3
  ([round 3](https://github.com/JN0V/workline/issues/1266#issuecomment-6803875764)).
- **The Verification is the tester's**: the product owner writes none
  ([#1277](https://github.com/JN0V/workline/pull/1277#tried)).
- **A no-test issue retested once**, unchanged; a rewritten blocked
  issue back to refine
  ([#1299](https://github.com/JN0V/workline/pull/1299),
  [#1301](https://github.com/JN0V/workline/pull/1301)).
- **The developer takes the ready issue it readied**: the pick of
  [ADR-0054](../../../docs/adr/0054-the-developer-takes-a-ready-issue-to-a-pull-request.md)
  ([#117](https://github.com/JN0V/workline/issues/117), closed).

## Missing, in the order to build it

1. **Bugs seen in real use**, kept for 1.0:
   - its code input misses the sources a Scope needs, one Kotlin Scope
     wrong in V1 round 3 (kept for 1.0,
     [#1332](https://github.com/JN0V/workline/issues/1332));
   - a refine dropped for one missing path, never retried (kept for
     1.0, [#1355](https://github.com/JN0V/workline/issues/1355));
   - an answer on a Validation question written in the Scope, the
     reviewer holding again (kept for 1.0,
     [#1356](https://github.com/JN0V/workline/issues/1356));
   - answers to the spec reviewer missed or read twice (kept for 1.0,
     [#1346](https://github.com/JN0V/workline/issues/1346));
   - a person's long answer cut at 3,000 characters (kept for 1.0,
     [#1364](https://github.com/JN0V/workline/issues/1364));
   - a question to the reporter never timing out on a local forge
     (kept for 1.0, [#1337](https://github.com/JN0V/workline/issues/1337));
   - an answer unreadable twice asked again in the same tick (kept for
     1.0, [#1090](https://github.com/JN0V/workline/issues/1090));
   - the role started twice in one tick after naming code (kept for
     1.0, [#1091](https://github.com/JN0V/workline/issues/1091)).
2. **Issue sections and kinds, the rest**
   ([ADR-0039](../../../docs/adr/0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md),
   [examples](proposed-headings.md)): established section names, a bug
   readied only with its steps. Built: sections read by meaning, each
   thing said once, the kind set by the role ([the kind](kind.md))
   (parked, [#435](https://github.com/JN0V/workline/issues/435)).
3. **Priority and effort where the forge keeps them**
   ([ADR-0040](../../../docs/adr/0040-priority-and-effort-are-the-forges-the-product-owner-estimates-them.md),
   [examples](priority-and-effort.md)): steps 1 to 5 built, off by
   default (Built); step 6, workline's labels renamed, and a project's
   own bands, not built; GitHub's fields with the job's token or an
   App's, and GitLab Premium's weight, not tried (parked,
   [#475](https://github.com/JN0V/workline/issues/475)).
4. **Tries on real use**, nothing to build: answers on the issue, a
   person's word first, issues a person reads, what no person wrote,
   refining, autonomy, sources and imports, ordering, a parent, what an
   issue waits on, next and stuck, the acts off by default once turned
   on (parked,
   [#1377](https://github.com/JN0V/workline/issues/1377)).
5. **Parked**, closed as not planned with `workline:parked`, reopened when
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
   ([#297](https://github.com/JN0V/workline/issues/297)); the Forgejo
   sample's backlog operations
   ([#65](https://github.com/JN0V/workline/issues/65)).
