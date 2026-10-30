# ADR-0058: The forge guards the project, not the agent's environment

- **Status:** proposed
- **Date:** 2026-10-30
- **Decided by:** the maintainer agreed to the direction after a
  critical review of [#726]; the choices are theirs,
  [below](#for-the-person)
- **Amends:** [ADR-0056] step 2 ("no credential in reach") and its
  amendment; [ADR-0054]'s [bounds](0054-the-developer-takes-a-ready-issue-to-a-pull-request.md#what-bounds-it),
  which move to the pull request
- **Rewords:** principle 6 ([PRINCIPLES.md](../PRINCIPLES.md)), at
  acceptance
- **Builds on:** [ADR-0011], [ADR-0042], [ADR-0044]; principles 7, 10,
  12, 13

## Context

- **[ADR-0056] step 2 keeps every forge credential out of the agent's
  reach**: [#711] refuses to start it while git would hand one
  (`credential-in-reach`); [#726] seals git, ssh, gh and glab to run
  it on a person's machine.
- **It cannot hold on a person's machine**: the agent has a shell as
  the person's user; it reads what they read — files, the keyring
  (`gh auth token`), `~/.netrc`, a git URL rewrite. Each review round
  of [#726] found a new hole; [#726]'s own limit says so.
- **It broke the developer locally**: nearly every machine has a
  credential helper, so every run was refused; principle 10 wants the
  same role on a laptop and in CI.
- **It piles up detection code** in the engine and the templates: the
  GitLab runner hid its token in an included file ([#722]).
- **The person already trusts that agent** with their credentials:
  Claude Code, or another, runs in their shell every day.

## Decision

### 1. The forge is the barrier

- **What guards the default branch**: its protection (pull requests
  only), required checks, the review ([ADR-0011]), the merge by the
  project's policy or a person ([ADR-0042]). Whoever pushes meets
  them: the engine, an agent with the person's account, the person.
- **A person's push is never merged alone**: [ADR-0042] already keeps
  a pull request with a commit not the App's for a person's yes.

### 2. Locally, the agent has the person's rights

- **As their own coding agent does**: same user, same files, same
  credentials. No refusal, no sealing.
- **Said plainly**: in the role's docs, and once in the run's output
  (`the agent runs with your account's rights; the forge's guards
  decide what merges`).
- **Isolation is an opt-in, later**: a container for whoever wants it;
  not built now.

### 3. In CI, the forge's own permissions

- **GitHub**: the agent's job holds a read-only `GITHUB_TOKEN`
  (`contents: read`), no App key, `persist-credentials: false`: as the
  templates do today.
- **GitLab**: the write token scoped to the `workline` environment,
  unset in the agent's job (as today); `CI_JOB_TOKEN` stays in the
  clone, read-only while the project keeps "Allow Git push requests"
  off, said in the GitLab docs.
- **The write step stays apart** (`apply`, [ADR-0044]): principle 6's
  split is kept where the forge gives it natively and cheaply.
- **The engine checks the config, not the environment**: `workline
  doctor` reports a write permission on the agent's job.

### 4. The developer's bounds become a check on its pull request

- **What it checks**, on the pull request's diff against its base: the
  issue's Scope, the tester's frozen acceptance tests, the files that
  steer agents, `.workline/`, CI workflows, a gate's definition, a
  symlink out of the tree ([the bounds](../../roles/developer/docs/bounds.md)).
- **Where**: a gate on the pull request, a required check, so it holds
  whoever pushed. It runs from the default branch's definition
  (GitHub: `pull_request_target`, reading the diff, never running the
  pull request's code), so a pull request cannot edit its own check.
- **Which pull requests**: a `workline/developer/<n>` branch, bounded
  by issue `n`; another is a person's, reviewed as one.
- **A refused file fails the check**, each named; the next round or a
  person reverts it. The engine's pass before its push stays, a fast
  first pass, no longer the guard.
- **What changes for [ADR-0054] step 2**: the bounds' rules are
  unchanged; where they hold moves from `apply` to the pull request,
  and a local run is held as a CI run.

### 5. Principle 6, reworded

> **The agent proposes; the forge's guards decide.** What an agent
> writes reaches the default branch only through the forge's guards —
> protection, required checks, review, the merge policy — which hold
> whoever pushes. Where the forge splits rights natively, as in CI, the
> agent's step holds no write token; a separate step applies its
> proposals.

- **Replaces** "The agent proposes, the engine acts… a separate step,
  holding the write token and no AI key, validates and applies them".
- **Lands with acceptance**: [PRINCIPLES.md](../PRINCIPLES.md) is the
  rule every role follows today; a proposed ADR does not change it.

### 6. What goes, what stays

- **Goes**:
  - [ADR-0056] step 2's refusal: [#711]'s `credential-in-reach`,
    `internal/developer/credential.go`, its cases, GitLab's
    `.workline:no-credential` step;
  - [#726]: closed, not merged; [#722] resolved by the removal.
- **Reframed**: [#719], the CI job's Claude login: the env scrub
  stays; isolation is the opt-in container.
- **Stays**: in CI, the scrub of forge tokens and the agent's login
  from its shell (`CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`, the templates'
  unsets) and `persist-credentials: false`; everywhere,
  `--setting-sources user` ([ADR-0056] amendment).

## For the person

- **One choice**: the forge as the barrier (this ADR), or keep sealing
  the agent's environment ([#726]).
- **What you accept**: on your machine, the developer's agent can do
  what your account can, as your own agent can today.

## Build plan

One pull request a step, its cases first in
`tests/conformance/cases/developer/`, `pending.txt` listing what does
not pass.

1. **The bounds as a check**: `workline developer bounds --pr <n>`,
   its verdict as a check run. Cases:
   `developer-pr-bounds-refuses-out-of-scope`,
   `developer-pr-bounds-refuses-frozen-test`,
   `developer-pr-bounds-refuses-steering-file`,
   `developer-pr-bounds-passes-in-scope`,
   `developer-pr-bounds-skips-a-persons-branch`.
2. **The check in the templates**, required on the default branch,
   from its definition: `ci-github-bounds-check-from-base`; GitLab's
   equivalent named, its limit said ([open questions](#open-questions)).
3. **No refusal, said once**: `credential.go` and
   `developer-worktree-refuses-persisted-credential` deleted;
   `developer-local-run-says-person-rights`,
   `developer-local-run-with-credential-helper-starts`.
4. **CI's permissions checked by `doctor`**:
   `doctor-warns-agent-job-can-write`; `.workline:no-credential`
   deleted.
5. **Docs**: the developer's
   [agent](../../roles/developer/docs/agent.md),
   [pull request](../../roles/developer/docs/pull-request.md) and
   [triggers](../../roles/developer/docs/triggers.md) pages; principle
   6.
6. **Tried for real**: a run on a person's machine with a credential
   helper, and on each template's runner.

## Consequences

- **The developer runs locally again**, with no setup; the refusal,
  the sealing and the GitLab unset go; the bounds hold for any push.
- **Risks, honestly**:
  - a local agent, steered by an injected prompt, can push a branch,
    comment, or open a pull request with the person's account; it
    can also read the person's other projects;
  - the forge's guards still stop it reaching the default branch: a
    pull request with the person's commits needs their yes
    ([ADR-0042]);
  - those guards must be set: an unprotected default branch is open to
    the agent as to the person; `doctor` says when it is.
- **CI keeps its split**: the agent's job cannot write, by the forge's
  own permissions.
- **The opt-in container, when built**, may also carry the project's
  toolchain, which the agent's job needs anyway ([#703]).

## Not chosen

- **Keep sealing**: a hole each round; it cannot hold against a shell
  as the same user, and it refused most machines.
- **A mandatory container**: Docker or a user namespace on every
  machine; a tool to install and keep for a guard the forge already
  gives. Offered later as an opt-in.
- **No agent with tools locally**: the developer would not run on a
  laptop (principle 10).

## Open questions

- **GitLab's check from the base**: a merge request pipeline runs the
  source branch's `.gitlab-ci.yml`; a pipeline execution policy
  (Ultimate) or a CODEOWNERS approval on CI files (Premium) holds it.
  Which one the template documents.
- **The message once**: the run's output only, or the pull request's
  body too.

[#703]: https://github.com/JN0V/workline/issues/703
[#711]: https://github.com/JN0V/workline/pull/711
[#719]: https://github.com/JN0V/workline/issues/719
[#722]: https://github.com/JN0V/workline/issues/722
[#726]: https://github.com/JN0V/workline/pull/726
[ADR-0011]: 0011-the-review-is-on-the-merge-request-not-the-push.md
[ADR-0042]: 0042-the-engine-merges-by-the-projects-policy.md
[ADR-0044]: 0044-the-apps-key-stays-with-the-default-branchs-runs.md
[ADR-0054]: 0054-the-developer-takes-a-ready-issue-to-a-pull-request.md
[ADR-0056]: 0056-the-developers-agent-works-in-a-sandbox-with-tools.md
