---
sources:
  - internal/role/config.go#config
  - internal/role/role.go#ProjectConfig
  - roles/committer/role.yaml#settings.subject-max
  - roles/committer/role.yaml#settings.rewrite
  - roles/documentalist/role.yaml#settings.whole-chars
  - roles/reviewer/role.yaml#settings.ai-findings
  - defaults/routing.yaml#budgets
checked: e81df7b34
verified: agent:claude-code
---
# A config example

Part of [the configuration](config.md#worklineconfigyaml).

## The example: agent, roles, routing

```yaml
ai: claude                  # the project's agent, one choice: none (default), claude, cmd:<command>…
autonomy: normal            # each role's level unless it sets its own: cautious | normal (default) | enterprising
forge: github               # github | gitlab | local | cmd:<command> | none (default)
roles:
  committer:
    settings: {subject-max: 60, rewrite: false}  # a role's settings, see its role.yaml; rewrite: an agent rewrites a refused message, 30 s at most (on)
    enforce: {internal-code: warn}       # block (default) | warn | off, per rule
  documentalist:
    settings:
      derive: {cases: "ls tests/*.yaml | wc -l"}   # fills <!-- workline:derive cases -->…<!-- workline:end --> in a doc
      history: ["docs/journal/**"]                 # records, beside changelogs and release notes (role.yaml)
      whole-chars: 25000                           # sources given whole to an agent that does not read the repository (25000 at most)
      versions: {pattern: '\d{2}\.\d+', files: [pyproject.toml]}   # calendar versions, and where the version is said
      language: fr                                 # the docs' language, for the removal rule; unset, read from each doc
      sample: {judge: "claude:opus", at-least: model, after: v1.4.0, ai-max-tokens: 400000}   # who reads the weekly sample, the least independence, nothing vouched for before your tag, the week's tokens; `panel: [three --ai values]` settles a dispute
  reviewer:
    settings:
      ai-findings: {correctness: block}   # a verified important finding blocks for these lenses; the others warn (one value, warn or block, sets every lens)
      judge-lines-max: 300                # lines of code a judge reads: the cause's function, then those it reaches, the rest named (200; 0: 31 lines around the cause)
      judge-read-rounds: 0                # the judge reads no more code than it is shown, a no drops the finding (2: it may ask twice, read at the commit)
      diff-alone: true                    # one call more a run, given the change only, not the files nor the messages (off by default, measured)
      spec-rounds: 3                      # on the forge: reviews of a spec in a row with a finding open, then a question to a person (5)
      specs-per-run: 2                    # on the forge: the specs a night reads, the rest the next night (5)
```

## The example: routing

The same file, continued:

```yaml
routing:                    # replaces the shipped line, event by event
  events: {merge-request: [committer, documentalist, gate:merge],
    schedule: [documentalist, product-owner, reviewer]}   # the reviewer reads each spec the product owner refines; ready waits on it
  handoffs: [{from: my-role, to: documentalist}]   # a role of your own (--roles); none shipped hands over
  max-handoffs: 3
  fail-fast: {schedule: false}   # every step of the schedule runs, the worst verdict stands (the default)
  budgets: {verdict: 300}        # the head's verdict may take 5 min here (3 by default); over it, said
  wip: {ready: 3}                # Ready holds 3 issues at most here (5 by default); the tester waits for room
  stale: {ready: 60}             # a ready issue is checked for staleness after 60 days here (30 by default)
```

## The example: gates, repositories, release, merge

The same file, continued:

```yaml
gates:                      # your own checks, as a step of the line
  merge:
    checks: [{id: secrets, run: "gitleaks detect --report-format sarif --report-path {out}/secrets.sarif", output: sarif, max: {error: 0}}]
repos:                      # other repositories docs may depend on
  api: {url: "https://github.com/acme/api.git", branch: main}
release:                    # how the project's release tool works
  branches: ["release/*"]   # its pull requests' branches; default: release-please's, releaser-pleaser's, release-plz's, changesets'
  tags: "v*"                # its tags (the default); the last release is the highest version merged, prereleases left out
merge: {autonomy: cautious} # the forge merges what a person labelled workline:accepted, its head's verdict passed
```

## `normal` crosses a line

**`merge: {autonomy: normal}` crosses a line**: a change an AI wrote and
an AI reviewed — workline's own docs, records and derived blocks, the
developer's fix of a reproduced bug — merges with no person's approval. That is past the two-person rule of
change control (SLSA's source level 4, SOC 2's change management): a
project's choice, never a default
([merge policy](merge-policy.md#normal-routine-work-with-no-yes)).

## Where the comments point

Where the comments point: `sample`'s independence,
[ADR-0005](adr/0005-independence-takes-the-best-level-available.md);
`diff-alone`, [#126](https://github.com/JN0V/workline/issues/126); the
reviewer on `schedule`, [#128](https://github.com/JN0V/workline/issues/128);
`gates`, [gates](spec/gates.md); `repos`, [multi-repo](spec/multi-repo.md);
`release`, [ADR-0017](adr/0017-the-release-manager.md).

