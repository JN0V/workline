package routing

// The line of states (ADR-0059, decisions 2 and 8): an issue's state is
// its label, one at a time; each state wakes one role, whose result moves
// the issue to the next. The table is the routing file's; what reads it
// is a pure function of an issue's labels, no forge, no agent.

import (
	"fmt"
	"slices"
	"strings"
)

// State is a row of the table: its name, the label that says it, the role
// it wakes (none: nobody, the issue waits there), and the state that
// role's result moves the issue to.
type State struct {
	Name  string `yaml:"name"`
	Label string `yaml:"label"`
	Role  string `yaml:"role,omitempty"`
	Next  string `yaml:"next,omitempty"`
}

// StateOf is the state an issue's labels say: the first row of the table
// whose label it bears, the table's order settling an issue that bears
// two; false when it bears none, an issue outside the line.
func (c *Config) StateOf(labels []string) (State, bool) {
	for _, s := range c.States {
		if slices.Contains(labels, s.Label) {
			return s, true
		}
	}
	return State{}, false
}

// State is the row named name; false when the table holds none.
func (c *Config) State(name string) (State, bool) {
	for _, s := range c.States {
		if s.Name == name {
			return s, true
		}
	}
	return State{}, false
}

// Move is the labels a move to state to sets, in one call: its label on,
// every other state's label off — states are exclusive.
func (c *Config) Move(to string) (add, remove []string, err error) {
	s, ok := c.State(to)
	if !ok {
		return nil, nil, fmt.Errorf("routing.states holds no state %q", to)
	}
	for _, o := range c.States {
		if o.Label != s.Label {
			remove = append(remove, o.Label)
		}
	}
	return []string{s.Label}, remove, nil
}

// Wakes is the role an issue's state wakes on event, given the steps the
// project routes there: the state's role when the event's line lists it;
// "" and why when it wakes none.
func (c *Config) Wakes(labels []string, event string) (State, string, string) {
	s, ok := c.StateOf(labels)
	switch {
	case !ok:
		return s, "", "it bears no state's label: it is not on the line"
	case s.Role == "":
		return s, "", fmt.Sprintf("its state, %s, wakes no role: it waits there", s.Name)
	case !slices.Contains(c.Events[event], s.Role):
		return s, "", fmt.Sprintf("its state, %s, wakes the %s, which the project does not route on %s", s.Name, s.Role, event)
	}
	return s, s.Role, ""
}

// checkStates refuses a table the line cannot read: a state with no name
// or label, two of the same name or label, a next naming no state.
func checkStates(states []State) error {
	names, labels := map[string]bool{}, map[string]bool{}
	for _, s := range states {
		switch {
		case s.Name == "" || s.Label == "":
			return fmt.Errorf("routing.states: each state has a name and a label (%+v)", s)
		case names[s.Name]:
			return fmt.Errorf("routing.states: two states named %q", s.Name)
		case labels[s.Label]:
			return fmt.Errorf("routing.states: two states labelled %q", s.Label)
		}
		names[s.Name], labels[s.Label] = true, true
	}
	for _, s := range states {
		if s.Next != "" && !names[s.Next] {
			return fmt.Errorf("routing.states: %s goes next to %q, which is no state", s.Name, s.Next)
		}
	}
	return nil
}

// Limit is state name's WIP limit (ADR-0059, decision 9): the issues it
// holds at most before the step that moves one into it waits; 0, none.
func (c *Config) Limit(name string) int { return c.WIP[name] }


// StaleDays is state name's age limit, in days (ADR-0059, decision 7):
// an issue in it longer, counted from when its label was put on, is
// checked for staleness; 0, none.
func (c *Config) StaleDays(name string) int { return c.Stale[name] }

// RevisitDays is how many days a blocked issue waits, nothing new on
// it, before the product owner reads it again (ADR-0059, amended); 0,
// never: Blocked waits for a person.
func (c *Config) RevisitDays() int {
	if c.Revisit == nil {
		return 0
	}
	return *c.Revisit
}

// checkStale refuses an age limit the line cannot read: on no state, or
// below 0.
func checkStale(stale map[string]int, states []State) error {
	for name, days := range stale {
		if !slices.ContainsFunc(states, func(s State) bool { return s.Name == name }) {
			return fmt.Errorf("routing.stale.%s: no state of routing.states is named so", name)
		}
		if days < 0 {
			return fmt.Errorf("routing.stale.%s is %d: a limit is a number of days, 0 for none", name, days)
		}
	}
	return nil
}

// checkWIPCycle refuses limits that can stop the line for good: a ring
// of states, each one's next the one after, every one limited. Each
// state's step would wait for room in the next, full of issues whose
// step waits for room in the one after, round to the first. A state
// with no limit on the ring never waits: it drains, and the ring with
// it. With no ring, waits form chains that end at a state with no next
// or no limit, and each chain drains from its end (ADR-0059, amended
// 2026-11-02).
func checkWIPCycle(wip map[string]int, states []State) error {
	next := map[string]string{}
	for _, s := range states {
		next[s.Name] = s.Next
	}
	for _, s := range states {
		ring, seen := []string{s.Name}, map[string]bool{s.Name: true}
		for n := next[s.Name]; n != ""; n = next[n] {
			if n == s.Name {
				limited := true
				for _, r := range ring {
					limited = limited && wip[r] > 0
				}
				if limited {
					return fmt.Errorf("routing.wip: %s, each with a limit, go next round to %s: each would wait for room in the next, full, for good; lift one's limit (0)",
						strings.Join(ring, " → "), s.Name)
				}
				break
			}
			if seen[n] {
				break // a ring that does not hold s: found from one of its own states
			}
			seen[n] = true
			ring = append(ring, n)
		}
	}
	return nil
}

// checkWIP refuses a limit the line cannot read: on no state, or below 0.
func checkWIP(wip map[string]int, states []State) error {
	for name, n := range wip {
		if !slices.ContainsFunc(states, func(s State) bool { return s.Name == name }) {
			return fmt.Errorf("routing.wip.%s: no state of routing.states is named so", name)
		}
		if n < 0 {
			return fmt.Errorf("routing.wip.%s is %d: a limit is a number of issues, 0 for none", name, n)
		}
	}
	return nil
}
