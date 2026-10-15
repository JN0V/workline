# ADR-0046: The auditor re-checks a sample of every role's acts, and its verdicts are counted

- **Status:** accepted (2026-10-14); being built: every role's acts
  drawn, re-checked, settled by a panel and counted; the reviewer's
  line per finding judged, the oldest going first past the comment's
  size ([the reviewer's judge](../../roles/reviewer/docs/judge.md#every-finding-judged-kept-for-the-weekly-sample));
  the product owner's fields and its issue folded in to come
- **Date:** 2026-10-14
- **Decided by:** the maintainer, on 2026-10-14: [#205] and [#276] built
  together, the share one in ten; the open points settled as below
  ("Decided by the maintainer")
- **Amends:** [ADR-0015] (the weekly sample), [ADR-0033] (the product
  owner's acts, drawn for a person)
- **Builds on:** [ADR-0005], [ADR-0014] (step 4), [ADR-0019], [ADR-0036];
  [research: auditing a sample of agent acts][research]

## Context

- **The sample reads only docs, and has read none yet** ([#155]). W40:
  nothing vouched for. W41: one doc drawn of six, its `checked` naming
  11d697f, a commit of [#136]'s branch that the squash left off `main`.
  GitHub still serves it: `refs/pull/136/head` reaches it.
- **The other roles act alone, re-checked by nobody.** From 2026-10-04 to
  10-13 the reviewer's judge let 56 findings through and dropped 173
  ([the judge, measured][judge-measured]). In W41 the product owner did 45
  acts alone. The committer rewrote at least 7 messages from 09-30 to
  10-13, counted in the run folders left on the author's machine.
- **One measure exists, made by hand**: 40 of the judge's verdicts read
  again by raters on another model, blind to the verdict; the 9
  disagreements went to a panel of three agents. Its "no" were wrong 8
  times in 20. The maintainer read the result, not the code.
- **The records are partial:**
  - documentalist: `checked`, with `Workline-Role` and `Workline-Model`
    or `verified: agent:documentalist`. Complete, but a squash drops the
    trailers from `main`;
  - product owner: `did` in each issue's state — kind, level, day, a line.
    Not the commit read at, the model, nor the text read;
  - reviewer: rows in its summary comment, rewritten at each run. GitHub
    keeps the earlier versions; GitLab does not;
  - committer: nothing on the commit. The author's message lives only in
    the machine's last 50 runs.

## Decision

### Which acts are drawn

One in ten of a week's acts **per role and kind**, rounded up, ranked by
a digest of the week and the act. That is [ADR-0015]'s draw, stratified
so a rare kind is still read ([research]).

| Role | Acts drawn | W41 on workline | Drawn |
|---|---|---|---|
| Documentalist | a `checked` it moved | 6 | 1 |
| Product owner | each act done alone: ready, refine, sources, order, ask; closing, split, milestone, rename where turned on | 45: 10, 12, 6, 14, 3 | 7 |
| Reviewer | an important finding its judge let through, or dropped | about 160 a week: 39, 121 | 17 |
| Committer | a message it rewrote | 7 in two weeks, at least | 1 |

Not drawn, and why:

- **What a person settles before it takes effect**: a proposal waiting
  on a yes, a doc left `judged` (still suspect, for a person), a question
  the reviewer puts to a person. The person's answer is the verdict.
- **Mechanical acts**: a rule refusing, a finding dropped because its
  quote is not found, the engine's own moves. Conformance proves them
  (principle 7).
- **The judge's answers**: read inside the act they decided — a finding
  kept or dropped, a doc vouched for, a closing as obsolete. One verdict
  an act.
- **A nit**: never judged, never blocking, left to the author.
- **A person's own coding agent** (`verified: agent:claude-code`): the
  person's act, not the line's.
- **The auditor's own verdicts**: measured by the panel, the evaluation
  and the maintainer (below).
- **A gate's verdict** (gitleaks, SonarQube): read, never rerun
  ([roles panorama][panorama]).

### Where each act is read from

At the act's commit, from the records that exist, three of them
completed:

| Role | Read from | Added to the record |
|---|---|---|
| Documentalist | the doc at the commit that moved `checked`, as now | nothing |
| Product owner | `did` in the issue's state | `at`, the commit read at; `model`; `body`, a digest of the text read |
| Reviewer | its summary comment | a hidden line per finding judged — lens, place, commit, yes or no, both models — kept for the merge request's life, on every forge |
| Committer | the commit and its diff | trailers `Workline-Role: committer` and `Workline-Model`; never the author's words |

- **A commit gone from `main`**, as in W41: fetched from the forge's ref
  of its merge request (`refs/pull/<n>/head`; on GitLab
  `refs/merge-requests/<iid>/head`), which a squash leaves. Still not
  found: `not-read`, said.
- **A squashed merge request's commits**: their trailers are read on its
  head, for each merge request merged in the week.
- **The issue's text as it was read**: its body now when the digest
  matches, else the version GitHub's edit history holds, else
  `not-read`. Its comments up to the act.

### How one act is re-checked

One auditor call per act. It gets what the role was given at that commit,
never the role's reasoning. For a finding, it is not told whether the
finding was kept: it answers blind, and the engine compares.

| Act | The question | False, with |
|---|---|---|
| A doc vouched for | Is every passage true of its sources at `checked`? As now | a passage and the code's words, or a name gone |
| Ready | One need, a Validation a person wrote, a Verification proving it, nothing open in the comments? | the issue's words showing what lacks |
| Refine | Do the sections it wrote say what the person asked, nothing more? | its words, and the person's they contradict |
| Sources | Does the code named hold what the issue is about? | the issue's words, the code's |
| Order, estimate | Do the issue's words contradict the priority or effort set? | those words |
| Ask | Was it already answered on the issue? | the answer, older than the question |
| Closing | Does the code do what the issue asks? A duplicate: the same need? | the issue's words, and the code's or a name gone |
| Split | Does each part hold one need, together the parent's? | the parent's words in no part |
| A finding | Does the code fail as the finding says? The lens's question | the code's words |
| A rewrite | Does the message say what the commit does, nothing more? Against the diff alone | the message's words, and the diff's |

- **Another model than every model that acted**: for a finding, neither
  the lens's nor the judge's. Workline has one provider and several
  models: a third one for findings. The level reached is said
  ([ADR-0005]), and `sample.at-least` stays the floor.
- **The proof, found again**: each quote is looked for in the act's
  material at that commit, code outside comments, as now. One not found
  is `unproven`, for a person, never counted false.
- **The author's words kept by a rewrite** are not re-checked here: the
  author's message never reaches history, its internal codes included.
  The weekly evaluation keeps measuring them (its rewrite cases).
- **Without an agent**, the acts drawn are listed for a person, as the
  product owner's are today (principle 5).

### A false verdict

- **A measure, never a change to the line.** Nothing is undone, no level
  lowered, no lens set back, no setting moved. The process engineer
  ([#89]) reads the counts and proposes ([ADR-0019]).
- **Except what [ADR-0014] already does for docs**: a false `checked`
  labels the issue `documentalist-step-0`, and a merge request puts it
  back, for a person ([ADR-0015]).
- **Shown where a person acts**: each false verdict links its issue,
  merge request or commit. A person who agrees with one on a product
  owner's act undoes the act there, which demotes its kind as today
  ([ADR-0033]). The auditor never undoes anything.
- **A true finding the judge dropped, its defect still on `main`**: its
  cause found again there, it is opened as an issue, `needs-triage`,
  through the one way every role opens issues ([ADR-0018]). At most 3 a
  week, the rest counted. A defect of the project's code, not a change
  to the line: the product owner takes it from there.

### Kept and counted, with the maintainer's agreement

- **One tracking issue for every role**: [#155], its title widened. The
  product owner's issue ([ADR-0033]) is folded in; its level suggestion
  is kept.
- **A comment a week**: a table per role, a row per act — its link,
  commit, verdict, the auditor's model and independence, the proof, the
  panel's vote, the maintainer's answer. The same rows sit below as
  hidden YAML the engine reads back.
- **Disputes go to a panel.** A false verdict is two models disagreeing.
  Three judges (the judge role, on the three models available) each get
  the material and both readings, as "a" and "b", never whose. Each
  picks one; the majority settles, a 2–1 vote flagged. It is what
  [the judge, measured][judge-measured] did by hand.
- **The maintainer agrees without reading code.** The week's comment
  opens with a synthesis: the acts read per role, how many stand, each
  false one in a line with the panel's vote. A 👍 from the maintainer
  agrees with the whole week. A 👎 and a reply naming rows (`not 3, 7`)
  contests those rows only. These are the forge's own reactions
  (GitHub's reactions, GitLab's award emoji): one click, with who gave it.
- **The count**: one comment kept in place on that issue. Each week it
  is written again from every week's hidden rows, never added to: per
  role and kind, drawn, stands, false, unproven, not read; agreed,
  contested, unanswered; the weeks that read nothing. A rerun writes the
  same.
- **[The auditor's status][status] links that count.** The pull request
  that moves its status quotes it, dated ([ADR-0043]). A weekly commit
  of work state is what [ADR-0015] refused.

### Cost a week

| | Drawn | Tokens each | A week |
|---|---|---|---|
| Docs | 1 | 8k–12k ([ADR-0015]) | 10k |
| Product owner's acts | 7 | about 10k: the issue, up to 300 lines of the code it names | 70k |
| Reviewer's findings | 17 | about 8k: the finding, its function and tests, as its judge reads them | 136k |
| Committer's rewrites | 1 | about 4k: the messages, the diff (at most 8,000 characters) | 4k |
| Panels | about 6 disputes | 3 × about 9k | 160k |
| **Total** | 26 reads | | **about 380k** |

- **Decided as proposed**: one in ten per role and kind, about 380k
  tokens a week, under the cap below.
- **Volumes**: W41, workline's busiest week so far. The reviewer's come
  from the judge's measure (229 verdicts in 10 days); the committer's
  are a floor.
- **Disputes**: the judge's drops were wrong 8 times in 20, so about 5
  of the 13 drawn; one more for the other roles is a guess. Fewer once
  the judge reads the code it needs ([#362]).
- **Once a week**, on the sample's schedule: about one night of the
  documentalist's gardening here (its cap is 400k).
- **A cap**: `sample.ai-max-tokens`, 400k by default. Each role's first
  act is read before any second one; past the cap, the acts left are
  `not-read`, said.

### Leaving beta

The three criteria of [ADR-0036], with the bar written before it is
measured:

1. **Nothing buildable in Missing**: the four roles drawn, the three
   records added, the panel and the count, with [#205]'s conformance
   cases. Parked: a judge of another provider, and a missed week caught
   up ([#108]).
2. **The main issue validated on real use**: [#205]'s Validation. A week
   on workline lists verdicts for the acts of at least three roles, and
   the maintainer agrees with each false one.
3. **AI verdicts measured:**
   - **Planted**: an evaluation case per kind with a wrong act — a
     closing whose code lacks what the issue asks, a dropped true finding,
     a rewrite saying what its diff does not do, a false doc — and a
     right act as control. In five runs, each wrong act is found false with
     its proof, and each control stands. An auditor answering "stands"
     to everything fails here; the real-use count cannot show that.
   - **Real use**: at least 60 verdicts answered by the maintainer, over
     four weeks or more, at most one contested. That keeps the contested
     share under one in ten at 95% (Wilson), about the size audit
     sampling asks to bound a 5% rate ([research]).
   - The result goes in docs/research, and status.md links it
     ([ADR-0043]).

## Consequences

- Each week reads the acts of every role that acted. W41 would have read
  26 acts instead of one doc it could not read.
- Each rewritten commit carries two trailers more, never the author's
  words. The reviewer's summary carries a hidden line per finding judged,
  and the product owner's `did` three fields more.
- The auditor gets a facet per kind of act: its question, and its answer
  as `claim`s with quotes. Its `sample` settings move to its own role;
  the documentalist's are still read.
- The maintainer spends one click a week, more only to contest.
- Up to 3 issues a week, from findings the judge dropped wrongly.
- The process engineer gets counts per role and kind, each verdict
  backed by a panel or the maintainer.

## Decided by the maintainer (2026-10-14)

1. **The volume**: one in ten per role and kind, about 380k tokens a
   week, under a 400k cap.
2. **Their agreement**: one 👍 a week on the weekly comment; a 👎 and a
   reply naming the rows contested.
3. **A rewrite is re-checked against its diff alone**: no trailer holds
   the author's subject, whose internal codes must not reach history.
   The author's words kept stay measured by the weekly evaluation.
4. **A true finding the judge dropped, its defect still on `main`**, is
   opened as an issue, `needs-triage`, through the one way roles open
   issues, at most 3 a week.

[#89]: https://github.com/JN0V/workline/issues/89
[#108]: https://github.com/JN0V/workline/issues/108
[#136]: https://github.com/JN0V/workline/pull/136
[#155]: https://github.com/JN0V/workline/issues/155
[#205]: https://github.com/JN0V/workline/issues/205
[#276]: https://github.com/JN0V/workline/issues/276
[#362]: https://github.com/JN0V/workline/issues/362
[ADR-0005]: 0005-independence-takes-the-best-level-available.md
[ADR-0014]: 0014-checked-is-earned-by-what-was-read.md
[ADR-0015]: 0015-the-weekly-sample-is-written-on-the-forge.md
[ADR-0018]: 0018-the-product-owner.md
[ADR-0019]: 0019-the-line-evaluates-itself.md
[ADR-0033]: 0033-the-weekly-sample-draws-the-product-owners-acts.md
[ADR-0036]: 0036-a-roles-status-is-earned-by-written-criteria.md
[ADR-0043]: 0043-a-try-is-told-in-its-pull-request.md
[research]: ../research/auditing-agent-acts.md
[judge-measured]: ../research/judge-measured.md
[panorama]: ../research/roles-panorama.md
[status]: ../../roles/auditor/docs/status.md
