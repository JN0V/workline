---
type: reference
sources: [internal/backlog/backlog.go, internal/backlog/assume.go, internal/backlog/proof.go, internal/backlog/plain.go, internal/backlog/repeat.go, internal/backlog/write.go, internal/work/headings.go, internal/work/work.go, internal/work/scenarios.go, internal/work/scope.go, internal/backlog/scenarios.go, internal/backlog/verification.go, internal/forge/forge.go, internal/engine/backlog.go]
status: draft
checked: db6103ad
verified: agent:claude-code
---
# Refining to ready

Part of [Acts on the backlog](../backlog-acts.md): writing the sections an issue lacks, moving it to ready and back, and naming its code.

An issue is `ready` when three sections are written in its body — `## Need`,
`## Validation`, `## Scope` (docs/spec/routing.md) — and
its Need and Validation are a person's (ADR-0018). Its `## Verification`
is not one: the tester writes it from its tests, in `needs-tests`, before ready
([ADR-0066](../../adr/0066-a-role-mirrors-a-human-role-ai-first-where-code-is-rigid.md),
[the tester's Verification](#the-testers-verification)). A person's text on top,
with no Need section holding text, is its Need (`backlog.NeedOnTop`,
ADR-0039, amended). The product owner writes
what it can and asks for the rest:

```yaml
- refine:
    issue: 12
    scope: "WriteRows in src/export/csv.go, and its test."
    sources: [src/export/csv.go]     # the files the scope names, in the commit
    need: "Every row exported, so a report counts what was sold."   # a draft
    example: "Ann makes 3 sales on 8 Oct and exports that day: 2 rows."   # a draft, never required; a bug: steps
    validation: "The maintainer opens an export of a known day."    # a draft
    why: "The issue says what is wrong, not how it will be proved."
- ready: {issue: 12, why: "Its Need, Validation and Scope are there, Need and Validation the reporter's."}
- ask: {issue: 14, questions: "Which export: CSV or JSON? What should a blank row become?", why: "The need is not clear."}
```

- **`refine`** adds the sections the body does not have, after its text,
  which stays as it is; one there but empty — an issue form's field left
  `_No response_` — is filled in place (an issue form writes its fields as
  `### ` headings, read as `## ` ones). A section with text — a person's,
  or one the role wrote before — is never rewritten (`section-kept`), but
  one the role may rewrite to answer the reviewer or a person's comment:
  the Scope always, whoever wrote it
  ([ADR-0060](../../adr/0060-the-person-is-on-the-loop.md)); the Need,
  the Validation, an example or a bug's steps while its own
  ([A spec read before ready](spec-review.md), [A person's answers](answers.md)).
  **In the reader's order** (the table's, `work.Headings`): the person's part — Need,
  an `example` (`## Example`) or a bug's `steps` (`## Steps to
  reproduce`), Validation — above a thematic break `---`; the builder's —
  Verification, Scope — below it. A `verification` the product owner
  gives is left out: the tester's. A section added goes after the nearest
  one before it in that order, else before the nearest one after it, at
  its heading's level, never moving a person's; the break is added when
  every section of the person's part comes before every one of the
  builder's and none is between them; a break alone after the person's
  part, as a footer, is that break. The example and the steps are never
  required. Need, Validation, the example and the steps are drafts: each
  follows a hidden marker alone (`<!-- workline:draft -->`), and the issue gets the label
  `workline:draft`. **A person accepts them with one label,
  `workline:accepted`** — on the issue, or on many at once from the list of
  issues; the engine creates that label with the first draft, so it is
  there to pick; only who may triage sets a label, so it is a person of the
  project's. Its description (`backlog.AcceptedSays`) names its other use:
  on a pull request, a yes to merge it, by the
  [merge policy](../../merge-policy.md), which creates it too. The engine then takes the draft marks out and moves the issue
  to ready, with no agent. **Editing a draft makes it a person's too**:
  its text no longer the one the role wrote (the state's `wrote`, a
  digest per section), the mark left in it or not (`backlog.IsDraft`). A
  draft an earlier engine wrote — a visible line, "*Draft by the product
  owner: …*", around the marker — with no digest kept is told by its
  marker alone; the role now writes the marker alone. Scope names every file the change edits; up to 5 of them are its `sources` (1 to 5, in
  the commit, as for naming its sources), which the issue's state gets when
  it has none; given none, the engine takes the paths the Scope's text
  names that the commit holds (`backlog.PathsIn`). A Scope naming none, or
  a path the commit does not hold, is left out, the rest of the refine
  written (`scope-without-files`, said in the summary with what a person
  may do). Each path the Scope writes in backticks is the developer's
  hint, so each must be found
  ([ADR-0065, change 2](../../adr/0065-code-only-what-converges-language-knowledge-in-packs.md#2-scope-as-paths-a-hint-not-a-fence),
  [#1228](https://github.com/JN0V/workline/issues/1228); `work.Unfound`):
  a file or folder the commit holds, a glob matching one of its files,
  or a file to create on a line `New:`. A Scope naming only new files,
  each on a `New:` line, needs no `sources`. A word with no `/` (a name,
  `os.Getenv`) is no path to check, nor one the repository's ignore
  rules match: a file the build writes, `app/build/outputs/…`, is a
  value in the Scope
  ([#1411](https://github.com/JN0V/workline/issues/1411)). A file the developer asked the Scope widened with, on its
  blocked issue, is kept though the commit does not hold it: its pull
  request adds it ([#1061](https://github.com/JN0V/workline/issues/1061)).
  That widening left out, an `unblock` beside it is refused
  (`scope-not-widened`): the issue stays blocked. Nothing to add drops the act (`nothing-to-refine`). The
  issue gets the label `workline:to-refine`; a blocked one keeps its
  label alone, an `unblock` beside the refine moving it.
- **A Scope waits for its code** (#970). A refine's Scope on an issue
  whose code the run did not give — not in pre's `code-read`, its state
  naming no sources, its title and body no file of the commit
  (`noCodeGiven`) — or whose code this run names anew (a `sources`
  done naming a file the task did not give: pre's `code-given` lists
  those it gave; a Scope written beside sources it gave stands, no
  `needs-code`, [#1089](https://github.com/JN0V/workline/issues/1089);
  so does one in a run the line started again for the code the agent
  said it lacked, on the issue it is about, each file of its `sources`
  given this time, whole or in part
  ([#1146](https://github.com/JN0V/workline/issues/1146)); so does one
  there whose own files were all given, or whose guessed paths a line
  `Guessed:` says, module folders named and cut or not
  ([#1433](https://github.com/JN0V/workline/issues/1433)); a first
  run's is left out as below),
  is left out, said `scope-unread`: its paths would be a guess,
  and a guessed folder can hold the issue behind an unrelated claim.
  Its paths, checked as above, are kept as the issue's sources, the
  rest of the refine written; the issue, still lacking its Scope, is
  read again with that code. A refine with its Scope alone is dropped
  (`scope-unread`), the agent told to name the code with `sources`. A
  person's yes to a proposal is done as it reads. A Scope marked a
  guess — a line `Guessed:` (`work.Guessed`); its words saying no code
  was given are no longer read, English prose being no fact
  ([#1228](https://github.com/JN0V/workline/issues/1228)) — is the role's to rewrite: once its code is given,
  a refine's new Scope replaces it, never dropped as there already
  ([#1126](https://github.com/JN0V/workline/issues/1126)). So is a
  Scope on an issue holding a person's comment the role has not read —
  an answer to the reviewer's question among them: what it settles goes
  into the Scope, the tester and the developer building from the body,
  never from its comments. The task quotes those comments
  ([#1331](https://github.com/JN0V/workline/issues/1331)). It gives
  the issue's body whole (cut past 20,000 characters only, for pasted
  logs) and every comment of a person who counts:
  only the role's and others' beyond the last five are left out, as a
  section rewritten whole loses what its writer could not see.
- **An answer read before ready**
  ([#1331](https://github.com/JN0V/workline/issues/1331)): the agent's
  `ready` on an issue whose run read a person's comment, its body as the
  role last read it, is held once (`answer-unwritten`) unless the run
  drafts a section of the spec on it (not the Verification alone) that
  is not dropped before the ready. The issue stays in
  refine, those comments unheard again (`heard`), the hold kept in the
  state (`answer-held`, their count). The next run quotes them again
  and says the ready was held; its `ready` is taken. Never held: the
  engine's own ready, a person's yes (`workline:accepted`).
- **Each thing said once** (ADR-0039, amended). The text above an
  issue's sections — its description — is read by whose it is, the
  engine telling so, never the agent (`backlog.RolesTop`): a role's when
  the issue holds an import's key or a finding's `opened-by`, and its
  description, spaces aside, matches the digest the state kept at
  opening (`wrote: {Description: …}`), an import's key, or an earlier
  engine's rewrite in plain words (`plain`); opened before that digest
  was kept, a finding opened by a bot (a GitHub app, a GitLab bot)
  counts as unedited while its body is unchanged since the role last
  read it. Otherwise it is a person's.
  - **A role's text on top becomes the Need draft**: when the refine
    writes a `need` — added, filled, or rewritten to answer a comment —,
    that text leaves the top; the engine's lines there — where it comes
    from ("Opened from …", "Found by …"), its hidden markers — stay, in
    their order, and the text before is in the forge's edit history.
    A block the role folded there (`<details>` … `</details>`, a cause
    and its fix) moves whole to the end of the last builder's section
    the refine writes (the Scope), or stays on top when
    it writes none ([#841](https://github.com/JN0V/workline/issues/841)).
    Never on an outsider's issue proposed in a comment. It follows the
    refine's mode and level as the drafts do: at `cautious`, proposed.
    The issue's one line says "Rewrote: the text on top as the Need, in
    plain words."
  - **A person's text on top, with no Need section, is the Need**, a
    person's: `ready` reads it so, and the task says it. A Need draft is
    written, and read instead, only when it does not say that text again.
  - **A draft that repeats is not written** (`backlog.Repeated`):
    containment, the share of its words already said above it, over
    one-word shingles, common words out. Compared: a Need draft with a
    person's text on top; an example or a bug's steps with all above
    them as the issue will read — the text on top, unless taken into the
    Need, and the Need. Never Validation, which restates the case by
    design. Past `repeated-percent-max` (40 by default, 1 to 100), the
    draft is left out, the rest of the refine kept (`repeated`, said),
    and the issue's line says "Not written: the Example, which said
    again what is above it." A person's text is only read; what a
    person said yes to — a proposal accepted with the label, a text the
    reporter agreed to — is written as they read it, measured when it
    was proposed, the top taken only if it said so.
  - A `description` given apart, as an earlier engine asked, is left out
    (`description-kept`, said).
- **What it writes is readable**, where the engine can tell
  (`backlog.LinkDecisions`, `backlog.MarkClosed`): a decision cited bare,
  `ADR-0038`, becomes a link to its page on the forge, when the commit
  holds it (a numbered page in a folder named `adr`, `adrs` or
  `decisions`); an issue cited that is closed is followed by "(closed)",
  once. Neither touches code — a code span, a fenced block. The task says, for each issue read, the issues it cites, open or
  closed and why (`Cites:`), and how a file of the repository is linked
  (`Links:`, the forge's file pages at `HEAD`).
- **An issue read and left with nothing that refines it** — no draft,
  no question, no sources nor closing; a kind, an order, a link or a
  title alone count for nothing — though it lacks a section is said in the
  summary (`nothing-proposed`) and on the issue, in the role's comment
  ([ADR-0059](../../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md), decision 1):
  the sections it lacks, each act of the agent's dropped on it with why,
  and the agent's note on it (`silent` in its state), kept until the
  role acts on it. Its next read tells the agent so: refine it now, or
  ask. The line wakes it again in refine
  ([ADR-0059, amended](../../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-03-an-issue-left-unrefined-is-woken-again)).
  The task lists the repository's
  folders for an issue that names no file: the agent names its sources
  from them, no line quoted. A `sources` naming only code the run did
  not give — no path in or holding one its state, title or body names
  (`unshown`) — needs no line quoted either
  ([#980](https://github.com/JN0V/workline/issues/980)). One mixing code
  given and not given, no line quoted, names the code not given and
  keeps the sources its state names; code given new to it is left out,
  said (`sources-unquoted`, info)
  ([#1151](https://github.com/JN0V/workline/issues/1151)).
- **`ready`** is checked by the engine, not taken from the agent: the four
  sections there and not empty — a person's text on top standing for a
  Need —, no draft of the role's left in Need or
  Validation unless the issue bears `workline:accepted` — read on the
  issue, never in a proposal (`not-ready` otherwise, naming what is
  missing); a mark left in a section a person edited is taken out.
  A Need or Validation whose writer opened it with `Draft:` is not
  final either: held the same way, never taken as an assumption, and the
  role's to rewrite (a `refine`, the agent told so) until a person
  accepts it ([#1019](https://github.com/JN0V/workline/issues/1019)). Done, the issue gets
  `workline:ready` and loses `workline:to-refine`, as `workline item ready`
  does, and loses `workline:draft` and `workline:accepted`. Where the
  project routes the tester, an issue whose Validation names scenarios
  gets `workline:needs-tests` instead: the tester's result makes it
  ready
  ([ADR-0059](../../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#3-tests-wakes-the-tester-its-result-makes-ready)).
  Scenarios are read in the forms people write Gherkin in Markdown
  (`work.Scenarios`, the one parser of the product owner, the tester
  and the developer): a bullet opening on its name in bold, `- **Name:**
  Given …`; a line naming it — `**Scenario: Name**`, `Scenario: Name`, a
  heading, a bold name alone — its steps below; the same in fenced
  code. A Validation with none of these — Given / When / Then in another
  form, or prose — is held in refine where the tester is routed
  (`scenarios-unread`): the Definition of Ready of Tests is named
  scenarios. It is said on the issue in the role's comment with the
  form expected, and kept until it reads
  ([#855](https://github.com/JN0V/workline/issues/855),
  [#1019](https://github.com/JN0V/workline/issues/1019)):
  the role's own text, the agent is told once a body to rewrite it (a
  `refine`, the Validation among those it revises); a person's, a person
  rewrites it, the agent told to `ask` its reporter.
  An issue ready from before the line, kept by a `ready`, goes there
  too while it has no Draft of the tester's open nor a proof kind
  recorded; otherwise its labels stay
  ([#788](https://github.com/JN0V/workline/issues/788)). At `merge:
  {autonomy: enterprising}`, a feature made ready in no milestone is put
  in the next open milestone neither released nor with its candidate cut, its
  candidate what a person tries (`milestone-set`; one that cannot be set
  is said, `milestone-not-set`, warn, the ready kept;
  [Release candidate](../../release-candidate.md)). An issue opened
  by someone without write access to the project is theirs: moving it to
  ready is proposed on the issue, never done (`reporter-outside`) —
  unless a person of the project accepted it with the label. A bot's
  issue, a split's children the line's App opened among them, and a
  role's finding are the project's
  ([#1195](https://github.com/JN0V/workline/issues/1195)). Who is of the
  project: on GitHub, the issue's author association (owner, member,
  collaborator); on GitLab, a member with the Planner role or above, read
  from the project's members (ADR-0023) — a token that may not list them
  fails the run, loud; on a plugged forge, its `insider`. One the forge
  does not say is outside.
- **`unready`** moves a ready issue, or one waiting for its tests,
  back to refine: `workline:ready` and `workline:needs-tests` off,
  `workline:to-refine` on, a developer's claim `workline:developing`
  off; the role's one comment on the issue says why
  and that putting `workline:ready` back undoes it. It is **done alone**,
  whatever the level or the settings, a changed need's included — cheap
  and undone by a label ([ADR-0059, amended](../../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-01-back-to-refine-needs-no-yes)) —;
  proposed only where a person undid one on that issue, or `undone-max`
  across the issues (ADR-0038). One proposed before, still waiting, is
  done at the next run with no agent. One bearing neither label is
  dropped (`unready-not-ready`). No quote; the state must read.

  ```yaml
  - unready: {issue: 13, why: "#12 now asks for JSON: its CSV row count no longer proves it."}
  ```

- **`ask`** comments on the issue, naming its reporter, with the agent's
  questions. The answer is a person's comment: the issue is read again at
  the next run. Unanswered past `acts.ask.wait`, it times out
  ([below](#a-question-nobody-answers)). An `ask` beside a `sources` the same run does alone for
  the same issue is not asked (`code-first`, `codeFirst`): the issue is
  read again with that code first, and its reporter never asked what the
  code answers ([#548](https://github.com/JN0V/workline/issues/548)),
  nor, for an imported issue, what the file it came from says around its
  item: the task gives it ([Reading](reading.md)).
- An issue whose body a person changed since it was read — a draft
  accepted, a section written — is read again (its state's `body`, a
  digest of the body when it was last read or written by the engine).

## A question nobody answers

[ADR-0060](../../adr/0060-the-person-is-on-the-loop.md#1-questions-time-out-as-lazy-consensus):
a question unanswered for the level's delay is taken as the role's
assumption, written on the issue, and the issue goes on.

```yaml
- assume: {issue: 14, assumption: "CSV, as the other exports.", why: "No answer in 24 h."}
```

- **The delay**, `acts.ask.wait` (`backlog.Delay`): `never` at
  `cautious`, `24h` at `normal`, `"0"` at `enterprising`
  ([settings](../../../roles/product-owner/docs/settings.md#how-long-a-question-waits)).
  Counted from the question's comment as the forge dates it
  (`created_at`, `backlog.Unanswered`), never from the issue's last
  update; a comment after it, anyone's but the engine's, is an answer
  and cancels it. A forge that dates no comment: never timed out
  (`assume-undated`).
- **Told once**: past the delay, the task says so for that issue
  (`told` in the state); the line wakes the role on it once
  ([routing](../routing.md#once-a-state-the-record)). At `"0"`, the task
  asks an `assume` beside each `ask`.
- **`assume`** is done alone, at every level the delay allows, checked:
  a question waiting past the delay, or asked in this run at `"0"`; not
  taken already (`assumed` in the state, the question's round). Dropped
  otherwise: `assume-early`, `assume-waits` (`never`),
  `assume-nothing-asked`, `assumed-already`, `assume-empty`.
- **Written in the body**, under `## Assumptions` at its end, above a
  footer (`backlog.WithAssumption`): "- Assumption taken on <day>, the
  question of <day> unanswered after 24 h: <assumption>." Need, Example,
  Validation and the builder's part are left as they are. Said in the
  role's one comment: "Assumption taken on #14: …. Answer here, or edit
  its line under Assumptions in the issue, to change it."
- **A draft is a question too**: a Need or Validation draft no person
  accepted, all that keeps the issue from ready (`backlog.OnlyDrafts`),
  waits the same delay, from when the forge says the issue got
  `workline:draft`, or a person's later comment. Past it, the engine
  moves the issue to ready with no agent, its draft marks out, a line
  under Assumptions saying the draft was taken as written; an agent's
  `ready` passes so too. At `cautious`, `ready` still needs a person.
- **A late answer**: a counted comment after the assumption
  (`assumed-at`) marks each line "Answered by @x on <day>: their answer
  stands over it", with no agent, and is read as a person's comment on
  what the role proposed: its own sections rewritten, `unready` if ready
  and no longer true ([answers](answers.md)).

## No test possible: a proof kind picked

The tester found no test possible and sent the issue back to refine
([ADR-0059](../../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#4-a-proof-kind-not-always-a-new-test)):

```yaml
- ready: {issue: 12, why: "Only the README's words change.", proof: {kind: docs-checks, reason: "The documentalist's checks prove it."}}
```

- **Read once**: the tester's reason, recorded with `none`, is given in
  the task until read (`no-test` in the state, its digest).
- **`ready` with `proof`** picks the kind in place of new tests:
  `existing-suite`, `docs-checks`, or `flaky-n-of-n` with its `test` (and
  `runs`). The issue goes to ready, not to its tests; the kind is kept
  in its state, `answers` naming the tester's word. The developer takes
  it while that word is the tester's last (`backlog.ReadProof`).
- **Refused**: a `proof` where the tester did not find none
  (`proof-not-asked`), with no reason, an unknown kind, a flaky test
  unnamed (`proof-reason-missing`, `proof-kind-unknown`,
  `proof-test-missing`).
- **Or back to its tests**: the role rewrites what keeps a test from
  being written — its Scope, its own Validation draft
  ([#1053](https://github.com/JN0V/workline/issues/1053)) — or asks; a
  `ready` with no proof beside that rewrite goes to
  `workline:needs-tests`.
- **Or back unchanged, once**
  ([#1053](https://github.com/JN0V/workline/issues/1053)): the reason
  is the tester's own — code missing from its task, a run that failed —
  not the issue's. `ready` with `retest: true` sends it to its tests as
  it stands; the state keeps the word it answers, with the body the
  tester read (`retest`, `backlog.RetestKey`), and the tester asks its
  agent again (`backlog.Retested`). The tester's next
  "no test possible" is recorded `retried`: a second retest is refused
  (`retest-spent`), as is one beside a `proof` (`retest-with-proof`) or
  with no "no test possible" to answer (`retest-not-asked`). On a text
  rewritten since the tester read it, or by a refine of the same run,
  it is a plain `ready`.

```yaml
- ready: {issue: 12, retest: true, why: "The tester lacked the counter's code, not a scenario."}
```

- **Never back unchanged otherwise**
  ([#815](https://github.com/JN0V/workline/issues/815)): the tester's
  record holds the digest of the body it read. While the body is that
  one, no kind picked and no retest sent (`backlog.NoTestStands`), a
  `ready` with no proof is refused (`no-test-unanswered`) unless a refine of the same
  run rewrites it; neither the spec review nor `workline:accepted`
  moves it. Left with nothing, the issue is said on it
  (`no-test-waits`), and the line moves it to `workline:blocked`, the
  tester's reason said (`no-test-unsettled`,
  [routing](../routing.md#the-line-of-states)). An older record with no digest holds nothing back:
  the tester's next word records one.
- **With no agent**, a person of the project comments `proof: <kind> —
  why` after the tester's comment (`backlog.PersonsPick`): the engine
  does that `ready`, whatever the role's mode, as their yes.

## A blocked issue read again

The line wakes the role on a blocked issue with news on it, or a week on
([routing](../routing.md#blocked-read-again),
[ADR-0059, amended](../../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-01-blocked-is-read-again)).
The task says why it was blocked, what is new, and the state it came
from.

```yaml
- unblock: {issue: 12, to: tests, why: "#9, which it waited on, is merged."}
- blocked: {issue: 12, why: "The objection needs the maintainer's call.", person: true}
```

- **`unblock`** moves it to `to`: refine, or the state it came from
  (`revisit-from`; Ready for one a developer had claimed). Every other
  state's label off, its card moved; the role's comment says why, and
  that putting `workline:blocked` back undoes it. Another `to` is
  dropped (`unblock-to`).
- **`blocked`** keeps it there, its reason said afresh in the role's
  comment; `person: true` says only a person can lift it. Nothing moved.
- **Done alone**, at every level, as `unready` is: cheap, undone by a
  label. Either on an issue not blocked is dropped (`not-blocked`); a
  `ready` on a blocked one too (`blocked-first`).
- **Or a `close`**, by its own rules.
- **A `refine` alone**, on an issue the line blocked from refine for
  the product owner's own read left lacking, its text rewritten and no
  `unblock`, `blocked`, `close` nor `split` on it: the engine moves it
  back to refine in the same run, said as an `unblock`. The read it was
  blocked on is gone, so it is read again in refine as it stands, not
  left blocked a week more
  ([#1206](https://github.com/JN0V/workline/issues/1206)). A refine
  proposed or dropped moves nothing; nor does any refine on a block a
  person or another role made, or one on a run's size: a rewrite does
  not lift those.
- **The developer's ask for its Scope**, from before
  [#1228](https://github.com/JN0V/workline/issues/1228) made the Scope a hint
  ([ADR-0059, amended](../../adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-03-a-fix-outside-the-scope-asks-the-product-owner)):
  the task names the files. To widen it, a `refine` rewrites the Scope
  whole, the files in it, then `unblock` to ready: the developer
  resumes on its pull request. Else `blocked`, why not.
- **A run too big for its role**
  ([ADR-0064, step 3](../../adr/0064-size-is-a-target-not-a-wall.md#2-two-checkpoints)):
  woken at once (`size`, the role), told what crossed with the run's
  numbers, and the issue's fresh tokens so far against its normal and
  its backstop ([#1193](https://github.com/JN0V/workline/issues/1193)).
  To narrow it, a `refine` rewrites the Scope whole, fewer files, a
  person's Scope too, then `unblock` to its state. A narrowing not
  written holds the unblock beside it (`scope-not-narrowed`); sent back
  as it is, the next crossing waits for a person. Its need atomic — no part
  passes its own scenarios apart — at the
  normal size: `size` with `tier: large`, then `unblock`
  ([ADR-0064, step 4](../../adr/0064-size-is-a-target-not-a-wall.md#3-the-product-owner-decides-once-a-state-an-issue)).
  Parts that pass apart, the issue no part itself: a `split`,
  `too-big: true` ([split on size](split.md#split-on-size)), then
  `blocked`, `person: true`: a person accepts it from its parts. Else
  `blocked`, `person: true`, why it cannot be narrowed.

## Its size at ready

On the line's refine step, the task asks the product owner to predict
the issue's size before `ready`
([ADR-0064, §2](../../adr/0064-size-is-a-target-not-a-wall.md#2-two-checkpoints)),
in the same call:

- **Against its normal**: the fresh tokens an issue of its kind spends
  here, the project's 85th percentile kept by `workline size`, else the
  seed (400k, doubled at the highest effort); times its tier.
- **From** its Scope's files, its scenarios and its effort.
- **Well past it**: a `split`, `too-big: true`, and no `ready`; or, its
  need atomic, `size` with `tier: large`, then `ready`. A part of a
  split is offered no split, an issue with a tier no grant.
- **A bug, its tests first**: in place of the split, `ready` with
  `too-big: true`; once its tests are written, the line asks its split
  ([split on size](split.md#split-on-size)).

## Granting a size

```yaml
- size: {issue: 12, tier: large, why: "one parser: no part of it passes its tests alone"}
```

- **What it does**: puts `workline:size/large` on the issue; every size
  target of the runs on it doubles (its context budget, token cap, code
  and change given at once). Said on the issue, kept in its state
  (`size: large`), counted and sampled by the auditor.
- **Checked**: `large` alone; `x-large` is a person's (`size-person`);
  an issue with a tier already is never granted (`size-kept`); one the
  role granted once, never again: a person who took the label off
  decided (`size-once`).
- **Never the backstop**: the issue's runs in all still stop at 4× its
  normal, for a person
  ([routing](../routing.md#the-line-of-states)).
- **Mode**: `act` at `normal` and `enterprising`, `propose` at
  `cautious` ([autonomy](autonomy.md)).

## The tester's Verification

How the work is proved is the tester's, as a human tester writes the
test plan ([ADR-0066](../../adr/0066-a-role-mirrors-a-human-role-ai-first-where-code-is-rigid.md),
[#1261](https://github.com/JN0V/workline/issues/1261)):

- **The product owner writes none**: a `verification` in its `refine` or
  a `split` child is left out (`work.Testers`); ready never waits for one.
- **The tester writes it** once its tests are proved red and linted, by
  the engine, not its agent (intention `verification`, `tester.testPlan`):
  each scenario, the test that proves it, or why none can. It is
  marked `<!-- workline:verification=tester -->`, below the line, and its
  comment on the issue says it wrote it.
- **Its own is rewritten** at its next tests, and so is one the product
  owner wrote before, as its state's `wrote` records it, unedited
  (`backlog.TestersToWrite`). **A person's is kept** word for word
  (`backlog.WithVerification`). When that text names not every test,
  the tester's comment says the tests prove it another way, naming each.
- **A reviewer's finding on it is the tester's**: it holds nothing from
  ready (`SpecReview.Testers`), it clears the spec as no finding would
  (`backlog.SpecCleared`: the engine's ready with no agent), it spends no
  round of the review, its rounds spent it is no person's, the product
  owner is not given it, and the tester's task quotes it
  (`backlog.SpecForTester`).
- **No tester routed, no Verification**: the product owner's ready puts
  `workline:ready` straight on, with no `needs-tests` step; the issue is
  built without one. Route the tester to have it written.

## A body's footer

A body's footer belongs to no section (`work.Footer`): what a tool or a
person appends after the last section is not the Scope's text. It is,
outside fenced code (CommonMark's fences: three or more backticks or
tildes open one, a backtick fence's info string holding no backtick; a
line of the same character alone, as long or longer, closes it):

- from a **thematic break** (`---`, `***`, `___`, CommonMark's, a blank
  line above it, so not a heading's underline) after the last heading, to
  the end — one with headings after it, as between the person's part and
  the builder's, is no footer, and is not the text of the section it
  ends (`work.Sections`): a draft's digest is the same with it or without;
- or else the body's **last line, when it is a tool's signature**: alone
  in its paragraph, reading "Generated with …" or "Generated by …", after
  an emoji or a mark if any — `🤖 Generated with [a tool](https://…)`.

Every reading of the sections leaves it out — `ready`'s check, the
state's `sections`, `wrote` and `deleted`, a finding's section —, and a
section `refine` adds, or a parent's `## Sub-issues`, goes above it,
the footer left last as it was.

## Naming its sources

```yaml
- sources:
    issue: 12
    sources: [src/export/csv.go]
    quote: {path: src/export/csv.go, text: "func WriteRows(rows []string, write func(string)) {"}
    why: "#12's WriteRows is defined here."
```

The files, or folders, must be in the commit the run is on (1 to 5;
`sources-unknown` otherwise). A quote, when given, is found in one of the
files named, an anchor after `#` aside, or in a file inside a folder named
(`no-quote` otherwise). None is needed only when no code was given —
the state names no sources, the title and body no file of the commit —:
named from the repository's folders
([#951](https://github.com/JN0V/workline/issues/951)). The sources its
state names already, their code given in the run, are dropped
(`sources-same`): named again, the issue would only be read again. A
folder is given at the next run as the list of its files. Done, the issue's
state gets them and loses `judged`: it is read again, with that code, at the
next run.
