# Are the documentalist's checks generic? (2026-10-02)

ADR-0014's mechanisms were built and measured on three repositories of one
author: WaterMeter, DomoticsCore, workline (C++ and Go, docs in English,
many written by an agent). This probe ran them on six public repositories,
with no agent and no token spent: the engine at 49e1e65, each check called
from a throwaway Go test, shallow copies in a scratch folder.

| Repository | Commit | Kind | Docs (default globs) |
|---|---|---|---|
| encode/httpx | b5addb6 | Python, MkDocs | 26 |
| prettier/prettier | 8dafda77 | JS/TS, docs/ | 28 |
| BurntSushi/ripgrep | 3fce3b5 | Rust, docs at the root | 7 |
| gip-inclusion/les-emplois | 9257b83 | Django, docs in French | 11 |
| backstage/backstage | 42adf961 | TS monorepo, 1,951 `.md` | 767 |
| pypa/pip | a7002c97 | Python, Sphinx, calver | 42 |

## Adoption

`workline init` runs without an agent (`--ai none`): it routes `pre-push`
and lists each doc with no `sources`, nothing more. Declaring them by hand
is the only way without an agent; 767 docs on backstage. They were declared
here by a heuristic, deterministic and crude:

1. a name with changelog, release-notes, news, history, license, code of
   conduct, security, authors → `sources: []`;
2. README.md → every top-level folder holding code;
3. else the doc's stem, then its folders up to `docs/`, matched to a code
   folder or file of that name (case, `-`/`_`, a leading `_` aside; index,
   readme, overview, intro left out), the shallowest first;
4. else the folder holding the most code.

Matched by name (3): httpx 3 of 23, prettier 6 of 27, ripgrep 0 of 6,
les-emplois 3 of 9, pip 3 of 41, backstage 335 of 524 (docs named after
plugins). Outside a monorepo, sources need an agent or a person. Once
adopted, 22 of httpx's 23 docs are suspect at once: `checked` is each doc's
last commit, and a folder's code moved since.

What the default globs (`docs/**`, `*.md`) and the `.md` suffix hard-coded
in `loadTree` leave out, unsaid: pip's 68 `.rst` docs and its README.rst;
113 `.mdx` (backstage); 241 READMEs of backstage's packages and plugins
and ripgrep's 11 crate READMEs. Worse, `git ls-files` quotes a path that
is not ASCII, the file is then not found, and the doc is skipped in
silence: a copy with `docs/déploiement.md` and `docs/plain.md` lists only
the second. Headers went into Docusaurus and MkDocs frontmatters as they
are. The same quoting made prettier's emoji-named test files a folder
`"tests`; written in a header it broke the YAML, and one doc's header
that does not parse stops the whole role.

## Per mechanism

| Mechanism | Verdict | Evidence |
|---|---|---|
| `count-off` | generic where it reads, idle elsewhere | 0 reports in 6 repos, with the heuristic's sources and with every file a source. No doc states a file's line count: ripgrep's `Line count` tables count search output (rows name no source: silent), "~1000 lines" a JSX test (prettier blog), "1 000 lignes par fichier" a limit (les-emplois migrations.md:24). The shape is agent-written docs |
| `count-off`, other languages | fitted | "1200 lignes", "1200 Zeilen" not read; "1 200 lines" read as **200**; limit words English only |
| `value-left`, same doc | generic mechanics, false alarms of one shape | a fix bumping each three-part version, one line at a time: 444 of 610 fixes report another line. Prettier: 67 of 86 lines reported are "_First available in v1.9.0_" markers, 99 of its 274 versions |
| `value-left`, siblings | fitted | 0 in every repo: two docs must name the same *file* that says the new version; folders (any heuristic, most agents) never do |
| `value-left`, formats | fitted | three parts only: pip's calver `26.2` and `26.3.dev0`, its 33 `versionadded:: 26.x`, unseen; prettier's 55 and backstage's 627 `vN` unseen |
| `value-left`, the real case | rare | no repo restates its current version in prose docs. ripgrep README.md:360–361 still installs 14.1.1 (Cargo.toml: 15.2.0): a fix to one line reports the other, real |
| Removal rule | generic, heavy everywhere | one-line rewordings of the docs' history refused for want of a claim: httpx 102/161, pip 912/1,837, ripgrep 25/28, les-emplois 9/13 (all carry a fact: a link, a name, a count) |
| Glue words | fitted | French prose (les-emplois templates, 6,079 one-line changes): 30 refused for `'` → `’` alone, 7 for an article alone ("Le connexion" → "La connexion", "du" → "de l’"); "a" (has) and "on" pass as English glue; an elided "l'usager" is one word |
| Comments | fitted to C-like code | read as code: Python docstrings (495 of pip's 661 files, 39 of httpx's 60), extensionless shebang scripts, `.sql`, `.pyi`, `.hbs`/`.mustache` (153 on backstage), Django `{# #}` (136 of 614 les-emplois templates), `<script>` in `.vue`. A stale docstring wins as #29's comment did |
| `isHistory`, ADR folders | fitted | missed: backstage's 296 `docs/releases/v1.x.0-changelog.md`, its 17 `architecture-decisions/adr001-…` (also unseen by `cites-superseded`: folder and `NNNN-` both differ), les-emplois `CHANGELOG_breaking_changes.md`, pip's `research-results/` (12), migration guides (backstage 17). Recognised: each root CHANGELOG.md |
| 20,000 characters whole | generic, and severe | docs whose sources fit: httpx 2/23, prettier 6/27, ripgrep 0/6, les-emplois 3/9, backstage 79/524, pip 2/41. One code file alone is over it: 1.2% (prettier) to 19% (ripgrep, max 246k), 14% on pip. DomoticsCore fit 0 of 8 |

## Fixes, ranked

1. **Paths not ASCII** (bug, PRINCIPLES 12): `git ls-files -z` (or
   `core.quotePath=off`) wherever the engine lists files. Small; one case.
2. **History and decision records by setting**: `history` globs, the
   present list as default, plus any stem holding `changelog` or
   `release-notes`; ADR folders and file pattern settable
   (`architecture-decisions`, `adr001-`). Small; two cases.
3. **The comment test, wider and loud**: Python docstrings as comments,
   a shebang names the type, `--` (SQL, Lua), `{{! }}`, `{# #}`; a quote
   from a type it does not know said so in the finding, not taken as
   code. Medium; a `quoteIn` case each.
4. **The 20k cap a setting**, in tokens, scaled to the model's context;
   or judging in parts on by default. Small to build; its cost is tokens,
   measured on the evaluation first.
5. **Docs beyond `.md`, said**: report the doc-like files the globs skip
   (`.rst`, `.mdx`, package READMEs) as `not-tracked`; reading `.rst`
   needs a header form and link reading. Small to report, large to read.
6. **`value-left` markers and formats**: skip a version after "since",
   "available in", "changed in", `versionadded` as ranges are; the
   version pattern settable, or read from a `version-source` file the
   project names, which also gives siblings a shared file. Small–medium.
7. **Words by language**: apostrophes and no-break spaces made plain
   before words are compared (tiny, any language); glue, fact, limit and
   count words per `language` (en, fr shipped); thousands written with a
   space. Medium; low yield until a French repository adopts.
8. **Sources without an agent**: the name match above, offered by
   `init --ai none` as a proposal. Worth it on a monorepo only (335 of
   524 on backstage, 0–6 elsewhere).

`count-off` needs no fix: it is quiet where docs state no counts, which is
everywhere but agent-written docs.
