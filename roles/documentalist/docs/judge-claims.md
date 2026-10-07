---
sources: [internal/builtin/documentalist/judge.go, internal/builtin/documentalist/words.go, internal/builtin/documentalist/cite.go, internal/builtin/documentalist/comments.go]
checked: fcc8f36
verified: agent:claude-code
---
# Documentalist — the judge's claims

Part of [the judge](judge.md): what backs a word taken out of a doc. The
steps cited are those of
[ADR-0014](../../../docs/adr/0014-checked-is-earned-by-what-was-read.md);
DomoticsCore is a C++ project the role runs on in CI
([tried for real](tried.md)).

## Removed words need a claim

`removal-uncited` (step 2): the agent had removed a true claim it could
not see backed (workline d43b3f2).

- **What needs one**: a run of changed lines whose removed lines say a word
  its added lines do not, case aside. It needs one claim beside the patch
  whose lines reach it (three lines either way).
- **The two claims**:
  - `contradicted`, quoting a file under the doc's sources;
  - `gone`, naming a name the removed lines say, in the code when the doc
    was last edited and gone now, as `identifier-gone` finds it.
- **A claim that does not hold** — the quote not in the file, the file not
  a source, the name still in the code — is `citation-unchecked`.
- ***Withheld***, each run of changed lines refused (as
  `citation-unchecked` and `comment-not-evidence`): asked again, the agent
  had withdrawn a right fix with the refused one beside it (16c660b), and
  rarely brought the refused place back right — 1 of 5 re-asks, on
  DomoticsCore and the evaluation (steps 3 and 4).

### Several docs in one answer

- Each claim names its doc (`doc:`); the task says so.
- One that does not is read for the only doc whose patch changes the lines
  it gives, with its quote found under that doc's sources or its name in
  the lines removed there. A right claim for HeapTracker's pitfall named
  no doc beside a second doc's patch, and the right fix was withheld
  (DomoticsCore, step 4).
- A claim that could stand for no doc, or for several, is read for none
  and reported (`claim-unattributed`); the place it was for is refused,
  saying a claim was given without `doc:`.

### What needs no claim

- Words only added; lines rewrapped.
- A line count the engine found off, brought to the engine's number: the
  stated number and the word making it rough ("~", "about",
  "approximately"), in prose, a table's cell or a fenced listing. Another
  number in its place needs a claim like any word. A table's total is a
  count the engine gives too.
- Where the run replaces line for line, the words said around a count it
  brings to the engine's number: the count changed what the line says
  ("Watch the 800-line limit" became "Over the 800-line hard limit" beside
  930 lines, DomoticsCore, step 4).
  - A fact of the line's own still needs one: another number, a version, a
    name (quoted as code, shaped as one, or capitalised past a sentence's
    start), a negation, a quantifier, a conjunction or a tense.
- A line reworded taking out only glue ("the", "per", "currently"; never
  such a fact).

The task and every refusal say it, the engine's count being the evidence:
the agent had left the counts it was given, believing a count "cannot be
quoted as a source" (step 3).

### How words are compared

- With their typography plain: `'` and `’`, quotes, a no-break space,
  "1 000" and "1000" alike.
- Glue and fact words are the doc's language's, English or French: the
  project's `language`, else read from the doc's most frequent words. In
  French, an elided "l’" is a word of its own, and "a" (has) is a fact.

## A comment is not evidence

`comment-not-evidence`: a change whose claims' quotes are found in the
file only inside comments. A stale comment had won over the code's setting
([workline #29](https://github.com/JN0V/workline/issues/29)).

- **Comments are read by the file's type**, strings read as strings:
  - `//` and `/* */`: Go, C, C++, JavaScript and the like;
  - `#`: YAML, shell, Python, TOML, a Makefile;
  - a Python docstring (a string standing as a statement);
  - `--`: SQL, Lua, Haskell;
  - `<!-- -->`: Markdown, HTML, XML;
  - `{# #}` and `{% comment %}`: Django and Jinja templates; `{{! }}`:
    Handlebars;
  - a script without an extension: by the interpreter its `#!` names.
- **The comment is reported** (`comment-disagrees`, at the comment's file
  and line), kept for the run's verdict, so a person or the committer
  fixes it. *Withheld*: the place resting on it.
- **A type whose comments the engine does not know**: its quote is taken
  as code, and said (`comment-style-unknown`, once a type a run), so a
  comment there passing as evidence is not silent.
