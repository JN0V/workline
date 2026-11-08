#!/bin/sh
# A cmd: agent playing the product owner on an issue too big for the line
# (ADR-0064, step 4): it splits issue $1 into two parts that pass apart,
# `too-big: true` unless $2 is `normal`; with $3 `blocked`, it keeps the
# parent blocked for a person, who accepts it from its parts.
cat >/dev/null
big=true
[ "${2:-}" = normal ] && big=false
cat <<TEXT
- split:
    issue: $1
    too-big: $big
    why: "multiplying and dividing each pass their own test"
    into:
      - title: "Multiply two numbers"
        need: "When I total an order, I want to multiply a price by a count, so I need not add it up by hand."
        verification: "A shell test multiplies 2 by 3."
        validation: "- **Two by three:** Given the library, When I multiply 2 by 3, Then I get 6."
        scope: "A mul function in lib.sh."
        sources: [lib.sh]
      - title: "Divide two numbers"
        need: "When I share a bill, I want to divide a total by a count, so each pays their part."
        verification: "A shell test divides 6 by 3."
        validation: "- **Six by three:** Given the library, When I divide 6 by 3, Then I get 2."
        scope: "A div function in lib.sh."
        sources: [lib.sh]
TEXT
if [ "${3:-}" = blocked ]; then
  echo "- blocked: {issue: $1, why: \"split into its parts: a person accepts it from what they deliver\", person: true}"
fi
