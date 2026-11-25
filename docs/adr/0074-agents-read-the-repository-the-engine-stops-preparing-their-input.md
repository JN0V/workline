# ADR-0074: Agents read the repository; the engine stops preparing their input

- **Status:** proposed, until the maintainer merges it; nothing built
- **In one sentence:** the reviewer's lenses, the documentalist and the
  product owner read the repository with read-only tools; the code
  that cut their input is deleted, step by step, each step measured
- **Date:** 2026-11-25
- **Proposed by:** the maintainer's assistant, from the root-cause
  analysis of 2026-11-25: seven independent analyses of the roles'
  scores, about 15 replays with a real agent; code read at `af68b1bb5`
  on `main`
- **Settles:** [#2188]; takes the place of the draft closed with
  [#2162]
- **Builds on:** [ADR-0005] (independence), [ADR-0056] (the agent's
  tool sets), [ADR-0063 §8] (the read tree, built for the tester),
  [ADR-0066], [ADR-0068] §2; principles 3, 4, 5, 6, 10, 12, 15
  ([PRINCIPLES.md](../PRINCIPLES.md))
- **Amends or supersedes, step by step, once built:** [ADR-0009],
  [ADR-0012], [ADR-0014], [ADR-0020], [ADR-0039], [ADR-0045],
  [ADR-0053], [ADR-0063] §3 and §8, [ADR-0064], [ADR-0065]
  ([what it amends](#what-it-amends))
- **Research:** [a fixed context, or the model reading](../research/judge-reads-code.md#a-fixed-context-or-the-model-reading)

## Context

- **No role is at 8 of 10**, the maintainer's bar to leave beta. The
  evaluation of 2026-11-25, on three repositories since v0.89.0:
  reviewer 5, product owner 5, documentalist 4.
- **Their first cause is the same: they do not see the repository.**
  Each gets a task a Go picker cut for it, and `claude` with no tools
  (`--tools ""`, [claude.go](../../internal/agent/claude.go)). Only the
  developer ([ADR-0056]) and the tester ([ADR-0063 §8]) read files.
  - **Reviewer**: about 12 of the 28 points the maintainer raised
    later were never raised; 33 of 63 pull requests were read by
    hunks, not whole files.
  - **Documentalist**: it finds suspect docs reliably and repairs
    almost none. Three pull requests stamped 5 docs and changed no
    line of prose; the same 3 spec pages were "left to a person" on
    19 pull requests.
  - **Product owner**: 5 of 28 issues it readied could not be built
    (an empty Scope among them); an issue not read in full gets its
    kind from its title and the first 600 characters of its body.
- **Replayed on the same cases, with a real agent:**

  | Case | Today's input | Reading the repository |
  |---|---|---|
  | Reviewer: 3 defects outside the lines changed ([DomoticsCore#260], [#252][DomoticsCore#252], [#247][DomoticsCore#247]) | 0 of 3 on Sonnet; Opus at high effort, no better | 3 of 3 on Sonnet at medium effort, about 20k tokens |
  | Documentalist: 7 wrong claims in 3 docs ([DomoticsCore#306]) | 0 of 7 | 3 of 7 on Sonnet, 5 of 7 on Opus |
  | Product owner: an issue's Scope | not replayed | found |

- **The model is not the cause**: a stronger model on the same cut
  input recovered nothing.
- **Reads served by the engine were tried on the same three defects**
  ([#2162], closed): a finder asked as a judge's question, its reads
  served within the judge's caps ([ADR-0045]). 0 of 3, at 35k to 63k
  tokens in each; on two it read the file holding the defect.
- **Preparing input is much of the engine's growth**: non-test Go
  went from 66.9k to 121k lines in a month; 316 fixes added 23.2k
  lines and deleted 2.8k.
- **Review products made the same move**, from a fixed context to the
  model reading with tools ([research](../research/judge-reads-code.md#a-fixed-context-or-the-model-reading)).
- **The mode exists.** The tester's `claude` runs in a read tree with
  `Read,Glob,Grep`, no shell, and what it read is logged
  (`claudeReadTools`, `ReadTree`; case
  `tester/write-claude-reads-the-code-itself`).

## Decision

**The reviewer's lenses, the documentalist and the product owner read
the repository themselves, read-only, in a copy at the commit they
judge. The engine gives them what only it knows — the change, the doc,
the issue — and deletes the code that chose what else they read.**

### One way to read, for every role

- **The read tree of [ADR-0063 §8], as the tester has it**: a checkout
  at one commit, `Read,Glob,Grep`, no shell, no edit, no network, the
  role's prompt alone. Each file read is logged with the call, its
  path and blob.
- **The task holds what the tree does not**: the change, the doc's
  name and its sources' names, the issue. Never code the engine
  picked.
- **Capped by what exists**: the run's `ai-max-tokens`, the answer's
  time, and its turns, capped as the tester's are (40 today) and told
  in the task. A call a cap cuts is a failed call, said; never "no
  finding" (principle 12).
- **Each agent**:
  - `claude`: its tools are the engine's choice, as above;
  - a `cmd:` agent is run in the tree and told its path. Whether it
    reads, and read-only, are its command's promise, as in a worktree
    ([role contract](../spec/role-contract.md#one-run)); the tree is a
    throwaway copy;
  - no agent: the checks alone, as today (principle 5).

### The reviewer

- **The lenses' call on the run on ready comes first**
  ([ready](../../roles/reviewer/docs/ready.md)): the whole change as
  `git diff`, what the author says, the issue, and the tree at the
  head. Then each push after ready, the same way on its new commits.
- **A finder's question** ([#2188]): what does this change break, here
  or elsewhere in the repository? Not the judge's "yes without a
  doubt".
- **No new finder beside the lenses**, no record of its own, no
  setting: [#2162]'s draft is not taken. The call that exists gets the
  tools.
- **No parts**: a change is one call. A diff past the context budget
  is a file in the tree, read as any other. A call its caps cut is
  `review-not-whole`, as a change past `parts-max` is today.

### The documentalist

- **It reads the doc whole and the sources it needs**, at the head.
  The task names the doc, its sources as its header lists them, the
  files changed since `checked`, and the commit `checked` must name.
- **`checked` is still earned by what was read** ([ADR-0014]), now
  from the log: it moves only when the log holds every source file
  the doc names. A task that gave a file never proved it was read.
- **No doc is too large for an agent**: none is "left to a person"
  for its size.

### The product owner

- **It reads the tree at the default branch**, and the issue whole,
  cut only as a full read cuts a pasted log.
- **The kind is judged on the whole body.**
- **The Scope is found by reading**: no "name the code, be read
  again".

### What the engine deletes

Lines of non-test Go, counted at `af68b1bb5`. The analysis estimates
more, with what `reviewer.go` and `documentalist.go` hold of the
same: about 3k to 4k for the reviewer, 5.9k for the documentalist.

| Step | Deleted | Lines |
|---|---|---|
| [2](#steps) | `roles/reviewer/code/parts.go`, `funcctx.go`; `marked` in `diff.go`; the material `ask` builds in `reviewer.go`; `code-lines-max`, `parts-max` | 580 in the two files |
| [3](#steps) | `roles/documentalist/code/parts.go` (the parts, their claims, `vouchFor`); the packing of sources and the change-alone task in `documentalist.go`; `whole-chars`, `judge-in-parts`, `parts-max`, `parts-max-per-run`, `parts-move-checked` | 1,376 in the file |
| [4](#steps), if the measure says so | `roles/documentalist/code/cite.go`, `notes.go` | 1,061 |
| [5](#steps) | the picker's call in `roles/product-owner/code/productowner.go` (`askOf`, `writeCode`, `needed`); the clip in `kinds.go` (`kindBodyMax`); `code-lines-max`, `code-bytes-max` | not counted |
| [6](#steps) | `roles/reviewer/code/reach.go`, `reads.go`; `ServeReads` in `internal/contract`; `judge-lines-max`, `judge-read-rounds`, `judge-read-lines-max`, `tests-lines-max` | 958 in the two files |
| [7](#steps) | `internal/kit/codepick`, `internal/codenav`; the packs' code-navigation rows | 3,208 |

A setting goes only when deleted code was its last reader. Each
role's pages on parts, cuts and picked code go with the code.

### What stays, and why

**A check stays where it reads a fact that converges, or guards a
write. A filter born of a starved input goes, or proves itself on the
bench.**

- **The checks with no AI stay**: the reviewer's rules, the
  documentalist's detection of a suspect doc
  ([ADR-0053] included), a patch that must apply, in the files the
  role may write. They converge (principle 4).
- **The quote stays as an address, no longer as a filter.**
  - Found again: the finding is on its line, keyed as today.
  - Not found: the finding is shown on the change, on no line, said.
    It is no longer dropped (`finding-unfounded`), and needs no path
    of its own (`finding-misquoted`).
  - Why: the evaluation counts 3 true findings dropped by the
    reviewer's own filters. A wrong finding shown costs a reader one
    line; a true one dropped costs a defect on `main`.
- **The judge stays where a block or a merge alone rests on it**: an
  important finding that may block, the question of [ADR-0073], a
  role's code judged by another model.
  - Why: independence is what lets a block stand with no person
    ([ADR-0005]). A finder that reads is still one reading.
  - Its no still quotes the lines that refute, found again; else the
    finding is shown not verified, never dropped ([ADR-0045]).
  - It reads the tree as the finder does, at [step 6](#steps): its
    material and its served reads are input the engine prepares too.
- **The documentalist's citations and notes go to the bench**
  ([step 4](#steps)). Every word removed is cited because an agent
  once removed a true claim "it could not see backed": a starved
  input. With the tools on, they stay only if the bench shows a true
  sentence removed without them.
- **Not decided here**: the judge's model and effort. The analysis
  counts the judge at 40% of fresh tokens; the bench measures a
  cheaper one.

## Cost

- **Reviewer**: about 8k fresh tokens more a pull request, for the run
  on ready (the analysis's estimate, from the replays). At the 63 pull
  requests of the evaluation's two days: about 0.5M fresh tokens.
  Each push after ready pays the same once step 2 is built.
- **Documentalist**: cheaper than today, by the same analysis. A doc
  in parts costs 130k to 250k tokens today
  ([measured](../research/token-costs-measured.md#documentalist));
  parts go.
- **Product owner, judges**: not measured. Each step says its fresh
  tokens a call under `## Tried` ([ADR-0043]).
- **Time**: a loop of reads is slower than one call. The run on
  ready's time is measured against its wait budget ([ADR-0050]) before
  step 1 merges. Not measured yet.

## Risks

- **An agent that wanders.** Its turns, its tokens and its time are
  capped; the task says where to start; the log shows what it read,
  and the auditor's sample may read it ([ADR-0046]). A cut call is a
  failure, said.
- **Prompt injection from the repository.** A file is data from
  whoever wrote the change; today's diff already is. The tree widens
  it. What bounds it:
  - the tools only read: no shell, no network, no write, no MCP
    server;
  - the tree holds tracked files only, and a file outside it is
    refused; no forge token is in it, nor in the agent's environment;
  - the project's own agent settings are not loaded;
  - the answer is a proposal: the forge's guards decide (principle 6),
    and the reviewer never approves.
  - Left: a planted text may talk a finder out of a finding, and a fix
    that merges alone counts on its silence. It can today, from a
    comment in the diff. The intent lens still answers each scenario:
    silence is no yes there.
- **A read is not a proof.** The log says a source was opened, not
  understood. The weekly sample of docs vouched for stays
  ([ADR-0014], step 4).
- **Conformance stays deterministic: it calls no real agent.**
  - `fake:` returns its proposals without reading
    ([the fakes](../spec/conformance.md#the-fakes)): cases on what the
    engine does with an answer do not change.
  - Tools are faked as the tester's case does: a stand-in `claude` on
    the `PATH` (`fixtures/claude-reads`) refuses any flag but the read
    tools, checks it runs in a copy at the right commit, reads, and
    answers a recorded stream. The case checks the tree, the flags and
    the log in `out/calls.jsonl`.
  - A case that asserts what a picker put in a task is deleted with
    its picker; `pending.txt` never lies.
  - How well an agent reads is not conformance's question: the bench
    answers it, with a real agent, as a rate over several runs.
- **A `cmd:` agent with no file tools reads less than today**: the
  change, the doc's name or the issue alone. The role's page says so.
  The task is the same for every agent (principle 10); reading is the
  agent's.
- **Deleting before the proof.** No picker goes before its step's bar
  is met.

## Steps

Each step is one small pull request: a conformance case first, the
switch, and the deletion of what it leaves dead. By the maintainer's
code budget until 1.0, each is net 0 lines of non-test Go or fewer and
adds no setting. Each is measured on the replay bench, told in its
`## Tried`. A bar not met: not merged, the picker stays, and this ADR
is amended with the measure.

0. **The bench**, built beside this ADR: real misses replayed with a
   real agent. The reviewer's 28 points, [DomoticsCore#306]'s 7
   claims, the 5 kinds the maintainer corrected, the readied issues
   that could not be built. Today's input gives its baseline.
1. **The reviewer's lenses on the run on ready**: the tree, the
   finder's question.
   - *Bar*: at least 2 of the 3 defects of [#2188]; more of the
     bench's points than the baseline; no more false findings shown.
   - If it cannot be net 0 alone, it lands with step 2.
2. **Every lens call**: the plain diff and the tree. Parts, widened
   hunks and marked files go; the quote becomes an address.
   - *Bar*: step 1's, on a change that takes two parts today.
     [#1113] and [#1975] then ask for nothing more.
3. **The documentalist**: it reads; `checked` moves by the log. Parts,
   packed sources and "too large" go.
   - *Bar*: at least 3 of the 7 claims on Sonnet; no `checked` moved
     on a doc whose source the log lacks.
4. **The documentalist's citations and notes**, on the bench with the
   tools on: deleted or kept, by the measure.
5. **The product owner**: it reads; the kind on the whole body.
   - *Bar*: more of the 5 kinds right than the baseline; a Scope of
     real paths on the issues it readied empty.
6. **The judges**, the reviewer's and the tester's: the tree in place
   of their material and served reads.
   - *Bar*: on the judge's 40 replayed questions
     (`tests/evaluation/judge-replay`), no fewer right than 32
     ([ADR-0045's amendment](0045-the-judge-asks-for-the-code-it-needs.md#amendment-2026-10-14-on-by-default)).
7. **The picker's last callers**: the tester's seed, the spec review's
   material. Then the picker and the code navigation go.
   - *Bar*: the tester's bench cases, no worse without the seed.

**Why this order**: the reviewer holds the clearest measure; the
documentalist the lowest score and the largest deletion; the judges
work today, so they move last, for the code they free.

## What it amends

Each at the step that builds it; the ADR amended is edited then.

- **[ADR-0020]**: [each file once, whole or by hunks](0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md#amendment-2026-10-06-147)
  and [a change read in parts](0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md#amendment-2026-10-17-446-a-change-too-large-read-in-parts)
  superseded (step 2); a quote not found no longer drops a finding.
- **[ADR-0009]** superseded whole (step 3): no doc is judged in parts.
- **[ADR-0012]**: its aim stands, a doc judged against its sources as
  they are now; the packing under `whole-chars` superseded (step 3).
- **[ADR-0014]**: [step 0](0014-checked-is-earned-by-what-was-read.md#0-the-engine-refuses-an-unearned-checked-now)
  reads the log, not the task; [`checked` earned in parts](0014-checked-is-earned-by-what-was-read.md#amendment-2026-10-18-checked-earned-in-parts)
  superseded (step 3); [step 2](0014-checked-is-earned-by-what-was-read.md#2-the-mechanical-checks-no-agent)'s
  citations by step 4's measure.
- **[ADR-0053]** stands for detection: a doc is suspect only when the
  part it names changes. A task no longer carries that part.
- **[ADR-0045]** superseded (step 6), but
  [what a no must hold](0045-the-judge-asks-for-the-code-it-needs.md#what-a-no-must-hold-what-the-engine-checks).
- **[ADR-0063]**: [§8][ADR-0063 §8]'s shared picker superseded; its
  read tools for the product owner and the reviewer (its step 9) are
  built here; a `cmd:` agent is run in the tree; [§3](0063-every-open-issue-is-due-waiting-or-done.md#3-the-input-of-a-step)'s
  input keeps the log of reads alone, no file "the picker gave".
- **[ADR-0065]**: [the picker, frozen, its tables in packs](0065-code-only-what-converges-language-knowledge-in-packs.md#the-four-changes):
  deleted instead (step 7).
- **[ADR-0064]**: [a cut of the code](0064-size-is-a-target-not-a-wall.md#amendment-2026-11-08-a-cut-is-not-a-crossing)
  no longer exists; its postponed code map is dropped.
- **[ADR-0039]**: the kind's light read gives the body whole (step 5).
- **The [role contract](../spec/role-contract.md#one-run)**: "a `cmd:`
  agent never sees a read tree" (step 1).
- **Principle 3 stands**: each role reads only what it needs. It now
  chooses; its context budget is its cap.

## Not chosen

- **A larger fixed context** (higher caps, more files whole): what a
  change breaks elsewhere is not known before reading, and every call
  pays it, the right ones too ([ADR-0045], option A).
- **A stronger model on today's input**: replayed, no better.
- **The engine serving the reads an answer asks** ([ADR-0045]'s way,
  [#2162]): 0 of 3 on the same defects, and a protocol of workline's
  own to keep.
- **A new finder beside the lenses** ([#2162]'s draft): one more
  call, record and setting, and every picker kept.
- **A shell** (`git log`, a build): the replays needed none. It widens
  what a planted text can do, and needs a sandbox on each machine. The
  tester's worktree is its own decision.
- **The picker kept as a seed** for agents that read: all its code
  stays. Step 7 measures the tester without it first.
- **An index, or search by meaning**: a store and a dependency
  (principle 11); search by name was enough on the replays.

## Consequences

- **The engine shrinks**: at least 6,122 lines of non-test Go in whole
  files, 7,183 if step 4's measure says so, and up to 13 settings.
- **A role's quality rests more on its agent's tools.** Any agent with
  file tools reads the same tree (principle 10).
- **A change, a doc or an issue is never too large to be judged**; a
  call may still be cut by its caps, said.
- **What an agent read is on record**, for a person and for the
  auditor.

[#1113]: https://github.com/JN0V/workline/issues/1113
[#1975]: https://github.com/JN0V/workline/issues/1975
[#2162]: https://github.com/JN0V/workline/pull/2162
[#2188]: https://github.com/JN0V/workline/issues/2188
[DomoticsCore#247]: https://github.com/JN0V/DomoticsCore/pull/247
[DomoticsCore#252]: https://github.com/JN0V/DomoticsCore/pull/252
[DomoticsCore#260]: https://github.com/JN0V/DomoticsCore/pull/260
[DomoticsCore#306]: https://github.com/JN0V/DomoticsCore/issues/306
[ADR-0005]: 0005-independence-takes-the-best-level-available.md
[ADR-0009]: 0009-docs-far-behind-are-judged-in-parts.md
[ADR-0012]: 0012-a-doc-is-judged-against-its-sources-whole.md
[ADR-0014]: 0014-checked-is-earned-by-what-was-read.md
[ADR-0020]: 0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md
[ADR-0039]: 0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md
[ADR-0043]: 0043-a-try-is-told-in-its-pull-request.md
[ADR-0045]: 0045-the-judge-asks-for-the-code-it-needs.md
[ADR-0046]: 0046-the-auditor-re-checks-every-roles-acts.md
[ADR-0050]: 0050-each-act-has-its-moment-and-a-wait-budget.md
[ADR-0053]: 0053-a-source-may-name-a-part-of-a-file.md
[ADR-0056]: 0056-the-developers-agent-works-in-a-sandbox-with-tools.md
[ADR-0063]: 0063-every-open-issue-is-due-waiting-or-done.md
[ADR-0063 §8]: 0063-every-open-issue-is-due-waiting-or-done.md#8-one-code-picker-and-tools-where-the-agent-has-them
[ADR-0064]: 0064-size-is-a-target-not-a-wall.md
[ADR-0065]: 0065-code-only-what-converges-language-knowledge-in-packs.md
[ADR-0066]: 0066-a-role-mirrors-a-human-role-ai-first-where-code-is-rigid.md
[ADR-0068]: 0068-the-line-steps-back-on-its-own-drift.md
[ADR-0073]: 0073-a-task-that-changes-no-behaviour-may-merge-alone.md
