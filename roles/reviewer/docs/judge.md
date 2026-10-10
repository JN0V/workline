---
sources: [roles/reviewer/role.yaml, roles/reviewer/lenses, internal/builtin/reviewer]
checked: 4464b84
verified: agent:claude-code
---
# Reviewer — the judge, the verdict, the budget

Part of [the reviewer](../README.md): steps 7 and 8 of [a run](run.md),
and what a run may spend.

## 7. The judge

Apart, for each important finding, at the best independence
(`judge-at-least`); its level and both models said. A no drops it, said
(`finding-judged-no`).

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
- shell, Python, Ruby, Lua and C-like files (C, C++, Java, C#, JavaScript,
  TypeScript, Rust, Kotlin, Swift, PHP, Scala, Dart) by a definition
  pattern and braces, indentation or `end`;
- references by `git grep -w` at the head; a name defined in the cause's
  file first, then its folder; in more than two other places, named, none
  shown; test files left to the tests judge.

**Its question** is the lens's (the front matter of `lenses/<lens>.md`),
else whether the quoted code fails:

- correctness, edge cases: whether the code quoted fails as the finding
  says;
- tests ([#223](https://github.com/JN0V/workline/issues/223)): whether a
  test exercises the behaviour and would fail were it wrong. The judge is
  shown the test files (`tests`) in the cause's folder and those naming
  its file, whole up to `tests-lines-max`, the rest named; a no names the
  test;
- intent ([#126](https://github.com/JN0V/workline/issues/126)): whether
  the issue asks what the change does not do, or the change does what the
  Scope leaves out. The judge is shown the issue, and for a cause in the
  issue the whole merge request's change, commits reviewed before
  included, up to `code-lines-max` lines: a part an earlier push did is
  refused, not reported missing;
- claims: whether the code quoted contradicts the claim quoted;
- two findings on one line are each judged by their own lens's question
  ([#229](https://github.com/JN0V/workline/issues/229)); a refused one is
  dropped alone, the rest still shown.

## 8. The verdict, and what goes wrong

- The rules block (the long comment warns).
- What the lenses find warns (`ai-findings: warn`) until the evaluation
  has measured it ([#90](https://github.com/JN0V/workline/issues/90)),
  lens by lens ([settings](../README.md#settings)).
- A question for a person neither warns nor blocks, the summary counting
  it.
- Past `findings-max` on the change, or `issues-max` outside it, the rest
  is counted.
- One summary comment on a merge request, edited each run
  (`forge-writes`), a blocked run's too ([on a merge request](run.md#on-a-merge-request)).

When something fails:

- A lens whose answer does not read is asked again once, with what the
  YAML reader said (`promote-after`).
- A lens that fails is said (`lens-failed`), the commits left unrecorded.
- An agent unreachable: `blocked-external`.
- No agent: the rules alone, the change left for a person (`not-reviewed`).

## The budget

- **`ai-max-tokens`**, 200000 a run: once spent, nothing more is asked,
  said (`ai-max-tokens`); the call that crosses it is paid.
- **Not whole**: a lens not asked, or a finding not judged
  (`review-not-whole`), leaves the commits unrecorded, reviewed again next
  run.
- **Said**: the summary gives the tokens used against the cap; each
  call's, by lens and by judged finding, is in `out/review.json`.
- **A call**: `context.budget`, 100000 tokens, estimated 711 + 0.82 a
  character before the call
  ([#235](https://github.com/JN0V/workline/issues/235)); a larger prompt
  is not sent, the lens failing, said (`lens-failed`).

What a run cost when measured: [status](status.md#cost-measured).
