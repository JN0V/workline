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
