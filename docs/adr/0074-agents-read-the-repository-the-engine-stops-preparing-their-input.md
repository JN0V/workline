# ADR-0074: Agents read the repository; the engine stops preparing their input

- **Status:** step 2 built and measured, its bar not met
  ([amendment](#amendment-2026-11-26-step-2s-bench-the-bar-not-met)); nothing merged.
  [Steps 7 and 8](#steps) go past the order he gave and wait for his
  own yes
- **Date:** 2026-11-25
- **Proposed by:** the maintainer's assistant, from the root-cause
  analysis of 2026-11-25 (seven independent analyses, about 15 replays
  with a real agent); code read at `af68b1bb5` on `main`
- **Settles:** [#2188]; replaces the draft closed with [#2162]
- **Builds on:** [ADR-0005], [ADR-0056](0056-the-developers-agent-works-in-a-sandbox-with-tools.md), [ADR-0063 §8];
  principles 3, 4, 5, 6, 10, 12 ([PRINCIPLES.md](../PRINCIPLES.md))
- **Amends or supersedes, once built:** [the list below](#what-it-amends)
- **Research:** [a fixed context, or the model reading](../research/judge-reads-code.md#a-fixed-context-or-the-model-reading)

## Context

- **No role is at 8 of 10**, the maintainer's bar to leave beta: on
  three repositories, reviewer 5, product owner 5, documentalist 4.
- **Their first cause is one: they do not see the repository.** Each
  gets a task that Go code cut for it, and `claude` with no tools
  ([claude.go](../../internal/agent/claude.go)).
  - **Reviewer**: about 12 of the maintainer's 28 later points were
    never raised; 33 of 63 pull requests were read by hunks.
  - **Documentalist**: three pull requests stamped 5 docs and changed
    no prose; 3 spec pages were "left to a person" on 19 pull requests.
  - **Product owner**: 5 of 28 issues it readied could not be built;
    a kind is read on the first 600 characters of a body.
- **Replayed on the same cases, with a real agent, once each:**

  | Case | Today's input | Reading the repository |
  |---|---|---|
  | Reviewer: 3 defects outside the lines changed ([DomoticsCore#260](https://github.com/JN0V/DomoticsCore/pull/260), [#252](https://github.com/JN0V/DomoticsCore/pull/252), [#247](https://github.com/JN0V/DomoticsCore/pull/247)) | 0 of 3 on Sonnet; Opus at high effort, no better | 3 of 3 on Sonnet at medium effort, about 20k tokens |
  | Documentalist: 7 wrong claims in 3 docs ([DomoticsCore#306](https://github.com/JN0V/DomoticsCore/issues/306)) | 0 of 7 | 3 of 7 on Sonnet, 5 of 7 on Opus |
  | Product owner: an issue's Scope | not replayed | found |

- **Reads served by the engine failed on the same three defects**
  ([#2162]): a finder asked a judge's question, its reads served
  within the judge's caps ([ADR-0045]). 0 of 3, at 35k to 63k tokens in
  each; on two it read the file holding the defect.
- **Preparing input is much of the engine's growth**: non-test Go went
  from 66.9k to 121k lines in a month. Review products moved to the
  model reading (the research above); the tester already reads so.

## Decision

**The reviewer's lenses, the documentalist and the product owner read
the repository themselves, read-only, in a copy at the commit they
judge. The engine gives them what only it knows — the change, the doc,
the issue — and deletes the code that chose what else they read.**

The maintainer's order, 2026-11-25:

- **Dead code first**, in its own pull requests, outside this ADR.
- **Then the input, role by role**: reviewer, documentalist, product
  owner.
- **A filter on answers goes only once its precision is measured.**
- **The line's mechanics are simplified after 1.0**, not here.
- **Every removal is gated by the bench** ([the gate](#the-gate)).

### One way to read

- **The tester's read tree** ([ADR-0063 §8]): a checkout at one commit,
  `Read,Glob,Grep`, no shell, no edit, no network, the role's prompt
  alone. Each file read is logged with the call, its path and blob.
  The task holds what the tree does not, never code the engine picked.
- **Capped by what exists**: the run's `ai-max-tokens`, the answer's
  time, and turns, as the tester's (40 today). A call a cap cuts is a
  failed call, said; never "no finding" (principle 12).
- **No agent**: the checks alone, as today (principle 5).

### An agent other than `claude`

- **A `cmd:` agent is run in a copy too**, and told its path: an
  export of the commit, with no `.git` ([risks](#risks)).
- **Its report may list what it read** (`reads`, in the file
  `WORKLINE_CALL` names); the engine logs them as `claude`'s. With no
  such list, it is taken as an agent that read nothing.
- **An agent that read nothing never passes in silence:**

  | Role | What it gets | What the engine says |
  |---|---|---|
  | Reviewer | the plain diff, what the author says, the issue | each run: the repository was not read, as a cut is said today (`input-cut`) |
  | Documentalist | the doc, its sources by name | `checked` never moves (`checked-unread`): the fix is kept, the doc stays suspect, for a person |
  | Product owner | the issue whole | a Scope not read in the code is written under `Guessed:`, as today |

### Each role

- **The reviewer's lenses**, first on the run on ready, then on each
  push: the change as a plain `git diff`, what the author says, the
  issue, and the tree at the head.
  - A finder's question ([#2188]): what does this change break, here
    or elsewhere? Not the judge's "yes without a doubt".
  - No parts: a change is one call. A diff past the context budget is
    a file in the tree. A call its caps cut is `review-not-whole`.
- **The documentalist** reads the doc whole and the sources it needs;
  the task names them, and the files changed since `checked`.
  - `checked` is still earned by what was read: it moves only when the
    log holds every source file the doc names; for a `cmd:` agent,
    only on the reads its report lists.
  - **Sources too wide to read within its caps**: the call is cut. The
    doc stays suspect, `checked` stays, and the finding names the
    sources not read, as `sources-too-wide` does today. Never stamped.
- **The product owner** reads the tree at the default branch and the
  issue whole. The kind is judged on the whole body; the Scope is
  found by reading, with no "name the code, be read again".

### What stays, and why

**A check stays where it reads a fact that converges, or guards a
write. A filter born of a starved input goes only on a measure.**

- **Every guard stays**: the forge's guards, each role's write fence
  (`writes:`), the tester's red-first proof, the secrets scan, the
  merge policy and a person's label, a run's caps.
- **The checks with no AI stay**: the reviewer's rules, the detection
  of a suspect doc ([ADR-0053]), a patch that must apply.
- **The reviewer's quote filter stays through the input steps.** It
  dropped true findings (3, with the judge, in the evaluation), and
  also drops invented ones. [Step 5](#steps) measures which, with the
  tools on: only then may a quote not found stop dropping a finding.
- **The documentalist's citations and notes stay** the same way, to
  [step 6](#steps). Every word removed is cited because an agent
  removed a true claim "it could not see backed".
- **The judge stays where a block or a merge alone rests on it**
  ([ADR-0073]): independence is what lets either stand with no person
  ([ADR-0005]). A finder that reads is still one reading. Its no still
  quotes the lines that refute, found again ([ADR-0045]).
- **Not decided here**: the judge's model and effort (40% of fresh
  tokens, by the analysis); the line's mechanics.

## The gate

- **Step 0 builds the bench**, three sets replayed with a real agent:
  - **real misses**: the reviewer's 28 points, the 7 wrong claims, the
    5 kinds the maintainer corrected, the issues readied unbuildable;
  - **kept catches**: true findings and true fixes the roles make
    today;
  - **clean pull requests**, where nothing important is to be found.
- **Three runs a case or more**: a bar reads "2 of 3 runs". One replay
  is no measure; the table in Context is one replay each.
- **Every step's bar**, against the baseline today's input gives:
  - the misses found: no fewer, and the step's own target below;
  - no kept catch lost;
  - no more false Important findings on the clean pull requests;
  - fresh tokens and seconds a pull request, the run on ready within
    its [wait budget](0050-each-act-has-its-moment-and-a-wait-budget.md).
- **One bar is run on a `cmd:` agent** that reads, and one on a
  `cmd:` agent that does not: what the engine says is checked.
- **A bar not met**: not merged, the code stays, this ADR is amended
  with the measure.

## Steps

One small pull request a step: a conformance case first, the switch,
the deletion of what it leaves dead, its pages too. By the
maintainer's code budget until 1.0, each is net 0 lines of non-test Go
or fewer, with no new setting; step 1 may land with step 2.

| Step | Switch | Goes, at most | Its own target, in 2 of 3 runs |
|---|---|---|---|
| 0 | The bench | | Today's input is the baseline |
| 1 | The lenses on the run on ready read the tree; a finder's question | | 2 of [#2188]'s 3 defects |
| 2 | Every lens call: the plain diff and the tree. The quote filter is kept | `roles/reviewer/code/parts.go`, `funcctx.go` (580 lines); `marked` in `diff.go`; the material in `reviewer.go`; `code-lines-max`, `parts-max` | Step 1's, on a change read in two parts today |
| 3 | The documentalist reads; `checked` by the log | `roles/documentalist/code/parts.go` (1,376), but `quotedAt`; the packed sources and the change-alone task in `documentalist.go`; `judge-in-parts`, `parts-max`, `parts-max-per-run`, `parts-move-checked` | 4 of the 7 claims on Sonnet; the 3 spec pages left to a person on 19 pull requests are judged |
| 4 | The product owner reads; the kind on the whole body | The picker's call (`askOf`, `writeCode`, `needed`), `kindBodyMax`; its `code-lines-max`, `code-bytes-max` | More of the 5 kinds right; real paths in a Scope readied empty |
| 5 | Filter: a quote not found puts the finding on no line, said | `finding-unfounded`'s drop, `finding-misquoted`'s path | More than half of what it drops is true; none becomes a false Important |
| 6 | Filter: the documentalist's citations and notes | `cite.go`, `notes.go` (1,061), but `quoteIn`, `bodyBlocks`, `wordSet` | No true sentence removed without them, in 3 of 3 |
| 7 | The judges, the reviewer's and the tester's, read the tree. **Needs the maintainer's yes** | `roles/reviewer/code/reach.go`, `reads.go` (958), but `quoted` and `answerIn`; `ServeReads` (`internal/contract/review.go`, its check in `contract.go`); the reviewer's `judge-lines-max`, `judge-read-rounds`, `judge-read-lines-max`, `tests-lines-max`; the tester's `judge-read-rounds`, `judge-read-lines-max` | 32 of the judge's 40 replayed questions right, or more ([ADR-0045]) |
| 8 | The tester's seed and the spec review's material go. **Needs the maintainer's yes** | `internal/kit/codepick`, `internal/codenav` (3,208); the tester's `code-lines-max` and `needs.go`; the packs' code-navigation rows; `TieredSettings` in `internal/role/tier.go` | The tester's cases, no worse, a `cmd:` agent's too |

## What a step leaves

- **The counts are ceilings**: lines of the whole files at
  `af68b1bb5`, 6,122 over steps 2, 3, 7 and 8, and 7,183 with step 6.
  What another caller uses stays, moved.
- **A setting goes when deleted code was its last reader.** Two stay:
  - `whole-chars` still caps what the weekly sample reads
    (`roles/documentalist/code/sample.go`, `internal/sample`);
  - `diff-lines-max` still bounds the judge's question on a task
    ([ADR-0073], `roles/reviewer/code/kept.go`) and the diff-alone
    facet.
- **The tester's seed** (step 8; a `cmd:` agent's whole input today,
  `roles/tester/code/write.go`): the Scope's paths and the nearest
  tests are named in the task, not given. A `cmd:` agent that reads
  none says what its tests assume on the Draft, as `claude` does. Kept
  as it is if the maintainer says no: the picker then stays.
- **Conformance cases retired, by order of size**: about 100 set a
  setting named here, about 45 are named for parts, and about 120 of
  the four roles' cases run a `cmd:` agent on a prepared task. Each
  goes or is rewritten with its step; `pending.txt` follows.

## Cost

- **Reviewer**: about 8k fresh tokens more a pull request, for the run
  on ready (the analysis's estimate): about 0.5M at the evaluation's
  63 pull requests in two days. Each push pays it from step 2.
- **Documentalist**: cheaper than today, by the same analysis; a doc
  in parts costs [130k to 250k tokens](../research/token-costs-measured.md#documentalist).
- **Time**: the replays' wall time was not reported. [#2162]'s served
  reads took 12 to 50 s a run. Every bar holds seconds.
- **Product owner, judges**: not measured. Each step says them.

## Risks

- **An agent that wanders**: turns, tokens and time are capped; the
  log shows what it read, for the auditor too.
- **Prompt injection from the repository.** A file is data from
  whoever wrote the change, as today's diff is; the tree widens it.
  - **Under `claude`**, the engine holds it: the tools only read, no
    shell, no network, no MCP server; a file outside the tree is
    refused; the project's agent settings stay unread.
  - **Under a `cmd:` agent**, nothing is the engine's promise but the
    copy: an export, with no link to the clone's `.git/config`, where
    a job's token may lie. Its environment holds no forge token, as
    today.
  - The answer is a proposal: the forge's guards decide (principle 6).
  - Left: a planted text may talk a finder out of a finding, and a fix
    that merges alone counts on its silence. A diff's comment can today.
- **A read is no proof**: a source opened is not a source understood.
  The weekly sample of docs vouched for stays.
- **Conformance stays deterministic: it calls no real agent.**
  - `fake:` answers without reading ([the fakes](../spec/conformance.md#the-fakes)).
  - Tools are faked as in `tester/write-claude-reads-the-code-itself`:
    a stand-in `claude` refuses any flag but the read tools, checks
    its tree and commit, reads, and answers a recorded stream.
  - How well an agent reads is the bench's question, not a case's.

## What it amends

Each at its step, the ADR amended edited then. Principle 3 stands:
each role reads only what it needs, and now chooses it.

| Decision | Becomes | Step |
|---|---|---|
| [ADR-0020](0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md#amendment-2026-10-17-446-a-change-too-large-read-in-parts): files whole or by hunks, a change in parts; a quote not found dropped | superseded; the quote by step 5's measure | 2, 5 |
| [ADR-0042](0042-the-engine-merges-by-the-projects-policy.md#amendment-2026-11-05-1107-a-person-who-read-it-whole-takes-a-review-not-whole): a person who read it whole takes a review not whole | stands; `review-not-whole` is a cut call, no longer a count of parts | 2 |
| [ADR-0009](0009-docs-far-behind-are-judged-in-parts.md): docs judged in parts | superseded whole | 3 |
| [ADR-0012](0012-a-doc-is-judged-against-its-sources-whole.md): sources whole in the task | the aim stands; the packing goes | 3 |
| [ADR-0013](0013-what-was-judged-is-not-asked-again.md): the `judged-in-parts` record, a merge request per doc judged in parts | read where written, written no more | 3 |
| [ADR-0014](0014-checked-is-earned-by-what-was-read.md#0-the-engine-refuses-an-unearned-checked-now): `checked` earned by what the task gave | earned by the log; its citations by step 6's measure | 3, 6 |
| [ADR-0015](0015-the-weekly-sample-is-written-on-the-forge.md#read-whole-by-a-judge-that-is-not-the-voucher): sources past `whole-chars` make a `checked` unearned | earned by the log; `whole-chars` caps the sample's own read | 3 |
| [ADR-0053]: a source names a part of a file | stands for detection; no task carries the part | 3 |
| [ADR-0039](0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md): the kind's light read | the body whole | 4 |
| [ADR-0045]: the judge asks, the engine serves | superseded, but what a no must hold | 7 |
| [ADR-0073]: a diff past `diff-lines-max` waits | stands | none |
| [ADR-0063 §8]: one shared picker; a `cmd:` agent sees no tree; [§3](0063-every-open-issue-is-due-waiting-or-done.md#3-the-input-of-a-step)'s files "the picker gave" | superseded; its read tools (its step 9) built here | 1 to 8 |
| [ADR-0055](0055-the-tester-writes-the-acceptance-tests-before-the-code.md): the code the tester is given, `needs:` | names and the tree | 8 |
| [ADR-0065](0065-code-only-what-converges-language-knowledge-in-packs.md#the-four-changes): the picker frozen, its tables in packs | deleted | 8 |
| [ADR-0064](0064-size-is-a-target-not-a-wall.md#amendment-2026-11-08-a-cut-is-not-a-crossing): a cut of the code, a code map postponed | no cut left; the map dropped | 8 |

## Not chosen

- **A larger fixed context**: every call pays for it, and what a
  change breaks elsewhere is not known before reading. **A stronger
  model on today's input**: replayed, no better.
- **The engine serving the reads an answer asks**, or **a new finder
  beside the lenses** ([#2162]): 0 of 3; a protocol, a record and a
  setting more, every picker kept.
- **A shell**: the replays needed none; it widens what a planted text
  can do. **An index**: a store and a dependency (principle 11).
- **Filters dropped with the input**: their precision is not measured.

## Amendment (2026-11-26): step 2's bench, the bar not met

- **Built**: steps 1 and 2 for the reviewer's lenses, in
  [#2273](https://github.com/JN0V/workline/pull/2273): the read tree,
  the finder's question, the plain diff, the pickers deleted (non-test
  Go −696 lines).
- **Measured**: the reviewer's replay set, 3 runs, Sonnet 5.5 medium,
  its judge Sonnet, against the baseline of
  [#2271](https://github.com/JN0V/workline/pull/2271):

  | Set | Today's input | Reading the repository |
  |---|---|---|
  | Misses, points | 53/102 | 51/102 |
  | Misses, cases full every run | 2/15 | 3/15 |
  | Kept catches, runs | 12/15 | 10/15 |
  | Clean, runs with no false Important | 6/12 | 9/12 |
  | Tokens a case run, cache included | about 48k | about 80k |

- **The bar** ([the gate](#the-gate)) is not met: one kept catch lost
  in 2 of 3 runs (DomoticsCore #262's stale replay), 2 miss points
  fewer; clean better. By the gate, not merged: the code stays.
- **Seen**: told only to read, a lens read one or two files and
  answered `[]`, its scenarios unanswered; told what to read first and
  that its last message is its whole answer, it answered them. Even so,
  it read a changed header less closely than when it was given whole.
- **Open, for the maintainer**: what step 2 tries next, if anything.
  [#2273](https://github.com/JN0V/workline/pull/2273) waits on that choice.

[#2162]: https://github.com/JN0V/workline/pull/2162
[#2188]: https://github.com/JN0V/workline/issues/2188
[ADR-0005]: 0005-independence-takes-the-best-level-available.md
[ADR-0045]: 0045-the-judge-asks-for-the-code-it-needs.md
[ADR-0053]: 0053-a-source-may-name-a-part-of-a-file.md
[ADR-0063 §8]: 0063-every-open-issue-is-due-waiting-or-done.md#8-one-code-picker-and-tools-where-the-agent-has-them
[ADR-0073]: 0073-a-task-that-changes-no-behaviour-may-merge-alone.md
