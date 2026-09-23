# Committer, release manager, documentalist — existing tools

**Verdict.** Commit and release tooling is mature; only a check on the *content*
of messages is missing. Documentation tooling detects drift from code to docs,
but nothing propagates technical changes to product docs, enforces size budgets
per document, or deduplicates.

## Committer

- No commit linter forbids a pattern out of the box. commitlint needs a local
  plugin; **gitlint** allows a negative-lookahead regex (T7/B8) but is slowing
  down. A ~30-line script is enough, run by a hook and again in CI.
- Format only: cocogitto `cog verify`, convco, committed, git-sumi.
- Hook managers: **lefthook** (Go binary), pre-commit (Python), husky (Node).
  The user's own tools use a global `core.hooksPath` with a dispatcher that
  hands over to the repository's hooks.
- The user's existing checks: `tools/forbidden-terms`, `tools/git-identity-guard`.
- [paepckehh/omc](https://github.com/paepckehh/omc) (Go): the LLM writes the message, a human
  signs and pushes.

## Release manager

| Tool | Forges | Notes |
|---|---|---|
| [git-cliff](https://github.com/orhun/git-cliff) | any | Rust binary; changelog from commits; `--bumped-version` |
| [releaser-pleaser](https://github.com/apricote/releaser-pleaser) | GitHub + GitLab | Go; release-PR flow like release-please |
| [changie](https://github.com/miniscruff/changie) | any | Go; changelog fragments, keeps explanations out of commits |
| cocogitto | git only | verify, bump, changelog, tag |
| semantic-release | both | Node, plugin-heavy |
| release-please | GitHub only | — |
| goreleaser | GitHub, GitLab, Gitea | publishing binaries |

Avoid: git-chglog (archived), go-semantic-release (stale).
Few projects focus on the release role; gstack `/ship` and claude-code-harness
`/harness-release` (changelog from verified evidence only) are the closest.

## Documentalist

**Link docs to code, detect drift (deterministic)**
- [mennanov/blockwatch](https://github.com/mennanov/blockwatch) (Rust): `<block affects="README.md:x">` markers; CI fails on drift.
- [Biaoo/docpact](https://github.com/Biaoo/docpact) (Rust): "changed X → these docs must be reviewed".
- [Ryu0118/docsync](https://github.com/Ryu0118/docsync): checksums tie docs to sources.
- [fireharp/coherence](https://github.com/fireharp/coherence) (Go): drift across code, docs, ADRs, tests; LLM pass is opt-in.
- docs-doctor, kontext: frontmatter `code:` globs, compare git history of code vs doc.
- [ocensis/doc-drift-agent](https://github.com/ocensis/doc-drift-agent): detection makes no model call; repair is separate.
- mdt, cog: single-source repeated blocks; `--check` fails on stale output.

**Maintain with AI**
- githubnext/agentics: `update-docs`, **`unbloat-docs`**, `glossary-maintainer`; draft PRs only, capped at 8 open.
- Delfini: findings typed *drift / additive / clarification*, clarification never auto-applied.
- [wei18/Upkeep](https://github.com/wei18/Upkeep): scheduled read-only drift audit, report only.
- langchain-ai/openwiki, DeepWiki-open: generate wikis — adds docs, the opposite of our goal.

**Prose and links**: Vale (Go), markdownlint-cli2, lychee (Rust).
**ADRs and cards**: adrs (Rust), MADR template, zk (Go, Zettelkasten notes).

**Lessons from Make My Dreams**: detection alone did not stop growth (a
1,757-line doc under a 200-line cap), because nothing forced action; counts
kept by hand drifted, so mechanical facts must be derived, and only intent
written by hand.
