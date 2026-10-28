#!/bin/sh
# A cmd: tester answering an objection's third exchange: kept, with the
# five whys when the prompt gives the shared skill after the task; without
# it, kept alone (which the run refuses when the whys are due).
prompt=$(cat)
echo '- note: "The scenario says Then I get 6: the test checks that value, nothing more."'
case "$prompt" in
*"# Skill: five-whys"*)
	cat <<'YAML'
- note: |
    Five whys
    Problem: the builder still finds the test too strict after two answers.
    1. Why too strict? The builder reads "Then I get 6" as any output holding 6 ("still too strict").
    2. Why can it be read so? The scenario names the value, not how it is printed.
    Root cause: the scenario leaves the output's form open.
    Action (doc): the product owner adds the printed form to the scenario's Then.
YAML
	;;
esac
