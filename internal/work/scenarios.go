package work

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Scenario is one of an issue's Validation scenarios: its name, as a test
// names it, and its Given / When / Then.
type Scenario struct {
	Name string `json:"name"`
	Text string `json:"text"`
}

// ScenarioForm is the form workline writes a scenario in, as told to a
// person whose scenarios it does not read.
const ScenarioForm = "`- **Name:** Given …, When …, Then …`"

var (
	// bullet opens a list item: `-`, `*`, `+` or `1.`, its indent kept.
	bullet = regexp.MustCompile(`^(\s*)(?:[-*+]|\d{1,3}[.)])\s+`)
	// keyword is Gherkin's for a scenario, its name after the colon.
	keyword = regexp.MustCompile(`(?i)^(?:scenario outline|scenario template|scenario|example)\s*:\s*(.*)$`)
	// step opens a Gherkin step.
	step = regexp.MustCompile(`(?i)^(?:given|when|then|and|but)\b`)
	// apart opens a Gherkin block that is no scenario: its steps are none's.
	apart = regexp.MustCompile(`(?i)^(?:feature|background|rule)\s*:`)
	// given opens a scenario written with no name; then, its outcome.
	given = regexp.MustCompile(`(?i)^given\b`)
	then  = regexp.MustCompile(`(?i)\bthen\s+`)
)

// Scenarios reads the named scenarios of a Validation section, in the forms
// people write Gherkin in Markdown:
//
//   - a bullet opening on its name in bold, `- **Old debt:** Given …`, the
//     colon inside the bold or after it, its lines until the next bullet,
//     those indented under it with it;
//   - a line naming a scenario, `**Scenario: Old debt**`, `Scenario: Old
//     debt`, `#### Scenario: Old debt`, a bullet or a bold name alone, its
//     steps on the lines below, bullets or not, until the next one;
//   - the same in fenced code (```gherkin), Feature and Background left out;
//   - a line or a bullet opening on Given, with no name: named by the first
//     words of its Then, as a person would, never asked of them.
//
// Gherkin's keywords are Scenario, Scenario Outline, Scenario Template and
// Example; the name is what follows the keyword. A name with no text
// after nor under it names none, nor a line naming one with no step.
func Scenarios(validation string) []Scenario {
	type read struct {
		Scenario
		header  bool // named on its own line: the steps below are its own
		indent  int  // a bullet's indent: one deeper is its own
		unnamed bool // opened on Given: named by its Then once read
	}
	var out []read
	var cur *read // the scenario its lines go to, nil once ended
	add := func(t string) {
		if t = strings.TrimSpace(t); t == "" || cur == nil {
			return
		}
		switch {
		case cur.Text == "":
			cur.Text = t
		case step.MatchString(t) && !strings.ContainsAny(cur.Text[len(cur.Text)-1:], ".,;:"):
			cur.Text += ", " + t
		default:
			cur.Text += " " + t
		}
	}
	fence := ""
	for _, l := range strings.Split(validation, "\n") {
		was := fence
		if fence = Fenced(l, fence); fence != "" && was == "" || fence == "" && was != "" {
			continue // the fence's own line
		}
		l = strings.TrimRight(l, "\r")
		t := strings.TrimSpace(l)
		if t == "" {
			if cur != nil && cur.unnamed {
				cur = nil // a blank line ends a scenario named by its Then
			}
			continue
		}
		indent, isBullet := -1, false
		if m := bullet.FindStringSubmatch(l); m != nil {
			indent, isBullet = len(m[1]), true
			t = strings.TrimSpace(l[len(m[0]):])
		}
		heading := false
		if h := strings.TrimLeft(t, "#"); h != t && strings.HasPrefix(h, " ") {
			t, heading = strings.TrimSpace(h), true
		}
		name, rest, bold := splitBold(t)
		if m := keyword.FindStringSubmatch(name); m != nil {
			// Gherkin's keyword: its name after it, or after the bold.
			n := strings.TrimSpace(m[1])
			switch {
			case n == "" && bold && rest != "" && !step.MatchString(rest):
				n, rest = rest, ""
			case n == "":
				n = strings.TrimSpace(strings.TrimSuffix(name, ":"))
			}
			out = append(out, read{Scenario: Scenario{Name: trimName(n)}, header: !isBullet || rest == "", indent: indent})
			cur = &out[len(out)-1]
			add(rest)
			continue
		}
		switch {
		case apart.MatchString(t):
			cur = nil
			continue
		case bold && isBullet && !heading:
			// The form workline writes: a bullet opening on its name.
			out = append(out, read{Scenario: Scenario{Name: trimName(name)}, header: rest == "", indent: indent})
			cur = &out[len(out)-1]
			add(rest)
			continue
		case heading && !bold, bold && !isBullet && (rest == "" || step.MatchString(rest)):
			// A heading, or a bold name on its own line, its steps after
			// it or below.
			out = append(out, read{Scenario: Scenario{Name: trimName(name)}, header: true, indent: -1})
			cur = &out[len(out)-1]
			add(rest)
			continue
		}
		switch {
		case given.MatchString(t) && (cur == nil || cur.unnamed || isBullet && !cur.header && indent <= cur.indent):
			out = append(out, read{indent: indent, unnamed: true})
			cur = &out[len(out)-1]
			add(t)
		case !isBullet && (cur == nil || !cur.unnamed || step.MatchString(t)):
			add(t) // a line going on, or a step under a name; prose ends one named by its Then
		case cur != nil && (cur.header || indent > cur.indent || cur.unnamed && step.MatchString(t)):
			add(t) // a bullet under the line naming it, indented under its bullet, or its next step
		default:
			cur = nil // a bullet with no name: not a scenario a test can cite
		}
	}
	var scenarios []Scenario
	for _, r := range out {
		if r.unnamed {
			if outcome := then.Split(r.Text, -1); len(outcome) > 1 { // no Then: prose, no scenario
				words := strings.Fields(outcome[len(outcome)-1])
				r.Name = strings.Trim(strings.Join(words[:min(6, len(words))], " "), " .,;:`*_")
			}
			if slices.ContainsFunc(scenarios, func(s Scenario) bool { return s.Name == r.Name }) {
				r.Name += " (" + strconv.Itoa(len(scenarios)+1) + ")" // two Thens alike: told apart by place
			}
		}
		if r.Name == "" || r.Text == "" || r.header && r.indent < 0 && !hasStep(r.Text) {
			continue // a name with nothing under it, or alone on its line with no step: a title, not a scenario
		}
		scenarios = append(scenarios, r.Scenario)
	}
	return slices.Clip(scenarios)
}

// splitBold reads a line opening in bold (or italics): the text inside, what follows,
// and true; the line whole and false otherwise.
func splitBold(t string) (inside, rest string, bold bool) {
	for _, mark := range []string{"**", "__", "*", "_"} {
		if !strings.HasPrefix(t, mark) {
			continue
		}
		end := strings.Index(t[len(mark):], mark)
		if end <= 0 {
			continue
		}
		inside = strings.TrimSpace(t[len(mark) : len(mark)+end])
		rest = strings.TrimSpace(t[len(mark)+end+len(mark):])
		rest = strings.TrimSpace(strings.TrimPrefix(rest, ":"))
		return inside, rest, true
	}
	return t, "", false
}

// trimName is a scenario's name without the colon or marks around it.
func trimName(n string) string {
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(n), ":")), "*_"))
}

// hasStep says whether a scenario's text opens on a Gherkin step.
func hasStep(text string) bool {
	return step.MatchString(text)
}
