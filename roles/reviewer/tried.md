# Reviewer — tried for real

Each try with a real agent, newest last. Tokens as the agent reports them,
the whole input counted (on a subscription, the tokens are what counts).

## 2026-10-04 — a copy of workline, planted defects, Sonnet finding, Opus judging

A clone of workline at v0.8.0, the engine on this machine, `workline review
--forge local`. On the copy's `main`, a defect from before any change:
`gitrange.Base` returning the first word *without* `^` (the head, not the
base). On a branch, two commits: `Args` splitting the range on one space
(`strings.Split`) where it used `strings.Fields` — empty arguments for two
spaces, tabs left inside a word — under the message "no change in
behaviour", and a comment telling its history ("Fields used / to drop the
empty words").

- **Run 1**: the story was not caught — "used to" ran over two lines of the
  comment, and the rule read a line at a time. The lenses ran (13.9k tokens
  in, 1.4k out); the `Args` defect found by all three, verified by Opus.
  **Fixed**: a story is looked for in a comment's lines joined
  (`TestStoryOverTwoLines`). The tests lens's "changed with no test" was
  dropped as a duplicate of the line's other finding, silently. **Fixed**:
  findings on one line are merged, each said beside the first
  (`local-review-outputs-json`).
- **Run 2**: `bug-story` at the comment's first line, no agent asked, 0
  tokens. The comment rewritten, the review went on.
- **Runs 3 to 5** (about 14k in, 1 to 1.2k out each, three lenses, one
  judge): the `Args` defect every time, verified at `model` independence
  (claude-sonnet-5-5 → claude-opus-5-5); the defect in `Base` never, though
  its file was given whole, even planted plainer (the condition inverted).
  The floor asked for one candidate: it was computed from the diff alone,
  600 bytes.
- **Run 6**: the floor from the change and its files (two candidates), and
  the task saying to read the files whole: `Base` found by the correctness
  lens — and dropped, `finding-unfounded`: its quote put two lines on one
  (`if !strings.HasPrefix(w, "^") { return … }`). **Fixed**: a quote is
  found spaces and line breaks aside (`TestLocate`, `quote-over-lines-found`).
- **Run 7**: `Base` opened as an issue in the local forge, `needs-triage`,
  the product owner's state naming `internal/gitrange/gitrange.go` and the
  commit; the `Args` defect reported on line 10, for the author. Its key
  hashed the agent's quote, which changes from run to run. **Fixed**: the
  key is the file and the line the cause starts at, as it reads.
- **Run 8**: the correctness lens counted as "not asked": its answer quoted a
  line starting with a tab, which the YAML library writes and then refuses to
  read. **Fixed**: proposals that would not read back are written as JSON,
  which YAML reads (`TestWriteReadsBackATabbedQuote`); the lens's message
  now says its answer did not read. The commits were left unrecorded, as
  designed.
- **Runs 9 to 11**: a whole run (three lenses, three judges: 14.1k + 11.9k
  in, 2.3k + 0.8k out) opened `Base` (#1) and a second issue from the tests
  lens, "Head/Base/Args consistency untested" (#2) — true, but a test
  missing outside the change is a weak issue; then the same commits asked
  nothing (0 calls); then, the record removed, a whole run again found
  `Base` and opened nothing: the open issue's key held it.

Tokens, all local runs: about 126k in and 15k out on Sonnet (twenty-seven lens
calls), about 48k in and 4k out on Opus (thirteen judge calls; the first
seven not kept, estimated from the six kept). Each lens call 5 to 9
seconds.

## 2026-10-04 — a merge request on JN0V/workline-sandbox, live

`review-base` holding `Expired` inverted (`<` where `>`), and a pull request,
JN0V/workline-sandbox#8, adding `Remaining`, whose `TokenTTL -
now.Sub(issued)` counts the lifetime in nanoseconds. The engine on this
machine: `run-role reviewer --event merge-request --forge github --target
merge-request:8`, with the range CI would give.

- **Push 1**, the correctness lens (4.1k in, 0.6k out; two judges, 7.4k in,
  0.6k out): the units defect on the change's line 13, verified; one
  summary comment on the pull request; `Expired` opened as #9,
  `needs-triage`, the product owner's state on it.
- **Push 2**, the units fixed: only the new commit given, the edge-cases
  lens next in turn (4.0k in, 0.3k out; one judge, 3.5k in, 0.3k out): no
  finding on the change; `Expired` found again, not opened again; the
  summary comment edited in place, its record holding both commits.

Not tried: the judging job in CI reading the record with its read token;
GitLab; a fork; a lens unreachable mid-run; `ai-findings: block`;
`workline init --review` on a real repository (unit-tested only); every
lens at once on a merge request (`lenses=all`). Left open on the sandbox for
a person to look at: pull request #8 and issue #9.
