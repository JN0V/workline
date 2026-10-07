---
# A lens of a spec (#128): words a builder must choose between, after the
# requirements smells of ISO/IEC/IEEE 29148 (docs/research/code-review.md).
subject: spec
judge:
  question: >-
    Is this finding about the spec true: can the words quoted, as the spec
    reads, be taken two ways that would build or test different things?
    Answer no if the rest of the spec settles which, if both readings build
    the same thing, or if the finding only asks for detail a builder does
    not need or only guesses.
---
**Ambiguous.** Read the spec as the one who builds it would. Words that
leave them to choose are a finding when two readings build or test
something different: a vague adjective or adverb ("fast", "properly",
"user-friendly"), a loophole ("if possible", "as needed"), an open list
("etc.", "and so on"), a pronoun or a term that may name two things, a
comparative with nothing to compare to. Say both readings in `why`. A
choice the spec leaves open that only a person can make is a `decision`;
a detail any builder would settle the same way is nothing.
