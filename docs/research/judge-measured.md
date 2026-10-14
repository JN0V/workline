# The judge, measured on real use (2026-10-14)

**Question.** When the reviewer's judge drops a finding ("no") or lets it
be shown ("yes"), is it right? The measure the
[judge's status](../../roles/judge/docs/status.md) asks for its third
criterion ([ADR-0036](../adr/0036-a-roles-status-is-earned-by-written-criteria.md#amendment-2026-10-13)).

**Verdict.** Its "yes" are right; its "no" drop true findings, mostly
because what it was shown did not hold the proof. It refuses what it is
not shown, not what is false. The judge will read the code it needs
([#362](https://github.com/JN0V/workline/issues/362)) before the
reviewer is measured ([#360](https://github.com/JN0V/workline/issues/360)).

## Results

| The judge said | Drawn | Right | Wrong |
|---|---|---|---|
| yes (finding shown) | 20 | 19 | 1: a latent fragility, not a crash |
| no (finding dropped) | 20 | 12 | 8 true findings dropped |

- **Of the 8 dropped**, 6 negligible by the panel, 2 worth raising:
  - [#280](https://github.com/JN0V/workline/pull/280): a final block
    after a refused proposal turns a run green. Still on `main`:
    [#361](https://github.com/JN0V/workline/issues/361).
  - [#238](https://github.com/JN0V/workline/pull/238): `whole-chars` can
    push a task past the role's budget. Fixed later in the same pull
    request (fcc8f36, a cap of 25,000).
- **The 6 negligible**: two tests not written for a branch, a stray note
  read from the working directory, a harmless link to a closed issue, a
  test helper's leak, a description saying "never blocking" where one
  unset setting blocks, by design.
- **Why it refuses.** Of all 173 refusals, 140 say the material was not
  shown or the finding only guesses: their reason matches, case ignored,
  `guess|not shown|not given|never shows?|nothing (shown|given)|cannot be (checked|confirmed)`.
- **Those words do not tell right from wrong**: they are in 7 of the 8
  true findings dropped, both worth raising among them, and in 9 of the 12
  right refusals. Only reading the code settles it.

## Method

- **The population**: every judge's verdict in the reviewer's comments on
  workline's pull requests, read through GitHub's API, every version of
  each edited comment included; one per pull request, lens, title and
  verdict. Kept when its `file:line` exists at the commit reviewed.
- **229 verdicts** on 58 pull requests, 2026-10-04 to 10-13: 173 no, 56
  yes. Lenses: correctness 93, edge cases 78, tests 53, claims 4, intent 1.
- **The draw**: 20 "no" and 20 "yes", spread over lenses and pull
  requests, one a place, by a seeded random draw (seed 20261014).
- **The raters**: four agents, ten items each, given the pull request,
  lens, title, `file:line` and commit, never the judge's verdict nor its
  reason. Each read the repository at that commit (12 of 40 also its
  later history) and said true or false, with its evidence.
- **Agreed**: rater and judge agreed on 31 of 40; that answer is taken as
  the truth.
- **The panel**: the 9 disagreements went to three agents, a developer, a
  tester and an architect, each given both readings as "a" and "b", not
  told which was the judge's. Each said whether the finding is true, and
  whether it matters (worth raising to a person). Unanimous on truth for
  all 9; on "matters" for 7 of 9.
- **Independence**
  ([ADR-0005](../adr/0005-independence-takes-the-best-level-available.md)):
  the finder on Sonnet, the judge on Opus, the raters on Fable; the
  panel's developer on Fable, tester on Opus, architect on Sonnet. One
  provider throughout.
- What the judge is given: [the reviewer's judge](../../roles/reviewer/docs/judge.md#7-the-judge).

## Limits

- **One sample of 40**, each item rated once.
- **Balanced 20/20, the population 173/56**: read a rate per verdict, not
  an overall accuracy. From 20 items, the share of true findings among
  the refusals lies between about a fifth and three fifths (8 of 20, 95%
  Wilson interval); the "yes" right at least three times in four.
- **31 truths rest on two readings that agree**, the judge's and a
  rater's, not on a panel: both could be wrong together.
- **"Matters" is the panel's judgement**, by majority, asked only of the
  9 disagreements.
- **The pattern is a word match** on the judge's reasons, not a reading
  of each.
- **The judge changed** over those ten days; the sample mixes its
  versions.
- The extracted verdicts and the agents' answers are not in the
  repository; the 40 are listed below, and
  [#362](https://github.com/JN0V/workline/issues/362) keeps them as an
  evaluation set.

## Decision that followed (2026-10-14)

- **The judge reads the code it needs** before saying no
  ([#362](https://github.com/JN0V/workline/issues/362)), an ADR first.
- **Then the reviewer's measure** before the developer role
  ([#360](https://github.com/JN0V/workline/issues/360)) runs on that
  judge, so the bench is paid once.

## The 40 items

Each place links to the file at the commit reviewed. "Truth" is the
agreed answer, or the panel's; "matters" was asked of the panel only.

### The judge said no

| Pull request | Place | Lens | Finding | Truth | Matters |
|---|---|---|---|---|---|
| [#136](https://github.com/JN0V/workline/pull/136) | [ci/github/workline-gardening.yml:85](https://github.com/JN0V/workline/blob/9ad8c5859d4633f8ae12ec2b3ed4f1fafbc0675b/ci/github/workline-gardening.yml#L85) | edge cases | Verdict step interpolates status into shell, empty status passes | false | — |
| [#137](https://github.com/JN0V/workline/pull/137) | [internal/backlog/backlog.go:929](https://github.com/JN0V/workline/blob/052290503dce4fe7a9f7f6dcd9f7b6d5e02c4589/internal/backlog/backlog.go#L929) | tests | Quoted-ask test never covers a quoted proposal or a later round | true, panel | no (1 of 3 yes) |
| [#144](https://github.com/JN0V/workline/pull/144) | [internal/engine/engine.go:161](https://github.com/JN0V/workline/blob/4eb33303a753e86f6484cef34fb5ec9ba184bfe5/internal/engine/engine.go#L161) | edge cases | Run reads files from the working directory when no run folder exists | true, panel | no |
| [#146](https://github.com/JN0V/workline/pull/146) | [internal/forge/gitlab.go:395](https://github.com/JN0V/workline/blob/953c6fff702ce82e825e579ecd4bfae3e746e1ef/internal/forge/gitlab.go#L395) | edge cases | Sticky edit swallows a 403 and writes a duplicate note every run | false | — |
| [#152](https://github.com/JN0V/workline/pull/152) | [internal/forge/command.go:275](https://github.com/JN0V/workline/blob/d2051d0a20821c259eb25925511ff9966657f5a3/internal/forge/command.go#L275) | correctness | Ticks error from command forge is swallowed, treated as no ticks | false | — |
| [#157](https://github.com/JN0V/workline/pull/157) | [internal/backlog/undo.go:168](https://github.com/JN0V/workline/blob/27b0bf36c52fae8aca2aab4d0ca4d35b3d51a177/internal/backlog/undo.go#L168) | tests | splitDrafts and the drafts-proposed branch have no unit test | false | — |
| [#170](https://github.com/JN0V/workline/pull/170) | [internal/backlog/depend.go:219](https://github.com/JN0V/workline/blob/3ba025139120a2e5bf645c6e8cdfe947c0535d0b/internal/backlog/depend.go#L219) | edge cases | Depend check misses a close decided later in the run | true, panel | no |
| [#190](https://github.com/JN0V/workline/pull/190) | [internal/builtin/documentalist/documentalist.go:649](https://github.com/JN0V/workline/blob/d5cfb70efc232e65197d0a3d53df48c5ab71d19e/internal/builtin/documentalist/documentalist.go#L649) | edge cases | source-gone emitted once per source and duplicated for cut docs | false | — |
| [#199](https://github.com/JN0V/workline/pull/199) | [internal/hooks/hooks.go:267](https://github.com/JN0V/workline/blob/1ab52cd49c4ee2f5183d5030ece3ec21f7958b50/internal/hooks/hooks.go#L267) | tests | XDG_CONFIG_HOME expansion untested | true, panel | no (1 of 3 yes) |
| [#219](https://github.com/JN0V/workline/pull/219) | [cmd/workline/cli_test.go:92](https://github.com/JN0V/workline/blob/7112b0ee1e123cf71b107e2c2c387fbd241609b7/cmd/workline/cli_test.go#L92) | edge cases | Reader goroutine never unblocked if f panics, and the pipe leaks | true, panel | no |
| [#238](https://github.com/JN0V/workline/pull/238) | [internal/builtin/documentalist/freshness.go:83](https://github.com/JN0V/workline/blob/4b27dd665275ef5145869b4b66375b1d43059d2f/internal/builtin/documentalist/freshness.go#L83) | correctness | whole-chars can push a task past the role's budget | true, panel | **yes** |
| [#248](https://github.com/JN0V/workline/pull/248) | [tests/…/spec-review-restarting-not-taken-as-answered.yaml:33](https://github.com/JN0V/workline/blob/b1f5dd3da151c8212517601c65d330d84375d325/tests/conformance/cases/product-owner/spec-review-restarting-not-taken-as-answered.yaml#L33) | tests | Unchanged-body half of the case asserts nothing | false | — |
| [#253](https://github.com/JN0V/workline/pull/253) | [internal/backlog/parent.go:188](https://github.com/JN0V/workline/blob/cd59ce82f6cc36b9aca632a207e6d9f254aa2a18/internal/backlog/parent.go#L188) | tests | ReadEvidence test-proof branches untested | false | — |
| [#257](https://github.com/JN0V/workline/pull/257) | [internal/builtin/documentalist/reader.go:57](https://github.com/JN0V/workline/blob/159cbebbeee00cf6c74c4ddeed5e3ad0244c6673/internal/builtin/documentalist/reader.go#L57) | claims | "Never blocking" contradicted by setting-missing | true, panel | no |
| [#272](https://github.com/JN0V/workline/pull/272) | [internal/backlog/ledger.go:639](https://github.com/JN0V/workline/blob/9fcc416d0d437d1231ecc893b633ab18c6e9d764/internal/backlog/ledger.go#L639) | claims | "Every line starting with /" is not escaped: indented lines skipped | false | — |
| [#279](https://github.com/JN0V/workline/pull/279) | [internal/backlog/ledger.go:535](https://github.com/JN0V/workline/blob/1ef286da4fb62289a98e2cbdd112b02e149adf8b/internal/backlog/ledger.go#L535) | correctness | Label removal for set-aside issues with no state | false | — |
| [#280](https://github.com/JN0V/workline/pull/280) | [internal/engine/engine.go:460](https://github.com/JN0V/workline/blob/5c4614774293fb5d780999a8fea5f465a5b79083/internal/engine/engine.go#L460) | correctness | Findings whose level is empty set to warn, hiding errors | true, panel | **yes** |
| [#315](https://github.com/JN0V/workline/pull/315) | [internal/engine/follow.go:62](https://github.com/JN0V/workline/blob/b91b620c3292f2f6ca861f11f528ec4ec5e7931f/internal/engine/follow.go#L62) | correctness | --gardening tends every role's workline branch | false | — |
| [#329](https://github.com/JN0V/workline/pull/329) | [tests/…/verdict-only-keeps-it-and-nothing-else.yaml:31](https://github.com/JN0V/workline/blob/022c017427d5fe9d5844bc7e610dfdfd9743f372/tests/conformance/cases/merge/verdict-only-keeps-it-and-nothing-else.yaml#L31) | intent | The verdict is kept as a commit status, not a git note | false | — |
| [#347](https://github.com/JN0V/workline/pull/347) | [internal/engine/routine.go:235](https://github.com/JN0V/workline/blob/0a3a5d3ba67843f7dadbc12c2df2cb5ca2eefee5/internal/engine/routine.go#L235) | correctness | Draft with own auto-merge on is skipped at start | false | — |

### The judge said yes

| Pull request | Place | Lens | Finding | Truth |
|---|---|---|---|---|
| [#133](https://github.com/JN0V/workline/pull/133) | [internal/engine/engine.go:1822](https://github.com/JN0V/workline/blob/03b1805a793df97df64d4bbe5a5c0abd73d94f7e/internal/engine/engine.go#L1822) | correctness | A quote that is not found changes the key silently | true |
| [#137](https://github.com/JN0V/workline/pull/137) | [internal/backlog/backlog.go:825](https://github.com/JN0V/workline/blob/d39c8f43718f7ea7e4374a8e5d8d068912565789/internal/backlog/backlog.go#L825) | edge cases | Dots in a question make different questions look the same | true |
| [#152](https://github.com/JN0V/workline/pull/152) | [internal/forge/github.go:423](https://github.com/JN0V/workline/blob/d2051d0a20821c259eb25925511ff9966657f5a3/internal/forge/github.go#L423) | correctness | Ticks array indexed wrongly when oldest edit truncated | true |
| [#158](https://github.com/JN0V/workline/pull/158) | [internal/engine/engine.go:703](https://github.com/JN0V/workline/blob/b76a7967831569a8ae04630478af5ec4004cfc0f/internal/engine/engine.go#L703) | edge cases | Unindented lines of an item silently dropped | true |
| [#177](https://github.com/JN0V/workline/pull/177) | [internal/builtin/productowner/productowner.go:448](https://github.com/JN0V/workline/blob/f695905446e1dee798236fff0b8b99e3c2163251/internal/builtin/productowner/productowner.go#L448) | correctness | One failing Closers call aborts the whole run | true |
| [#185](https://github.com/JN0V/workline/pull/185) | [internal/backlog/change.go:307](https://github.com/JN0V/workline/blob/b8d0f44c73c0af51e7941904cd72a5d2d4354ae2/internal/backlog/change.go#L307) | edge cases | Second change overwrites the first for the same touched issue | true |
| [#199](https://github.com/JN0V/workline/pull/199) | [internal/hooks/hooks.go:208](https://github.com/JN0V/workline/blob/dc42fb2929d70ec86fe9256ec7c45f5ac1c4a093/internal/hooks/hooks.go#L208) | edge cases | Guard line with exit hides later hand-over | true |
| [#219](https://github.com/JN0V/workline/pull/219) | [tests/conformance/help_test.go:24](https://github.com/JN0V/workline/blob/e2add4fe2acf42f3459d715a69993bf893ad6ded/tests/conformance/help_test.go#L24) | correctness | Conformance test crashes on blank line in help listing | **false**, panel |
| [#234](https://github.com/JN0V/workline/pull/234) | [internal/builtin/reviewer/reviewer.go:707](https://github.com/JN0V/workline/blob/9484f74faa61cdd6ef2acee13d298e758172b0e2/internal/builtin/reviewer/reviewer.go#L707) | tests | Duplicate findings share one judge key, untested | true |
| [#237](https://github.com/JN0V/workline/pull/237) | [internal/builtin/reviewer/reach.go:296](https://github.com/JN0V/workline/blob/7592b240ab00a03762f8eb273d9baeaf27b24bcb/internal/builtin/reviewer/reach.go#L296) | correctness | Brace matching counts braces in rune literals and raw regex | true |
| [#245](https://github.com/JN0V/workline/pull/245) | [internal/agent/mend.go:35](https://github.com/JN0V/workline/blob/3ab81f50964a82ef4849f5f53934fafa231ce413/internal/agent/mend.go#L35) | edge cases | Two slips in one answer are not mended | true |
| [#253](https://github.com/JN0V/workline/pull/253) | [internal/backlog/undo.go:166](https://github.com/JN0V/workline/blob/cd59ce82f6cc36b9aca632a207e6d9f254aa2a18/internal/backlog/undo.go#L166) | tests | Unread state in the open-parent split branch untested | true |
| [#262](https://github.com/JN0V/workline/pull/262) | [internal/backlog/report.go:68](https://github.com/JN0V/workline/blob/c950651796cd2523a14b5b1156f8de981de462a1/internal/backlog/report.go#L68) | tests | A page with only items to settle claims nothing waits on the person | true |
| [#280](https://github.com/JN0V/workline/pull/280) | [internal/builtin/documentalist/condense.go:331](https://github.com/JN0V/workline/blob/5c4614774293fb5d780999a8fea5f465a5b79083/internal/builtin/documentalist/condense.go#L331) | correctness | movedWhole accepts any new problem fitting a vanished size | true |
| [#307](https://github.com/JN0V/workline/pull/307) | [ci/gitlab/workline.gitlab-ci.yml:71](https://github.com/JN0V/workline/blob/c01886f42dab8bb611d8b15d720a5439b8f239f3/ci/gitlab/workline.gitlab-ci.yml#L71) | edge cases | Only the first 100 notes are read | true |
| [#316](https://github.com/JN0V/workline/pull/316) | [internal/builtin/reviewer/reviewer.go:953](https://github.com/JN0V/workline/blob/adf03f59e739af216603bdaec7b37d7c55dc1ecb/internal/builtin/reviewer/reviewer.go#L953) | tests | Fallback of the review to the removed-line "On" has no test | true |
| [#329](https://github.com/JN0V/workline/pull/329) | [internal/engine/follow.go:83](https://github.com/JN0V/workline/blob/3e401fb03200f250a827a6cd50e2ff7159913421/internal/engine/follow.go#L83) | tests | Verdict-only mode has no test | true |
| [#347](https://github.com/JN0V/workline/pull/347) | [internal/engine/routine.go:437](https://github.com/JN0V/workline/blob/9077b5bd7d80640425d675168379eebb680bfd0c/internal/engine/routine.go#L437) | tests | History read failure path has no test | true |
| [#347](https://github.com/JN0V/workline/pull/347) | [internal/forge/gitlab_merge.go:119](https://github.com/JN0V/workline/blob/05180255a035022c7286be7bc2adca06e7a1d80b/internal/forge/gitlab_merge.go#L119) | claims | GitLab pushers are read only inside a window and may be empty | true |
| [#347](https://github.com/JN0V/workline/pull/347) | [internal/engine/demote.go:140](https://github.com/JN0V/workline/blob/05180255a035022c7286be7bc2adca06e7a1d80b/internal/engine/demote.go#L140) | claims | Reverts older than 100 commits are never read | true |
