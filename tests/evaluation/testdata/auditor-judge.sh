#!/bin/sh
# An auditor answering blind (ADR-0046): `true` or `false` to every act,
# quoting nothing. It can let an act stand, never prove one false.
cat > /dev/null
echo "- note: \"$1: answered without reading\""
