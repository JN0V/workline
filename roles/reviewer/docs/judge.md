---
sources: [roles/reviewer/role.yaml, roles/reviewer/lenses, internal/builtin/reviewer]
checked: 7327530a
verified: agent:claude-code
---
# Reviewer — the judge, the verdict, the budget

Part of [the reviewer](../README.md): steps 7 and 8 of [a run](run.md),
and what a run may spend.

## 7. The judge

Apart, for each important finding, at the best independence
(`judge-at-least`); its level and both models said. A no drops it, said
(`finding-judged-no`), when the lines it quotes are found again
([below](#when-the-judge-reads)).

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
- **Unsure**: shown not verified, never dropped, never blocking
  (`judge-unsure`). So is a judge still asking to read past its caps, or
  one whose reads `ai-max-tokens` stopped.
- **First rounds first**: every judge answers once before any read is
  served.
- **Said**: what it read, beside its answer; each round in
  `out/review.json` (`reads`), its calls in `calls`.
- **The code's judges only**: a spec's judge reads nothing more.

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
- A finding that blocks stands until a commit answers it
  ([below](#a-block-stands-until-a-commit-answers-it)).
- Each finding judged leaves a hidden line in the summary comment, every
  run's, for the weekly sample
  ([below](#every-finding-judged-kept-for-the-weekly-sample)).

When something fails:

- A lens whose answer does not read is asked again once, with what the
  YAML reader said (`promote-after`).
- A lens that fails is said (`lens-failed`), the commits left unrecorded.
- An agent unreachable: `blocked-external`.
- No agent: the rules alone, the change left for a person (`not-reviewed`).

### A block stands until a commit answers it

[#326](https://github.com/JN0V/workline/issues/326): the record keeps
each finding that blocked, its cause quoted, beside the commits reviewed
([the record](run.md), step 3).

- **Nothing new to read** — the job run again on the same commits, a merge
  of the base that settled nothing, new commits changing no code, a rule
  holding the run, no agent: the block stands. No agent is asked; the
  summary comment names the finding again, marked `**blocks**`.
- **Its file moved**: it stands, found and kept on the new path
  ([#358](https://github.com/JN0V/workline/issues/358)).
- **It goes** when a commit answers it:
  - its cause is no longer found on the change (the line changed, or
    removed);
  - or its lens reads the cause's file again, in a whole review, and does
    not find it, or its judge says no. Found again and verified, it is
    the new review's finding, shown once.
- **It stays past a commit elsewhere**: a lens that did not read the cause
  again leaves it. One quoted from an issue stays until its lens reads the
  issue again.
- **Its lens set back to `warn`** (`ai-findings`): it goes.
- **Resolving its comment on the line does not lift it**: a person who
  disagrees merges where the forge lets them, or sets its lens to `warn`.

### Every finding judged, kept for the weekly sample

[ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md):
the auditor re-checks a share of the findings the judge kept or dropped
([the weekly sample](../../../docs/weekly-sample.md)). The summary
comment hides a line for each.

- **A line**: the lens, the place, the head the review read, yes (kept)
  or no (dropped), the lens's model and the judge's, the title.
- **No line**: a finding shown not verified, a nit, a question for a
  person, a spec's finding.
- **Every run's**, for the merge request's life, on GitHub and GitLab:
  each run writes the comment again, the lines before kept, its own
  added. A run that asks no lens keeps them too, and one applied after
  another it never read keeps that one's
  ([#375](https://github.com/JN0V/workline/issues/375)). GitLab keeps no
  earlier version of a comment: the lines do.
- **Within 60,000 characters**, under GitHub's 65,536: past it, the
  oldest lines go first, and the comment counts them
  (`<!-- workline:judged-lines gone=N -->`). The sample says them not
  drawn. A line takes about 300 characters: about 200 fit.
- **On a machine**: no line; the local record keeps none.

## The budget

- **`ai-max-tokens`**, 300000 a run: once spent, nothing more is asked,
  said (`ai-max-tokens`); the call that crosses it is paid. Raised from
  200000 so a large review is judged whole when its judges read
  ([ADR-0045](../../../docs/adr/0045-the-judge-asks-for-the-code-it-needs.md#the-token-budget));
  a run spends what it uses.
- **Not whole**: a lens not asked, or a finding not judged
  (`review-not-whole`), leaves the commits unrecorded, reviewed again next
  run.
- **Said**: the summary gives the tokens used against the cap; each
  call's, by lens and by judged finding, is in `out/review.json`.
- **A call**: `context.budget`, 100000 tokens, estimated 711 + 0.82 a
  character before the call
  ([#235](https://github.com/JN0V/workline/issues/235)); a larger prompt
  is not sent, the lens failing, said (`lens-failed`).

What a run cost when measured: [research](../../../docs/research/code-review.md#the-reviewers-cost-measured).
