---
sources: [internal/gate/added.go, internal/line/line.go, internal/report/gates.go, internal/report/report.go, internal/engine/annotate.go, cmd/workline/main.go]
checked: cae7a5d3
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
  `failed`, `error`, `fix`; `kept`, the merge base's findings read as
  main kept them), and each new finding: check, tool, rule, SARIF level,
  file, line, message, and a fingerprint. On `main`, `present` names
  every finding the head holds, by its fingerprint
  ([main after a merge](../main-after-merge.md)).
- **The fingerprint** hashes the finding's identity
  ([gates.md](gates.md#on-a-merge-request-the-findings-the-change-adds))
  and how many alike come before it: no line number.

## The gates alone

- **`route <event> --gates-only`**: every gate step of the event runs,
  none of its roles. Their tools run the change's code: that job holds no
  write token and no agent's ([gates in CI](../gates-in-ci.md)).
- **`--gates-from <file>`**: the job judging the line reads each gate
  step's verdict there, and runs no gate. One missing from the file, or a
  file for other commits than the line judges, is a `check-error`: the
  gates' result says the commits its range named when they ran
  (`commits`), a branch moved since names others.
- **A gate's findings are reported by the job that ran it**: a step read
  from a file adds none to `--sarif` nor `--code-quality`.
- **The run apart** (`--warn-only --gates-from <file>`) reads no verdict
  there: it hands that result to the inspector, which triages the new
  findings, only for the commits it judges; for others, said
  (`gates-unread`), none given
  ([the inspector](../../roles/inspector/README.md)).

## Where they go

- **GitLab, Code Quality** (`--code-quality`): each new finding, its
  `check_name` `gate:<gate>/<check>/<rule>`, its severity the tool's
  level (`error` major, `warning` minor, `note` info), its fingerprint
  the one above. GitLab compares the report with the target branch's.
- **GitHub, a check and one comment** (`apply --gates <file>
  --from-merge-request <n> --head <sha>`), in the job holding the write
  token and no AI key:
  - the annotations of the check `workline gates` on the head judged,
    fifty a call, 500 at most, the rest counted;
  - one summary comment (`<!-- workline:sticky=gates/summary -->`),
    edited on each push: a row per check, a line per rule with its first
    three places, as many rules as the reviewer's `findings-max`.
- **Never code scanning** on a merge request: `--sarif` leaves the gates'
  findings out, and the inspector's triage of them.
- **Nothing to say**: with no new finding and every check passing, no
  check is written; the comment is only edited where an earlier push
  left one.
- **A check with `fix`**: its findings are one line of the comment, the
  command with them, and no annotation.

## The run's data

- **Its places** must be inside the repository; its texts are cut and
  their mentions defused.
- **The merge request it names** must have the head judged as its head,
  else nothing is shown (`moved-on`).
- **Said, never an error**: `gates-shown`, `gates-not-shown` (a refused
  check, a comment not written), `moved-on`.
