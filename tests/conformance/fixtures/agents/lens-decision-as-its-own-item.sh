#!/bin/sh
# A cmd: agent answering the intent lens as Sonnet did on PRs #490 and
# #492: a finding, then the open point written as an item of its own,
# `decision` and its `cause` side by side where one kind goes. Asked again
# with a reader saying those are no kinds but a finding's keys, it writes
# the decision inside a finding. Told only to send valid YAML, it sends
# the same answer again, as Sonnet did.
prompt=$(cat)
case "$prompt" in
*"Your previous answer was refused"*"cause and decision are no kind"*"finding"*)
	cat <<'YAML'
- finding:
    lens: intent
    severity: nit
    title: No test of the zeros
    why: "#4 asks a test of Average on [0, 2, 4]; the change adds none"
    cause: {path: "#4", quote: "A test of Average on [0, 2, 4]."}
- finding:
    lens: intent
    title: What an empty list gives
    why: "#4 leaves it open, and the change had to choose"
    decision: Should an empty list give 0, or an error?
    cause: {path: "#4", quote: "What an empty list gives is left open."}
YAML
	;;
*)
	cat <<'YAML'
- finding:
    lens: intent
    severity: nit
    title: No test of the zeros
    why: "#4 asks a test of Average on [0, 2, 4]; the change adds none"
    cause: {path: "#4", quote: "A test of Average on [0, 2, 4]."}
- decision: "Should an empty list give 0, or an error? #4 leaves it open."
  cause:
    path: "#4"
    quote: "What an empty list gives is left open."
YAML
	;;
esac
