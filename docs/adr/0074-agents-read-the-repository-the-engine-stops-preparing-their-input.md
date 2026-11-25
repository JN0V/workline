# ADR-0074: Agents read the repository; the engine stops preparing their input

- **Status:** proposed, until the maintainer merges it; nothing built
- **Date:** 2026-11-25
- **Proposed by:** the maintainer's assistant, from the root-cause
  analysis of 2026-11-25 (seven independent analyses, about 15 replays
  with a real agent); code read at `af68b1bb5` on `main`
- **Settles:** [#2188]; replaces the draft closed with [#2162]
- **Builds on:** [ADR-0005], [ADR-0056], [ADR-0063 §8] (the read tree,
  built for the tester); principles 3, 4, 5, 6, 10, 12
  ([PRINCIPLES.md](../PRINCIPLES.md))
- **Amends or supersedes, once built:** [the list below](#what-it-amends)
- **Research:** [a fixed context, or the model reading](../research/judge-reads-code.md#a-fixed-context-or-the-model-reading)

## Context

- **No role is at 8 of 10**, the maintainer's bar to leave beta. The
  evaluation of 2026-11-25, on three repositories: reviewer 5, product
  owner 5, documentalist 4.
- **Their first cause is one: they do not see the repository.** Each
  gets a task that Go code cut for it, and `claude` with no tools
  ([claude.go](../../internal/agent/claude.go)). Only the developer
  and the tester read files.
  - **Reviewer**: about 12 of the maintainer's 28 later points were
    never raised; 33 of 63 pull requests were read by hunks.
  - **Documentalist**: three pull requests stamped 5 docs and changed
    no prose; 3 spec pages were "left to a person" on 19 pull requests.
  - **Product owner**: 5 of 28 issues it readied could not be built;
    a kind is read on the first 600 characters of a body.
- **Replayed on the same cases, with a real agent:**

  | Case | Today's input | Reading the repository |
  |---|---|---|
  | Reviewer: 3 defects outside the lines changed ([DomoticsCore#260], [#252][DomoticsCore#252], [#247][DomoticsCore#247]) | 0 of 3 on Sonnet; Opus at high effort, no better | 3 of 3 on Sonnet at medium effort, about 20k tokens |
  | Documentalist: 7 wrong claims in 3 docs ([DomoticsCore#306]) | 0 of 7 | 3 of 7 on Sonnet, 5 of 7 on Opus |
  | Product owner: an issue's Scope | not replayed | found |

- **Reads served by the engine failed on the same three defects**
  ([#2162]): a finder asked a judge's question, its reads served
  within the judge's caps ([ADR-0045]). 0 of 3, at 35k to 63k tokens in
  each; on two it read the file holding the defect.
- **Preparing input is much of the engine's growth**: non-test Go went
  from 66.9k to 121k lines in a month.
- **Review products made the same move** (the research above), and the
  mode exists: the tester's `claude` reads a tree with `Read,Glob,Grep`
  (case `tester/write-claude-reads-the-code-itself`).

## Decision

**The reviewer's lenses, the documentalist and the product owner read
the repository themselves, read-only, in a copy at the commit they
judge. The engine gives them what only it knows — the change, the doc,
the issue — and deletes the code that chose what else they read.**

### One way to read

- **The tester's read tree** ([ADR-0063 §8]): a checkout at one commit,
  `Read,Glob,Grep`, no shell, no edit, no network, the role's prompt
  alone. Each file read is logged with the call, its path and blob.
- **The task holds what the tree does not**, never code the engine
  picked.
- **Capped by what exists**: the run's `ai-max-tokens`, the answer's
  time, and turns, as the tester's (40 today). A call a cap cuts is a
  failed call, said; never "no finding" (principle 12).
- **A `cmd:` agent is run in the tree** and told its path. Whether it
  reads, and read-only, are its command's promise, as in a worktree.
  **No agent**: the checks alone, as today (principle 5).

### Each role

- **The reviewer's lenses**, first on the run on ready, then on each
  push: the change as a plain `git diff`, what the author says, the
  issue, and the tree at the head.
  - A finder's question ([#2188]): what does this change break, here
    or elsewhere? Not the judge's "yes without a doubt".
  - No finder beside the lenses, no record, no setting: the call that
    exists gets the tools.
  - No parts: a change is one call. A diff past the context budget is
    a file in the tree. A call its caps cut is `review-not-whole`.
- **The documentalist** reads the doc whole and the sources it needs.
  The task names the doc, its sources, the files changed since
  `checked`.
  - `checked` is still earned by what was read ([ADR-0014]), now from
    the log: it moves only when the log holds every source file the
    doc names.
  - No doc is "left to a person" for its size.
- **The product owner** reads the tree at the default branch and the
  issue whole. The kind is judged on the whole body; the Scope is
  found by reading, with no "name the code, be read again".

### What stays, and why

**A check stays where it reads a fact that converges, or guards a
write. A filter born of a starved input goes, or proves itself on the
bench.**

- **The checks with no AI stay**: the reviewer's rules, the detection
  of a suspect doc ([ADR-0053]), a patch that must apply.
- **The quote stays as an address, no longer as a filter.** Found
  again, the finding is on its line. Not found, it is shown on the
  change, on no line, said; never dropped (`finding-unfounded`).
  - Why: the evaluation counts 3 true findings dropped by the
    reviewer's own filters. A wrong finding shown costs a reader one
    line; a true one dropped costs a defect on `main`.
- **The judge stays where a block or a merge alone rests on it**
  ([ADR-0073]): independence is what lets either stand with no person
  ([ADR-0005]). A finder that reads is still one reading.
  - Its no still quotes the lines that refute, found again; else the
    finding is shown not verified ([ADR-0045]).
  - It reads the tree too, last: its material and served reads are
    input the engine prepares.
- **The documentalist's citations and notes go to the bench.** Every
  word removed is cited because an agent removed a true claim "it
  could not see backed". They stay only if, with the tools on, the
  bench shows a true sentence removed without them.
- **Not decided here**: the judge's model and effort (40% of fresh
  tokens, by the analysis). The bench measures a cheaper one.

## Steps

- **One small pull request a step**: a conformance case first, the
  switch, the deletion of what it leaves dead.
- **The maintainer's code budget until 1.0**: each is net 0 lines of
  non-test Go or fewer, and adds no setting. Step 1 lands with step 2
  if it cannot be alone.
- **Measured on the replay bench**, told in its `## Tried`. A bar not
  met: not merged, the picker stays, this ADR is amended.
- **A setting goes** when deleted code was its last reader; a role's
  pages on parts and cuts go with the code.

| Step | Switch | Deleted (non-test Go lines at `af68b1bb5`) | Bar on the bench |
|---|---|---|---|
| 0 | The bench: the reviewer's 28 points, the 7 claims, the 5 kinds the maintainer corrected, the readied issues that could not be built | | Today's input is the baseline |
| 1 | The lenses on the run on ready read the tree; a finder's question | | 2 of [#2188]'s 3 defects; more points than the baseline, no more false findings |
| 2 | Every lens call: the plain diff and the tree; the quote an address | `roles/reviewer/code/parts.go`, `funcctx.go` (580); `marked` in `diff.go`; the material in `reviewer.go`; `code-lines-max`, `parts-max` | Step 1's, on a change read in two parts today; [#1113] and [#1975] ask nothing more |
| 3 | The documentalist reads; `checked` by the log | `roles/documentalist/code/parts.go` (1,376); the packed sources and the change-alone task in `documentalist.go`; `whole-chars` and four parts settings | 3 of the 7 claims on Sonnet; no `checked` moved past the log |
| 4 | Its citations and notes, tools on | `cite.go`, `notes.go` (1,061), if the measure says so | No true sentence removed without them |
| 5 | The product owner reads; the kind on the whole body | The picker's call (`askOf`, `writeCode`, `needed`), `kindBodyMax`; `code-lines-max`, `code-bytes-max` | More of the 5 kinds right; real paths in a Scope readied empty |
| 6 | The judges, the reviewer's and the tester's, read the tree | `roles/reviewer/code/reach.go`, `reads.go` (958); `ServeReads`; four judge settings | 32 of the judge's 40 replayed questions right, or more ([ADR-0045]) |
| 7 | The tester's seed and the spec review's material go | `internal/kit/codepick`, `internal/codenav` (3,208); the packs' code-navigation rows | The tester's cases, no worse |

- **In whole files**: 6,122 lines, 7,183 with step 4. The analysis
  counts more with what `reviewer.go` and `documentalist.go` hold:
  about 3k to 4k for the reviewer, 5.9k for the documentalist.
- **Why this order**: the reviewer has the clearest measure; the
  documentalist the lowest score and the largest deletion; the judges
  work today, and move for the code they free.

## Cost

- **Reviewer**: about 8k fresh tokens more a pull request, for the run
  on ready (the analysis's estimate). At the evaluation's 63 pull
  requests in two days: about 0.5M. Each push pays it from step 2.
- **Documentalist**: cheaper than today, by the same analysis. A doc
  in parts costs 130k to 250k tokens
  ([measured](../research/token-costs-measured.md#documentalist)).
- **Product owner, judges, and the time of a run on ready** against
  its wait budget ([ADR-0050]): not measured. Each step says them.

## Risks

- **An agent that wanders**: turns, tokens and time are capped; the
  task says where to start; the log shows what it read, for the
  auditor too.
- **Prompt injection from the repository.** A file is data from
  whoever wrote the change, as today's diff is; the tree widens it.
  - The tools only read: no shell, no network, no write, no MCP server.
  - The tree holds tracked files only; no forge token is in it, nor in
    the agent's environment. The project's agent settings are not
    loaded.
  - The answer is a proposal: the forge's guards decide (principle 6).
  - Left: a planted text may talk a finder out of a finding, and a fix
    that merges alone counts on its silence. A comment in a diff can
    today.
- **A read is not a proof**: the log says a source was opened, not
  understood. The weekly sample of docs vouched for stays.
- **Conformance stays deterministic: it calls no real agent.**
  - `fake:` returns its proposals without reading
    ([the fakes](../spec/conformance.md#the-fakes)).
  - Tools are faked as the tester's case does: a stand-in `claude`
    (`fixtures/claude-reads`) refuses any flag but the read tools,
    checks its tree and commit, reads, and answers a recorded stream.
  - A case on what a picker put in a task goes with its picker.
  - How well an agent reads is the bench's question: a real agent, a
    rate over several runs.
- **A `cmd:` agent with no file tools reads less than today.** The
  role's page says so.

## What it amends

Each at its step; the ADR amended is edited then.

| Decision | Becomes | Step |
|---|---|---|
| [ADR-0020](0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md#amendment-2026-10-17-446-a-change-too-large-read-in-parts): files whole or by hunks, a change in parts, a quote not found dropped | superseded | 2 |
| [ADR-0009](0009-docs-far-behind-are-judged-in-parts.md): docs judged in parts | superseded whole | 3 |
| [ADR-0012](0012-a-doc-is-judged-against-its-sources-whole.md): sources whole in the task | the aim stands; the packing goes | 3 |
| [ADR-0014](0014-checked-is-earned-by-what-was-read.md#0-the-engine-refuses-an-unearned-checked-now): `checked` earned by what the task gave | earned by the log; its citations by step 4's measure | 3, 4 |
| [ADR-0053]: a source names a part of a file | stands for detection; no task carries the part | 3 |
| [ADR-0039](0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md): the kind's light read | the body whole | 5 |
| [ADR-0045]: the judge asks, the engine serves | superseded, but what a no must hold | 6 |
| [ADR-0063 §8]: one shared picker; a `cmd:` agent sees no tree; [§3](0063-every-open-issue-is-due-waiting-or-done.md#3-the-input-of-a-step)'s files "the picker gave" | superseded; its read tools (its step 9) built here | 1 to 7 |
| [ADR-0065](0065-code-only-what-converges-language-knowledge-in-packs.md#the-four-changes): the picker frozen, its tables in packs | deleted | 7 |
| [ADR-0064](0064-size-is-a-target-not-a-wall.md#amendment-2026-11-08-a-cut-is-not-a-crossing): a cut of the code, a code map postponed | no cut left; the map dropped | 7 |

Principle 3 stands: each role reads only what it needs. It now
chooses; its context budget is its cap.

## Not chosen

- **A larger fixed context**: what a change breaks elsewhere is not
  known before reading, and every call pays for it.
- **A stronger model on today's input**: replayed, no better.
- **The engine serving the reads an answer asks** ([#2162]): 0 of 3,
  and a protocol of workline's own to keep.
- **A new finder beside the lenses** ([#2162]'s draft): one more call,
  record and setting, every picker kept.
- **A shell**: the replays needed none; it widens what a planted text
  can do. The tester's worktree is its own decision.
- **The picker kept as a seed**: all its code stays. Step 7 measures
  the tester without it.
- **An index, or search by meaning**: a store and a dependency
  (principle 11).

[#1113]: https://github.com/JN0V/workline/issues/1113
[#1975]: https://github.com/JN0V/workline/issues/1975
[#2162]: https://github.com/JN0V/workline/pull/2162
[#2188]: https://github.com/JN0V/workline/issues/2188
[DomoticsCore#247]: https://github.com/JN0V/DomoticsCore/pull/247
[DomoticsCore#252]: https://github.com/JN0V/DomoticsCore/pull/252
[DomoticsCore#260]: https://github.com/JN0V/DomoticsCore/pull/260
[DomoticsCore#306]: https://github.com/JN0V/DomoticsCore/issues/306
[ADR-0005]: 0005-independence-takes-the-best-level-available.md
[ADR-0014]: 0014-checked-is-earned-by-what-was-read.md
[ADR-0045]: 0045-the-judge-asks-for-the-code-it-needs.md
[ADR-0050]: 0050-each-act-has-its-moment-and-a-wait-budget.md
[ADR-0053]: 0053-a-source-may-name-a-part-of-a-file.md
[ADR-0056]: 0056-the-developers-agent-works-in-a-sandbox-with-tools.md
[ADR-0063 §8]: 0063-every-open-issue-is-due-waiting-or-done.md#8-one-code-picker-and-tools-where-the-agent-has-them
[ADR-0073]: 0073-a-task-that-changes-no-behaviour-may-merge-alone.md
