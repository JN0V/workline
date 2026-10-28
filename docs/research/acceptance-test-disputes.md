# When the builder disagrees with an acceptance test — what teams do (2026-10-28)

**Question.** The tester writes an issue's acceptance tests before the
code ([tests-first-and-corpus.md](tests-first-and-corpus.md)); the
developer must not edit them. When a test does not fit for a real
reason, how do teams settle it without a hard block, and without a
cheat path?

**Verdict.**

- **Teams negotiate the test, they do not lock it**: in ATDD and BDD the
  acceptance test is a shared agreement of three views (business,
  development, testing); a builder who finds it wrong reopens the talk,
  the test's owner rewrites it.
- **A doubt about the need goes to the person who owns the need**, not
  to the test's author: example mapping writes it as a question card
  for the product owner.
- **The owner of a file approves its change**: forges ship it as code
  owners; contract testing does the same between two teams.
- **Nothing found bounds the talk**: teams rely on a meeting. An agent
  pair needs a number and a way out.
- **What is left for workline**: the objection as a traced forge
  comment, the tester's answer as a fresh run, a bound, the five whys,
  a person last.

## Three amigos and specification by example

- **Three amigos**: business, development and testing look at a story
  together, before and during the work; each brings what the others
  miss ([Agile Alliance](https://www.agilealliance.org/glossary/three-amigos/)).
- **Specification by example**: the examples are specified together
  and refined until each side agrees they mean the need; the tests are
  then that shared specification (Gojko Adzic,
  [Specification by Example](https://www.manning.com/books/specification-by-example), 2011).
- **Example mapping**: rules, examples, and red *question* cards; a
  question nobody in the room can answer goes to the product owner, and
  the story waits for it
  ([Cucumber](https://cucumber.io/blog/bdd/example-mapping-introduction/)).
- **Taken**: the developer raises an objection, the tester rewrites, a
  doubt about the need goes to the product owner.

## Code owners: a change needs its owner

- **GitHub and GitLab code owners**: a file's owner is asked to review
  any change to it; a branch rule can require that approval
  ([GitHub](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners),
  [GitLab](https://docs.gitlab.com/user/project/codeowners/)).
- **The limit for an agent**: the owner approves another's edit; an
  agent's edit to a test it must pass is the edit to avoid.
- **Taken, tightened**: only the tester writes its tests; the developer
  asks, it does not propose the diff.

## Consumer-driven contracts

- **Pact**: the consumer writes the contract the provider must meet;
  the provider cannot change it alone, and a disagreement is settled
  between the two teams, then the consumer publishes a new contract
  ([how Pact works](https://docs.pact.io/getting_started/how_pact_works)).
- **Taken**: the one who wrote the expectation changes it; the one held
  to it explains why it cannot hold.

## What keeps it from being a cheat path

- **The tester answers to the criterion**, not to the code: a rewrite
  must still check the scenario's Then, and still fail on the base
  ([red first](tests-first-and-corpus.md#a-report-every-runner-writes)).
- **Each rewrite is traced** to its objection, visible to the reviewer
  in the same pull request: a test weakened in the open, never quietly.
- **The agents studied cheat when they can edit the tests**
  ([agents and tests](tests-first-and-corpus.md#agents-and-the-tests-they-can-edit));
  an objection changes nothing by itself.
