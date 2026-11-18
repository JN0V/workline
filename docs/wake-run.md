---
sources:
  - internal/wake/wake.go#Read
  - cmd/workline/wake.go#wakeCmd
  - .github/workflows/workline-apply.yml#jobs.wake
  - internal/forge/wake.go#AppStatuses
  - internal/forge/wake.go#github.AppStatus
  - internal/wake/wake.go#readForge
  - internal/forge/forge.go#Open
checked: e2e165c2d
verified: agent:claude-code
---
# Running the wake

Part of [The wake](wake.md).

## Run it

```sh
workline wake                                      # HEAD's first parent..HEAD
workline wake --forge github --range <before>..<after>
workline wake --forge github --range <before>..<after> --issue   # keep the issue
```

- **`--forge none`**, or no forge set: git's triggers alone; the
  forge's are said unread.
- **`--json`**: what it read, as data.
- **On GitHub**: the job `wake` of
  [workline-apply.yml](../.github/workflows/workline-apply.yml), on a
  push to `main`, with `issues: write`, `pull-requests: read` and
  `statuses: read`; the App's token when there is one, and its slug
  in `WORKLINE_APP`: a `workline/accepted` only that App set is a yes,
  another app's is not; with none named, the yes is said unread.
- **GitLab and other CI**: the same command after a push to the default
  branch, `--forge gitlab`; GitLab keeps no `workline/accepted`, so that
  trigger is not read there. Not in the templates yet.
