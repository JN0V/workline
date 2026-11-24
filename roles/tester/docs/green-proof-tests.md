---
sources: [roles/tester/code/green.go, roles/tester/code/runcache.go, roles/tester/code/frozen.go, roles/tester/code/renamed.go, roles/tester/code/tester.go]
checked: e81df7b34
verified: agent:claude-code
---
# The tests the green proof requires

Part of [the green proof](green-proof.md), and of [the tester](../README.md).

The ids the tester's record keeps, found new by its red proof, never
"the head's tests the base lacks": a change to a build file or a filter
that stops one running blocks, named
([ADR-0065, change 2](../../../docs/adr/0065-code-only-what-converges-language-knowledge-in-packs.md#2-scope-as-paths-a-hint-not-a-fence)).

## An id from a report, or a name from a file

- **An id from a report** must be in each head run, green.
- **A test whose build broke on the base** is kept as `file:name`:
  no report named it yet. A head id holding that name as a whole
  part is it (`calc.Name`, `Name[1]`, `Name/case`), qualified by:
  - the file's folders (a Go package), or
  - the file, after its last folders or nothing (`tests.test_x`), or
  - a class named after the file whose package leaves the folders
    aside (Kotlin, C#).
  - Not it: a nested folder's `Name` (`calc/sub.Name`), or one of a
    same-named folder elsewhere (`app/cmd/calc.Name` for
    `internal/calc`).
- **A test kept as its file alone** (its build broke, no
  `test-name` to name it,
  [#1293](https://github.com/JN0V/workline/issues/1293)): met by a
  head id of that file, its name after its last folders or nothing
  (`test_mul.main`, `tests.test_mul` for `tests/test_mul.c`); none,
  `frozen-test-missing`, or the judge's beside other new tests.

## When no report names it

- **A record with no class, failed, on the red run** is kept marked
  `errored`: maybe the runner's record of a file it could not collect
  (pytest's collection error), not a test case. Gone on the code, it
  goes to the judge too. So is one red by an error, its name in none
  of the tester's files, alone in its suite; beside the suite's
  tests, it is not frozen at all
  ([#1325](https://github.com/JN0V/workline/issues/1325)).
- **Not found by that name, while the head has other new tests**: the
  match is a guess, so the judge decides
  ([rule A](../../../docs/adr/0065-code-only-what-converges-language-knowledge-in-packs.md#rule-a-a-guess-may-only-pass)).
  - It reads the ids, the head's new tests, the frozen files, and the
    change beside them (build and runner files among it).
  - "Ran under another name" (a display name): passes, said.
  - "Dropped": `frozen-test-missing` blocks.
  - Unsure, failed, or no agent: `frozen-unproven`, a warning for a
    person.
  - Either way it is written on the pull request, in the tester's
    comment there.
- **No new test the head could run it as**: it blocks, no judge.

## Older records, and a rewrite

- **A record written before the ids were kept**: the head's tests
  the base lacks, as before, said by `frozen-ids-unknown`.
- **A rewrite on an objection** moves the ids: the file's old ones,
  from one run of the base with the file as frozen, come off; the
  rewrite's go on.
