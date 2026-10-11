# Priority and effort on the forges — focused research (2026-10-11)

**Question.** The product owner sets its own labels, `workline:priority/1`
to `/4`. Where does each forge keep an issue's priority, and its effort
(development complexity, a "weight"), so the role can use them instead?
Which scales do projects use, how do tools combine the two, who
estimates, and how is a person's value kept? Extends
[product-owner.md](product-owner.md#priority-and-ranking-2026-10-04).

**Verdict.**

- **GitHub has native Priority and Effort**, for organisations only:
  issue fields, generally available since 2026-07-02. A user's repository
  has none and drops the values silently.
- **GitLab Free** has a priority (prioritized labels, the "Label
  priority" sort) and no effort; **Premium** adds `weight` and exclusive
  `priority::` labels. Time estimate is time, not complexity.
- **Gitea** ships exclusive `Priority/*` labels and sorts by them (1.24);
  **Forgejo** has the labels only.
- **Scales converge**: four priority levels (Urgent to Low, or 1 to 4, 1
  the highest); effort a coarse scale of 3 to 5 steps.
- **The value-vs-effort grid** reads in one line; the ratio scores need
  data a role cannot observe.
- **No tracker estimates effort natively.** Tools that do suggest, and
  keep a person's value by writing only where empty, holding a
  suggestion, or reading who changed it last.

Searched in the ecosystem's words: *issue fields*, *priority labels*,
*issue weight*, *scoped labels*, *prioritized labels*, *exclusive
labels*, *story points*, *estimate*, *effort*, *value vs effort matrix*,
*WSJF*, *RICE*, *ICE*, *suggestions*, *rationale and confidence*. Pages,
API schemas and source code read on 2026-10-11; "inferred" marks what no
page states.

## GitHub: issue fields

| What | Found | Source |
|---|---|---|
| When | preview for some organisations 2026-03-12, for all 2026-05-21; generally available 2026-07-02 on Free, Team, Enterprise; GHES 3.23 | [03-12](https://github.blog/changelog/2026-03-12-issue-fields-structured-issue-metadata-is-in-public-preview/), [05-21](https://github.blog/changelog/2026-05-21-issue-fields-are-now-in-public-preview-for-all-organizations/), [07-02](https://github.blog/changelog/2026-07-02-issue-fields-are-now-generally-available/) |
| Defaults | `Priority` Urgent, High, Medium, Low; `Effort` High, Medium, Low; `Start date`, `Target date`; editable, deletable | [managing fields](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/managing-issue-fields-in-your-organization) |
| Types, limits | single-select, text, number, date (the API lists multi-select too); 25 fields, 100 options a field | same; [org fields API](https://docs.github.com/rest/orgs/issue-fields) |
| Where | per organisation, set by its admins; **not on user-owned repositories**, values sent there "silently dropped" | [issues API](https://docs.github.com/en/rest/issues/issues) |
| Visibility | per field, "Organization only" (default) or "Public"; outsiders see only Public ones | managing fields |
| Not on | pull requests, issue templates, URL prefill | [adding fields](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/adding-and-managing-issue-fields) |
| Search, sort | `field.priority:high`, `field.story-points:>5`; the REST list sorts by created, updated, comments only, filters by `issue_field_values`; a sort by field needs a project view | [filtering](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/filtering-and-searching-issues-and-pull-requests), issues API |
| History | the timeline records who changed a field, when; the `issues` webhook sends `field_added`, `field_removed` with the previous value | 03-12, adding fields |
| Helpers | a Copilot skill bulk-copies labels or project fields into fields; GitHub's MCP server reads and sets them | 05-21, 07-02 |

**API and tokens**

| What | Found | Source |
|---|---|---|
| REST per issue | `GET`, `POST`, `PUT /repos/{o}/{r}/issues/{n}/issue-field-values`, `DELETE …/{issue_field_id}`; `{field_id, value}`, a single-select by its option's **name**; header `X-GitHub-Api-Version: 2026-03-10`; push access | [field values](https://docs.github.com/en/rest/issues/issue-field-values) |
| REST on the issue | `PATCH`, `POST …/issues` take `issue_field_values`, each with `rationale`, `suggest`, `confidence` (low, medium, high) | issues API |
| GraphQL | `setIssueFieldValue(input: {issueId, issueFields})`: `fieldId`, `singleSelectOptionId`, …, `delete`, `suggest`, `rationale` (280 characters), `confidence` | [schema](https://docs.github.com/public/fpt/schema.docs.graphql) |
| Tokens | "Issues: write", from an App's installation token, a user token, a fine-grained token; `GITHUB_TOKEN` holds `issues: write`, so should too (inferred, to verify) | [App](https://docs.github.com/en/rest/authentication/permissions-required-for-github-apps), [fine-grained](https://docs.github.com/en/rest/authentication/permissions-required-for-fine-grained-personal-access-tokens), [installation](https://docs.github.com/en/rest/authentication/endpoints-available-for-github-app-installation-access-tokens), [workflow](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax) |

## GitHub: Projects fields, and projects' labels

- **Projects (v2) fields are per project**: the
  [quickstart](https://docs.github.com/en/issues/planning-and-tracking-with-projects/learning-about-projects/quickstart-for-projects)
  creates `Priority` and `Estimate` by hand; "there is no standard
  meaning" ([community](https://github.com/orgs/community/discussions/54055)).
  Written by GraphQL `updateProjectV2ItemFieldValue` or REST
  ([items](https://docs.github.com/en/rest/projects/items)); `GITHUB_TOKEN`
  "cannot access projects"
  ([automating](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/automating-projects-using-actions),
  [API](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-api-to-manage-projects));
  a user's project needs a classic token (inferred: the App tables list
  only `/orgs/` endpoints).

| Project | Priority | Effort |
|---|---|---|
| [kubernetes](https://github.com/kubernetes/kubernetes/labels) | `priority/critical-urgent`, `important-soon`, `important-longterm`, `backlog`, `awaiting-more-evidence` | none: `size/XS` to `XXL` count a pull request's lines, set by a bot |
| [rust](https://github.com/rust-lang/rust/labels) | `P-critical`, `P-high`, `P-medium`, `P-low` | — |
| [flutter](https://github.com/flutter/flutter/labels) | `P0` ("build break or regression") to `P3` | — |
| [angular](https://github.com/angular/angular/labels) | `P0` to `P5` | `effort1: hours`, `effort2: days`, `effort3: weeks` |
| [dotnet/runtime](https://github.com/dotnet/runtime/labels) | `Priority:0` "can't release without" to `Priority:3` "nice to have" | — |
| [kibana](https://github.com/elastic/kibana/labels) | `impact:critical`, `high`, `medium`, `low` | `effort:high`, `medium`, `low`; `estimate:small`, `medium`, `needs-research` |
| [grafana](https://github.com/grafana/grafana/labels) | `severity:*` | `effort/small`, `medium`, `large`, `x-large` |

One open issue each in vscode, github/docs, flutter, kubernetes and cli
showed no public field value: a field may be "Organization only" or unset.

## GitLab

| | Free | Premium, Ultimate |
|---|---|---|
| Priority | [prioritized labels](https://docs.gitlab.com/user/project/labels/): a project label starred and ranked, `priority` on the label ([labels API](https://docs.gitlab.com/api/labels/)) | the same, and scoped `priority::1` labels, one per scope |
| Effort | none; [time estimate](https://docs.gitlab.com/user/project/time_tracking/) is time | [`weight`](https://docs.gitlab.com/user/project/issues/issue_weight/), a whole number, 0 or more |
| Sorts ([lists](https://docs.gitlab.com/user/project/issues/sorting_issue_lists/)) | Label priority; Priority (milestone due date, then label priority); Manual | Weight, lowest first, none last |
| Custom fields | — | [select, number, text](https://docs.gitlab.com/user/work_items/custom_fields/), since 18.0 |

- **Prioritized labels**: a project's, not a group's; the
  [triage tutorial](https://docs.gitlab.com/tutorials/issue_triage/)
  treats them as Free and stars `priority::1` to `::4`. On Free, `::`
  labels "aren't mutually exclusive". A lower rank is more pressing:
  `label_priorities.priority ASC`, nulls last
  ([label.rb](https://gitlab.com/gitlab-org/gitlab/-/raw/master/app/models/label.rb),
  [issuable.rb](https://gitlab.com/gitlab-org/gitlab/-/raw/master/app/models/concerns/issuable.rb)).
- **Weight**: Premium since 13.9
  ([archive](https://archives.docs.gitlab.com/16.11/ee/user/project/issues/issue_weight.html));
  a new value replaces the old; Planner role; milestones sum weights,
  parents roll them up. API `weight` on `POST` and `PUT …/issues`,
  "Premium and Ultimate only"; `order_by` takes `label_priority`,
  `priority`, `weight`; a manual rank by `PUT …/reorder`
  ([issues API](https://docs.gitlab.com/api/issues/)).
- **GitLab's own convention**
  ([handbook](https://handbook.gitlab.com/handbook/product-development/how-we-work/issue-triage/)):
  `priority::1` Urgent, "as soon as possible regardless of the limit on
  our team capacity", 30 days; `::2` High, 60 to 90; `::3` Medium, 90 to
  120; `::4` Low, no timeline; set by the product group's owner.
- **Time estimate**: `/estimate 1w 2d`, one an item, a new one
  overwrites. Severity is for
  [incidents](https://docs.gitlab.com/operations/incident_management/incidents/) only.

## Gitea and Forgejo

| What | Gitea | Forgejo |
|---|---|---|
| Priority labels | "Advanced" set: `Priority/Critical`, `High`, `Medium`, `Low`, exclusive, `exclusive_order` 1 to 4 ([Advanced.yaml](https://github.com/go-gitea/gitea/blob/main/options/label/Advanced.yaml), [label.go](https://github.com/go-gitea/gitea/blob/main/modules/label/label.go)) | the same, no order ([Advanced.yaml](https://codeberg.org/forgejo/forgejo/src/branch/forgejo/options/label/Advanced.yaml)) |
| Sort by them | since 1.24, `sort=scope-Priority`, web only ([#33206](https://github.com/go-gitea/gitea/pull/33206), [release](https://blog.gitea.com/release-of-1.24.0/), [search](https://github.com/go-gitea/gitea/blob/main/models/issues/issue_search.go)) | none ([search](https://codeberg.org/forgejo/forgejo/src/branch/forgejo/models/issues/issue_search.go)) |
| API | cannot set `exclusive_order` ([#35740](https://github.com/go-gitea/gitea/issues/35740), [structs](https://github.com/go-gitea/gitea/blob/main/modules/structs/issue_label.go)); no sort on the issue list ([issue.go](https://github.com/go-gitea/gitea/blob/main/routers/api/v1/repo/issue.go)) | — |
| Effort | a time estimate (1.23), read by the API, set on the web only ([structs](https://github.com/go-gitea/gitea/blob/main/modules/structs/issue.go), [timetrack](https://github.com/go-gitea/gitea/blob/main/routers/web/repo/issue_timetrack.go)); an old `Priority` column not in the API ([model](https://github.com/go-gitea/gitea/blob/main/models/issues/issue.go)) | none ([structs](https://codeberg.org/forgejo/forgejo/src/branch/forgejo/modules/structs/issue.go)) |
| Time spent | tracked ([routes](https://github.com/go-gitea/gitea/blob/main/routers/api/v1/api.go)) | tracked |
| Versions read | [v28.1.0](https://github.com/go-gitea/gitea/releases) | [v16.0.5](https://codeberg.org/forgejo/forgejo/releases) |

## Other trackers' scales

| Tracker | Priority | Effort |
|---|---|---|
| [Linear](https://linear.app/docs/issue-property) | none, Low, Medium, High, Urgent; API 1 Urgent to 4 Low ([schema](https://github.com/linear/linear/blob/master/packages/sdk/src/schema.graphql)) | per team: exponential, Fibonacci, 1 to 5, T-shirt; label names priority, estimate, effort reserved |
| [Jira](https://support.atlassian.com/jira-cloud-administration/docs/configure-priorities-for-projects/) | Highest to Lowest, set by admins | Story Points, two incompatible fields ([help](https://support.atlassian.com/jira/kb/story-points-not-showing-in-custom-board-for-a-team-managed-jira-project/)) |
| [Azure DevOps](https://learn.microsoft.com/en-us/azure/devops/boards/queries/planning-ranking-priorities) | `Priority` 1, highest, to 4; Stack Rank by drag | Story Points, Effort or Size, "subjective" ([fields](https://learn.microsoft.com/en-us/azure/devops/boards/queries/query-numeric)) |
| [Shortcut](https://www.shortcut.com/help/fields-and-features/custom-fields) | P1 to P4, renamable | a scale per workspace, "Unestimated" first ([settings](https://www.shortcut.com/help/admin/workspace-features/)) |

The pattern: four levels, or 1 to 4 with 1 the highest; effort a coarse
scale of complexity, not time.

## Combining priority and effort

| Rule | How | Source |
|---|---|---|
| Value vs effort grid | quick wins (high value, low effort) first; big bets one at a time; fill-ins in the gaps; time sinks last; pitfalls: value badly estimated, effort underestimated | [savio](https://www.savio.io/product-roadmap/value-vs-effort-matrix/) |
| WSJF | cost of delay ÷ job size, in modified Fibonacci numbers; Azure DevOps ships fields for it | [2016](https://webarchive.library.unt.edu/web/20161225132214mp_/http://www.scaledagileframework.com/wsjf), [today](https://framework.scaledagile.com/wsjf) |
| RICE | reach × impact × confidence ÷ effort | [Intercom](https://www.intercom.com/blog/rice-simple-prioritization-for-product-managers/) |
| ICE | impact × confidence × ease; a low ease "can sharply drag down an item's total" | [ProductPlan](https://www.productplan.com/glossary/ice-scoring-model) |

For a role (inferred): the grid needs only the two values; a ratio lets
a noisy effort jump a priority level; reach and confidence are not
observable, which invites made-up numbers.

## Who estimates, and keeping a person's value

| Tool | What | Source |
|---|---|---|
| GitHub agent automation controls (preview) | an agent's change to labels, fields, type or assignees carries a rationale and a confidence; below the repository's level it waits as a suggestion (`has:suggestions`); "not a security control"; any integration may send `suggest: true` | [07-23](https://github.blog/changelog/2026-07-23-agent-automation-controls-in-github-issues-in-public-preview/), [about](https://docs.github.com/en/enterprise-cloud@latest/copilot/concepts/agents/cloud-agent/about-automation-rationale-and-approvals), [how-to](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/cloud-agent/manage-rationale-confidence-approvals) |
| Linear Triage Intelligence | suggests teams, projects, assignees, labels with its reasoning; not priority nor estimate | [docs](https://linear.app/docs/triage-intelligence) |
| Jira | no native estimate found; a marketplace app suggests points "with a confidence level and a short explanation", for "every unpointed ticket" | [AI Story Point Estimator](https://marketplace.atlassian.com/apps/334357954/ai-story-point-estimator-for-jira) |
| Zenhub | estimation only "exploring" | [blog](https://blog.zenhub.com/introducing-zenhub-ai-the-next-frontier-for-zenhub) |
| GitLab | no guard: a new weight or estimate replaces the old | issue weight, time tracking |

Three ways to keep a person's value: write only where empty; write a
suggestion (GitHub's `suggest`, Linear's "show"); read who changed it
last (GitHub's timeline, the webhook's previous value).

## Checked live (2026-10-11, read only)

- **jn0v-lab**, an organisation on GitHub Free: Priority, Effort, Start
  date, Target date, all "organization members only". Effort's options
  are listed **High first**: an option's position is not its rank.
- **JN0V/workline**, a user's repository: GraphQL `repository.issueFields`
  answers null; `GET …/suggestions` answers `200 []`.
- **workline's labels**: 18 open issues carry `workline:priority/2` (2),
  `/3` (10) or `/4` (6), all set by the role's bot, none changed by a
  person since; `/1` was never created. 8 of 9 `ready` issues have none.
- **GitLab**: an issue's JSON has no `weight` key on Free, has one on
  Ultimate. **GitHub's `PATCH`** answers `ignored` and `ignored_reason`
  per suggestion; a suggestion ends `pending`, `applied`, `approved`,
  `dismissed`, `replaced` or `invalidated`.
- **jn0v-lab/workline-sandbox** is a fork with issues off; `gh` in a
  fork's clone resolves to the parent unless `GH_REPO` is set.

## What to borrow, and what is missing

- **Borrowed**: GitHub's fields and their levels; GitLab's ranked
  `priority::` labels and `weight`; a project's own label family; the
  grid; writing only where empty.
- **Missing everywhere**: a tool that estimates effort from the code an
  issue names, and keeps a person's correction. Decided in
  [ADR-0040](../adr/0040-priority-and-effort-are-the-forges-the-product-owner-estimates-them.md).

## Not verified

- Whether `GITHUB_TOKEN` sets field values, and whether each token reads
  "organization members only" values on a public repository.
- Whether GitHub suggestions work on a user's repository, whom they
  notify; whether a listing gives field values without the version header.
- Which Projects templates create Priority, Size or Estimate; Shortcut's
  default estimate values (its help pages redirect).
