package scale

// Where a project keeps each axis (ADR-0040, "Where the values live"):
// detected each run, with no agent, when the project's settings ask —
// the setting's field, weight or family of labels; else the forge's own
// field, or GitLab's weight for the effort; else the project's own
// family; else the forge's words, created when the role first sets one.
// A value is read in one place an axis: another place holding one on an
// issue is never written over, nor mirrored.

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// How a setting says where an axis is kept: detected, a family of
// labels named by its prefix, a field named, or GitLab's weight (the
// effort only).
const (
	InAuto   = "auto"
	InLabels = "labels"
	InField  = "field"
	InWeight = "weight"
)

// Setting is where a project's settings say it keeps an axis. Unset (In
// ""), the engine reads as before the detection (Before).
type Setting struct {
	In     string
	Prefix string         // with In "labels": the family's prefix
	Field  string         // with In "field": the issue field's name
	Names  map[string]int // a label's name, of the family read, to the level the settings give it
}

// Set says whether the project's settings name where the axis is kept.
func (s Setting) Set() bool { return s.In != "" }

// ReadSetting reads an axis's setting: unset, `auto`, or a map — `in`
// (auto, labels, field, or weight for the effort), `prefix` with labels,
// `name` with field, `names` mapping a label of the family to a level.
// Anything else is an error, never read as unset.
func ReadSetting(s Scale, v any) (Setting, error) {
	bad := func(format string, a ...any) (Setting, error) {
		return Setting{}, fmt.Errorf("%s: "+format, append([]any{s.Axis}, a...)...)
	}
	switch v := v.(type) {
	case nil:
		return Setting{}, nil
	case string:
		if v != InAuto {
			return bad("%q is not %s, nor {in: %s, prefix: …}, nor {in: %s, name: …}", v, InAuto, InLabels, InField)
		}
		return Setting{In: InAuto}, nil
	case map[string]any:
		var set Setting
		for _, k := range slices.Sorted(maps.Keys(v)) {
			switch k {
			case "in":
				in, _ := v[k].(string)
				switch {
				case in == InWeight && s.Axis != Effort.Axis:
					return bad("in: %s is GitLab's weight, an effort: not a %s", InWeight, s.Axis)
				case in != InAuto && in != InLabels && in != InField && in != InWeight:
					return bad("in: %v is not one of %s, %s, %s, %s", v[k], InAuto, InLabels, InField, InWeight)
				}
				set.In = in
			case "prefix":
				p, _ := v[k].(string)
				if strings.TrimSpace(p) == "" {
					return bad("prefix: %v is not the start of a label's name", v[k])
				}
				set.Prefix = p
			case "name":
				n, _ := v[k].(string)
				if strings.TrimSpace(n) == "" {
					return bad("name: %v is not a field's name", v[k])
				}
				set.Field = strings.TrimSpace(n)
			case "names":
				m, ok := v[k].(map[string]any)
				if !ok {
					return bad("names: %v is not a map of a label's name to a level", v[k])
				}
				set.Names = map[string]int{}
				for name, level := range m {
					w, _ := level.(string)
					l, known := s.Word(w)
					if !known {
						return bad("names: %s: %v is not a level (%s)", name, level, strings.Join(s.Names, ", "))
					}
					set.Names[name] = l
				}
			default:
				return bad("%s is not a key it reads (in, prefix, name, names)", k)
			}
		}
		switch {
		case set.In == "":
			return bad("in is missing: %s, %s, %s or %s", InAuto, InLabels, InField, InWeight)
		case set.In == InLabels && set.Prefix == "":
			return bad("in: %s names its family by a prefix, missing", InLabels)
		case set.In == InField && set.Field == "":
			return bad("in: %s names its field by a name, missing", InField)
		case set.In != InLabels && set.Prefix != "":
			return bad("a prefix names a family of labels: in: %s, not %s", InLabels, set.In)
		case set.In != InField && set.Field != "":
			return bad("a name names a field: in: %s, not %s", InField, set.In)
		}
		return set, nil
	}
	return bad("%v is not %s, nor {in: %s, prefix: …}, nor {in: %s, name: …}", v, InAuto, InLabels, InField)
}

// Field is a single-select issue field of the forge an axis may be kept
// in (ADR-0040): its options as its owner arranged them; Hidden, its
// values visible to the organisation's members only on a public
// repository, or a visibility not read; Blind, why this token reads no
// hidden value, "" when it does.
type Field struct {
	ID      int64
	Name    string
	Options []string
	Hidden  bool
	Blind   string
}

// Source is where the engine reads and writes an axis on a project.
type Source struct {
	Family Family
	// Said is where, as the run's summary says it: "labels P0 to P3, the
	// project's"; "" when the axis is read nowhere.
	Said string
	// Words: the forge's words, each created when the role first sets
	// one — all of them, ranked where the forge ranks labels.
	Words bool
	// Others are the axis's other families on the project: read to tell
	// a value kept in another place, never written.
	Others []Family
	// Unknown names the labels of the family no table knows, nor the
	// settings map: read as none, said once.
	Unknown error
	// Field: the issue field the axis is kept in, its Family its options;
	// "" when it is kept in labels. OtherField, a field of the axis the
	// project has beside the labels read: a value there is elsewhere.
	Field      Field
	OtherField string
	// Weight: the effort is kept in GitLab's weight (Premium, Ultimate),
	// read in bands, written 1, 3 or 5, never on a parent (ADR-0040).
	// OtherWeight: the project keeps a weight beside the labels read, a
	// value there elsewhere.
	Weight      bool
	OtherWeight bool
}

// InField says whether the axis is kept in an issue field.
func (s Source) InField() bool { return s.Field.Name != "" }

// Native says whether the axis is kept in the forge's own place, a field
// or GitLab's weight, rather than in labels.
func (s Source) Native() bool { return s.InField() || s.Weight }

// Place says where the axis is kept, as a sentence names it: "the field
// Priority", "GitLab's weight", "P0 to P3".
func (s Source) Place() string {
	switch {
	case s.InField():
		return "the field " + s.Field.Name
	case s.Weight:
		return WeightSaid
	}
	return s.Family.Span()
}

// WeightSaid names GitLab's weight where the role says an effort is kept.
const WeightSaid = "GitLab's weight"

// On is what an issue carries where an axis may be kept: its labels, its
// values in the forge's issue fields, by name, and its GitLab weight, nil
// for none.
type On struct {
	Labels []string
	Fields map[string]string
	Weight *int
}

// Hidden says whether the axis's values are visible to the organisation's
// members only, on a public repository: never shown in the role's
// comment, its state, nor the job's summary (ADR-0040).
func (s Source) Hidden() bool { return s.Field.Hidden }

// Read are the levels an issue carries on the axis: its field's value,
// its weight read in bands, or its labels of the family.
func (s Source) Read(on On) []int {
	switch {
	case s.Weight:
		if on.Weight != nil {
			if l := Weight(*on.Weight); l > 0 {
				return []int{l}
			}
		}
		return nil
	case !s.InField():
		return s.Family.Levels(on.Labels)
	}
	if l := s.Family.Of(on.Fields[s.Field.Name]); l > 0 {
		return []int{l}
	}
	return nil
}

// Elsewhere are where an issue carries the axis outside its source — its
// labels of another family, or a field or a weight beside them —: a
// value there is never written over, nor mirrored (ADR-0040).
func (s Source) Elsewhere(on On) []string {
	var out []string
	for _, f := range s.Others {
		for _, n := range on.Labels {
			if f.Of(n) > 0 && !slices.Contains(out, n) {
				out = append(out, n)
			}
		}
	}
	if s.OtherField != "" && on.Fields[s.OtherField] != "" {
		out = append(out, "the field "+s.OtherField)
	}
	if s.OtherWeight && on.Weight != nil {
		out = append(out, WeightSaid)
	}
	return out
}

// Span says a family as the summary does: "P0 to P3" for numbers after
// a letter, else its prefix and a star, "priority/*".
func (f Family) Span() string {
	last, _ := utf8.DecodeLastRuneInString(f.Prefix)
	if n := len(f.Labels); n > 1 && f.Prefix != "" && isAlnum(last) {
		names := make([]string, n)
		for i, l := range f.Labels {
			names[i] = l.Name
		}
		slices.SortFunc(names, func(a, b string) int {
			x, _ := number(strings.TrimPrefix(a, f.Prefix))
			y, _ := number(strings.TrimPrefix(b, f.Prefix))
			return x - y
		})
		return names[0] + " to " + names[n-1]
	}
	return f.Prefix + "*"
}

// The families a project keeps an axis in, tried in this order (ADR-0040,
// "Where the values live"). Never size/*: Kubernetes' counts lines.
var families = map[string][]string{
	Priority.Axis: {"P", "P-", "priority/", "Priority/", "priority::"},
	Effort.Axis:   {"effort", "effort/", "effort::"},
}

// anchors say what each effort level means to build.
var anchors = []string{"hours", "days", "weeks"}

// effortColors are the forge's effort words' colours, the lightest the
// palest.
var effortColors = []string{"c2e0c6", "fef2c0", "f9d0c4"}

// Words are the forge's words for an axis, created when the role first
// sets one: on GitHub, the fields' option names, priority/urgent to
// priority/low and effort/low to effort/high, so a repository moved to
// an organisation maps them one to one; on GitLab (scoped), the ranked
// priority::1 to ::4, its own "Label priority" sort then following the
// role's, and effort::low to ::high.
func Words(s Scale, scoped bool) Family {
	f := Family{Scale: s, Prefix: s.Axis + "/"}
	if scoped {
		f.Prefix = s.Axis + "::"
	}
	for i, name := range s.Names {
		l := Label{Name: f.Prefix + strings.ToLower(name), Level: i + 1}
		switch s.Axis {
		case Priority.Axis:
			l.Color, l.About = Own.Labels[i].Color, fmt.Sprintf("Priority %s, of %s", name, strings.Join(s.Names, ", "))
			if scoped {
				l.Name, l.Rank = f.Prefix+strconv.Itoa(i+1), i+1
			}
		default:
			l.Color, l.About = effortColors[i], fmt.Sprintf("Effort %s, %s to build", name, anchors[i])
		}
		f.Labels = append(f.Labels, l)
	}
	return f
}

// ownSaid is where workline's own priority labels are said to be.
var ownSaid = "labels " + Own.Span() + ", workline's own"

// Detect finds where a project keeps an axis (ADR-0040, "Where the values
// live"), with no agent. labels are the project's, open those on its open
// issues; scoped says the forge's labels are GitLab's; fields, its issue
// fields; weights, that its issues carry GitLab's weight. Unset, as
// before the detection (Before). Else the setting's field, weight or
// family; else the forge's own field named as the axis, Priority or
// Effort; else, for the effort, GitLab's weight; else the project's own
// family, two values or more and one on an open issue; else workline's
// own priority labels, one on an open issue; else the forge's words. A
// setting naming a field, a weight or a family the project lacks is an
// error: the run stops, never reading another.
func Detect(s Scale, set Setting, labels, open []string, scoped bool, fields []Field, weights bool) (Source, error) {
	if !set.Set() {
		if s.Axis == Priority.Axis {
			return Before.Priority, nil
		}
		return Before.Effort, nil
	}
	words := Words(s, scoped)
	read := func(prefix string) (Family, error) { return named(s, prefix, labels, set.Names) }
	used := func(f Family) bool {
		return slices.ContainsFunc(f.Labels, func(l Label) bool { return slices.Contains(open, l.Name) })
	}
	// The forge's words already there are the role's, read as such: a
	// word missing created when first set.
	theWords := func(f Family) bool {
		return f.Prefix == words.Prefix && !slices.ContainsFunc(f.Labels, func(l Label) bool { return words.Of(l.Name) == 0 })
	}
	// A field, read by its options' names, never their places.
	field := func(name string) (Field, bool) {
		i := slices.IndexFunc(fields, func(f Field) bool { return strings.EqualFold(f.Name, name) })
		if i < 0 {
			return Field{}, false
		}
		return fields[i], true
	}
	fromField := func(f Field, how string) Source {
		fam, err := Options(s, f.Options)
		said := "the field " + f.Name + how
		if f.Hidden {
			said += ", its values visible to organization members only: never shown"
		}
		return Source{Family: fam, Field: f, Said: said, Unknown: err}
	}
	var src Source
	candidates := slices.Clone(families[s.Axis])
	if s.Axis == Priority.Axis {
		candidates = append(candidates, Own.Prefix)
	}
	native, hasNative := field(s.Axis)
	nativeLevels, _ := Options(s, native.Options)
	weight := Source{Family: WeightFamily, Weight: true, Said: WeightSaid}
	switch {
	case set.In == InWeight:
		switch {
		case s.Axis != Effort.Axis:
			return Source{}, fmt.Errorf("%s: %s keeps an effort, never a %s", s.Axis, WeightSaid, s.Axis)
		case !weights:
			return Source{}, fmt.Errorf("%s: the settings name %s, and its issues carry none (GitLab Premium and Ultimate keep one): name a family of labels, or auto", s.Axis, WeightSaid)
		}
		src = weight
		src.Said += ", as the settings say"
	case set.In == InField:
		f, ok := field(set.Field)
		if !ok {
			return Source{}, fmt.Errorf("%s: the settings name the field %s, and the project has none: name a field it has, or auto", s.Axis, set.Field)
		}
		src = fromField(f, ", as the settings say")
	case set.In == InLabels:
		f, err := read(set.Prefix)
		if len(f.Labels) == 0 {
			return Source{}, fmt.Errorf("%s: the settings name the labels %s*, and the project has none that names a level: name a family it has, or auto", s.Axis, set.Prefix)
		}
		src = Source{Family: f, Said: "labels " + f.Span() + ", as the settings say", Unknown: err}
		if !slices.Contains(candidates, set.Prefix) {
			candidates = append(candidates, set.Prefix)
		}
	case hasNative && len(nativeLevels.Labels) > 0:
		// The forge's own field, Priority or Effort, before any label.
		src = fromField(native, "")
	case weights && s.Axis == Effort.Axis:
		// GitLab's weight, Premium's and Ultimate's, before any label.
		src = weight
	default:
		for _, p := range candidates {
			f, err := read(p)
			own := p == Own.Prefix
			if (len(f.Labels) >= 2 || own && len(f.Labels) > 0) && used(f) {
				src = Source{Family: f, Said: "labels " + f.Span() + ", the project's", Unknown: err}
				if own {
					src = Source{Family: Own, Said: ownSaid}
				}
				break
			}
		}
	}
	switch {
	case src.Native():
	case src.Family.Scale.Axis == "":
		src = Source{Family: words, Words: true, Said: "labels " + words.Span() + ", created when first set"}
		if slices.ContainsFunc(words.Labels, func(l Label) bool { return slices.Contains(labels, l.Name) }) {
			src.Said = "labels " + words.Span() + ", created"
		}
	case theWords(src.Family):
		src.Family, src.Words, src.Unknown = words, true, nil
		src.Said = "labels " + words.Span() + ", created"
	}
	for _, p := range candidates {
		if p == src.Family.Prefix && !src.Native() {
			continue
		}
		if f, _ := read(p); len(f.Labels) > 0 {
			src.Others = append(src.Others, f)
		}
	}
	if !src.InField() && hasNative {
		src.OtherField = native.Name
	}
	if !src.Weight && weights && s.Axis == Effort.Axis {
		src.OtherWeight = true
	}
	return src, nil
}
