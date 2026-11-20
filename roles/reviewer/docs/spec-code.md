---
sources:
  - roles/reviewer/code/spec.go#askSpec
  - roles/reviewer/code/spec.go#specMaterial
  - roles/reviewer/code/spec.go#namedCode
  - internal/kit/codepick/names.go#NamedIn
  - internal/kit/codepick/names.go#codeWords
  - internal/kit/codepick/names.go#APath
  - internal/kit/codepick/codepick.go#Picker.codeNames
checked: e2e165c2d
verified: agent:claude-code
---
# Reviewer — the code a spec names

Part of [Reviewer — a spec before it is built](spec.md).

What the `contradicted` lens is shown, and its judge when a finding
quotes no code ([#425](https://github.com/JN0V/workline/issues/425)),
found by [the code picker](../../../docs/spec/code-picker.md) the
product owner and the tester share, up to `code-lines-max` (the judge's
`judge-lines-max`), the rest said
([in order](../../../docs/spec/code-picker.md#what-it-gives-in-order)):

- **The lines it names after a file**, and each definition of a name it
  writes as code, whole, read as Markdown reads it:
  - every word in backticks;
  - in a fenced block, a word called (`f(`) or holding a capital or an
    underscore;
  - in the prose, a word called, or shaped as a name (`answerItems`,
    `load_all`).
- **A name defined in a file it names** is that one: one of the same
  name elsewhere is counted, not shown.
- **The files it names**, whole while they fit; else by those pieces.
- **Then the lines naming a name**, a few a name.
- **Never a plain word**: a capitalised one (`Parse` opening a
  sentence, `When`) is English as often as a name. Write a function's
  name in backticks.
- **Never a path, a flag or a key**: `cmd/x/main.go`, `--all` and
  `code-lines-max` name no function.
