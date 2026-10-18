---
sources: [internal/gate/added.go, internal/line/line.go, internal/report/report.go, cmd/workline/main.go]
checked: 6ac16e6d
verified: agent:claude-code
---
# Gates — the new findings shown on a merge request

What [gates.md](gates.md#shown-on-the-merge-request) says in short: where
the findings a change adds go, and how
([ADR-0049](../adr/0049-the-inspector-judges-the-new-findings-of-the-projects-tools.md),
[ADR-0047](../adr/0047-findings-on-a-merge-request-open-no-conversation.md)).

## What a gate step holds

- **On an event given a range**, its result in `--json` holds `new`: the
  merge base, each check's count (`new`; `before`, `-1` when not known;
  `failed`, `error`, `fix`), and each new finding: check, tool, rule,
  SARIF level, file, line, message, and a fingerprint.
- **The fingerprint** hashes the finding's identity
  ([gates.md](gates.md#on-a-merge-request-the-findings-the-change-adds))
  and how many alike come before it: no line number.

## The gates alone

- **`route <event> --gates-only`**: every gate step of the event runs,
  none of its roles. Their tools run the change's code: that job holds no
  write token and no agent's ([gates in CI](../gates-in-ci.md)).
- **`--gates-from <file>`**: the job judging the line reads each gate
  step's verdict there, and runs no gate. One missing from the file, or a
  file for other commits than the line judges, however the range spells
  them, is a `check-error`.
- **A gate's findings are reported by the job that ran it**: a step read
  from a file adds none to `--sarif` nor `--code-quality`.

## Where they go

- **GitLab, Code Quality** (`--code-quality`): each new finding, its
  `check_name` `gate:<gate>/<check>/<rule>`, its severity the tool's
  level (`error` major, `warning` minor, `note` info), its fingerprint
  the one above. GitLab compares the report with the target branch's.
- **Never code scanning** on a merge request: `--sarif` leaves the gates'
  findings out.
- **GitHub**: the judge's job summary says each gate's verdict and its
  first findings; a check's annotations and one summary comment are not
  built yet.
