#!/bin/sh
# A cmd: inspector, on the `linted` fixture: each finding its task asks to
# triage (`## t1 — …, clean.go:10`) gets a triage. One in cleanup.go — a
# temporary file removed on the way out — is called a likely false
# positive, put last; the others are defects, in the task's order. Each
# finding triaged before (`- r1 — …`) gets its place only, after the new
# ones. TRIAGE_BROKEN set: an answer that never reads.
task=$(cat)
if [ -n "${TRIAGE_BROKEN:-}" ]; then
	echo 'triage: [this does not read'
	exit 0
fi
printf '%s\n' "$task" | awk '
	/^## t[0-9]+ — / {
		id = $2
		if (index($0, "cleanup.go")) { fp[++nf] = id } else { real[++nr] = id }
	}
	/^- r[0-9]+ — / { before[++nb] = $2 }
	END {
		n = 0
		for (i = 1; i <= nr; i++) {
			printf "- triage:\n    finding: %s\n    matters: an unchecked os.Remove leaves the temporary file behind when it fails, and the next run reads it as its own\n    fix: return the error, or log it\n    false-positive: false\n    order: %d\n", real[i], ++n
		}
		for (i = 1; i <= nb; i++) {
			printf "- triage: {finding: %s, order: %d}\n", before[i], ++n
		}
		for (i = 1; i <= nf; i++) {
			printf "- triage:\n    finding: %s\n    matters: the deferred os.Remove cleans a temporary file on the way out\n    false-positive: true\n    why: the file is created in the same function and its removal failing changes nothing the caller sees\n    order: %d\n", fp[i], ++n
		}
		if (n == 0) print "[]"
	}'
