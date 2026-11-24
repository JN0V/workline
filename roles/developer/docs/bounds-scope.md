---
sources:
  - internal/work/scope.go
  - roles/developer/code/bounds.go
  - roles/developer/code/developer.go
checked: 7c650fc50
verified: agent:claude-code
---
# The bounds: what the Scope holds

Part of [the bounds](bounds.md). A file the Scope does not hold is
beyond it: applied and listed, never refused
([beyond the Scope](bounds.md#beyond-the-scope)).

## Paths and sources

- the paths and globs in backticks in the issue's Scope
  ([ADR-0065, change 2](../../../docs/adr/0065-code-only-what-converges-language-knowledge-in-packs.md#2-scope-as-paths-a-hint-not-a-fence)):
  - a path, a line after it taken off (`parts.go:943`); a bare file
    name read as the one tracked file whose path ends with it;
  - a glob as the files it matches, new ones too (`core/session/**`,
    `src/calc/*.sh`), never its whole folder;
  - a new file's path, on a line `New:` as the product owner writes it;
  - a path with a `/` only when it has a file's extension, ends with
    `/`, or its first folder is the repository's (`dorny/paths-filter`
    or `actions/checkout@v4.1`, an Action, holds nothing);
  - never a name alone: a class (`TracingViewModel`) or a module
    (`core:session`) holds no file. Its code is beyond the Scope;
- the paths in plain words in the Scope that the repository holds, a
  file or a folder (`docs/ci.md, internal/doctor`);
- the product owner's `sources` for the issue;
- the tests beside them, new or not: a test file in a Scope file's
  folder or under a Scope folder (`mul_test.go` beside `mul.go`). A
  test file is one the project's
  [one definition](../../tester/docs/settings.md) names: the tester's
  `test-paths`, else the listed [packs'](../../../docs/packs.md) with
  the built-in generic patterns, else those alone
  ([ADR-0065](../../../docs/adr/0065-code-only-what-converges-language-knowledge-in-packs.md#3-one-definition-of-a-test-file)).
  A Scope `src` holds no test at the root.

## A new test file beside the Scope

The Scope also holds a new test file beside it
([ADR-0063, §9](../../../docs/adr/0063-every-open-issue-is-due-waiting-or-done.md#9-the-developers-write-fence)):

- a test file by that definition, not in the commit the rules are
  read at;
- its home, its code's folder by the tester's `test-homes`, is a
  Scope path's folder, holds one, or is in one:
  `app/src/test/kotlin/x/FakeRepo.kt` beside a Scope
  `app/src/main/kotlin/x/Repo.kt`, with
  `{tests: "**/src/test/kotlin/**", code: "**/src/main/kotlin/**"}`;
- unset, the listed packs' homes, then the generic ones: the folder
  holding its `test` or `tests` folder, else its own:
  `Storage/test/test_name/test_name.cpp` beside a Scope
  `Storage/include/Storage.h`;
- a test the definition misses is code: beyond the Scope, listed;
  `workline doctor` says how to set `test-paths`;
- one named like a class of the main code is beside it too: the
  reviewer's [`tests` lens](../../reviewer/prompts/lenses/tests.md) judges
  whether it replaces that class when the tests run (a JVM
  classpath);
- the root is a home only to a Scope file at the root: a Scope
  `src/x.py` holds no `tests/test_x.py`;
- a test written so stays beside the Scope in the next rounds, its
  record naming it, though the pull request's head holds it;
- a folder pattern takes any new file under it: `*/test/**` takes a
  `test/CMakeLists.txt` or a `conftest.py` too; name files
  (`**/test_*.cpp`) for tests alone;
- said and folded in: listed on the pull request and on the issue
  (`workline:scope-kept`), added to the Scope at the product owner's
  next refine;
- an existing helper or fixture a frozen test may read, away from
  the Scope's folders, is never one: it is beyond the Scope; the
  reviewer checks a listed helper that sets up the test runner
  (Kotest's `ProjectConfig`): it changes how every test runs.

## Never in the Scope

- **The root**: a Scope or source naming the repository's root
  (`.`, `./`): it is dropped, else every file would be in it.

