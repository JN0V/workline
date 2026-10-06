# ADR-0035: The engine writes the job's summary

- **Status:** accepted
- **Date:** 2026-10-06
- **Builds on:** ADR-0010 (docs judged on each merge request and when
  gardening), ADR-0011 (the review is the merge request's); principles 10,
  12, 14
- **Settles:** #173 (a readable job summary on GitLab and any CI)

## Context

- GitHub's templates built a summary of `line.json` with jq and wrote it
  to `$GITHUB_STEP_SUMMARY`; the gardening and sample templates each kept
  a copy of that jq.
- GitLab's template had none: a merge request showed the findings in its
  Code Quality report, but a scheduled job, or one another tool starts
  (docs/gitlab-trigger.md), left a person the log and `line.json`.
- GitLab does not show a Markdown artifact: its job page offers it as a
  download. With Pages (gitlab.com), it previews an `.html`, `.txt`,
  `.json`, `.xml` or `.log` artifact in the browser. A job's page links
  out through `artifacts:reports:annotations` (`external_link`); a merge
  request's page links an artifact through `artifacts:expose_as`.

## Decision

- **The engine renders the summary**, from its result: `--summary <file>`
  on `route`, `run-role`, `apply` and `sample` ([usage.md](../usage.md)).
  The verdict, each step's verdict with its findings under it, the agent's
  calls and notes, what was applied, what waits to be. Every finding is
  there, those that name no file too, unlike SARIF and Code Quality.
- **Markdown, the look GitHub's jq gave**; **HTML for a file named
  `.html`**, every text escaped. Repeatable, to write both.
- **Added to, never overwritten**, as `$GITHUB_STEP_SUMMARY` is: the
  applying job receives the judging job's file and adds what it applied;
  one page tells the whole run.
- **GitHub**: the templates pass `--summary` and copy the file to
  `$GITHUB_STEP_SUMMARY`; no jq rebuilds it.
- **GitLab**: each job writes both files, keeps them as artifacts
  (`when: always`), prints the Markdown at the end of the log, and links
  the HTML from the job's page (annotations) and, on a merge request, from
  its page (`expose_as`).

## Consequences

- One rendering for every CI: another CI shows the Markdown or the HTML,
  whichever it reads ([ci.md](../ci.md)).
- A template copied from `main` needs the first release after v0.16.0:
  an older engine refuses `--summary`.
- Where GitLab Pages is off (a self-managed instance may have it off), the
  HTML link downloads the file; the log keeps the Markdown.
