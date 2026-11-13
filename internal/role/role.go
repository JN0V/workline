// Package role loads a role's contract (role.yaml) and the project's settings
// for it, as described in docs/spec/role-contract.md.
package role

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/packs"
	"github.com/JN0V/workline/internal/testdef"
)

// Role is a parsed role.yaml, plus the folder it was loaded from.
type Role struct {
	Contract   int      `yaml:"contract"`
	Name       string   `yaml:"name"`
	Mission    string   `yaml:"mission"`
	Events     []string `yaml:"events"`
	Requires   []string `yaml:"requires"`
	Uses       []string `yaml:"uses"`
	Intentions []string `yaml:"intentions"`
	// PartIntentions are what a part of a question may answer with
	// (in/parts): claims when unset; a reviewer's lens answers findings.
	PartIntentions []string `yaml:"part-intentions"`
	// OnBlock are the intentions a run still applies when post blocks:
	// those that only tell why, never change what was judged (the
	// reviewer's summary comment, #226). Empty: nothing applied on a block.
	OnBlock []string `yaml:"on-block"`
	// OnMovedOn are the intentions a merge request's run still applies
	// once its branch moved on from the head the run judged (#375): those
	// that only say what that head held, never a patch nor a label (the
	// reviewer's review). Empty: nothing applied once the branch moved on.
	OnMovedOn []string `yaml:"on-moved-on"`
	// Apart are the events where what the role finds that only warns can
	// run apart from the verdict (ADR-0050): the verdict's line leaves it
	// (`route --warn-apart`, input `warn=apart`), and a line of its own
	// runs it alone (`route --warn-only`, input `warn=only`), its verdict
	// never the head's. Empty: the role runs whole on the verdict's line.
	Apart []string `yaml:"apart"`
	// Personal are the settings a person may set for themselves, in their
	// own config (PersonalSettings): what only changes their own wait or
	// their own words, never what the project's line holds to.
	Personal []string `yaml:"personal"`
	Model    Model    `yaml:"model"`
	Duties   struct {
		Reads  []string `yaml:"reads"`
		Writes []string `yaml:"writes"`
	} `yaml:"duties"`
	Context struct {
		Knowledge []string `yaml:"knowledge"`
		// Skills are the shared skills a run of the role may load, when
		// its pre names them in in/skills (docs/spec/role-contract.md).
		Skills []string `yaml:"skills"`
		Budget int      `yaml:"budget"`
	} `yaml:"context"`
	WithoutAI string `yaml:"without-ai"`
	// MergeRequestAI `ready`: on a merge request, the agent is asked on the
	// run at ready alone, never on a push after it (ADR-0051).
	MergeRequestAI string         `yaml:"merge-request-ai"`
	Settings       map[string]any `yaml:"settings"`
	// Levels are named sets of settings a project picks with one setting
	// (docs/spec/role-adapting.md, "Levels"): setting -> value -> the
	// settings laid over the role's before the project's own.
	Levels map[string]map[string]map[string]any `yaml:"levels"`

	Dir string `yaml:"-"`
	// InstructionFacet is the facet read as the instruction: "" for
	// instruction.md; a kind of task's own (ForTask).
	InstructionFacet string `yaml:"-"`
	// TaskIntentions are the intentions the agent answers a kind of task
	// with, a few of the role's; none, the role's (ForTask, Asked).
	TaskIntentions []string `yaml:"-"`
}

// Model is what kind of thinking a role needs (docs/spec/model-grid.md).
type Model struct {
	Capability   string `yaml:"capability"`
	Tier         string `yaml:"tier"`
	Effort       string `yaml:"effort"`
	PromoteAfter int    `yaml:"promote-after"`
	Timeout      string `yaml:"timeout"` // how long one answer may take, e.g. "10m"; default 3m
	// Tasks gives a kind of task, named by pre in in/task-kind, other needs
	// than the role's: condensing a doc needs more than judging one.
	Tasks map[string]Model `yaml:"tasks"`
	// Instruction and Intentions, on a kind of task only: the facet read
	// in place of instruction.md, and the intentions it answers with, a
	// few of the role's — a light read pays for little prompt (the
	// product owner's kind, #878). Unset, the role's.
	Instruction string   `yaml:"instruction"`
	Intentions  []string `yaml:"intentions"`
}

// For returns the needs of a task of that kind: the role's, with what
// model.tasks sets for the kind in their place.
func (m Model) For(kind string) Model {
	t, ok := m.Tasks[kind]
	if !ok {
		return m
	}
	out := m
	out.Tasks = nil
	if t.Capability != "" {
		out.Capability = t.Capability
	}
	if t.Tier != "" {
		out.Tier = t.Tier
	}
	if t.Effort != "" {
		out.Effort = t.Effort
	}
	if t.PromoteAfter != 0 {
		out.PromoteAfter = t.PromoteAfter
	}
	if t.Timeout != "" {
		out.Timeout = t.Timeout
	}
	return out
}

// AnswerTimeout is how long the agent may take for one answer.
func (m Model) AnswerTimeout() (time.Duration, error) {
	if m.Timeout == "" {
		return 3 * time.Minute, nil
	}
	d, err := time.ParseDuration(m.Timeout)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("model.timeout %q is not a duration like 10m", m.Timeout)
	}
	return d, nil
}

// Load reads roles/<name>/role.yaml from rolesDir.
func Load(rolesDir, name string) (*Role, error) {
	dir := filepath.Join(rolesDir, name)
	data, err := os.ReadFile(filepath.Join(dir, "role.yaml"))
	if why, retired := Retired[name]; retired && os.IsNotExist(err) {
		return nil, fmt.Errorf("role %q: %s", name, why)
	}
	if err != nil {
		return nil, fmt.Errorf("role %q: %w", name, err)
	}
	var r Role
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("role %q: role.yaml: %w", name, err)
	}
	if r.Contract != 1 {
		return nil, fmt.Errorf("role %q: contract version %d is not supported", name, r.Contract)
	}
	if r.Name != name {
		return nil, fmt.Errorf("role %q: role.yaml names itself %q", name, r.Name)
	}
	for _, k := range r.OnBlock {
		if !slices.Contains(r.Intentions, k) {
			return nil, fmt.Errorf("role %q: on-block names %q, not one of its intentions", name, k)
		}
	}
	for _, k := range r.OnMovedOn {
		switch {
		case !slices.Contains(r.Intentions, k):
			return nil, fmt.Errorf("role %q: on-moved-on names %q, not one of its intentions", name, k)
		case k == "patch" || k == "label" || k == "commit-message":
			return nil, fmt.Errorf("role %q: on-moved-on names %q: what changes the branch or the merge request applies on the head judged only", name, k)
		}
	}
	for _, e := range r.Apart {
		if !slices.Contains(r.Events, e) {
			return nil, fmt.Errorf("role %q: apart names %q, not one of its events", name, e)
		}
	}
	for _, k := range r.Personal {
		if _, ok := r.Settings[k]; !ok {
			return nil, fmt.Errorf("role %q: personal names %q, not one of its settings", name, k)
		}
	}
	for kind, t := range r.Model.Tasks {
		if t.Instruction != "" && !taskFacet.MatchString(t.Instruction) {
			return nil, fmt.Errorf("role %q: model.tasks.%s.instruction %q is not a facet in tasks/, like tasks/%s.md", name, kind, t.Instruction, kind)
		}
		for _, k := range t.Intentions {
			if !slices.Contains(r.Intentions, k) {
				return nil, fmt.Errorf("role %q: model.tasks.%s.intentions names %q, not one of its intentions", name, kind, k)
			}
		}
	}
	r.Dir = dir
	return &r, nil
}

// taskFacet is where a kind of task's own instruction is: a facet in the
// role's tasks/ folder.
var taskFacet = regexp.MustCompile(`^tasks/[a-z0-9]+(-[a-z0-9]+)*\.md$`)

// ForTask is the role for a task of that kind: the needs, the instruction
// facet and the agent's intentions model.tasks gives it, in place of the
// role's; r itself when it names no such kind. The role's intentions stay
// whole: what pre falls back on is the role's (Asked narrows the agent's).
func (r *Role) ForTask(kind string) *Role {
	t, ok := r.Model.Tasks[kind]
	if !ok {
		return r
	}
	out := *r
	out.Model = r.Model.For(kind)
	out.InstructionFacet = t.Instruction
	out.TaskIntentions = slices.Clone(t.Intentions)
	return &out
}

// Asked is the role as the agent is asked: the intentions its task allows
// (TaskIntentions), else r itself.
func (r *Role) Asked() *Role {
	if len(r.TaskIntentions) == 0 {
		return r
	}
	out := *r
	out.Intentions = r.TaskIntentions
	return &out
}

// PartAnswers are the intentions a part of a question may answer with:
// `part-intentions`, else claims (docs/spec/role-contract.md, "In parts").
func (r *Role) PartAnswers() []string {
	if len(r.PartIntentions) > 0 {
		return r.PartIntentions
	}
	return []string{"claim"}
}

// ApartOn says whether what the role finds that only warns can run apart
// from the verdict on event (Apart).
func (r *Role) ApartOn(event string) bool { return slices.Contains(r.Apart, event) }

// Accepts reports whether the role can run on event.
func (r *Role) Accepts(event string) bool {
	for _, e := range r.Events {
		if e == event {
			return true
		}
	}
	return false
}

// Allows reports whether the role may emit an intention of this kind.
func (r *Role) Allows(kind string) bool {
	for _, k := range r.Intentions {
		if k == kind {
			return true
		}
	}
	return false
}

// ProjectConfig is the part of .workline/config.yaml the engine reads today.
type ProjectConfig struct {
	AI    string `yaml:"ai"`    // default agent for this project: none, claude...
	Forge string `yaml:"forge"` // github, gitlab, or none
	Roles map[string]struct {
		Settings map[string]any    `yaml:"settings"`
		Enforce  map[string]string `yaml:"enforce"`
	} `yaml:"roles"`
	// Autonomy is the project's default level (ADR-0060, §3): each role
	// with an `autonomy` level runs at it unless its own settings pick
	// one; "" none. The merge keeps its own, `merge: {autonomy: …}`.
	Autonomy string `yaml:"autonomy"`
	// Merge is the project's merge policy (ADR-0042): nil, none — no
	// pull request's auto-merge is touched.
	Merge *Merge `yaml:"merge"`
	// GitLab.Bots are the users GitLab does not name bots that are: a
	// service account, an account made for the line (ADR-0023, amended);
	// the gitlab forge reads them.
	GitLab struct {
		Bots []string `yaml:"bots"`
	} `yaml:"gitlab"`
	// Board overrides how the forge's board is matched (ADR-0057): a
	// column's name for a slot, a GitHub project's number among several.
	Board Board `yaml:"board"`
	// Packs are the language packs the project uses (ADR-0065), written
	// by `workline init` from their markers; none, the generic pack.
	Packs []string `yaml:"packs"`
	// PackDefaults are the settings the listed packs give each role, by
	// role name, read by LoadProjectConfig: under the project's own, over
	// the role's defaults. A pack never decides a verdict.
	PackDefaults map[string]map[string]any `yaml:"-"`
}

// Board is the `board:` setting: empty, the columns are matched by their
// names and the only board linked is taken.
type Board struct {
	Ready   string `yaml:"ready"`
	Doing   string `yaml:"doing"`
	Refine  string `yaml:"refine"`
	Tests   string `yaml:"tests"`
	Blocked string `yaml:"blocked"`
	Project int    `yaml:"project"`
}

// Merge is the project's merge policy: which pull requests `workline
// follow` turns the forge's auto-merge on for (ADR-0042).
type Merge struct {
	Autonomy string `yaml:"autonomy"`
	// Alone are the kinds of workline's own pull requests that wait for a
	// person's yes at `normal` the project lets merge with none (MergeAlone).
	Alone []string `yaml:"alone"`
}

// MergeLevels are the merge policy's levels, as the product owner's are
// named (ADR-0042): `cautious`, a person's yes on each pull request;
// `normal`, workline's own routine ones merged with none; `enterprising`,
// the developer's features too, the person's yes moved to the milestone's
// release candidate (ADR-0060).
var MergeLevels = []string{"cautious", "normal", "enterprising"}

// MergeAlone are the kinds of workline's own pull requests that wait for a
// person's yes at `normal` until the project lets them merge with none
// (`merge: {alone: [...]}`, ADR-0042 amended): `reader`, an agent's
// rewording of pages for their reader.
var MergeAlone = []string{"reader"}

// LoadProjectConfig reads <repo>/.workline/config.yaml. A missing file is an
// empty config; an unreadable or invalid one is an error.
func LoadProjectConfig(repo string) (*ProjectConfig, error) {
	var c ProjectConfig
	data, err := os.ReadFile(filepath.Join(repo, ".workline", "config.yaml"))
	if os.IsNotExist(err) {
		return &c, nil
	}
	if err != nil {
		return nil, err
	}
	if err := CheckConfig(data); err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, &ConfigError{err.Error()}
	}
	if len(c.Packs) > 0 {
		listed, err := packs.Listed(repo, c.Packs)
		if err != nil {
			return nil, &ConfigError{err.Error()}
		}
		c.PackDefaults = map[string]map[string]any{"tester": packs.TesterDefaults(listed)}
	}
	if c.Autonomy != "" && !slices.Contains(AutonomyLevels, c.Autonomy) {
		return nil, &ConfigError{fmt.Sprintf("autonomy is %q: the levels are %s", c.Autonomy, strings.Join(AutonomyLevels, ", "))}
	}
	if c.Merge != nil && !slices.Contains(MergeLevels, c.Merge.Autonomy) {
		return nil, &ConfigError{fmt.Sprintf("merge.autonomy is %q: the levels are %s", c.Merge.Autonomy, strings.Join(MergeLevels, ", "))}
	}
	if c.Merge != nil {
		for _, kind := range c.Merge.Alone {
			if !slices.Contains(MergeAlone, kind) {
				return nil, &ConfigError{fmt.Sprintf("merge.alone lists %q: the kinds it may let merge alone are %s", kind, strings.Join(MergeAlone, ", "))}
			}
		}
		if len(c.Merge.Alone) > 0 && c.Merge.Autonomy == "cautious" {
			return nil, &ConfigError{fmt.Sprintf("merge.alone lets kinds merge with no person's yes, which only autonomy normal or enterprising does: autonomy is %q", c.Merge.Autonomy)}
		}
	}
	return &c, nil
}

// AutonomyLevels are the levels the project's default may name
// (ADR-0060, §3); what each means is the role's, in its `levels`.
var AutonomyLevels = []string{"cautious", "normal", "enterprising"}

// ProjectDefault is the level setting the project's top-level `autonomy`
// sets (ADR-0060, §3), in each role that has it and does not set its own.
const ProjectDefault = "autonomy"

// LevelFrom is the setting a role with levels gets beside its own: where
// each level setting's value comes from, for its task to say.
const LevelFrom = "level-from"

// Where a role's level comes from, the first that holds (ADR-0060, §3):
// its own setting, the project's default, the role's own default.
const (
	FromRoleSetting    = "role setting"
	FromProjectDefault = "project default"
	FromRoleDefault    = "role default"
)

// ByLevel is the setting a role with levels gets beside its own: its
// settings as the level alone gives them, the project's left out — what a
// project set is what differs from it.
const ByLevel = "by-level"

// ProjectSet is the setting a role with levels gets beside its own: the
// project's settings for it as written, so a value the project set equal
// to the level's is still told the project's (ADR-0064: a `split` the
// project turned off stays off).
const ProjectSet = "project-set"

// MergedSettings returns the role's defaults overridden by the project's
// settings, as docs/spec/role-contract.md says: maps merged key by key, at
// every depth; a list or a scalar replaced whole; a null removes the key.
// A level is laid over the defaults first: the role's own setting picks
// it, else the project's default (a level the role has), else the role's
// default (ADR-0060, §3).
func (r *Role) MergedSettings(c *ProjectConfig) map[string]any {
	var project map[string]any
	if rc, ok := c.Roles[r.Name]; ok {
		project = rc.Settings
	}
	own := r.Settings
	if d := c.PackDefaults[r.Name]; len(d) > 0 {
		merged := MergeSettings(own, d) // the packs' over the role's defaults, the project's over both
		// A pack never narrows what the role's default knows: its
		// build-errors rows come first, the role's others follow.
		if rows, ok := mergeErrorRows(d["build-errors"], own["build-errors"]); ok {
			merged["build-errors"] = rows
		}
		own = merged
	}
	if len(r.Levels) == 0 {
		return MergeSettings(own, project)
	}
	base := own
	names := make([]string, 0, len(r.Levels))
	for name := range r.Levels {
		names = append(names, name)
	}
	slices.Sort(names)
	picked, from := map[string]any{}, map[string]any{}
	for _, name := range names {
		v, ok := project[name]
		from[name] = FromRoleSetting
		if !ok || v == nil {
			v, from[name] = own[name], FromRoleDefault
			if _, has := r.Levels[name][c.Autonomy]; name == ProjectDefault && c.Autonomy != "" && has {
				v, from[name] = c.Autonomy, FromProjectDefault
				picked[name] = v
			}
		}
		if over, ok := r.Levels[name][fmt.Sprint(v)]; ok {
			base = MergeSettings(base, over)
		}
	}
	out := MergeSettings(base, project)
	for name, v := range picked {
		out[name] = v
	}
	out[ByLevel] = MergeSettings(base, nil)
	out[ProjectSet] = MergeSettings(nil, project)
	out[LevelFrom] = from
	return out
}

// LevelOf reads a level setting from a role's merged settings: its value
// and where it comes from (FromRoleSetting, FromProjectDefault or
// FromRoleDefault); "" for a role with no such level.
func LevelOf(settings map[string]any, name string) (level, from string) {
	level, _ = settings[name].(string)
	if f, ok := settings[LevelFrom].(map[string]any); ok {
		from, _ = f[name].(string)
	}
	return level, from
}

// MergeSettings lays over on defaults into a new map that shares nothing
// with either: changing the result changes neither of them.
func MergeSettings(defaults, over map[string]any) map[string]any {
	out := make(map[string]any, len(defaults)+len(over))
	for k, v := range defaults {
		out[k] = cloneSetting(v)
	}
	for k, v := range over {
		if v == nil {
			delete(out, k)
			continue
		}
		if om, ok := v.(map[string]any); ok {
			dm, _ := out[k].(map[string]any) // nil when the default is no map: nulls inside are dropped all the same
			out[k] = MergeSettings(dm, om)
			continue
		}
		out[k] = cloneSetting(v)
	}
	return out
}

// cloneSetting copies a setting's maps and lists, at every depth.
func cloneSetting(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return MergeSettings(t, nil)
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = cloneSetting(x)
		}
		return out
	}
	return v
}

// Enforcement returns how hard each rule bites for this role in this project.
func (r *Role) Enforcement(c *ProjectConfig) map[string]string {
	if rc, ok := c.Roles[r.Name]; ok && rc.Enforce != nil {
		return rc.Enforce
	}
	return map[string]string{}
}

// Writes returns the path patterns the role may write, with `$settings.<name>`
// replaced by the setting's value (a string or a list of strings).
func (r *Role) Writes(settings map[string]any) []string {
	var out []string
	for _, w := range r.Duties.Writes {
		name, ok := strings.CutPrefix(w, "$settings.")
		if !ok {
			out = append(out, w)
			continue
		}
		var vals []string
		switch v := settings[name].(type) {
		case string:
			vals = append(vals, v)
		case []any:
			for _, x := range v {
				if s, ok := x.(string); ok {
					vals = append(vals, s)
				}
			}
		}
		if len(vals) == 0 && name == testdef.PathsSetting {
			vals = testdef.Generic().Patterns() // unset: the generic test files (ADR-0065)
		}
		out = append(out, vals...)
	}
	return out
}

// TestDef is the project's one definition of a test file (ADR-0065,
// change 3): the tester's test-paths and test-homes, in its role.yaml
// under rolesDir and the project's config, each unset the generic one.
// Not read, the generic one, with the error.
func TestDef(rolesDir, repo string) (testdef.Def, error) {
	cfg, err := LoadProjectConfig(repo)
	if err != nil {
		return testdef.Generic(), err
	}
	r, err := Load(rolesDir, "tester")
	if err != nil {
		// The project's own settings still say it.
		d, _ := testdef.FromSettings(cfg.Roles["tester"].Settings)
		return d, fmt.Errorf("the tester's role, whose test-paths say which files are tests: %w", err)
	}
	return testdef.FromSettings(r.MergedSettings(cfg))
}

// mergeErrorRows are the pack's build-error rows, then the role's whose
// error pattern the pack's rows do not hold.
func mergeErrorRows(pack, role any) ([]any, bool) {
	p, ok1 := pack.([]any)
	r, ok2 := role.([]any)
	if !ok1 || !ok2 {
		return nil, false
	}
	seen := map[string]bool{}
	for _, x := range p {
		if m, ok := x.(map[string]any); ok {
			if e, ok := m["error"].(string); ok {
				seen[e] = true
			}
		}
	}
	out := slices.Clone(p)
	for _, x := range r {
		if m, ok := x.(map[string]any); ok {
			if e, ok := m["error"].(string); ok && seen[e] {
				continue
			}
		}
		out = append(out, x)
	}
	return out, true
}
