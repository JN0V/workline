package scripts

import (
	"path/filepath"
	"strings"
	"testing"
)

const (
	dispatchCall = "workflow run workline-merge.yml -R JN0V/TraceGlass-trial"

	readyGreen    = `{"__typename":"CheckRun","name":"test","conclusion":"SUCCESS"}`
	readyAccepted = `{"__typename":"StatusContext","context":"workline/accepted","state":"SUCCESS"}`
)

// keepingGh is the forge's gh that also keeps the tags created on the copy
// from one run to the next, as GitHub does: creating a tag that exists fails
// with "Reference already exists", reading one answers its ref or a 404, and
// deleting one removes it. A marker kept as a tag on the copy is thus seen by
// a rerun.
func keepingGh() string {
	funcs := `tagfile() { mkdir -p "$dir/tags"; echo "$dir/tags/${1//\//_}"; }
mark_write() {
  local a name=
  case ${args[2]} in
    POST)
      for a in "${args[@]}"; do case $a in ref=refs/tags/*) name=${a#ref=refs/tags/} ;; esac; done
      [ -n "$name" ] || return 0
      if [ -e "$(tagfile "$name")" ]; then echo "gh: Reference already exists (HTTP 422)" >&2; exit 1; fi
      : > "$(tagfile "$name")" ;;
    DELETE) name=${args[3]##*/refs/tags/}; rm -f "$(tagfile "$name")" ;;
  esac
}
mark_read() {
  local p=${args[1]} name
  case $p in */tags/*) ;; *) echo "unexpected gh $*" >&2; exit 1 ;; esac
  name=${p##*/tags/}
  if [ -e "$(tagfile "$name")" ]; then
    case $p in
      */matching-refs/*) jq -r "$jqx" <<<"[{\"ref\":\"refs/tags/$name\"}]" ;;
      *) jq -r "$jqx" <<<"{\"ref\":\"refs/tags/$name\"}" ;;
    esac
  else
    case $p in
      */matching-refs/*) jq -r "$jqx" <<<'[]' ;;
      *) echo "gh: Not Found (HTTP 404)" >&2; exit 1 ;;
    esac
  fi
}
case "$1 $2" in
`
	script := strings.Replace(forgeGh, "case \"$1 $2\" in\n", funcs, 1)
	script = strings.Replace(script, "  \"api -X\") ;;\n",
		"  \"api -X\") mark_write ;;\n  \"api repos/JN0V/TraceGlass-trial/\"*) mark_read ;;\n", 1)
	return script
}

// readyFix sets a copy whose canned issue is open, its fix pull request open
// and green, with the given labels and the policy's acceptance.
func readyFix(t *testing.T, labels string) forge {
	t.Helper()
	f := newForge(t)
	f.set("pr-merged.json", `[{"number":5,"url":"https://github.com/JN0V/TraceGlass-trial/pull/5"}]`)
	f.set("issues.json", `[{"number":8,"title":"An outline of 65536 by 65536 pixels with no pixels crashes instead of being refused","state":"OPEN","stateReason":null,"body":"<!-- workline-try: v9.9.9 -->"}]`)
	f.set("issue-8.json", `{"state":"OPEN","closedByPullRequestsReferences":[{"number":10}]}`)
	f.set("pr-10.json", `{"state":"OPEN","isDraft":false,"labels":[`+labels+`],"statusCheckRollup":[`+readyGreen+`,`+readyAccepted+`]}`)
	return f
}

// Whole yes: Given a fix pull request that carries the label
// workline:accepted-whole, When the try waits for the merge, Then the script
// dispatches nothing.
func TestTryReleaseDispatchesNothingForAFixAcceptedWhole(t *testing.T) {
	f := readyFix(t, `{"name":"workline:accepted-whole"}`)
	out, code := f.runEnv([]string{"TRY_DISPATCH_MINUTES=0"}, "v9.9.9")
	if code != 1 || !strings.Contains(out, "no pull request merged for the canned issue") {
		t.Fatalf("exit %d, want 1 when the time runs out:\n%s", code, out)
	}
	if c := f.calls(); strings.Contains(c, "workflow run") {
		t.Errorf("the merge workflow was dispatched for a fix a person accepted whole:\n%s", c)
	}
	if strings.Contains(out, "dispatched the copy's workline merge workflow") {
		t.Errorf("the output says a dispatch happened:\n%s", out)
	}
}

// Rerun: Given a try that dispatched the merge workflow once and was stopped,
// When the same try is run again, Then it does not dispatch a second time.
func TestTryReleaseRerunDoesNotDispatchASecondTime(t *testing.T) {
	f := readyFix(t, `{"name":"bug"}`)
	write(t, filepath.Join(f.gh, "gh"), keepingGh())
	env := []string{"TRY_DISPATCH_MINUTES=0"}
	out, code := f.runEnv(env, "v9.9.9")
	if code != 1 || !strings.Contains(out, "no pull request merged for the canned issue") {
		t.Fatalf("first run: exit %d, want 1 when the time runs out:\n%s", code, out)
	}
	if got := strings.Count(f.calls(), dispatchCall); got != 1 {
		t.Fatalf("the first run dispatched %d times, want 1:\n%s", got, f.calls())
	}
	out, code = f.runEnv(env, "v9.9.9")
	if code != 1 || !strings.Contains(out, "no pull request merged for the canned issue") {
		t.Fatalf("rerun: exit %d, want 1 when the time runs out:\n%s", code, out)
	}
	if got := strings.Count(f.calls(), dispatchCall); got != 1 {
		t.Errorf("the two runs dispatched %d times in all, want 1:\n%s", got, f.calls())
	}
	if strings.Contains(out, "dispatched the copy's workline merge workflow") {
		t.Errorf("the rerun says it dispatched:\n%s", out)
	}
}
