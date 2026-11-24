---
sources: [roles/product-owner/role.yaml, roles/product-owner/prompts/instruction.md, roles/product-owner/prompts/policy.md, roles/product-owner/code, internal/backlog]
checked: e81df7b34
verified: agent:claude-code
---
# Product owner — a run: people

Part of [a run](run.md), steps 4 and 5, then imports: a person's comment,
the reporter, a file imported, the issues other roles open.

## 4. A person's comment

On an issue with nothing waiting on you — set aside, or read and left —,
a comment of yours or of its reporter is read first, the agent told your
word decides.

- **Set aside, then commented on**: it is brought back, your comment
  quoted to the agent; it completes, proposes or moves it to `ready` as
  for any issue, unless you say otherwise ("not now", "leave it").
- **Said with the "not now"**: a comment written before you took
  `workline:proposed` off brings nothing back.
- **Asking a split**: your latest comment saying to split the issue has
  it split, though `split` is off by default — proposed at `cautious`; never where
  the project set `split` off itself
  ([#1667](https://github.com/JN0V/workline/issues/1667)).

On an issue waiting on you, it is "revise":

- the issue is read first, the agent told to revise;
- it rewrites its own sections in place — a draft, or a text as it wrote
  it — never yours;
- 👀 on each comment read, then one line in reply;
- one revision a run, two at most; past them, one line "left to a person"
  in the summary, and said in its own comment on the issue, edited in
  place, never a new one
  ([ADR-0059](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#1-every-refusal-claim-wait-or-block-is-on-the-issue)).

## 5. The reporter

- **Never asked what the code answers**: an ask on an issue whose code
  the same run names (`sources`) is not asked; the issue is read again
  with that code at the next run
  ([#548](https://github.com/JN0V/workline/issues/548)).
- **Asked again** only after they answered, never the same question
  twice, `acts.ask.rounds` times at most
  ([ADR-0021](../../../docs/adr/0021-the-product-owner-talks-with-the-reporter.md)):
  a question asked before is left out, the rest of the proposal kept.
- **An outsider's issue** gets no section in its body: the engine comments
  to its reporter what it understood and the sections it would write. A
  reply `agreed`, or `workline:accepted`, has them written; with the
  label, an answer since from the reporter or the project is read by the
  agent first, and its `refine` is written instead. If that
  comment's hidden block does not read, nothing is written, said each run
  (`proposal-unread`): copy the sections in by hand.

## Importing a file

On `import`, `workline issues import <file>` has the agent read a committed
file a share at a time
([the command](../../../docs/commands-work.md#workline-issues-import)):

- each item still to do opened as an issue quoting it, never twice;
- every other line answered with why (`skip`);
- every item mapped to its issue or reason; a line with neither listed for
  a person (exit 2).

## Opening issues

Every role opens an issue through one way the engine keeps for the
product owner
([backlog acts, "Opening issues"](../../../docs/spec/backlog-acts/opening.md)):
a key per subject, never opened twice; the role named, `needs-triage`; a
cap a run (`issues-max`).
