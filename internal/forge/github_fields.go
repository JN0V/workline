package forge

// GitHub's issue fields (ADR-0040): an organisation's single-select
// Priority and Effort, listed for a repository by GraphQL, read with each
// issue, written by REST. A value "visible to organization members only"
// is shown to a member's token; an outsider's reads none, as though the
// issue had none.

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// fieldsQuery lists the repository's single-select issue fields, whether
// it is public, and whether the token's user is a member of the
// organisation owning it; issueFields is null on a user's repository.
const fieldsQuery = `query($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    isPrivate
    owner { ... on Organization { viewerIsAMember } }
    issueFields(first: 50) { nodes { ... on IssueFieldSingleSelect { fullDatabaseId name visibility options { name } } } }
  }
}`

// fieldsVersion is the REST API version issue field values are written
// with (docs/research/priority-and-weight.md).
const fieldsVersion = "X-GitHub-Api-Version: 2026-03-10"

func (g *github) issueFields() (Fields, error) {
	out, err := g.api("graphql", "-f", "query="+fieldsQuery, "-F", "owner={owner}", "-F", "name={repo}")
	if err != nil {
		return Fields{}, err
	}
	var v struct {
		Data struct {
			Repository struct {
				IsPrivate bool `json:"isPrivate"`
				Owner     struct {
					ViewerIsAMember *bool `json:"viewerIsAMember"`
				} `json:"owner"`
				IssueFields *struct {
					Nodes []struct {
						ID         json.Number `json:"fullDatabaseId"`
						Name       string      `json:"name"`
						Visibility string      `json:"visibility"`
						Options    []struct {
							Name string `json:"name"`
						} `json:"options"`
					} `json:"nodes"`
				} `json:"issueFields"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := decode(out, &v); err != nil {
		return Fields{}, err
	}
	r := v.Data.Repository
	fs := Fields{Public: !r.IsPrivate, Blind: g.blind(r.Owner.ViewerIsAMember)}
	if r.IssueFields == nil {
		return fs, nil
	}
	for _, n := range r.IssueFields.Nodes {
		id, err := n.ID.Int64()
		if n.Name == "" || err != nil {
			continue // a field of another type: a date, a text
		}
		f := Field{ID: id, Name: n.Name, Public: n.Visibility == "ALL"}
		for _, o := range n.Options {
			f.Options = append(f.Options, o.Name)
		}
		fs.List = append(fs.List, f)
	}
	return fs, nil
}

// blind says why this token reads no value visible to an organisation's
// members only (ADR-0040, "Never read as empty"): a token the live try has
// not shown to read them — the job's own, an App's —, or a person who is
// no member; "" for a member's token.
func (g *github) blind(member *bool) string {
	switch {
	case strings.TrimSpace(os.Getenv("WORKLINE_APP")) != "":
		return "an App's token is not shown yet to read values visible to organization members only"
	case os.Getenv("GITHUB_ACTIONS") == "true":
		return "the job's own token (GITHUB_TOKEN) is not shown yet to read values visible to organization members only"
	case member == nil || !*member:
		return "the token's user is no member of the organization: its values visible to members only read as none"
	}
	return ""
}

func (g *github) setField(issue int, field Field, option string) error {
	body, _ := json.Marshal(map[string]any{"issue_field_values": []map[string]any{{"field_id": field.ID, "value": option}}})
	_, err := runInput(g.repo, body, "gh", "api", "-X", "POST", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/issue-field-values", issue),
		"-H", fieldsVersion, "--input", "-")
	return err
}
