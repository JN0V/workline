<!-- workline
sources: [cmd/workline]
-->
# Develop

```sh
git clone https://github.com/JN0V/workline && cd workline
go build -o ~/.local/bin/workline ./cmd/workline   # or anywhere on your PATH
go test -count=1 ./...    # unit tests and the conformance suite
```

`-count=1` matters: the conformance suite builds the engine itself, which Go's
test cache does not see. The evaluation grades the roles with a real agent on
real cases, and costs tokens, so it runs only when asked:
`WORKLINE_EVAL=claude go test -count=1 -timeout 60m ./tests/evaluation/`
(docs/spec/conformance.md, "Evaluation").
