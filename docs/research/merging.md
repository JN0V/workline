# Who merges pull requests — focused research (2026-10-12)

**Question.** What does a person do when they merge workline's pull
requests? How do other projects and tools decide who, or what, merges?
Where in workline would a written merge policy live?

**Verdict.**

- **The click is mostly mechanical.** Of 191 merges in 19 days, 142 (74%)
  needed the click alone, 33 more only an Update branch. The person
  commented on, approved and edited nothing.
- **What the person adds is order and timing**: 38 Update branch clicks,
  and the release order kept by hand in 9 of 11 releases.
- **Elsewhere, a bot merges what people voted for**, and alone only a
  narrow class people wrote down in advance. No AI coding agent merges
  its own work by default.
- **The forge already merges** (auto-merge on GitHub and GitLab). A user's
  GitHub repository lacks only the queue keeping a pull request up to date.
- **Not a role**: whether a pull request may merge is rules, not
  judgement. Proposed in [ADR-0042]: `workline follow` turns the forge's
  auto-merge on, by a written project policy.

## What the person does at merge (2026-09-24 → 2026-10-12)

### Method

- **196 pull requests closed**: 191 merged, 5 not (four trials, one
  replaced by an issue). Kinds: an agent's (a coding agent's session), the
  documentalist's (`workline/documentalist/*`), the release's
  (release-please); the last two opened by workline's GitHub App.
- **Read with `gh api`**: the checks on the merged head, the reviewer's
  comment, the timeline, every commit pushed.
- **Mechanical**: every check green on the merged head, no action by the
  person but the merge. **Update branch**: a "Merge branch 'main' into …"
  commit made through GitHub (committer `web-flow`). **Ready**: the last
  check on the merged head, or the reviewer's last edit, whichever later.
- **Wrong merge**: a fix within 48 h naming the merged pull request; 18
  such citations read by hand.

### What was found

| Merges | All, from 09-24 (191) | Since the ruleset settled, 10-04 (120) |
|---|---|---|
| The click alone | 142 (74%) | 93 (78%) |
| Only an Update branch more | 33 | 27 |
| Something else | 16 | 0 |

- **By kind since 10-04**, the click alone: an agent's 86%, the
  documentalist's 62%, the release's 53%; the rest, one Update branch.
- **38 Update branch clicks**: 21 on agents' pull requests, 7 on the
  documentalist's, 10 on release-please's. Behind `main` before the click:
  the documentalist's a median of 41 min (90th percentile 376 min, 13.9 h
  in all), an agent's 8 min (47 min). Then a median of 2.4 min to merge.
- **The release order, by hand, in 9 of 11 releases since 10-05**: the
  release pull request held (its `judge` red) until the documentalist's
  release docs pull request merged, then updated, then merged (no docs
  due for [#181], [#250]). Timing too: one stayed open 95.7 h ([#267]).
- **Nothing else**: no comment by the person, no approval, no commit but
  Update branch, no merge message edited.
- **Merged anyway**: 15 with nits listed, 2 with an Important finding
  ([#234] answered; [#260] not, still in `main`), 1 with a lens not
  reviewed whole ([#282]).
- **Ready to merge**: a median of 7.1 min, 90th percentile 54 min; 17
  waited over an hour, mostly overnight; [#284] 20.7 h.
- **Wrong merges: 4 of 191 (2.1%), all from 10-01 to 10-04**, none of the
  120 since: [#33] (70 MB of binaries), [#38] and [#113] (stacked on a
  branch merged already), [#73] (merged while an agent still pushed); the
  same days, 6 merges over a red `judge`. After: two defects no check saw
  ([#280], [#284]), fixed within 48 h; no revert.

### How sure

- **Medium-high on who did what**: the person and the agents share one
  account. Update branch is read as the person's from its timing and the
  agents' rule never to merge; every comment under the account was read,
  all agents'. Latency ±2 min: reading the diff is not seen.
- **Not measured**: when a conflict appeared; a defect fixed after 48 h
  or never named. [merge-conflicts.md](merge-conflicts.md#clean-updates-forced-by-up-to-date)
  counts 30 clean Update branch merges: from 09-28, the updates it replayed.

## Who merges elsewhere

| Pattern | Who decides | Examples |
|---|---|---|
| People vote, a bot merges once checks pass | a reviewer's `lgtm`, an owner's `approve` | Prow's [Tide][tide], bors (`r+`), [Zuul's gate][zuul], GitHub's merge queue, GitLab's merge trains |
| The bot tests the exact merge on the latest tip | — | the ["not rocket science rule"][nrsr] |
| Owners write the policy in the repository; no bot edits its own | owners | OWNERS files, [Mergify's queue rules][mergify], Renovate's `packageRules`, rulesets |
| No person approves, for a narrow class | people, in advance | [Renovate][renovate] (lock files, then lint tools, then non-major; `minimumReleaseAge`), Dependabot's semver-patch, Chromium's [Rubber Stamper][chromium] (clean reverts, cherry-picks; never an owner's approval) |

- **What serious automation asks**: the checks green on the merge result;
  an approval by someone other than the author, with authority over the
  paths; approvals reset by a new push (SLSA's "final revision"); no hold
  label; threads resolved; out of the queue on a failure; a cheap veto.
- **Autonomy, graded**: by the person's role, five levels from operator
  to observer, scoped and renewed ([Feng, McDonald, Zhang, 2025][levels]);
  earned per scope and revocable, as a Kubernetes approver after 3 months
  and 30 pull requests reviewed ([membership][k8s]), Claude Code users
  auto-approving about 20% of sessions at first, over 40% after about 750
  ([Anthropic][autonomy]); by change class ([Ship / Show / Ask][ssa]).

### AI agents, and the governance line

- **No agent merges its own work by default**: Copilot's coding agent and
  OpenHands' resolver open drafts; Google's AI migrations are reviewed by
  owners ([2025][google]).
- **Evidence**: agents' pull requests merge less often than people's
  ([AIDev][aidev], 456k pull requests); of Claude Code's, 83.8% merged,
  54.9% of those unmodified ([Watanabe et al.][cc]).
- **Two persons**: SLSA's source level 4 has changes "agreed to by two or
  more trusted persons", each "a human", on the final revision; a robot
  only by an explicit exception ([SLSA 1.2][slsa]). SOC 2's CC8.1 asks
  authorised personnel; no guidance accepts an AI approver.
- **"The product owner merges"**: almost no precedent. Prow splits `lgtm`
  (quality) from `approve` (ownership): a product owner's acceptance is
  an approval or a release, not the merge.

## The forges' own merge

| | GitHub | GitLab |
|---|---|---|
| Merge once checks pass | [auto-merge][gh-auto], allowed per repository; GraphQL `enablePullRequestAutoMerge` | [auto-merge][gl-auto], every tier; the [merge API][gl-api]'s `auto_merge` |
| Turned off by | a push by someone without write access; the base changed; Disable auto-merge | new commits; the target moving under fast-forward merges, unless [automatic rebase][gl-methods] is on; `cancel_auto_merge` |
| A branch behind | waits, when the ruleset requires it up to date; no merge queue on a user's repository | merge commits merge anyway; fast-forward: automatic rebase, or merge trains (Premium) |
| Veto | Draft; Disable auto-merge | Draft; Cancel auto-merge |

- **The token**: a merge made with `GITHUB_TOKEN` starts no workflow on
  `main` — no `follow`, no release-please; the App's token does
  ([ci-and-forge.md](ci-and-forge.md#a-bots-commit-on-the-pull-request)).
- **This repository**: `main` takes pull requests with `test` and `judge`
  green and up to date, no approval, squash only, auto-merge allowed.

## Inside workline

- **Its texts say it never merges**: [principle 1](../PRINCIPLES.md),
  [concepts.md](../concepts.md#people), [ADR-0011], [ADR-0020].
- **A green check is not a pass**: with no agent, the documentalist only
  lists docs; a run may end `human` or `blocked-external`; the reviewer's
  AI findings only warn here. A policy reads the verdicts.
- **A release pull request**: merging it is the decision to release; a
  tag cannot be taken back ([ADR-0017]).
- **`follow`** holds the write token, calls no agent ([ADR-0034]).

## Role or not: four agents, then the person

- **The arbiter**, by workline's criteria: a role brings judgement, not
  screens the forge has ([principle 13](../PRINCIPLES.md),
  [roles-panorama.md](roles-panorama.md)); mechanical work goes to
  scripts; a role must beat the tool ([ADR-0017]); every shipped role has
  an agent ([role contract](../spec/role-contract.md)). A merge by policy
  fails three, the fourth weakly: a command, as `follow` and `sample` are.
- **No role**: the forge merges, Draft is a veto; `follow` adds the
  queue [ADR-0041] left to this decision; trust needs a record, not a role.
- **An existing role**: the committer fits best, but judging a message is
  not accepting a change ([principle 2](../PRINCIPLES.md)); the
  documentalist would approve its own pull requests ([ADR-0005]); the
  reviewer never approves ([ADR-0020]); the product owner works on
  issues, and acceptance is the person's ([ADR-0018]).
- **A new role**, its own objection: rules with no agent are a command.
- **The person, 2026-10-12**: no new role; `follow` turns the forge's
  auto-merge on by a `merge:` policy, `cautious` first; a release pull
  request never merged without a person's yes; the veto is the forge's.

[ADR-0005]: ../adr/0005-independence-takes-the-best-level-available.md
[ADR-0011]: ../adr/0011-the-review-is-on-the-merge-request-not-the-push.md
[ADR-0017]: ../adr/0017-the-release-manager.md
[ADR-0018]: ../adr/0018-the-product-owner.md
[ADR-0020]: ../adr/0020-the-reviewer-finds-the-engine-verifies-the-person-merges.md
[ADR-0034]: ../adr/0034-the-release-fix-follows-its-base.md
[ADR-0041]: ../adr/0041-workline-settles-its-own-lines-when-the-base-moves.md
[ADR-0042]: ../adr/0042-the-engine-merges-by-the-projects-policy.md
[#33]: https://github.com/JN0V/workline/pull/33
[#38]: https://github.com/JN0V/workline/pull/38
[#73]: https://github.com/JN0V/workline/pull/73
[#113]: https://github.com/JN0V/workline/pull/113
[#181]: https://github.com/JN0V/workline/pull/181
[#234]: https://github.com/JN0V/workline/pull/234
[#250]: https://github.com/JN0V/workline/pull/250
[#260]: https://github.com/JN0V/workline/pull/260
[#267]: https://github.com/JN0V/workline/pull/267
[#280]: https://github.com/JN0V/workline/pull/280
[#282]: https://github.com/JN0V/workline/pull/282
[#284]: https://github.com/JN0V/workline/pull/284
[tide]: https://docs.prow.k8s.io/docs/components/core/tide/config
[zuul]: https://zuul-ci.org/docs/zuul/latest/gating.html
[nrsr]: https://blog.janestreet.com/making-never-break-the-build-scale/
[mergify]: https://docs.mergify.com/merge-queue/rules
[renovate]: https://docs.renovatebot.com/key-concepts/automerge/
[chromium]: https://chromium.googlesource.com/chromium/src/+/HEAD/docs/code_review_owners.md
[levels]: https://arxiv.org/abs/2506.12469
[k8s]: https://github.com/kubernetes/community/blob/master/community-membership.md
[autonomy]: https://www.anthropic.com/research/measuring-agent-autonomy
[ssa]: https://martinfowler.com/articles/ship-show-ask.html
[google]: https://arxiv.org/abs/2501.06972
[aidev]: https://arxiv.org/abs/2507.15003
[cc]: https://arxiv.org/abs/2509.14745
[slsa]: https://slsa.dev/spec/v1.2/source-requirements
[gh-auto]: https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/incorporating-changes-from-a-pull-request/automatically-merging-a-pull-request
[gl-auto]: https://docs.gitlab.com/user/project/merge_requests/auto_merge/
[gl-api]: https://docs.gitlab.com/api/merge_requests/
[gl-methods]: https://docs.gitlab.com/user/project/merge_requests/methods/
