#!/bin/sh
# A cmd: agent playing the product owner on a bug in refine it predicts
# past its normal, its tests not yet written (#1212): it readies issue $1
# `too-big: true`, so the tester writes them first and the line asks its
# split once they are.
cat >/dev/null
cat <<TEXT
- ready: {issue: $1, too-big: true, why: "multiplying and dividing each pass their own test, past its normal: its reproduction tests first"}
TEXT
