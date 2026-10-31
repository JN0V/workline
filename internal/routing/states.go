package routing

// The line of states (ADR-0059, decisions 2 and 8): an issue's state is
// its label, one at a time; each state wakes one role, whose result moves
// the issue to the next. The table is the routing file's; what reads it
// is a pure function of an issue's labels, no forge, no agent.

import (
	"fmt"
	"slices"
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
