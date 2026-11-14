// Package wake reads, after a merge to the default branch, where the line
// may be drifting, with no AI (ADR-0068 §8): facts from git and the
// forge, printed; it never blocks a merge and never calls an agent. Each
// trigger is a rule read from files and the forge, never from a title or
// a commit's type, which the judged choose.
package wake

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/forge"
	"go.yaml.in/yaml/v3"
)

// The sections a trigger is listed under, in this order (ADR-0068 §8's
// table).
const (
	Contract = "contract" // growth of the contract
	Prompt   = "prompt"   // prompts steer the line
	Symptom  = "symptom"  // symptoms without cause
	Need     = "need"     // need laundering
	Bypass   = "bypass"   // nobody bypasses
	Cord     = "cord"     // the person's unease, or a role's
)

// Label is the cord: anyone may set it on an issue or a merge request.
const Label = "step-back"

// Window is how far back the merges are read for what repeats: the same
// area fixed again, an issue fixed again.
const Window = 14 * 24 * time.Hour

// AreaFixes is how many merges fixing a bug, in the window, make a file
// an area fixed again and again.
const AreaFixes = 3

// areasShown caps the areas listed, the most fixed first; the rest are
// counted.
const areasShown = 10

// emptyTree is git's tree of nothing: what a range from the first commit
// compares with.
const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// Options is what one read takes.
type Options struct {
	Repo   string
	Before string      // the default branch's tip before the merge; "" for its first parent
	After  string      // its tip after
	Forge  forge.Forge // nil: none, git's facts alone
}

// Entry is one trigger met.
type Entry struct {
	Kind string `json:"kind"`
	Key  string `json:"key"` // what it is about, the same from one run to the next
	Text string `json:"text"`
	At   string `json:"at"`  // the commit of the default branch it was read at
	Day  string `json:"day"` // that commit's day, YYYY-MM-DD
}

// Result is what one read found.
type Result struct {
	Before  string   `json:"before"`
	Head    string   `json:"head"`
	Day     string   `json:"day"`
	Entries []Entry  `json:"entries"`
	Facts   string   `json:"facts,omitempty"`
	Unread  []string `json:"unread,omitempty"` // what could not be read, and why
}

// Read reads the triggers the range before..after meets, and those the
// forge says now.
func Read(o Options) (*Result, error) {
	g := gitIn(o.Repo)
	after, err := g.commit(o.After)
	if err != nil {
		return nil, fmt.Errorf("%s is no commit here", o.After)
	}
	before := ""
	if o.Before != "" && strings.Trim(o.Before, "0") != "" {
		before, _ = g.commit(o.Before)
	}
	if before == "" {
		before, _ = g.commit(after + "^1") // the first commit: none
	}
	at, err := g.date(after)
	if err != nil {
		return nil, err
	}
	r := &Result{Before: before, Head: after, Day: at.UTC().Format(time.DateOnly), Entries: []Entry{}}
	add := func(kind, key, text string) {
		r.Entries = append(r.Entries, Entry{Kind: kind, Key: kind + ":" + key, Text: text, At: after, Day: r.Day})
	}
	if err := readTree(g, before, after, add); err != nil {
		return nil, err
	}
	if o.Forge == nil {
		r.Unread = append(r.Unread, "the forge, with none named (`--forge github`, `gitlab`): which merges fix a bug or close an issue, who opened them, what bypassed, and the cord")
		r.Facts = facts(g, after, at, nil, nil)
		return r, nil
	}
	readForge(g, o, r, before, after, at, add)
	return r, nil
}

// readTree reads what the range changed of the contract and the prompts,
// from the trees at before and after.
func readTree(g git, before, after string, add func(kind, key, text string)) error {
	from := before
	if from == "" {
		from = emptyTree
	}
	changed, err := g.changed(from, after)
	if err != nil {
		return err
	}
	goChanged, cmdChanged := false, false
	roles := map[string]bool{}
	for _, c := range changed {
		switch {
		case isADR(c.path):
			add(Contract, "adr:"+c.path, fmt.Sprintf("ADR %s: `%s`", c.verb(), c.path))
		case strings.HasSuffix(c.path, engineSource) && !strings.HasSuffix(c.path, engineTests):
			goChanged = true
			cmdChanged = cmdChanged || strings.HasPrefix(c.path, "cmd/")
		}
		if name, rest, ok := roleFile(c.path); ok {
			switch {
			case rest == "role.yaml":
				roles[name] = true
			case isPrompt(rest):
				was, now := g.words(from, c.path), g.words(after, c.path)
				add(Prompt, "prompt:"+c.path, fmt.Sprintf("prompt %s: `%s`, %d → %d words", c.verb(), c.path, was, now))
			}
		}
	}
	for _, name := range sortedKeys(roles) {
		readRole(g, from, after, name, add)
	}
	if goChanged {
		for _, code := range added(g.literals(from, ruleCode, engineGlob), g.literals(after, ruleCode, engineGlob)) {
			add(Contract, "rule:"+code, fmt.Sprintf("rule code added: `%s`", code))
		}
	}
	if cmdChanged {
		for _, name := range added(g.literals(from, command, commandGlob), g.literals(after, command, commandGlob)) {
			add(Contract, "command:"+name, fmt.Sprintf("command added: `%s`", name))
		}
	}
	return nil
}

// readRole reads a role's role.yaml before and after: the role added, its
// settings added, its levels table and its default level changed.
func readRole(g git, from, after, name string, add func(kind, key, text string)) {
	file := "roles/" + name + "/role.yaml"
	was, had := g.roleFile(from, file)
	now, has := g.roleFile(after, file)
	if !has {
		return
	}
	if !had {
		add(Contract, "role:"+name, fmt.Sprintf("role added: `%s`", name))
	}
	for _, k := range added(sortedKeys(was.Settings), sortedKeys(now.Settings)) {
		add(Contract, "setting:"+name+"."+k, fmt.Sprintf("setting added: `%s.%s`", name, k))
	}
	if !had {
		return
	}
	if a, b := marshal(was.Levels), marshal(now.Levels); a != b {
		add(Prompt, "levels:"+name, fmt.Sprintf("levels table changed: `%s`", name))
	}
	if a, b := level(was), level(now); a != b {
		add(Prompt, "level:"+name, fmt.Sprintf("default level changed: `%s`, `%s` → `%s`", name, orNone(a), orNone(b)))
	}
}

// readForge reads the merges of the window, the issues they close, and
// the forge's state now: each read that fails is said, never fatal.
func readForge(g git, o Options, r *Result, before, after string, at time.Time, add func(kind, key, text string)) {
	f := o.Forge
	unread := func(what string, err error) {
		r.Unread = append(r.Unread, what+": "+forge.Said(err))
	}
	window := g.firstParents(after, at.Add(-Window))
	inRange := map[string]bool{}
	for _, c := range g.rangeCommits(before, after) {
		inRange[c] = true
	}
	var merged []forge.MergeRequest
	listed := false
	if m, ok := f.(forge.Merged); ok {
		all, err := m.MergedSince(at.Add(-Window - 24*time.Hour))
		if err != nil {
			unread("the merge requests merged", err)
		}
		listed = err == nil
		for _, mr := range all {
			if _, ok := window[mr.MergedAs]; ok && mr.MergedAs != "" {
				merged = append(merged, mr)
			}
		}
	} else {
		r.Unread = append(r.Unread, "the merge requests merged: this forge does not list them")
	}
	closes := map[int][]forge.Issue{}
	w, woken := f.(forge.Woken)
	if woken && len(merged) > 0 {
		ids := make([]int, 0, len(merged))
		for _, mr := range merged {
			ids = append(ids, mr.ID)
		}
		var err error
		if closes, err = w.Closes(ids); err != nil {
			unread("the issues each merge closes", err)
			closes = map[int][]forge.Issue{}
		}
	}
	ref := func(id int) string { return forge.MergeRef(f, id) }

	// Symptoms without cause: an area fixed again and again; an issue
	// fixed again.
	fixes := map[string][]int{} // file -> the merges fixing a bug that touch it
	byIssue := map[int][]int{}
	open := map[int]bool{}
	for _, mr := range merged {
		bug := false
		for _, is := range closes[mr.ID] {
			byIssue[is.ID] = append(byIssue[is.ID], mr.ID)
			open[is.ID] = open[is.ID] || !is.Closed
			bug = bug || isBug(is.Labels)
		}
		if bug {
			for _, file := range window[mr.MergedAs] {
				fixes[file] = append(fixes[file], mr.ID)
			}
		}
	}
	files := sortedKeys(fixes)
	sort.SliceStable(files, func(i, j int) bool { return len(fixes[files[i]]) > len(fixes[files[j]]) })
	hot := 0
	for _, file := range files {
		n := len(fixes[file])
		if n < AreaFixes {
			continue
		}
		if hot++; hot <= areasShown {
			add(Symptom, "area:"+file, fmt.Sprintf("the same area fixed again: `%s`, by %d merges in %d days (%s)", file, n, int(Window.Hours()/24), refs(fixes[file], ref)))
		}
	}
	if hot > areasShown {
		add(Symptom, "area:more", fmt.Sprintf("and %d more %s, each fixed by %d merges or more in %d days", hot-areasShown, plural(hot-areasShown, "file", "files"), AreaFixes, int(Window.Hours()/24)))
	}
	for _, id := range sortedKeys(byIssue) {
		by := byIssue[id]
		switch {
		case open[id]:
			add(Symptom, fmt.Sprintf("again:%d", id), fmt.Sprintf("fixed again: #%d, closed by %s, open again", id, refs(by, ref)))
		case len(by) > 1:
			add(Symptom, fmt.Sprintf("again:%d", id), fmt.Sprintf("fixed again: #%d, closed by %s", id, refs(by, ref)))
		}
	}

	// The range's own merges: a need stated, the App's yes, a merge
	// request at all. The App's yes is read where the project's merge
	// policy, as the merged tree holds it, asks for it (`merge:`).
	accepted := g.mergePolicy(after)
	seen := map[string]bool{}
	for _, mr := range merged {
		if !inRange[mr.MergedAs] {
			continue
		}
		seen[mr.MergedAs] = true
		if woken && len(closes[mr.ID]) == 0 {
			add(Need, fmt.Sprintf("none:%d", mr.ID), fmt.Sprintf("closes no issue: %s, by `%s`", ref(mr.ID), orNone(mr.Author)))
		}
		if !accepted {
			continue
		}
		s, ok := f.(forge.AppStatuses)
		switch {
		case !ok:
		case mr.HeadSHA == "":
			r.Unread = append(r.Unread, fmt.Sprintf("the head %s merged at: the forge does not say", ref(mr.ID)))
		default:
			yes, err := s.AppStatus(mr.HeadSHA, forge.Accepted)
			switch {
			case err != nil:
				unread(fmt.Sprintf("%s's status on %s", forge.Accepted, ref(mr.ID)), err)
			case !yes:
				add(Bypass, fmt.Sprintf("unaccepted:%d", mr.ID), fmt.Sprintf("merged without the App's `%s`: %s, head `%s`", forge.Accepted, ref(mr.ID), short(mr.HeadSHA)))
			}
		}
	}
	if listed {
		for _, c := range g.rangeCommits(before, after) {
			if !seen[c] {
				add(Bypass, "pushed:"+c, fmt.Sprintf("on the default branch with no merge request: `%s`", short(c)))
			}
		}
	}

	// The ruleset changed in the range's time: its public updated_at.
	if b, ok := f.(forge.Bypasser); ok {
		since := at.Add(-Window)
		if before != "" {
			if t, err := g.date(before); err == nil {
				since = t
			}
		}
		branch, sets, err := b.Bypass("")
		if err != nil {
			unread("the default branch's rulesets", err)
		}
		for _, s := range sets {
			t, err := time.Parse(time.RFC3339, s.Updated)
			if err == nil && t.After(since) && !t.After(at) {
				add(Bypass, fmt.Sprintf("ruleset:%d:%s", s.ID, s.Updated), fmt.Sprintf("ruleset changed: `%s` on %s, at %s", s.Name, orNone(branch), s.Updated))
			}
		}
	}

	// The cord.
	if woken {
		items, err := w.Labelled(Label)
		if err != nil {
			unread("the items bearing `"+Label+"`", err)
		}
		for _, t := range items {
			name := fmt.Sprintf("issue #%d", t.ID)
			if t.Kind == "merge-request" {
				name = "merge request " + ref(t.ID)
			}
			add(Cord, fmt.Sprintf("%s:%d", t.Kind, t.ID), fmt.Sprintf("`%s` on %s", Label, name))
		}
	} else {
		r.Unread = append(r.Unread, "what each merge closes, and the cord: this forge does not say")
	}
	r.Facts = facts(g, after, at, merged, closes)
}

// facts sums up the window: the merges, by people's accounts and apps,
// and those fixing a bug.
func facts(g git, after string, at time.Time, merged []forge.MergeRequest, closes map[int][]forge.Issue) string {
	n := len(g.firstParents(after, at.Add(-Window)))
	days := int(Window.Hours() / 24)
	s := fmt.Sprintf("**The %d days to %s**: %d %s on the default branch", days, at.UTC().Format(time.DateOnly), n, plural(n, "commit", "commits"))
	if merged == nil {
		return s + "."
	}
	people, apps, fixes := 0, 0, 0
	for _, mr := range merged {
		if mr.ByBot() {
			apps++
		} else {
			people++
		}
		if slices.ContainsFunc(closes[mr.ID], func(is forge.Issue) bool { return isBug(is.Labels) }) {
			fixes++
		}
	}
	return s + fmt.Sprintf("; %d merge %s, %d by people's accounts, %d by apps; %d closing a bug.", len(merged), plural(len(merged), "request", "requests"), people, apps, fixes)
}

// isBug says whether labels say a bug: `bug`, or a scoped one ending so
// (`type::bug`, `kind/bug`), whatever its case.
func isBug(labels []string) bool {
	for _, l := range labels {
		l = strings.ToLower(strings.TrimSpace(l))
		if i := strings.LastIndexAny(l, ":/"); i >= 0 {
			l = strings.TrimSpace(l[i+1:])
		}
		if l == "bug" {
			return true
		}
	}
	return false
}

var adrPath = regexp.MustCompile(`(^|/)(adr|adrs|decisions)/[^/]+\.md$`)

// isADR: a decision record, a Markdown file in a folder named adr, adrs
// or decisions; its README, an index, is none.
func isADR(p string) bool {
	return adrPath.MatchString(p) && !strings.EqualFold(path.Base(p), "README.md")
}

// roleFile splits roles/<name>/<rest>.
func roleFile(p string) (name, rest string, ok bool) {
	after, found := strings.CutPrefix(p, "roles/")
	if !found {
		return "", "", false
	}
	name, rest, ok = strings.Cut(after, "/")
	return name, rest, ok && name != ""
}

// isPrompt: a Markdown file of a role's folder that its agent reads, its
// README and its docs/ left out: instruction.md, persona.md, policy.md,
// lenses, kinds, tasks.
func isPrompt(rest string) bool {
	return strings.HasSuffix(rest, ".md") && rest != "README.md" && !strings.HasPrefix(rest, "docs/")
}

// The contract is read as workline's engine lays it out, in Go: its rule
// codes and commands are its own source's literals, whatever the language
// of a project the line runs on.
const (
	engineSource = ".go"      // converges: workline's own engine is Go; a project's language is never read here
	engineTests  = "_test.go" // converges: the engine's own tests, left out of its contract
	engineGlob   = "*.go"     // converges: workline's own engine is Go
	commandGlob  = "cmd/*.go" // converges: workline's commands, in its own Go source
)

var (
	ruleCode = regexp.MustCompile(`Rule: +"([^"]+)"`)
	command  = regexp.MustCompile(`\{name: +"([a-z][a-z0-9-]*)"`)
)

// roleYAML is what the wake reads of a role.yaml.
type roleYAML struct {
	Settings map[string]any `yaml:"settings"`
	Levels   any            `yaml:"levels"`
}

func level(r roleYAML) string {
	if s, ok := r.Settings["autonomy"].(string); ok {
		return s
	}
	return ""
}

func marshal(v any) string {
	if v == nil {
		return ""
	}
	out, _ := yaml.Marshal(v)
	return string(out)
}

// added lists what now holds and was did not, sorted.
func added(was, now []string) []string {
	var out []string
	for _, s := range now {
		if !slices.Contains(was, s) {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return slices.Compact(out)
}

func refs(ids []int, ref func(int) string) string {
	ids = slices.Clone(ids)
	sort.Ints(ids)
	ids = slices.Compact(ids)
	var out []string
	for i, id := range ids {
		if i == 6 {
			out = append(out, fmt.Sprintf("%d more", len(ids)-6))
			break
		}
		out = append(out, ref(id))
	}
	if len(out) == 2 {
		return out[0] + " and " + out[1]
	}
	return strings.Join(out, ", ")
}

func sortedKeys[V any, K string | int](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

func short(c string) string { return c[:min(7, len(c))] }

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// git reads one repository, never its working copy.
type git struct{ repo string }

func gitIn(repo string) git { return git{repo: repo} }

func (g git) run(args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", g.repo}, args...)...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return "", errors.New(strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}

func (g git) commit(rev string) (string, error) {
	out, err := g.run("rev-parse", "--verify", "-q", rev+"^{commit}")
	return strings.TrimSpace(out), err
}

func (g git) date(c string) (time.Time, error) {
	out, err := g.run("show", "-s", "--format=%cI", c)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, strings.TrimSpace(out))
}

// change is a path the range changed: A, M or D.
type change struct{ status, path string }

func (c change) verb() string {
	switch c.status {
	case "A":
		return "added"
	case "D":
		return "removed"
	}
	return "changed"
}

func (g git) changed(from, to string) ([]change, error) {
	out, err := g.run("diff", "--no-renames", "--name-status", from, to)
	if err != nil {
		return nil, err
	}
	var all []change
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		status, p, ok := strings.Cut(l, "\t")
		if ok {
			all = append(all, change{status: status[:1], path: p})
		}
	}
	return all, nil
}

func (g git) show(rev, p string) (string, bool) {
	if rev == emptyTree {
		return "", false
	}
	out, err := g.run("show", rev+":"+p)
	return out, err == nil
}

func (g git) words(rev, p string) int {
	s, _ := g.show(rev, p)
	return len(strings.Fields(s))
}

func (g git) roleFile(rev, p string) (roleYAML, bool) {
	var r roleYAML
	s, ok := g.show(rev, p)
	if !ok {
		return r, false
	}
	_ = yaml.Unmarshal([]byte(s), &r)
	return r, true
}

// mergePolicy says whether the project's config at rev sets the merge
// policy, `merge:`, which holds a merge to workline/accepted (ADR-0042).
func (g git) mergePolicy(rev string) bool {
	s, ok := g.show(rev, ".workline/config.yaml")
	if !ok {
		return false
	}
	var cfg map[string]any
	_ = yaml.Unmarshal([]byte(s), &cfg)
	return cfg["merge"] != nil
}

// literals lists the first group of re in the tree's files matching
// glob, tests left out.
func (g git) literals(rev string, re *regexp.Regexp, glob string) []string {
	if rev == emptyTree {
		return nil
	}
	out, _ := g.run("grep", "-h", "-o", "-E", re.String(), rev, "--", glob, ":!*"+engineTests)
	var all []string
	for _, m := range re.FindAllStringSubmatch(out, -1) {
		all = append(all, m[1])
	}
	sort.Strings(all)
	return slices.Compact(all)
}

// firstParents maps each commit of the default branch's first parents
// from since to after to the files it changed against its first parent.
func (g git) firstParents(after string, since time.Time) map[string][]string {
	out, _ := g.run("log", "--first-parent", "--diff-merges=first-parent", "--name-only", "--format=@@%H",
		"--since="+since.UTC().Format(time.RFC3339), after)
	all := map[string][]string{}
	cur := ""
	for _, l := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(l, "@@"):
			cur = l[2:]
			all[cur] = nil
		case strings.TrimSpace(l) != "" && cur != "":
			all[cur] = append(all[cur], l)
		}
	}
	return all
}

// rangeCommits lists the first parents from before, left out, to after.
func (g git) rangeCommits(before, after string) []string {
	spec := after
	if before != "" {
		spec = before + ".." + after
	}
	out, _ := g.run("rev-list", "--first-parent", spec)
	return strings.Fields(out)
}
