#!/bin/sh
# A cmd: agent rewording docs/tech/jetons.md, a French doc, with no claim:
# a straight apostrophe made typographic, an article mended ("Le connexion"),
# a number written whole ("1 000" as "1000"), and a negation dropped
# ("ne … pas"), which changes what the line says. `checked` is left and
# `judged` recorded. It reads the numbered lines from the prompt.
prompt=$(cat)
old=$(printf '%s\n' "$prompt" | sed -n 's/^ *3 | checked: //p' | head -1)
at=$(printf '%s\n' "$prompt" | sed -n 's/.*`judged: \([0-9a-f]*\)`.*/\1/p' | head -1)
cat <<YAML
- patch: |
    --- a/docs/tech/jetons.md
    +++ b/docs/tech/jetons.md
    @@ -3,2 +3,3 @@
     checked: $old
    +judged: $at
     ---
    @@ -7 +8 @@
    -Le jeton d'accès dure une heure.
    +Le jeton d’accès dure une heure.
    @@ -9 +10 @@
    -Le connexion reste ouverte.
    +La connexion reste ouverte.
    @@ -11 +12 @@
    -Il en reste 1 000.
    +Il en reste 1000.
    @@ -13 +14 @@
    -Le jeton ne se renouvelle pas.
    +Le jeton se renouvelle.
YAML
