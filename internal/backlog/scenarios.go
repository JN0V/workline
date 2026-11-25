package backlog

import (
	"regexp"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/work"
)

// ScenariosNoneSilent opens the product owner's word on an issue it holds
// in refine, its Validation prose, no scenario the tester reads (#1019):
// ready, it would skip its tests without a word.
var ScenariosNoneSilent = "its " + work.Name(work.Validation) + " names no scenario the tester reads"

// HeldForScenarios says whether the product owner's word on an issue is
// that it holds it for its Validation's scenarios.
func HeldForScenarios(silent string) bool {
	return strings.HasPrefix(silent, ScenariosNoneSilent)
}

// rewritesOwn closes that word when the Validation is the role's own text,
// its to rewrite (Revisable); a person's otherwise.
const rewritesOwn = "; the text is the product owner's own"

// ScenariosUnread says whether an issue's Validation names no scenario the
// tester reads — prose (#1019); Given / When / Then with no name is read,
// named by its Then —, and whether it is the
// role's to rewrite: its own draft or text, or one its writer marked
// `Draft:`, never a person's (Revisable). An empty Validation is not:
// it is missing (NotReady).
func ScenariosUnread(is forge.Issue, st *State) (unread, own bool) {
	v := work.ByMeaning(is.Body)[work.Validation]
	if strings.TrimSpace(markup.ReplaceAllString(v.Text, "")) == "" || len(work.Scenarios(v.Text)) > 0 {
		return false, false
	}
	return true, len(Revisable(is, st, &SpecReview{In: []string{v.Heading}})) > 0
}

// scenariosUnreadWhy is the reason said on an issue held for its unread
// scenarios.
func scenariosUnreadWhy(own bool) string {
	if own {
		return ScenariosNoneSilent + rewritesOwn
	}
	return ScenariosNoneSilent
}

// scenariosUnreadSaid is what the issue's state tells a person of it: why
// it waits in refine, and who rewrites its Validation, how.
func scenariosUnreadSaid(silent string) string {
	next := "a person rewrites each scenario of its " + work.Name(work.Validation) + " as " + work.ScenarioForm +
		", or as a `Scenario: <name>` line with its Given / When / Then below; it then goes to its tests"
	if strings.HasSuffix(silent, rewritesOwn) {
		next = "the product owner rewrites its " + work.Name(work.Validation) + ", each scenario " + work.ScenarioForm +
			"; or a person does; it then goes to its tests"
	}
	return Said("The product owner", "leaves this issue in refine: no test would be written first", strings.TrimSuffix(silent, rewritesOwn), next)
}

// ScenariosUnreadTask tells the agent, on an issue whose Validation no
// test can name, what holds it from ready and how it answers: a `refine`
// rewriting the Validation when it is its own; an `ask` otherwise.
func ScenariosUnreadTask(own bool) string {
	s := "I" + ScenariosNoneSilent[1:] + ", so no test can name one: it stays out of ready until it does. The form: each scenario a bullet " + work.ScenarioForm + ". "
	if own {
		return s + "It is yours to rewrite: answer with a `refine` giving its whole new " + work.Name(work.Validation) + " in that form, its meaning unchanged.\n"
	}
	return s + "It is a person's, never rewritten: `ask` its reporter to write each scenario in that form.\n"
}

// unreadStill says whether an issue held for its unread scenarios still
// is: in refine, its Validation as unread.
func unreadStill(is forge.Issue, st *State) bool {
	unread, _ := ScenariosUnread(is, st)
	return unread && !Readied(is)
}

// markedDraft opens a section its writer marked unfinished: `Draft:`, bold
// or not, on its first line.
var markedDraft = regexp.MustCompile(`(?i)^(?:\*\*|__)?draft\s*:`)

// MarkedDraft says whether the section of meaning m in body opens with
// `Draft:` (#1019): its writer, a person or an import, said it is not
// final. It is not ready until it is rewritten — the role may, Rewritable
// — or a person accepts it (workline:accepted); never taken as written.
func MarkedDraft(body string, m work.Meaning) bool {
	text := strings.TrimSpace(markup.ReplaceAllString(work.ByMeaning(body)[m].Text, ""))
	return drafted(m) && markedDraft.MatchString(text)
}
