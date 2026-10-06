# Code navigation without a build — focused research (2026-10-06)

**Question.** How do code-navigation and code-review tools find a
function's bounds, what it calls and who calls it, with no build and in any
language — so that a reviewer's judge reads the code that proves or refutes
a finding (#127)?

**Verdict.** Two families. *Precise* navigation needs a build or a
compiler's index (SCIP, stack graphs, go/types). *Search-based*
navigation needs neither: a parser or a regular expression finds the
definitions, and plain word search finds the references, accepting a name
taken for another. Every tool that works across languages with no setup
is search-based. workline takes that: Go parsed with the standard library,
other languages by a definition pattern per extension and braces or
indentation for the bounds, references by `git grep -w` at the head; no
ctags nor tree-sitter, so the result is the same on every machine
(principles 10, 11; ADR-0001).

## Bounds and definitions

- **universal-ctags**
  ([ctags(1)](https://docs.ctags.io/en/latest/man/ctags.1.html)): a tag per
  definition, its kind and line; `--fields=+e` adds `end`, the line the
  object ends at. References only with `--extras=+r`, "limited to specific
  areas of specific languages"; no caller to callee link — its FAQ sends
  to GLOBAL, id-utils, cscope or cflow
  ([ctags-faq(7)](https://docs.ctags.io/en/latest/man/ctags-faq.7.html)).
  An external command: present on some machines, not on others.
- **tree-sitter tags queries**
  ([code navigation](https://tree-sitter.github.io/tree-sitter/4-code-navigation.html)):
  a grammar's `queries/tags.scm` captures `@definition.function` and
  `@reference.call` around a `@name`; `tree-sitter tags` runs them. One
  grammar per language, in C; the Go binding
  ([go-tree-sitter](https://github.com/tree-sitter/go-tree-sitter)) ships C
  sources (cgo) and no grammar by default, each one a module of its own.
- **Go's own parser** ([go/parser](https://pkg.go.dev/go/parser),
  [go/ast](https://pkg.go.dev/go/ast)): `ParseFile` reads one file, no
  build, no type check, a partial tree on a syntax error; a `FuncDecl`'s
  `Pos` and `End` are its exact bounds, its `CallExpr`s its callees — by
  name only: resolving them is `go/types`' work.
- **dumb-jump** ([jacktasia/dumb-jump](https://github.com/jacktasia/dumb-jump)):
  "jump to definition" for 60+ languages with no index: ag, rg or grep
  and a set of regular expressions by file extension, candidates ranked
  by heuristics (same file, then nearer).

## References and call graphs

- **Sourcegraph, search-based**
  ([search-based navigation](https://sourcegraph.com/docs/code-search/code-navigation/search_based_code_navigation)):
  definitions by ctags symbol search, references by case-sensitive,
  word-bounded plain text search filtered by extension; "text search and
  syntax-level heuristics (no language-level semantic information)",
  against precise navigation from compile-time data (SCIP)
  ([code navigation](https://sourcegraph.com/docs/code-search/code-navigation)).
- **GitHub code navigation**
  ([docs](https://docs.github.com/en/repositories/working-with-files/using-files/navigating-code-on-github)):
  tree-sitter, no configuration, 21 languages, repositories under 100k
  files. Search-based by default, every definition of the same name
  shown; precise for Python through stack graphs
  ([blog, 2021](https://github.blog/news-insights/product-news/precise-code-navigation-python-code-navigation-pull-requests/),
  [stack graphs](https://github.blog/open-source/introducing-stack-graphs/)),
  whose repository was archived on 2025-09-09
  ([github/stack-graphs](https://github.com/github/stack-graphs)).
- **Aider's repo map** ([docs](https://aider.chat/docs/repomap.html),
  [blog](https://aider.chat/2023/10/22/repomap.html),
  [repomap.py](https://raw.githubusercontent.com/Aider-AI/aider/main/aider/repomap.py)):
  tree-sitter definitions and references (ctags before), a graph of files
  ranked by PageRank, cut to a token budget (`--map-tokens`, 1k by
  default) by a binary search. The budget is the idea taken: what is
  shown is ranked, the rest left out.

## Review tools that read past the hunk

- **PR-Agent / Qodo**
  ([dynamic context](https://github.com/qodo-ai/pr-agent/blob/main/docs/docs/core-abilities/dynamic_context.md)):
  `allow_dynamic_context`, looking back up to
  `max_extra_lines_before_dynamic_context` (8) lines for the enclosing
  function or class; otherwise `patch_extra_lines_before` 3 and `_after` 1.
- **Greptile**
  ([graph-based context](https://www.greptile.com/docs/how-greptile-works/graph-based-codebase-context)):
  functions, classes and variables parsed, calls and imports linked; a
  review checks what a changed function calls and where it is called.
- **CodeRabbit**
  ([context engineering](https://www.coderabbit.ai/blog/context-engineering-ai-code-reviews)):
  a dependency graph built each review, definitions as context, "a 1:1
  ratio of code-to-context". Its code-graph documentation page was not
  found (404) and is not cited.

## What workline takes

| Taken | From |
|---|---|
| Search-based, no build: a definition pattern per extension, references by word search | Sourcegraph's search-based navigation, dumb-jump |
| Go's bounds and calls from its own parser, in the binary | go/parser, go/ast |
| A function's body by its braces, or by its indentation (Python), or to its `end` (Ruby, Lua) | dumb-jump's per-language rules, simplified |
| A candidate defined in the same file first, then in its folder; a name defined in many places named, not shown | dumb-jump's ranking; GitHub's "every definition of the name" |
| The function enclosing the change, not only lines around it | PR-Agent's dynamic context |
| What the function calls and who calls it | Greptile, CodeRabbit |
| A budget, the rest named | Aider's `--map-tokens` |

| Refused | Why |
|---|---|
| ctags or tree-sitter when installed | the same review would differ from one machine to the next (principle 10) |
| tree-sitter in the binary | cgo and a grammar module per language (principle 11, ADR-0001) |
| Precise navigation (SCIP, go/types, stack graphs) | a build or an index per language; the judge needs the code, not proof of the link |
