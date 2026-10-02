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

## After fixes 1 to 3

Measured on the same copies, no agent, each mechanism from a throwaway
Go test as before.

- **Paths not ASCII**: the copy with `docs/déploiement.md` and
  `docs/plain.md` reports both suspect.
- **History**: backstage's docs taken as history go from 0 to 314: all
  296 of `docs/releases/` (240 `-changelog.md`, 56 `v1.x.0.md`), the 17
  of `architecture-decisions/` (16 numbered records and their
  `index.md`), and `docs/.release-notes-template.md`, a template, the one
  doc caught that is no record. Elsewhere: les-emplois
  `CHANGELOG_breaking_changes.md`, pip's 12 `research-results/`; httpx,
  prettier and ripgrep unchanged (their root CHANGELOG.md). Migration
  guides are left out on purpose: they say how to move now.
- **Decision records**: backstage's 16 are found, but none read as
  superseded: it says so in its title (`ADR013: [superseded] …`) and a
  `:::note[Superseded]` box, which no status reader knows. So
  docs/tutorials/corporate-proxy.md, citing ADR013, is still not
  reported: a status read from the title is still to do.
- **Comments**: Python files with a docstring now read as comment: pip
  490 of the 495 holding a triple-quoted string, httpx 39 of 39,
  les-emplois 322 of 363 (the rest are strings given to a value, which
  stay code); Django `{# #}` in all 109 les-emplois templates holding
  one; `{{! }}` in 16 of backstage's 155 Handlebars files and 13 of
  prettier's 92; every extensionless `#!` script by its interpreter
  but two of prettier's. Types still unknown, now reported when quoted:
  0.1% to 5% of the text files, mostly test data (les-emplois' 128
  `.ambr` snapshots, prettier's `.prettierrc` and fixtures).

## After fixes 4 to 8

The same copies, no agent, each mechanism from a throwaway Go test.

- **`whole-chars`**: docs whose sources fit, at 20,000 / 40,000 /
  80,000 / 160,000 characters, with the probe's sources: httpx 2 / 3 /
  3 / 3 of 23, prettier 6 of 27 at every cap, ripgrep 0 of 6, les-emplois
  3 of 9, pip 2 / 2 / 2 / 3 of 41, backstage 79 / 109 / 167 / 177 of
  524. Past backstage, a cap raised wins little: the probe's sources are
  whole folders, and one file alone is often over any cap. Narrower
  sources, or judging in parts, matter more than the cap.
- **Docs not read**, as `workline doctor` says them: pip 92 (91 `.rst`,
  1 README), backstage 388 (113 `.mdx`, 275 READMEs outside the globs),
  ripgrep 15 READMEs (its crates and benchmark runs), prettier 7 (1
  `.mdx`, 6 READMEs), les-emplois 2, httpx none. Test folders left out:
  prettier's 17 `.mdx` under `tests/format/` are its formatter's test
  data.
- **`value-left`**: a fix bumping each version of each doc line, one at a
  time. Prettier: 43 lines reported before, 39 of them markers ("_First
  available in v1.9.0_", "_Added in v2.3.0_", "until v1.13.0", "default
  value changed from `es5` to `all` in v3.0.0"); 4 after: the
  `"version": "1.0.0"` of three JSON examples and a blog link's
  `1.15.0.html`. httpx, ripgrep (its real 14.1.1 kept), backstage and pip
  unchanged in three parts. pip with `versions.pattern: '\d{2}\.\d+'`:
  its calendar versions read, 53 of them; 15 lines reported without the
  markers, 10 of them `{versionadded}`, "prior to pip 18.0" and kin, so 5
  with them, all in examples (a report's `"pip_version": "22.2"`, a
  progress bar's `26.2 MB/s`); 6 with `versions.files:
  [src/pip/__init__.py]`, one doc more sharing the version.
  Prettier's `vN` with `v\d+(?:\.\d+)*`: 4 lines, the same examples.
- **Removal rule**, on one-line changes of les-emplois' templates
  (5,403, from its whole history): refused before 3,183, 26 of them for
  an apostrophe alone and 14 for French articles alone; after, the
  language read from each line, 3,150, none for an apostrophe, 2 for an
  article (a line too short to tell French); with `language: fr`, 3,155,
  none for either — more refused than read line by line, French "a"
  (has) and "on" being facts now. English unchanged: httpx 127 of 189,
  pip 851 of 1,659, ripgrep 22 of 24, as before.
- **Sources by name** (`init --ai none`), history and the root README
  aside: httpx 3 of 24, prettier 5 of 26, ripgrep 0 of 5, les-emplois 2
  of 8, pip 3 of 30, backstage 261 of 452. Tests left out of the code
  matched (prettier's `docs/api.md` had matched a test's `API.js`). On
  backstage, a sample of 20: about half right (`connections/`,
  `gerrit/`, `kubernetes`, `search`, `techdocs`, `packages/cli/`), the
  rest a folder's generic name matched far away (`features/`,
  `provider/`, `plugins/`): a proposal for a person, as it is written.
