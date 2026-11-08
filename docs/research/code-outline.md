# A file's declarations and its imports, with no parser — focused research (2026-11-05)

**Question.** How do tools give an agent a file's declarations without
their bodies, and follow what a file includes or imports, in any
language and with no parser per language — so that the tester gets what
its test builds against within `code-lines-max`?

**Why.** On the C++ copy's #36, the tester got the functions the issue
names whole, then the files it asked for, within 1000 lines. At each of
three runs it asked for a new set: the HAL headers, then the stubs those
include and an events header, then `MQTT.h` whole. It wrote no test
([#1106](https://github.com/JN0V/workline/pull/1106), `## Tried`). A
panel of three chose: the include and import closure, two steps deep
inside the project, as declarations only; the nearest test's imports
first; the tester told to ask for every missing file at once. Not a
larger `code-lines-max`, not another ask, not tools
([ADR-0063](../adr/0063-every-open-issue-is-due-waiting-or-done.md#working-agreements)).

**Verdict.** Every tool that outlines code for a model either parses
(tree-sitter: aider, Repomix) or matches lines by a table of patterns
per language (ctags' optlib, editors' indentation folding). workline
keeps its own rule of
[code navigation](code-navigation.md): no parser, no outside command.
Bodies are cut by brace depth or indentation, one rule a family; a
block is kept when its header names a container (`class`, `struct`,
`namespace`, `interface`, `impl`, …), a list of words; imports are one
row of data per family of extensions: a pattern, how a name becomes a
path. A language missing from the table imports nothing followed and
keeps its files whole or cut, as before.

## Declarations without bodies

- **aider's repository map**
  ([repomap](https://aider.chat/docs/repomap.html),
  [building it with tree-sitter](https://aider.chat/2023/10/22/repomap.html)):
  each file's key definitions, "the critical lines of code for each
  definition", the rest shown as `⋮...`; files ranked on a graph whose
  edges are dependencies, the most referenced first, within
  `--map-tokens` (1k by default). Built with tree-sitter and a
  `tags.scm` query per language; it replaced a map from universal-ctags,
  which users had to install.
- **Repomix `--compress`**
  ([code compression](https://repomix.com/guide/code-compress)):
  keeps signatures, interfaces, types, class structures; drops function
  bodies, loops, conditions; bodies shown as `⋮----`. Tree-sitter,
  "experimental".
- **Editors' folding**
  ([VS Code](https://code.visualstudio.com/docs/editing/codebasics)):
  "folding regions are by default evaluated based on the indentation of
  lines"; a language that ships a folding provider replaces it. The
  language-agnostic default is indentation, no parser.
- **ctags' optlib**
  ([universal-ctags](https://docs.ctags.io/en/latest/optlib.html)): a
  language defined in options, no C: `--langdef`, `--map-<LANG>` to its
  extensions, `--regex-<LANG>` per kind of definition, and scope
  tracking by flags (`{scope=push}` on a class, `{scope=pop}` on `}`).
  A table of patterns, kept as data; the scope a stack of braces.

What workline takes: aider's and Repomix's output (signatures kept,
bodies as one mark), ctags' scope stack and editors' indentation for the
cut, no tree-sitter (cgo and one grammar module per language, against
[principle 11](../PRINCIPLES.md)). The cut, for every language of a
family:

| Family | Rule |
|---|---|
| Braces (C, C++, Go, Java, C#, JavaScript, TypeScript, Rust, Kotlin, Swift, PHP, …) | a block whose header holds a parenthesis or no container word is a body, its opening line kept, `… }` after it; where a type's header holds its parameters (Kotlin, Scala, Java, C#, TypeScript), a container word before them keeps it |
| Indentation (Python) | a header ending in `:` whose first word opens a body (`def`) is cut: its header kept, `…` after it; `class`, `if TYPE_CHECKING:` kept |
| A closing word (Ruby, Lua), shell | no outline: whole or cut, as before |

Each language is one row of `codenav.Languages`, as ctags' optlib keeps
one per language: its container words (Kotlin's `object`, Java's
`record`, Rust's `impl`), what a header holds that says nothing (an
annotation, `__attribute__((packed))`, `alignas(8)`, `template <…>`),
whether `#` opens a preprocessor line (C, C++, C#), whether a line ends
a statement (Go, Kotlin, Scala, Swift), and the pattern of the types a
type derives from or embeds (C#'s `: B`, Kotlin's `: B()`, Rust's
`impl T for`, Go's embedded fields).

Lines holding only a comment go too: the lines are the budget. A piece
the picker gives whole (a function the issue names) stays whole inside
the outline.

## Following imports

- **tree-sitter tags and aider** read references from the parse tree;
  the graph of files is theirs, by name.
- **Language servers and build tools** resolve imports exactly
  (`go list`, `tsc`, the compiler's include path): a build, a toolchain
  per language.
- **Pattern tables**: GitHub's
  [linguist](https://github.com/github-linguist/linguist) keeps
  languages as data (extensions, file names); ctags' optlib keeps
  patterns as data. Neither resolves an import.

What workline takes: a table, one row a family, in
[imports.go](../../internal/codepick/imports.go):

| Family | Pattern (group 1) | Resolved to |
|---|---|---|
| C, C++, Objective-C, Arduino | `#include "a.h"`, `<a/b.h>` | beside the file, from the root, else the one file whose path ends so |
| Python | `from .a.b import`, `import a.b` | `a/b.py` or `a/b/__init__.py`; leading dots climb |
| JavaScript, TypeScript | `from './a'`, `import './a'`, `require('./a')` | `./` and `../` only, with `.ts`, `.js`, `/index.ts`, …; `./a.js` written for `a.ts` |
| Go | `"host/mod/pkg"` in an import | past the prefix `go.mod` declares, the folder from the root, its files, tests aside; another prefix is from outside |
| Java, Kotlin, Scala | `import a.b.C;`; a type of its package, no import | `a/b/C.java`, `.kt`, … anywhere; the files of its package's folders it names ([#1109](https://github.com/JN0V/workline/issues/1109)) |
| Rust, Ruby, PHP, Lua, shell | `mod a;`, `require`, `include`, `source` | beside the file, else by its ending |

A package from outside resolves to no file of the project and is left
out. C#'s `using` and Swift's `import` name namespaces and modules, not
files: not followed.

## How much it saves

Measured on the C++ copy's headers, the outline against the file:

| Files | Lines | Declarations |
|---|---|---|
| `DomoticsCore-MQTT/include/DomoticsCore/*.h` (8) | 2,227 | 486 |
| `DomoticsCore-Core/include/DomoticsCore/*.h` (28) | 5,748 | 1,560 |
| `DomoticsCore-Wifi/include/DomoticsCore/*.h` (15) | 2,249 | 528 |
| `MQTT_impl.h` (out-of-class bodies) | 743 | 53 |
| workline's `internal/codepick/*.go` (8) | 1,963 | 243 |
| the copy's `tools/check_versions.py` | 315 | 28 |

A quarter of the lines in C++ headers, about a tenth in Go and Python,
blank lines left out. A file's declarations still cost: the
closure's files are ranked by the names they declare that the issue,
the test or the code given use and no other file declares, per line,
the test's own includes first; a file two steps away counts the names
the declarations one step away use. What does not fit stays a path,
"one step away".

## Limits

- **Braces that do not balance**, a regular expression's (`/\{/`) or the
  branches of an `#if` each opening one, give no outline: the file is
  given whole or cut, as with no outline, never the rest of it hidden.
- A container word after a parenthesis is a body, and before one too where
  a type's header holds no parameters (C's `struct foo *make(int x) {`):
  `class A extends
  mixin(B) {` keeps its members, `val x = object : T {` does not. An
  annotation's or attribute's parenthesis is
  dropped first.
- `#if` branches are all followed: a HAL header's ESP32, ESP8266 and
  stub includes are all candidates; the ranking puts the used ones
  first.
- A Go import names a package: all its files are candidates, a big
  package many of them.
