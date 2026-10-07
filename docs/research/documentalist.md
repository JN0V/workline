# Documentalist — focused research

**Verdict.** Drift from code to docs is well covered by small deterministic
tools. Three things are not covered by anyone: propagation from technical docs
to product docs, size budgets beyond one file, and splitting into cards. The
first has a proven model in requirements tools (*suspect links*); the other two
are ours to build.

## Mechanisms worth taking

| Tool | Mechanism |
|---|---|
| [Spexcode](https://github.com/shuxueshuxue/Spexcode) (303★) | A spec's frontmatter anchors code (`code: src/x.ts#fn`). Drift is read from **git history**: a commit after the spec's last edit touching the anchored lines. No stored hash, so no merge conflicts with many committers. |
| [doorstop](https://github.com/doorstop-dev/doorstop), [throughline](https://github.com/rhodium-org/throughline) | **Suspect links**: when a parent item changes, every link to it becomes suspect until reviewed; throughline cascades suspicion across the graph and keeps AI-made items `proposed` until a human ratifies them. |
| [docpact](https://github.com/Biaoo/docpact) (Rust) | Rules "changed X → review these docs". A review can be recorded **without editing the doc**. Baselines and waivers with expiry. `route`: which docs to read before coding. |
| [coherence](https://github.com/fireharp/coherence) (Go) | Normalized hash, so a typo fix is not a change. Docs citing a superseded ADR are flagged. Identical claims across docs are detected. LLM opt-in, capped at 3 calls. |
| [doc-drift-agent](https://github.com/ocensis/doc-drift-agent) | Truth classes per glob: is the code or the doc authoritative? Exit 2 = "can't tell", which fails. Blocking vs advisory findings. Never calls the forge itself. |
| [Delfini](https://github.com/Legends-of-Tech/delfini) | Findings typed drift / additive / clarification; clarification never auto-applied. **Checks the agent's quoted text really sits at the cited lines**, dropping hallucinated findings. |
| agentics `unbloat-docs` | One file per run, target ≥20% shorter, draft PR only, skipped when 8 docs PRs are already open (backpressure). |
| [docs-keeper](https://github.com/kostiantyn-matsebora/docs-keeper) | **Extract rather than compress** above ~200 lines: rewording plateaus near −10%, extracting to a companion doc reaches −60%. Generated per-folder `index.md`. |
| [openwiki](https://github.com/langchain-ai/openwiki) (16.7k) | Claims carry versioned evidence; only stale ones are rewritten; a clean update calls no model. Has GitLab CI examples. |
| [blockwatch](https://github.com/mennanov/blockwatch) (Rust) | Fine-grained code↔doc blocks; `line-count`, `keep-unique`. |
| throughline, [mdox](https://github.com/bwplotka/mdox) (Go), embedmd (Go) | Facts derived between markers, `--check` fails if regenerating would change the file. The fix for counts kept by hand. |

## Budgets, duplicates, cards

- **Vale metric rules** give a per-section word budget, deterministically.
- A Japanese team capped CLAUDE.md at 12 KB with a hook and the advice
  "relocate, don't delete": growth fell from +182% a week to about +1%.
- **jscpd v5** (Rust binary) finds duplicated Markdown prose and reports in
  GitLab Code Quality format. Near-duplicates need a MinHash, ~150 lines of Go.
- **Card format**: Google's OKF v0.2 — one concept per file, frontmatter `type`,
  `sources`, `verified` (`human:` prefix = reviewed by a person), `status`,
  `stale_after`; reserved `index.md`.
- **Over-splitting is a smell too** ("Fragmented" in Khan & Uddin's API-doc
  smells): budget a minimum card size, not only a maximum.

## Docs for agents

"Evaluating AGENTS.md" (arXiv 2602.11988): context files did not improve task
success and raised cost by more than 20%, whether written by humans or models.
Agent entry points should be short routing files — indexes pointing to cards —
not content dumps. Japanese practice converges on ≤200 lines at the root.

## Decay rules from research

- DOCER: an identifier named in a doc that existed at the doc's last edit and is
  gone from the code now is outdated. Cheap and deterministic.
- Google's SWE book: each doc has an owner and a freshness date.
- Superseded decisions (2026-09-27): coherence reads `supersedes:` in the
  successor's frontmatter, and reports a doc citing the old id, by link or by
  `ADR-###`, without naming the new one. adr-tools writes
  `Superseded by [N. Title](file)` under the old record's `## Status`; MADR puts
  `status: superseded by ADR-NNNN` in its frontmatter. Records live in
  `doc/adr` (adr-tools), `docs/adr` (log4brains) or `docs/decisions` (MADR).
- lychee 0.24.2 (2026-09-27): `--format json` gives, per file, each failed
  link with its line and its HTTP status code; a link not reached (no
  network, a proxy down) has no code, and a timeout goes to `timeout_map`.
  Exit code 2 when a link fails. Example domains (`example.com`,
  `.invalid`) are excluded by default, with no request made; `lychee.toml`
  and `.lycheeignore` are read from the working directory.

## Commercial doc agents (2026-09-29)

Missed by the first pass, which looked for mechanisms in open source. None
runs at commit or push: they act on the merge request or after the merge.

| Tool | Does | When, and how |
|---|---|---|
| [Swimm](https://docs.swimm.io/continuous-integration/github-app/) | The closest: detects when code a doc embeds changes; renames fixed silently, the rest proposed | A check on each pull request; accepted changes go in one commit of their own. Paid, docs in its own format |
| [Mintlify](https://www.mintlify.com/docs/agent/suggestions) | A docs host; its agent proposes updates | After a merge, as a pull request in the docs repository, reviewers taken from the code's authors. Paid, docs hosted there |
| [Dosu](https://dosu.dev/blog/august-2025-dosu-drop) | Names the docs a pull request concerns, updates them | A comment on the pull request, then a docs pull request after the merge. A third party with write access |
| [CodeRabbit](https://docs.coderabbit.ai/finishing-touches/docstrings) | Code review; docstrings on request | A follow-up pull request. Not Markdown docs kept true |
| [Promptless](https://github.com/Promptless/promptless.ai/pull/1057) | Proposed doc changes in a review queue, per file, to edit or reject | On a pull request, a merge, or a sweep of recent merges |

What none covers: working without AI and offline, on a machine and any
forge, with any agent; propagation from technical to product docs, cut at
the release; size budgets, duplicates, cards, derived facts. What to take:
detection without AI first, the AI only on what changed and never inside the
push, its changes in one commit of their own, reviewed per doc, the rest in
a merge request later.

## Judging a doc in parts (2026-09-29)

For a doc far behind sources too large for one task. Verification research
splits by claim: FActScore, SAFE, VeriScore, RefChecker break the text into
claims and check each against evidence, cut in chunks when too long
(MiniCheck: claim × chunk, supported if any chunk supports it). Splitting by
source is the same grid batched by chunk, with known failures: a claim true
only across two sources, a claim about code since deleted (every part says
"not here"), "not here" read as "false", the same line flagged by several
parts ([LLM×MapReduce](https://arxiv.org/abs/2410.09342)).

- **Aggregation**: [RefChecker](https://github.com/amazon-science/RefChecker)
  (434★), strict: any contradiction makes it wrong, all entailed makes it
  right, the rest neutral. For code, support must not outvote a
  contradiction: a deprecated path supports what the current one contradicts.
  LLM×MapReduce has each part return a structured answer — the facts, the
  reason, a confidence — merged in a last pass.
- **Claim extraction costs more than it gives**: MiniCheck's authors measured
  GPT-4 +0.3 points for 2 to 4 times the cost; Claimify, the best extractor,
  still loses 16% of the content. Line ranges of the doc serve as claims.
- **Whole or in parts**: accuracy falls as the context grows (lost in the
  middle, Chroma's context rot; FIND: recall −2.6% per 10k tokens of input).
  Divide and conquer wins when that loss grows faster than the dependencies
  between parts (arXiv 2506.16411), where a weaker model in parts can beat a
  stronger one reading the whole. Nothing settles our case: measure it.
- **Fan-out costs**: Anthropic's multi-agent research system spends about 15
  times a chat's tokens; [drift-detect](https://github.com/agent-sh/drift-detect)
  went back from several agents to one call, 77% fewer tokens; agentics'
  wiki writer forbids sub-agents and caches a summary per source file.
  [Delfini](https://github.com/Legends-of-Tech/delfini) drops the quotes an
  agent gives that are not at the lines it cites.
- **Small models as checkers** (LLM-AggreFact): gpt-4o-mini 74.0 against
  gpt-4o 75.9, MiniCheck-7B 77.4; short documents, one claim, no reasoning
  across distant evidence. On long documents larger models win (FIND). No
  Haiku figure published.
- **Measuring**: FIND plants inconsistencies in long documents, matches found
  against planted, and checks precision by hand — many "false positives" were
  real errors. Spread planted defects through the doc; FIND's cluster early.

Measured on DomoticsCore, the 40 docs left (2026-09-29): 305 source files,
806 pairs of a doc and a file. One call per file is out of reach; packing
files into parts that fill a task, 393 calls, about 3.2 million tokens in,
149 sources that are docs left out. Most go to a few docs naming whole
libraries (`docs/index.md`: 168 files); at most 8 parts a doc, 22 docs in 81
calls. Four docs are themselves too long to leave room for any source.

## Generated sections and parallel branches (2026-10-05)

A fact generated into a doc and required equal on every pull request makes
two pull requests that both change it conflict on its line (#149). What
others do:

- **Merge drivers** (`merge=union`, a custom driver in `.gitattributes`):
  GitHub, GitLab and Bitbucket do not run them when merging a pull request
  ([gitlab#18830](https://gitlab.com/gitlab-org/gitlab/-/issues/18830)),
  and `union` keeps both values of a count, so the line is wrong anyway.
- **Nothing shared on a branch, compiled later**: towncrier's news
  fragments and changesets' files are one per change, compiled into the
  changelog at the release: "there cannot be merge conflicts on the
  changelog because the changelog does not exist until you release"
  ([pyinstaller](https://pyinstaller.org/en/stable/development/changelog-entries.html)).
- **Written on the default branch by a bot**: the all-contributors bot
  opens a pull request of its own for README's table; release-please writes
  its `extra-files` only in its release pull request.
- **Computed when read**: a dynamic badge (shields.io) keeps no value in
  the file.
- **Equal on every pull request** (mdox `--check`, Kubernetes'
  `hack/verify-*` scripts): the conflicts and rebases are accepted.

Taken (ADR-0027): a branch's derived block may lag; gardening regenerates
it on the default branch, in a merge request of its own.

## Docs for their reader (2026-10-07)

Whether a user page serves its reader, beside whether it is true
([#193](https://github.com/JN0V/workline/issues/193)). What exists:

- **Vale**: an `occurrence` rule scoped to `paragraph` caps the words of a
  paragraph, and `metric` rules give readability scores. It reads the text
  with the markup taken out, so it cannot tell `ADR-0026` from a link to
  it, and knows nothing of the pages around one. A binary and rules to keep
  ([#95](https://github.com/JN0V/workline/issues/95)).
- **markdownlint**: line length (MD013), headings, lists; no word count,
  nothing across pages.
- **Pages no navigation reaches**: Sphinx warns "document isn't included in
  any toctree", MkDocs lists the pages "not included in the nav"; site
  crawlers call them orphan pages. Each reads its own navigation file.
- **A diagram where a page explains a flow**, a decision or an issue named
  without a link: no linter found.

Taken: the idea of the orphan page — reached from an entry point, link
after link — and Vale's paragraph cap, both computed in the engine as the
other checks are
([ADR-0003](../adr/0003-documentalist-checks-in-the-engine.md)): no binary,
and one pass sees links and pages together. Vale stays for a project that
keeps a config ([#95](https://github.com/JN0V/workline/issues/95)).
