package forge

import (
	"time"
)

// Budget is what the token has left of the forge's API budget, on the
// resource nearest spent: GitHub's REST calls (core) or its GraphQL
// points (graphql), each counted by the hour (#804).
type Budget struct {
	Resource  string    `json:"resource"`
	Remaining int       `json:"remaining"`
	Limit     int       `json:"limit"`
	Reset     time.Time `json:"reset"`
}

// Low says whether less than a tenth of the budget is left: a run of
// the line that started then would likely stop half-way on the rate.
func (b Budget) Low() bool { return b.Limit > 0 && b.Remaining*10 < b.Limit }

// Budgeted is a forge that says the token's API budget at no cost to it:
// GitHub's rate_limit, which counts against no limit
// (docs/github/calls.md). GitLab has no such call; it says its budget
// only in each answer's headers.
type Budgeted interface {
	Budget() (Budget, error)
}

// nearest is the resource with the least share left.
func nearest(all []Budget) (Budget, bool) {
	var low Budget
	found := false
	for _, b := range all {
		if b.Limit <= 0 {
			continue
		}
		if !found || b.Remaining*low.Limit < low.Remaining*b.Limit {
			low, found = b, true
		}
	}
	return low, found
}

// Budget reads GitHub's rate_limit: the token's REST and GraphQL budgets,
// the one nearest spent returned.
func (g *github) Budget() (Budget, error) {
	out, err := g.api("rate_limit", "--jq", ".resources | to_entries[] | select(.key == \"core\" or .key == \"graphql\") | {resource: .key, remaining: .value.remaining, limit: .value.limit, reset: .value.reset}")
	if err != nil {
		return Budget{}, err
	}
	found, err := lines[struct {
		Resource  string `json:"resource"`
		Remaining int    `json:"remaining"`
		Limit     int    `json:"limit"`
		Reset     int64  `json:"reset"`
	}](out)
	if err != nil {
		return Budget{}, err
	}
	var all []Budget
	for _, f := range found {
		all = append(all, Budget{Resource: f.Resource, Remaining: f.Remaining, Limit: f.Limit, Reset: time.Unix(f.Reset, 0)})
	}
	b, _ := nearest(all)
	return b, nil
}

// FakeBudget is the simulated forge's API budget: its reset ResetIn
// seconds after it is read.
type FakeBudget struct {
	Resource  string `json:"resource,omitempty"`
	Remaining int    `json:"remaining"`
	Limit     int    `json:"limit"`
	ResetIn   int    `json:"reset-in,omitempty"`
}

// Budget says the simulated forge's budget, when its file sets one; none
// set, a budget that is never low.
func (f *Fake) Budget() (Budget, error) {
	s, err := f.load()
	if err != nil {
		return Budget{}, err
	}
	if s.Budget == nil {
		return Budget{}, nil
	}
	r := s.Budget.Resource
	if r == "" {
		r = "core"
	}
	return Budget{Resource: r, Remaining: s.Budget.Remaining, Limit: s.Budget.Limit,
		Reset: time.Now().Add(time.Duration(s.Budget.ResetIn) * time.Second)}, nil
}
