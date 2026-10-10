# Issue formats — focused research (2026-10-09)

**Question.** When a project already has an issue format, where does the
forge hold it, and which tools fill it? How do projects describe a bug,
a feature, a task, and how is an issue's kind recorded? How do templates
handle languages? For the product owner's sections
([writing an issue](../../roles/product-owner/docs/writing.md)); extends
[product-owner.md](product-owner.md) (forms, labels, triage).

**Verdict.** Every forge keeps the format in the repository, as files:
GitHub's YAML forms, also read by Gitea and Forgejo, and Markdown
templates everywhere, GitLab included. A submitted form is plain
Markdown, each field under its label as a heading. The kind is the
forge's issue type (GitHub organizations) or a label, often set by the
template itself. Bugs ask the same few things everywhere; features ask
"what" and "why", never a story sentence. Only one tool fills a
project's own template, mapping to its fields, never to fixed meanings;
none checks that a filled template is enough to build from. Forms have
no language key.

Searched in the ecosystem's words: *issue forms*, *issue templates*,
*description templates*, *issue types*, *type labels*, *steps to
reproduce*, *expected behavior*, *acceptance criteria*, *bug report
template*, *generate issue description*. Template contents were read
from the repositories' default branches on 2026-10-09.

## Where the forge holds the format

| Forge | Files | Form or Markdown | Kind set by the template |
|---|---|---|---|
| GitHub ([forms](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/syntax-for-issue-forms), [templates](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/configuring-issue-templates-for-your-repository)) | `.github/ISSUE_TEMPLATE/*.yml`, `*.md`, `config.yml` | both | `labels`, `type` (an organization's issue type) |
| GitLab ([description templates](https://docs.gitlab.com/user/project/description_templates/)) | `.gitlab/issue_templates/*.md` | Markdown only | a `/label ~type::bug` quick action in the template |
| Gitea ([templates](https://docs.gitea.com/usage/issue-pull-request-templates)) | `ISSUE_TEMPLATE/` at the root, in `.gitea/`, `.github/` or `.gitlab/` | both, GitHub's YAML shape | `labels` |
| Forgejo ([templates](https://forgejo.org/docs/latest/user/issue-pull-request-templates/)) | `issue_template/` in `.forgejo/`, `.gitea/`, `.github/` or `docs/` | both, as Gitea | `labels` |

- **A submitted form is Markdown**: each field becomes `### <label>`, then
  the answer; an empty optional field reads `_No response_` (golang/go
  [#81670](https://github.com/golang/go/issues/81670)). It is read back by
  its headings; the field's `id` is not in the body.
- **A form holds in the web page only**: an issue opened by the API or a
  command line has no form, and its headings are what its author typed
  (kubernetes [#142886](https://github.com/kubernetes/kubernetes/issues/142886):
  the form's labels under `##`, where the form writes `###`). `required`
  is enforced there alone ([product-owner.md](product-owner.md)).
- **GitLab's tiers**: `Default.md` in the folder is the default on every
  plan; a default set in the project's settings, group templates and
  instance templates are Premium.
- **GitHub's order**: alphanumeric, YAML before Markdown (`00-bug.yml`);
  `config.yml` has `blank_issues_enabled` and `contact_links`, which send
  questions to a forum. An organization's `.github` repository gives
  defaults.

## What projects ask

| Project | Kinds offered | Bug fields |
|---|---|---|
| kubernetes | bug, enhancement, failing test, flaking test; support sent to a forum | What happened? / What did you expect to happen? / How can we reproduce it (as minimally and precisely as possible)? / versions, cloud, OS |
| golang/go | bug, pkgsite, gopls, vuln, proposal, language change, telemetry | Go version / `go env` / What did you do? / What did you see happen? / What did you expect to see? |
| rust-lang/rust | bug, regression, ICE, diagnostics, docs, tracking issue | "I tried this code:" / "I expected to see this happen:" / "Instead, this happened:" / `rustc --version` / backtrace |
| facebook/react | bug only; blank issues off | version / Steps To Reproduce / link to code / current / expected |
| microsoft/vscode | bug, feature | version, OS / Steps to Reproduce; the app's "Report Issue" pre-fills it |
| GitLab itself ([templates](https://gitlab.com/gitlab-org/gitlab/-/tree/master/.gitlab/issue_templates)) | Bug, Feature proposal (basic, lean, detailed), Documentation, Refactoring, Spike… | Summary / Steps to reproduce / Example project / current behavior / expected behavior / logs / environment |

- **Bugs, everywhere**: steps or code to reproduce, expected, actual,
  version and environment, logs. React: "Issues without reproduction
  steps … may be immediately closed".
- **Features**: two questions. kubernetes: "What would you like to be
  added?" / "Why is this needed?"; GitLab's lean proposal: "Problem to
  solve" / "Proposal" / "Intended users". None asks for "As a …, I want
  …, so that …".
- **Other kinds**: failing and flaky tests (kubernetes), a regression with
  the version it worked on (Rust), docs with the page's address (Rust),
  refactoring (GitLab). Questions are sent away (`contact_links`).
- **Short**: Go's bug form has 5 fields, kubernetes' 10; GitLab offers a
  lean and a detailed feature proposal.

## How an issue's kind is recorded

| Forge | A type of the forge | Labels |
|---|---|---|
| GitHub | [issue types](https://docs.github.com/en/issues/tracking-your-work-with-issues/configuring-issues/managing-issue-types-in-an-organization), organizations only: task, bug, feature by default, up to 25 | [defaults](https://docs.github.com/en/issues/using-labels-and-milestones-to-track-work/managing-labels): `bug`, `enhancement`, `documentation`, `question`… |
| GitLab | [`issue_type`](https://docs.gitlab.com/api/issues/): `issue`, `incident`, `test_case`, `task` | `type::bug`, `type::feature`, `type::maintenance`: GitLab's own [convention](https://docs.gitlab.com/ee/development/labels.html), "one and only one" per issue, not enforced |
| Gitea, Forgejo | none | the "Advanced" [label set](https://github.com/go-gitea/gitea/blob/main/options/label/Advanced.yaml): `Kind/Bug`, `Kind/Feature`, `Kind/Enhancement`, `Kind/Documentation`… |

- A repository of a personal GitHub account has no issue types: labels
  carry the kind there. kubernetes writes `kind/bug`.

## Tools that draft or fill an issue

| Tool | Follows the project's template? | How |
|---|---|---|
| GitHub Copilot ([docs](https://docs.github.com/en/copilot/how-tos/copilot-on-github/copilot-for-github-tasks/use-copilot-to-create-or-update-issues), [changelog](https://github.blog/changelog/2025-07-16-support-for-issue-forms-in-chat-and-file-uploads-in-spaces)) | yes | "Copilot maps your prompt to the relevant fields"; suggests labels, assignees and the issue type; a person reviews the draft before it is created; "ensure all required fields are completed" |
| GitLab Duo, "Generate issue description" | no | expands a short summary; a request to pick a template ([gitlab#579104](https://gitlab.com/gitlab-org/gitlab/-/issues/579104)): "users cannot leverage existing issue templates" |
| Linear ([templates](https://linear.app/docs/issue-templates)) | n/a | standard and form templates, defaults per team; no AI filling documented |
| Jira with Rovo ([docs](https://support.atlassian.com/jira-software-cloud/docs/create-work-items-with-rovo/)) | not documented | drafts work items; acceptance criteria on request |
| GitHub Models triage ([blog](https://github.blog/ai-and-ml/generative-ai/automate-your-project-with-github-models-in-actions/)) | no, its own checks | looks for steps, expected and actual, environment; comments what is missing |
| [request-info](https://github.com/behaviorbot/request-info) | partly | flags an issue whose body equals the template, label `needs-more-info` |
| [github-issue-parser](https://github.com/stefanbuck/github-issue-parser) | reads it | the form and the `### label` body to JSON, by field `id` |

- Only Copilot fills a project's own template, and it maps to the form's
  fields: no tool maps them to fixed meanings (the need, how it is
  checked, the code it touches), nor checks that a filled template is
  enough to build from.

## Writing a need, a bug, and how it is accepted

- **A bug**, the guides agree:
  - [Mozilla](https://bugzilla.mozilla.org/page.cgi?id=bug-writing.html):
    one bug a report; steps to reproduce are "the most important part";
    expected, actual, the build; say whether it happens always,
    sometimes or never; a title that "quickly and uniquely" identifies
    it, about 60 characters, no solution in it.
  - [Simon Tatham](https://www.chiark.greenend.org.uk/~sgtatham/bugs.html):
    "let the programmer see the failure with their own eyes"; facts apart
    from guesses; the symptoms even with a diagnosis.
  - [Chromium](https://www.chromium.org/for-testers/bug-reporting-guidelines/):
    a reduced test case attached when there can be one.
  - No guide writes a bug as a wish. Its example is the reproduction; its
    acceptance is the same reproduction with the expected result.
- **A need**: GOV.UK's [service manual](https://www.gov.uk/service-manual/agile/writing-user-stories)
  accepts any form naming who, what and why, the goal being what matters.
- **Acceptance, two shapes**: scenarios as Given / When / Then
  ([martinfowler.com](https://martinfowler.com/bliki/GivenWhenThen.html)),
  the same shape as a test's arrange, act, assert, for behaviour a person
  sees; a checklist, "done when …", for constraints that hold everywhere
  (a task, a doc). A rule whose outcome is unclear is a question, not a
  criterion ([Cucumber's blog](https://cucumber.io/blog/bdd/example-mapping-introduction/)).
- **Before work starts**: a team's checklist of what an item needs can
  become "a big, burly bouncer"
  ([Mountain Goat Software](https://www.mountaingoatsoftware.com/blog/the-dangers-of-a-definition-of-ready)):
  keep it to what a builder cannot start without.
- **Readable by anyone**: [Google's guide](https://developers.google.com/style/translation)
  for a global audience: short sentences, the main idea first, one term
  per idea, no idioms; a real case before the rule.

## Languages

- **Forms have no language key.** Projects copy a template per language
  ([vant](https://github.com/youzan/vant/tree/main/.github/ISSUE_TEMPLATE):
  `…en-US.yml` beside `…zh-CN.yml`), write two-language labels
  ([Paddle](https://github.com/PaddlePaddle/Paddle/tree/develop/.github/ISSUE_TEMPLATE):
  "bug描述 Describe the Bug"), or write in one language
  ([etalab/data.gouv.fr](https://github.com/etalab/data.gouv.fr/tree/master/.github/ISSUE_TEMPLATE):
  "Description du bug", "Comment reproduire le bug", "Comportement
  attendu").
- **Given / When / Then** is translated in Cucumber's
  [Gherkin](https://cucumber.io/docs/gherkin/reference/), "over 70
  languages", chosen by `# language: fr` (Étant donné / Quand / Alors).
- Copilot's docs say nothing of the language it writes in.

## What to borrow

- **The forge's own templates, untouched**: the reporter's fields, their
  words, order and heading level; `_No response_` as empty.
- **The kind as the forge holds it**: GitHub's issue type where an
  organization has them, else the labels a template sets: `bug`,
  `type::bug`, `Kind/Bug`.
- **A bug's fields**: steps, expected, actual, version; its acceptance
  is its steps with the expected result.
- **A feature's**: the problem before the proposal.
- **Scenarios for behaviour, a checklist for constraints**; no clear
  outcome, a question.
- **Given / When / Then's words per language**, as Gherkin translates them.

## What is missing, for workline to complete

- **A field read as a meaning**: nothing tells that "What did you expect
  to happen?" or "Comportement attendu" is how a person checks the
  result. workline's `ready` needs it to read a template's fields.
- **What a template lacks**: most bug forms have no field for the code a
  change touches nor for how the machine proves it; added after the
  reporter's fields, never inside them.
- **Headings in the project's language** for what workline writes; a
  documentalist setting already takes headings in any language
  ([#260](https://github.com/JN0V/workline/pull/260)).
- **Whether a filled template is enough to build from**: no tool checks
  it; it is the role's judgment.
