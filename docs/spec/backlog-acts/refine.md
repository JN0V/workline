---
type: reference
sources: [internal/backlog/backlog.go#hidden, internal/backlog/backlog.go#drafts, internal/backlog/backlog.go#theirs, internal/backlog/backlog.go#moves, internal/backlog/backlog.go#PathsIn, internal/backlog/backlog.go#builder, internal/backlog/backlog.go#named, internal/backlog/backlog.go#IsDraft, internal/backlog/backlog.go#AcceptedSays, internal/backlog/backlog.go#bullet, internal/backlog/backlog.go#questions, internal/backlog/backlog.go#question, internal/backlog/backlog.go#holds, internal/backlog/assume.go, internal/backlog/proof.go, internal/backlog/plain.go, internal/backlog/repeat.go, internal/backlog/write.go, internal/work/headings.go, internal/work/work.go, internal/work/scenarios.go, internal/work/scope.go, internal/backlog/scenarios.go, internal/backlog/verification.go, internal/engine/backlog.go#report]
status: draft
checked: a19e8cb0
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
  issue gets the label `workline:to-refine`; a blocked one, or one
  already `ready`, `needs-tests` or `developing`, keeps its label alone:
  an `unblock` beside the refine moves a blocked one, only `unready` moves
  a ready one back.
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

What a refine reads before ready, says once and writes is in [refine-rules.md](refine-rules.md).

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

A question nobody answers is in [refine-after.md](refine-after.md#a-question-nobody-answers).

## No test possible: a proof kind picked

A proof kind picked is in [refine-after.md](refine-after.md#no-test-possible-a-proof-kind-picked).

## A blocked issue read again

A blocked issue read again is in [refine-after.md](refine-after.md#a-blocked-issue-read-again).

## Its size at ready

A size at ready is in [refine-after.md](refine-after.md#its-size-at-ready).

## Granting a size

Granting a size is in [refine-after.md](refine-after.md#granting-a-size).

## The tester's Verification

The tester's Verification is in [refine-after.md](refine-after.md#the-testers-verification).

## A body's footer

A body's footer is in [refine-after.md](refine-after.md#a-bodys-footer).

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
