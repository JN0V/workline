<!-- workline
sources: [ci/github, cmd/workline]
verified: agent:documentalist
-->

# Importing a file on GitHub Actions

The workflow [ci-github.md](../ci-github.md#importing-a-file) tells of, run by hand:

```yaml
on: {workflow_dispatch: {inputs: {file: {required: true}}}}
permissions: {}
env: {WORKLINE_RUNS_DIR: "${{ github.workspace }}/.workline-runs"}
jobs:
  judge:                                   # the agent, a read token
    runs-on: ubuntu-latest
    permissions: {contents: read, issues: read}
    steps:
      - uses: actions/checkout@v4
      # + the engine and the agent
      # AGENT: your choice — claude, claude:<model>, cmd:<command> (another agent, its own install and token)
      - env: {AGENT: claude, GH_TOKEN: "${{ github.token }}", CLAUDE_CODE_OAUTH_TOKEN: "${{ secrets.CLAUDE_CODE_OAUTH_TOKEN }}", FILE: "${{ inputs.file }}"}
        run: workline issues import "$FILE" --ai "$AGENT" --forge github --summary "$GITHUB_STEP_SUMMARY" --json > line.json
      - uses: actions/upload-artifact@v4
        if: always()
        with: {name: workline-import, path: "line.json\n.workline-runs", include-hidden-files: true}
  apply:                                   # the write token, no agent
    needs: judge
    if: always()
    runs-on: ubuntu-latest
    permissions: {contents: read, issues: write}
    steps:
      - uses: actions/checkout@v4
      # + the engine
      - uses: actions/download-artifact@v4
        with: {name: workline-import}
      - env: {GH_TOKEN: "${{ github.token }}"}
        run: workline apply --line line.json --summary "$GITHUB_STEP_SUMMARY"
```
