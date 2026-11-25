---
sources: [roles/reviewer/role.yaml, roles/reviewer/prompts/lenses, roles/reviewer/code, internal/codenav]
checked: e81df7b34
verified: agent:claude-code
---
# Reviewer — the judge

Part of [the reviewer](../README.md): step 7 of [a run](run.md). Step 8, and what a run may spend: [the verdict](verdict.md).

## 7. The judge

Apart, for each important finding, at the best independence
(`judge-at-least`; `judge-at-least-by-role` raises it for a role's
commits, the developer's to `model`); its level and both models said.
A no drops it, said (`finding-judged-no`), when the lines it quotes are
found again ([below](#when-the-judge-reads)). One judged before, the same
finding on the same patch, keeps its verdict ([judged once](once.md)).

### What it reads

**What it reads**: the finding, the code it stands on, and what the change
did within 40 lines of the cause
([#147](https://github.com/JN0V/workline/issues/147)). The code
([#127](https://github.com/JN0V/workline/issues/127)) is found with no
build ([research](../../../docs/research/code-navigation.md)), up to
`judge-lines-max` lines all together, the rest named with where it lies:

- the function the cause lies in, whole (the symptom's too); none found,
  or longer than the cap, the 31 lines around it;
- then, each whole while it fits: the functions the finding names, those
  the cause's function calls, those calling it, and last a capitalised
  word opening a sentence that names a function;
- Go by its own parser, an unexported name looked for in its package only;
- other languages by a [language pack's row](../../../docs/packs.md#code-navigation)
  (shell, Python, C, C++, Java, Kotlin, Rust…): a definition pattern and
  braces, indentation or `end`; a C++ method defined out of its class
  (`void Class::name() {`) and a destructor too;
- references by `git grep -w` at the head; a name defined in the cause's
  file first, then its folder; in more than two other places, named, none
  shown; test files left to the tests judge.

### A code comment is not the spec

**A code comment is not the spec**: where the judge is shown the issue's
text, it is told that the issue and a person's answer in it outrank a
comment in the code.

- A comment calling a behaviour intended does not drop a finding against
  the issue's spec.
- It drops one only when no spec contradicts it, the issue being silent
  on the behaviour.
- For a cause not quoted from the issue, the judge is shown the issue
  for this; never the tests judge, nor a spec's judge.

### Its question

**Its question** is the lens's (the front matter of `lenses/<lens>.md`),
else whether the quoted code fails:

- correctness, edge cases: whether the code quoted fails as the finding
  says;
- tests ([#223](https://github.com/JN0V/workline/issues/223)): whether a
  test exercises the behaviour and would fail were it wrong. The judge is
  shown the test files (the tester's `test-paths`,
  [one definition](../../../docs/adr/0065-code-only-what-converges-language-knowledge-in-packs.md#3-one-definition-of-a-test-file))
  in the cause's folder and those naming
  its file, whole up to `tests-lines-max`, the rest named; a no names the
  test;
- intent ([#126](https://github.com/JN0V/workline/issues/126)): whether
  the issue asks what the change does not do, or the change touches a
  file beyond the Scope, listed by the author or not, that the Need does
  not need. The judge is shown the issue, and for a cause in the
  issue the whole merge request's change, commits reviewed before
  included, up to `code-lines-max` lines: a part an earlier push did is
  refused, not reported missing;
- claims: whether the code quoted contradicts the claim quoted;
- simplicity: whether the code quoted adds what the issue's Need and
  Verification do not need, and the smaller change the finding names
  closes the issue as well; shown the issue, as intent's judge; asked
  once the lens is calibrated: until then its findings are nits;
- two findings on one line are each judged by their own lens's question
  ([#229](https://github.com/JN0V/workline/issues/229)); a refused one is
  dropped alone, the rest still shown.

### A task's behaviour

- **Asked on a task**, where the project lets one merge alone
  (`merge: {autonomy: normal, alone: [task]}`) or asks in shadow
  (`merge: {shadow: [task]}`,
  [ADR-0073](../../../docs/adr/0073-a-task-that-changes-no-behaviour-may-merge-alone.md)):
  whether a user or caller can observe anything the change adds,
  removes or changes; its examples: a public interface, the toolchain,
  a chosen limit or value, a test fixture changed.
- **Shown** the issues it closes and the whole diff; past
  `diff-lines-max`, not asked.
- **Said** on the summary (`Behaviour kept`); a yes adds a hidden line
  by head, which `workline follow` reads. A no, unsure or no answer
  adds none: the task waits for a person. A yes or a no joins the
  findings judged, for the auditor's sample.

## When the judge reads

[ADR-0045](../../../docs/adr/0045-the-judge-asks-for-the-code-it-needs.md),
[#362](https://github.com/JN0V/workline/issues/362): on since 2026-10-14
(`judge-read-rounds: 2`), after
[its replay](../../../docs/research/judge-measured.md#replayed-2026-10-14);
`0` turns it off, a no then dropping the finding.

- **It may ask, instead of a verdict**: a file's lines, a definition by
  its name (found as its material's functions are), or the lines naming
  a word (`grep`).
- **The engine reads them** from git at the head reviewed: files the
  repository tracks, never the working tree nor a path outside. A read
  refused is said to the judge.
- **Caps**: `judge-read-rounds` rounds, `judge-read-lines-max` lines a
  judge, all rounds together; past them, cut, the rest named. A search
  past 50 places is not read. Then the judge is asked again; with no read
  left, it must answer.
- **A no quotes** the lines that refute the finding. Found again at the
  head, in what the judge was shown or read, the finding is dropped, the
  place said. Otherwise it is shown not verified (`judge-no-unquoted`).
  A finding that something lacks (a check, a test, an explanation of
  why) is refuted by lines that do it, or show it is not needed or not
  asked; the line it names refutes it only if that line does it or
  shows it is not needed. A guess is refuted by what the judge read
  showing it wrong. With no such lines, the judge answers yes only when
  what it was shown or read covers where it would be, else unsure,
  never no.
- **Unsure**: shown not verified, never dropped, never blocking
  (`judge-unsure`). So is a judge still asking to read past its caps, or
  one whose reads `ai-max-tokens` stopped.
- **First rounds first**: every judge answers once before any read is
  served.
- **Said**: what it read, beside its answer; each round in
  `out/review.json` (`reads`), its calls in `calls`.
- **The code's judges only**: a spec's judge reads nothing more.
