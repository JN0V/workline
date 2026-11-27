#!/bin/sh
# A cmd: agent for the `documented` fixture, its doc given a sentence its
# sources, cut to a part, do not show ("Sessions last a week."). It takes the
# sentence out of docs/tech/auth.md, citing nothing, as an agent does that
# takes what a part does not show for false.
cat > /dev/null
printf '%s\n' \
	'- patch: |' \
	'    --- a/docs/tech/auth.md' \
	'    +++ b/docs/tech/auth.md' \
	'    @@ -8,5 +8,3 @@' \
	'     ## Token refresh' \
	'     ' \
	'     Access tokens last one hour.' \
	'    -' \
	'    -Sessions last a week.'
