# Issue formats — focused research (2026-10-09, extended 2026-10-11)

**Question.** When a project already has an issue format, where does the
forge hold it, and which tools fill it? Which names do projects give the
problem, a case, what a person checks, the tests and the code? How do
they mark a bug with no reproduction, and how do tools record and set an
issue's kind? For the product owner's sections
([writing an issue](../../roles/product-owner/docs/writing.md)); extends
[product-owner.md](product-owner.md) (forms, labels, triage).

**Verdict.**

- **Templates are files** in the repository, GitHub's YAML forms read by
  Gitea and Forgejo too; a submitted form is Markdown, a heading a field.
- **Established names**: a bug asks "Steps to reproduce", "Expected
  behavior", "Actual behavior", a version; what a person checks is
  "Acceptance criteria"; the builder's tests, a "Test plan".
  "Validation" and "Verification" mean something else in trackers.
- **A bug with no reproduction** gets a waiting label beside the decision
  label, often with a bot that closes it.
- **The kind** is GitHub's issue type in an organization, else a label.
  Bots differ on a person's correction: one never re-adds a label
  removed, one re-adds it on every edit.
- **No tool** checks that a filled template is enough to build from.
  Forms have no language key; workline writes English only.
- **A section that repeats another** is found as near-duplicate text
  is: the share of its words already said, over shingles.

Searched in the ecosystem's words: *issue forms*, *description
templates*, *issue types*, *steps to reproduce*, *expected behavior*,
*acceptance criteria*, *test plan*, *needs repro*, *waiting for info*,
*near-duplicate detection*, *shingling*, *containment*.
Templates and label lists read on the default branches on 2026-10-09
and 2026-10-11; counts from GitHub's code search on 2026-10-11.

## Where the forge holds the format

| Forge | Files | Form or Markdown | Kind set by the template |
|---|---|---|---|
| GitHub ([forms](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/syntax-for-issue-forms), [templates](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/configuring-issue-templates-for-your-repository)) | `.github/ISSUE_TEMPLATE/*.yml`, `*.md`, `config.yml` | both | `labels`, `type` (an organization's issue type) |
| GitLab ([description templates](https://docs.gitlab.com/user/project/description_templates/)) | `.gitlab/issue_templates/*.md`; a default in settings, group templates: Premium | Markdown only | a `/label ~type::bug` quick action in the template |
| Gitea ([templates](https://docs.gitea.com/usage/issue-pull-request-templates)) | `ISSUE_TEMPLATE/` at the root, in `.gitea/`, `.github/` or `.gitlab/` | both, GitHub's YAML shape | `labels` |
| Forgejo ([templates](https://forgejo.org/docs/latest/user/issue-pull-request-templates/)) | `issue_template/` in `.forgejo/`, `.gitea/`, `.github/` or `docs/` | both, as Gitea | `labels` |

- **A submitted form is Markdown**: `### <label>`, then the answer; an
  empty optional field reads `_No response_` (golang/go
  [#81670](https://github.com/golang/go/issues/81670)). The field's `id`
  is not in the body.
- **A form holds in the web page only**: an issue opened by the API has
  the headings its author typed (kubernetes
  [#142886](https://github.com/kubernetes/kubernetes/issues/142886): the
  form's labels under `##`). `required` is enforced there alone.

## Section names

Files under `.github/ISSUE_TEMPLATE` holding the phrase, by GitHub's code
search: files, not headings, case ignored, copies counted. "Large forms":
the bug forms of 21 and the feature forms of 16 large projects
(kubernetes, react, terraform, node, flutter, TypeScript, GitLab…).

| Meaning | Names found | Template files |
|---|---|---|
| The problem | GitHub's built-in feature template: "Is your feature request related to a problem"; "Problem statement"; node "What is the problem this feature will solve?"; GitLab "Problem to solve" | 49,536; 17,024; a problem heading in 6 of 16 large feature forms |
| A case | TypeScript "Motivating Example"; "Use case", under which terraform and flutter put the need | "Use case" 40,448 |
| A bug's case | "Steps to reproduce", "Expected behavior", "Actual behavior", a version | 185,088; 176,128; 66,048; "Steps to reproduce" word for word in 8 of 21 large bug forms |
| What a person checks | "Acceptance criteria"; "Definition of done"; "Done when" | 39,104; 4,784; 1,380 |
| The tests | "Test cases"; "Test plan" | 1,452; 846 |
| The code | "Implementation details"; "Technical details"; "Implementation notes"; "Implementation plan" | 3,304; 3,136; 2,412; 1,350 |
| In or out of the work | "Scope"; "Out of scope"; "Non-goals" | 39,808; 6,944; 2,848 |
| Open points | "Open questions"; "Unresolved questions" | 1,752; 547 |

## What the names mean where they are used

| Name | Used by, holding |
|---|---|
| Acceptance criteria | an Azure DevOps [field](https://learn.microsoft.com/en-us/azure/devops/boards/queries/titles-ids-descriptions) on Bug and Feature, "the criteria that must be met before the bug … can be closed"; [Atlassian](https://www.atlassian.com/work-management/project-management/acceptance-criteria): "clear, concise, and testable statements"; [GOV.UK](https://www.gov.uk/service-manual/agile/writing-user-stories): "a list that begins with 'it's done when…'"; Dan North's [story template](https://dannorth.net/blog/whats-in-a-story/): "Acceptance Criteria: (presented as Scenarios)", each a Given / When / Then; GitHub's [coding agent guide](https://docs.github.com/en/copilot/tutorials/coding-agent/get-the-best-results): "complete acceptance criteria" |
| Test plan | Kubernetes enhancement proposals' [template](https://github.com/kubernetes/enhancements/blob/master/keps/NNNN-kep-template/README.md): unit, integration, e2e tests; [Phabricator](https://secure.phabricator.com/book/phabricator/article/differential_test_plans/): "a repeatable list of steps"; Docusaurus's pull-request template. Tests are mostly asked in pull-request templates |
| The code | GitLab's ["Implementation plan"](https://gitlab.com/gitlab-org/gitlab/-/blob/master/.gitlab/issue_templates/Implementation.md): "the parts of the code that will need to get updated"; its bug template's ["Possible fixes"](https://gitlab.com/gitlab-org/gitlab/-/blob/master/.gitlab/issue_templates/Bug.md); GitHub's coding agent guide: "which files need to be changed" |
| Verification | a person re-checking a fix: VS Code's `verification-needed` ([wiki](https://github.com/microsoft/vscode/wiki/Issue-Tracking)); GitLab's `workflow::verification`, "the responsible engineer will manually test" ([handbook](https://handbook.gitlab.com/handbook/product-development/how-we-work/product-development-flow/)); GitLab's template field "Verification steps" |
| Validation | checking that a problem is real: GitLab's "Problem Validation"; Kubernetes' triage, "validate if the problem is a bug by trying to reproduce it" ([guide](https://github.com/kubernetes/community/blob/main/contributors/guide/issue-triage.md)); the engineering sense "can be inconsistent" ([Wikipedia](https://en.wikipedia.org/wiki/Verification_and_validation)) |
| Scope | what is in or out of the work, not which files |

## Bugs with no reproduction

Label lists read on each repository on 2026-10-11.

| Project | Waiting label | Decision label | Bot |
|---|---|---|---|
| [vscode](https://github.com/microsoft/vscode/labels) | `info-needed`, `repro-steps-needed` | `confirmed` | closes after 7 days ([wiki](https://github.com/microsoft/vscode/wiki/Issues-Triaging)) |
| [kubernetes](https://github.com/kubernetes/kubernetes/labels) | `triage/needs-information` | `triage/accepted` | stale after 90 days |
| [golang/go](https://github.com/golang/go/labels) | `WaitingForInfo` | `NeedsFix` | closes after a month ([gopherbot](https://github.com/golang/build/blob/master/cmd/gopherbot/gopherbot.go)) |
| [rust](https://github.com/rust-lang/rust/labels) | `S-needs-repro` | `C-feature-accepted` | — |
| [terraform](https://github.com/hashicorp/terraform/labels) | `waiting for reproduction` | `confirmed` | — |
| [angular](https://github.com/angular/angular/labels) | `needs reproduction` | `state: confirmed` | — |
| [next.js](https://github.com/vercel/next.js/labels) | `please add a complete reproduction` | — | closes after 2 days ([triaging](https://github.com/vercel/next.js/blob/canary/contributing/repository/triaging.md)) |
| [electron](https://github.com/electron/electron/labels) | `blocked/need-repro` | `status/confirmed` | — |
| Gitea, Forgejo ([Advanced](https://github.com/go-gitea/gitea/blob/main/options/label/Advanced.yaml)) | `Status/Need More Info` | `Reviewed/Confirmed` | — |

- "repro" is the word of 8 projects for a missing reproduction.
- **Accepted, but waiting**: two independent labels. Go's `NeedsFix`
  "may have the labels `Blocked` or `WaitingForInfo`"
  ([HandlingIssues](https://go.dev/wiki/HandlingIssues)); Gitea's
  `Reviewed/*` and `Status/*` are separate scopes. Or a section,
  "Unresolved questions" in Rust's tracking issues.
- **A reproduction is broader than steps**: angular, next.js and electron
  ask for a link to one, TypeScript for a playground, rust for code.

## Kinds: recorded and set by tools

| Forge | A type of the forge | Default labels |
|---|---|---|
| GitHub | [issue types](https://docs.github.com/en/issues/tracking-your-work-with-issues/configuring-issues/managing-issue-types-in-an-organization), an organization's: task, bug, feature by default | `bug`, `documentation`, `enhancement`, `question`… ([labels](https://docs.github.com/en/issues/using-labels-and-milestones-to-track-work/managing-labels)) |
| GitLab | [`issue_type`](https://docs.gitlab.com/api/issues/): `issue`, `incident`, `test_case`, `task` | `bug`, `enhancement`, `documentation`… when generated; `type::bug`, `type::feature`, `type::maintenance` by GitLab's own [convention](https://docs.gitlab.com/development/labels/) |
| Gitea, Forgejo | none | `bug`, `enhancement`… ("Default"); `Kind/Bug`, `Kind/Feature`… ("Advanced"), not exclusive |

- **Setting a GitHub type**: "Only users with push access can set the
  type … type changes are silently dropped"
  ([REST](https://docs.github.com/en/rest/issues/issues)). On a personal
  account's repository, REST lists Task, Bug, Feature while GraphQL's
  `issueTypes` is null (JN0V/workline, observed 2026-10-11).
- **GitHub's agent suggestions** (preview, 2026-07-23): an automation
  suggests a type or a label; "High-confidence changes apply
  automatically", the rest wait for a person
  ([changelog](https://github.blog/changelog/2026-07-23-agent-automation-controls-in-github-issues-in-public-preview)).
- **GitLab**: an issue's type is edited to `issue`, `incident` or
  `test_case` only; scoped labels (`key::value`) replace each other on
  Premium ([labels](https://docs.gitlab.com/user/project/labels/)).
- **A person's correction**: Go's gopherbot skips any issue whose label
  was ever removed (`HasEvent("unlabeled")`);
  [issue-labeler](https://github.com/github/issue-labeler) adds every
  matching label again on each edit. VS Code (`triage-miss`) and GitLab
  (`automation:ml wrong`) keep a label saying the bot was wrong.

## Tools that draft or fill an issue

| Tool | Follows the project's template? | How |
|---|---|---|
| GitHub Copilot ([docs](https://docs.github.com/en/copilot/how-tos/copilot-on-github/copilot-for-github-tasks/use-copilot-to-create-or-update-issues), [changelog](https://github.blog/changelog/2025-07-16-support-for-issue-forms-in-chat-and-file-uploads-in-spaces)) | yes | "Copilot maps your prompt to the relevant fields"; suggests labels and the issue type; a person reviews the draft |
| GitLab Duo, "Generate issue description" | no | a request to pick a template ([gitlab#579104](https://gitlab.com/gitlab-org/gitlab/-/issues/579104)): "users cannot leverage existing issue templates" |
| GitHub Models triage ([blog](https://github.blog/ai-and-ml/generative-ai/automate-your-project-with-github-models-in-actions/)) | no, its own checks | looks for steps, expected and actual; comments what is missing |
| [request-info](https://github.com/behaviorbot/request-info) | partly | flags a body equal to the template, label `needs-more-info` |
| [github-issue-parser](https://github.com/stefanbuck/github-issue-parser) | reads it | the form and its body to JSON, by field `id` |

## Writing a bug, and how it is accepted

- **A bug**: steps to reproduce are "the most important part"
  ([Mozilla](https://bugzilla.mozilla.org/page.cgi?id=bug-writing.html));
  "let the programmer see the failure with their own eyes"
  ([Simon Tatham](https://www.chiark.greenend.org.uk/~sgtatham/bugs.html)).
  Its acceptance is the reproduction with the expected result.
- **Acceptance, two shapes**: Given / When / Then scenarios
  ([martinfowler.com](https://martinfowler.com/bliki/GivenWhenThen.html))
  for behaviour; "done when …" items for constraints. A checklist
  required before work starts can become "a big, burly bouncer"
  ([Mountain Goat Software](https://www.mountaingoatsoftware.com/blog/the-dangers-of-a-definition-of-ready)).

## A draft that repeats

- **Near-duplicate detection** cuts each text into shingles, every run
  of k consecutive words, and compares the sets
  ([Manning, Raghavan, Schütze, *Introduction to Information Retrieval*, 19.6](https://nlp.stanford.edu/IR-book/html/htmledition/near-duplicates-and-shingling-1.html)):
  for web pages, k = 4, near duplicates past a Jaccard of 0.9.
- **Two measures** ([Broder, "On the resemblance and containment of
  documents", 1997](https://doi.org/10.1109/SEQUEN.1997.666900)):
  resemblance, the Jaccard of the two sets; containment of A in B, the
  share of A's shingles found in B.
- **For a short draft against a longer text**, containment: Jaccard
  dilutes a draft wholly inside a long text. One-word shingles, common
  words left out: a restated problem is a paraphrase, not a copy.
  [#103](https://github.com/JN0V/workline/issues/103)'s Example scores
  54% against its text on top at one word, 18% at three
  ([ADR-0039](../adr/0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md),
  amended 2026-10-11); the engine's own, with its common words, in
  [repeat_test.go](../../internal/backlog/repeat_test.go).

## Languages

- **Forms have no language key.** Projects copy a template per language
  ([vant](https://github.com/youzan/vant/tree/main/.github/ISSUE_TEMPLATE)),
  write each label in two languages
  ([Paddle](https://github.com/PaddlePaddle/Paddle/tree/develop/.github/ISSUE_TEMPLATE)),
  or write in one language other than English
  ([etalab/data.gouv.fr](https://github.com/etalab/data.gouv.fr/tree/master/.github/ISSUE_TEMPLATE)).
- **For workline: English only**: another language, supported for real,
  needs every engine line, conformance case and blind measure again
  ([ADR-0039](../adr/0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md)).

## What to borrow, and what is missing

- **Borrowed**: the forge's templates, untouched; the established names;
  two labels for "accepted, but waiting", as Go and Gitea do; the kind as
  the forge holds it, a label a person removed never set again, as
  gopherbot does; containment over shingles, to find a draft that
  repeats.
- **Missing**: a template's field read as a meaning (a documentalist
  setting already takes a project's own headings,
  [#260](https://github.com/JN0V/workline/pull/260)); what a template
  lacks, added after the reporter's fields; a bug kept from being worked
  on for want of a reproduction — bots close such bugs instead; whether
  a filled template is enough to build from, the role's judgment.
