package work

import (
	"regexp"
	"slices"
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
	// gherkinWords: a text written as Given / When / Then.
	givenWord    = regexp.MustCompile(`(?i)\bgiven\b`)
	whenThenWord = regexp.MustCompile(`(?i)\b(?:when|then)\b`)
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
//   - the same in fenced code (```gherkin), Feature and Background left out.
//
// Gherkin's keywords are Scenario, Scenario Outline, Scenario Template and
// Example; the name is what follows the keyword. A line naming a scenario
// with no step under it names none.
func Scenarios(validation string) []Scenario {
	type read struct {
		Scenario
		header bool // named on its own line: the steps below are its own
		indent int  // a bullet's indent: one deeper is its own
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
		case !isBullet:
			add(t) // a line going on, or a step under a name
		case cur != nil && (cur.header || indent > cur.indent):
			add(t) // a bullet under the line naming it, or indented under its bullet
		default:
			cur = nil // a bullet with no name: not a scenario a test can cite
		}
	}
	var scenarios []Scenario
	for _, r := range out {
		if r.Name == "" || r.header && r.indent < 0 && !hasStep(r.Text) {
			continue // a name alone on its line, no step under it: a title, not a scenario
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

// hasStep says whether a scenario's text holds a Gherkin step.
func hasStep(text string) bool {
	return givenWord.MatchString(text) || whenThenWord.MatchString(text)
}

// ScenariosUnread says whether a Validation is written as Given / When /
// Then — a Given, and a When or a Then — yet holds no scenario Scenarios
// reads: no test can name one, so the tester writes none.
func ScenariosUnread(validation string) bool {
	return givenWord.MatchString(validation) && whenThenWord.MatchString(validation) && len(Scenarios(validation)) == 0
}
