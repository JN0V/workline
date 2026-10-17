package backlog

import "testing"

// Three of workline's own issues as a night of the role left them on
// 2026-10-11 (their edit history), and the two examples kept in the
// worked examples of ADR-0039: each draft that said again what is above
// it is past 40, each example kept is under it.
func TestRepeatedOnRealIssues(t *testing.T) {
	const (
		top103  = "On Linux without brew or cargo, `workline setup` cannot install lychee, the link checker. It only names the lychee page. Wanted: fetch lychee's release binary, pinned."
		need103 = "A Linux user without brew or cargo needs `workline setup` to install lychee for them, so link checks work without manual steps."
		ex103   = "On a Linux machine with neither brew nor cargo, `workline setup` prints the lychee page and stops. The user installs it by hand."
		top104  = "On macOS, `config.yaml` and the global hooks are read from Go's config folder, `~/Library/Application Support/workline`. Your own role facets are looked for in `~/.config/workline/roles/`. A user must know two folders. Wanted: one folder for all, or both named in the docs.\n\n<details><summary>For the builder</summary>\n\nSee `docs/usage.md` and `.workline/config.yaml`.\n\n</details>"
		need104 = "A macOS user needs one config folder for settings, hooks and facets, so they do not look in two places."
		ex104   = "A macOS user puts a facet in `~/Library/Application Support/workline` next to `config.yaml`. It is not found, because facets are read from `~/.config/workline/roles/`."
		top111  = "The removal rule reads English and French glue and fact words, but its count and limit words are still English only. A line like \"1 200 lignes\" is not read."
		need111 = "A French-language project needs the count and limit words read in French too, so its docs are checked like English ones."
		ex111   = "A French doc says \"1 200 lignes\". The `count-off` check does not read it, while \"1,200 lines\" is read."
		// The worked examples' (roles/product-owner/docs/proposed-headings.md).
		problemFeature = "Accountants closing a month cannot export it as one file: the export\ntakes one day at a time."
		exFeature      = "Closing September 2026: `shopctl export --day 2026-09-01`, then 29 more\ndays, 30 CSV files pasted into one sheet, two hours."
		problemBug     = "Users who exclude a folder with a trailing slash still upload all of it."
		stepsBug       = "1. Run `snapsync push --exclude node_modules/ ~/projects/site`.\n2. Read the last line of output.\n\n- **Expected behavior:** about 40 files; `node_modules` skipped.\n- **Actual behavior:** `uploaded 18412 files (1.2 GB) in 9m02s`.\n- **Version:** 2.3.1, Linux, every time."
	)
	for _, c := range []struct {
		name         string
		draft, above string
		want         int
	}{
		{"#103's Need, in its text on top", need103, top103, 60},
		{"#104's Need, in its text on top", need104, top104, 64},
		{"#111's Need, in its text on top", need111, top111, 50},
		{"#103's Example, in all above", ex103, top103 + "\n\n" + need103, 53},
		{"#104's Example, in all above", ex104, top104 + "\n\n" + need104, 69},
		{"#111's Example, in all above", ex111, top111 + "\n\n" + need111, 55},
		{"a feature's Example kept", exFeature, problemFeature, 17},
		{"a bug's steps kept", stepsBug, problemBug, 3},
	} {
		if got := Repeated(c.draft, c.above); got != c.want {
			t.Errorf("%s: %d%%, want %d%%", c.name, got, c.want)
		}
	}
}

func TestRepeatedReadsWords(t *testing.T) {
	for _, c := range []struct {
		name         string
		draft, above string
		want         int
	}{
		{"no word", "<!-- workline:draft -->", "Anything.", 0},
		{"common words only", "It is the one.", "Nothing here.", 0},
		{"case and marks aside", "<!-- workline:draft -->\n\nExports LOSE rows.", "exports lose rows", 100},
		{"code read as words", "`WriteRows` stops early.", "WriteRows stops", 67},
		{"each word once", "rows rows rows lost", "rows", 50},
		{"a fold's tags are no words", "<details><summary>Cause</summary>loop</details>", "cause loop", 100},
	} {
		if got := Repeated(c.draft, c.above); got != c.want {
			t.Errorf("%s: %d%%, want %d%%", c.name, got, c.want)
		}
	}
}

// The setting reads 1 to 100, 40 when not set; out of range, an error,
// never a default (principle 12).
func TestRepeatedPercentSetting(t *testing.T) {
	if c, err := ReadConfig(map[string]any{}); err != nil || c.RepeatedPercent != RepeatedPercentMax {
		t.Fatalf("not set: %d, %v", c.RepeatedPercent, err)
	}
	if c, err := ReadConfig(map[string]any{"repeated-percent-max": 100}); err != nil || c.RepeatedPercent != 100 {
		t.Fatalf("100: %d, %v", c.RepeatedPercent, err)
	}
	for _, v := range []any{0, 101, "half"} {
		if _, err := ReadConfig(map[string]any{"repeated-percent-max": v}); err == nil {
			t.Errorf("%v: read", v)
		}
	}
}
