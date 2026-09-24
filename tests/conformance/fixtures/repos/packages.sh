#!/bin/sh
# A library released as one, made of two components with their own versions,
# repeated in their sources — the shape of DomoticsCore. Released as v2.6.1.
set -eu
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"
lib() { printf '{\n  "name": "%s",\n  "version": "%s",\n  "dependencies": [{"name": "Other", "version": ">=1.0.0"}]\n}\n' "$1" "$2"; }
mkdir -p Lib-Core/src Lib-Core/tests Lib-OTA/src Lib-OTA/tests
lib Lib 2.6.1 > library.json
lib Lib-Core 1.10.0 > Lib-Core/library.json
lib Lib-OTA 1.9.1 > Lib-OTA/library.json
printf 'metadata.version = "1.10.0";\n' > Lib-Core/src/core.h
printf 'metadata.version = "1.9.1";\n' > Lib-OTA/src/ota.h
printf '# Changelog\n\nNotes kept above every entry.\n\n## [2.6.1] - 2026-01-01\n\n- Earlier.\n' > CHANGELOG.md
git add . && git commit -q -m "feat: first release"
git tag -a v2.6.1 -m v2.6.1
printf '// new event\n' >> Lib-Core/src/core.h
git commit -qam "feat(core): publish an event when the clock is set"
printf '// more cases\n' > Lib-OTA/tests/t.cpp && git add . && git commit -q -m "test(ota): cover a refused upload"
