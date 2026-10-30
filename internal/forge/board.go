package forge

// The boards a forge shows the work on (ADR-0057), read-only: GitHub's
// projects linked to the repository, their Status field's options; GitLab's
// project and group boards, their label lists. workline finds them each
// run and creates none.

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Board is one board, its columns in the forge's order.
type Board struct {
	Forge  string `json:"forge"`            // github | gitlab
	Number int    `json:"number,omitempty"` // GitHub: the project's number; GitLab: the board's id
	Name   string `json:"name"`
	// Owner: GitHub, organization or user; GitLab, project or group.
	Owner string `json:"owner,omitempty"`
	URL   string `json:"url,omitempty"`
	// Columns: GitHub, the Status field's options; GitLab, the labels of
	// its label lists. None: the board has no column workline can set.
	Columns []string `json:"columns"`
	// Workflows: GitHub, its built-in workflows by name, whether on;
	// unread when WorkflowsUnread says why.
	Workflows       map[string]bool `json:"workflows"`
	WorkflowsUnread string          `json:"workflows-unread,omitempty"`
	// ClosedList: GitLab, the board shows its Closed list.
	ClosedList bool `json:"closed-list,omitempty"`
}

// Boards reads f's boards; ok is false when f keeps none (the local
// forge, a command, none). An error beside boards says what part was
// not read: the boards found stand.
func Boards(f any) (boards []Board, ok bool, err error) {
	b, ok := f.(interface{ boards() ([]Board, error) })
	if !ok {
		return nil, false, nil
	}
	boards, err = b.boards()
	return boards, true, err
}

func (f *Fake) boards() ([]Board, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	if s.BoardsUnread != "" {
		// The boards listed are those read, the rest unread: as a group's
		// boards GitLab did not answer.
		return s.Boards, errors.New(s.BoardsUnread)
	}
	return s.Boards, nil
}

// projectsQuery lists the open projects linked to the repository, their
// owner's kind and Status field. Kept apart from their workflows, which
// a token may not read while it reads the rest.
const projectsQuery = `query($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    projectsV2(first: 20) { nodes { number title url closed owner { __typename }
      field(name: "Status") { ... on ProjectV2SingleSelectField { options { name } } } } }
  }
}`

const workflowsQuery = `query($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    projectsV2(first: 20) { nodes { number workflows(first: 20) { nodes { name enabled } } } }
  }
}`

func (g *github) boards() ([]Board, error) {
	out, err := g.api("graphql", "-f", "query="+projectsQuery, "-F", "owner={owner}", "-F", "name={repo}")
	if err != nil {
		return nil, projectsUnread(err)
	}
	var v struct {
		Data struct {
			Repository struct {
				ProjectsV2 struct {
					Nodes []struct {
						Number int    `json:"number"`
						Title  string `json:"title"`
						URL    string `json:"url"`
						Closed bool   `json:"closed"`
						Owner  struct {
							Type string `json:"__typename"`
						} `json:"owner"`
						Field *struct {
							Options []struct {
								Name string `json:"name"`
							} `json:"options"`
						} `json:"field"`
					} `json:"nodes"`
				} `json:"projectsV2"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := decode(out, &v); err != nil {
		return nil, err
	}
	var boards []Board
	for _, n := range v.Data.Repository.ProjectsV2.Nodes {
		if n.Closed {
			continue
		}
		b := Board{Forge: "github", Number: n.Number, Name: n.Title, URL: n.URL, Owner: strings.ToLower(n.Owner.Type), Columns: []string{}}
		if n.Field != nil {
			for _, o := range n.Field.Options {
				b.Columns = append(b.Columns, o.Name)
			}
		}
		boards = append(boards, b)
	}
	if len(boards) == 0 {
		return nil, nil
	}
	flows, why := g.workflows()
	for i := range boards {
		if why != "" {
			boards[i].WorkflowsUnread = why
			continue
		}
		boards[i].Workflows = flows[boards[i].Number]
	}
	return boards, nil
}

// workflows reads each linked project's built-in workflows, by its
// number; why says what kept them unread.
func (g *github) workflows() (map[int]map[string]bool, string) {
	out, err := g.api("graphql", "-f", "query="+workflowsQuery, "-F", "owner={owner}", "-F", "name={repo}")
	if err != nil {
		return nil, Said(projectsUnread(err))
	}
	var v struct {
		Data struct {
			Repository struct {
				ProjectsV2 struct {
					Nodes []struct {
						Number    int `json:"number"`
						Workflows struct {
							Nodes []struct {
								Name    string `json:"name"`
								Enabled bool   `json:"enabled"`
							} `json:"nodes"`
						} `json:"workflows"`
					} `json:"nodes"`
				} `json:"projectsV2"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := decode(out, &v); err != nil {
		return nil, Said(err)
	}
	flows := map[int]map[string]bool{}
	for _, n := range v.Data.Repository.ProjectsV2.Nodes {
		flows[n.Number] = map[string]bool{}
		for _, w := range n.Workflows.Nodes {
			flows[n.Number][w.Name] = w.Enabled
		}
	}
	return flows, ""
}

// SetStatus sets the Status of issue's card in the GitHub project
// numbered project to option (ADR-0057): one value, the field holding
// one. Nothing written, why says it: the issue in no item of the project
// (adding it is the project's Auto-add), its card there already, f
// keeping no project.
func SetStatus(f any, issue, project int, option string) (why string, err error) {
	s, ok := f.(interface {
		setCardStatus(issue, project int, option string) (string, error)
	})
	if !ok {
		return "the forge keeps no project", nil
	}
	return s.setCardStatus(issue, project, option)
}

func (f *Fake) setCardStatus(issue, project int, option string) (why string, err error) {
	err = f.write(func(s *FakeState) error {
		if s.BoardsUnwritable != "" {
			return fmt.Errorf("%w: %s", ErrProjectUnwritable, s.BoardsUnwritable)
		}
		it, err := s.item(Target{Kind: "issue", ID: issue})
		if err != nil {
			return err
		}
		key := strconv.Itoa(project)
		was, in := it.Status[key]
		switch {
		case !in:
			why = notInProject(issue, project)
		case was == option:
			why = "its card is there already"
		default:
			it.Status[key] = option
		}
		return nil
	})
	return why, err
}

func notInProject(issue, project int) string {
	return fmt.Sprintf("#%d sits in no item of the project #%d: adding it is the project's Auto-add", issue, project)
}

// itemQuery reads an issue's cards: each project's item, its Status, and
// the project's Status field, its options' ids.
const itemQuery = `query($owner: String!, $name: String!, $issue: Int!) {
  repository(owner: $owner, name: $name) { issue(number: $issue) { projectItems(first: 20) { nodes { id
    fieldValueByName(name: "Status") { ... on ProjectV2ItemFieldSingleSelectValue { name } }
    project { id number field(name: "Status") { ... on ProjectV2SingleSelectField { id options { id name } } } } } } } }
}`

const statusMutation = `mutation($project: ID!, $item: ID!, $field: ID!, $option: String!) {
  updateProjectV2ItemFieldValue(input: {projectId: $project, itemId: $item, fieldId: $field, value: {singleSelectOptionId: $option}}) { projectV2Item { id } }
}`

func (g *github) setCardStatus(issue, project int, option string) (string, error) {
	out, err := g.api("graphql", "-f", "query="+itemQuery, "-F", "owner={owner}", "-F", "name={repo}", "-F", fmt.Sprintf("issue=%d", issue))
	if err != nil {
		return "", projectsUnread(err)
	}
	var v struct {
		Data struct {
			Repository struct {
				Issue struct {
					ProjectItems struct {
						Nodes []struct {
							ID     string `json:"id"`
							Status *struct {
								Name string `json:"name"`
							} `json:"fieldValueByName"`
							Project struct {
								ID     string `json:"id"`
								Number int    `json:"number"`
								Field  *struct {
									ID      string `json:"id"`
									Options []struct {
										ID   string `json:"id"`
										Name string `json:"name"`
									} `json:"options"`
								} `json:"field"`
							} `json:"project"`
						} `json:"nodes"`
					} `json:"projectItems"`
				} `json:"issue"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := decode(out, &v); err != nil {
		return "", err
	}
	for _, n := range v.Data.Repository.Issue.ProjectItems.Nodes {
		if n.Project.Number != project || n.Project.Field == nil {
			continue
		}
		if n.Status != nil && n.Status.Name == option {
			return "its card is there already", nil
		}
		for _, o := range n.Project.Field.Options {
			if o.Name != option {
				continue
			}
			_, err := g.api("graphql", "-f", "query="+statusMutation, "-f", "project="+n.Project.ID, "-f", "item="+n.ID,
				"-f", "field="+n.Project.Field.ID, "-f", "option="+o.ID)
			return "", projectUnwritable(err)
		}
		return fmt.Sprintf("the project #%d's Status has no option %q any more", project, option), nil
	}
	return notInProject(issue, project), nil
}

// ErrProjectUnwritable: the forge refused a project's write for want of
// the right, the token's (ADR-0057).
var ErrProjectUnwritable = errors.New("the token may not write the project")

// projectUnwritable tells a write GitHub refused for want of the right:
// an App without Organization projects' write ("Resource not accessible
// by integration"), a person's token without the project scope.
func projectUnwritable(err error) error {
	if err == nil {
		return nil
	}
	m := err.Error()
	if strings.Contains(m, "Resource not accessible") || strings.Contains(m, "required scopes") || strings.Contains(m, "FORBIDDEN") {
		return fmt.Errorf("%w: %s", ErrProjectUnwritable, m)
	}
	return err
}

// projectsUnread names the right a token lacks to read projects, as GitHub
// says it: a person's token, the read:project scope.
func projectsUnread(err error) error {
	if strings.Contains(err.Error(), "read:project") {
		return errors.New("the token lacks the read:project scope (`gh auth refresh -s read:project`; a GitHub App: Organization projects, read)")
	}
	return err
}

// gitlabBoardsQuery reads a project's or a group's boards and their lists;
// `first` keeps it under GitLab's complexity limit.
const gitlabBoardsQuery = `query($p: ID!) { %s(fullPath: $p) { %s boards(first: 20) { nodes { id name webUrl hideClosedList
  lists(first: 20) { nodes { listType label { title } } } } } } }`

type gitlabBoards struct {
	Nodes []struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		WebURL         string `json:"webUrl"`
		HideClosedList bool   `json:"hideClosedList"`
		Lists          struct {
			Nodes []struct {
				ListType string `json:"listType"`
				Label    *struct {
					Title string `json:"title"`
				} `json:"label"`
			} `json:"nodes"`
		} `json:"lists"`
	} `json:"nodes"`
}

func (g *gitlab) boards() ([]Board, error) {
	// Connected first: out of CI, the project's path is the remote's,
	// read there; often the first call of a run.
	if err := g.connect(); err != nil {
		return nil, err
	}
	var p struct {
		Project *struct {
			Group *struct {
				FullPath string `json:"fullPath"`
			} `json:"group"`
			Boards gitlabBoards `json:"boards"`
		} `json:"project"`
	}
	ok, err := g.graphql(fmt.Sprintf(gitlabBoardsQuery, "project", "group { fullPath }"), map[string]any{"p": g.projectPath()}, &p)
	if err != nil {
		return nil, err
	}
	if !ok || p.Project == nil {
		return nil, errors.New("GitLab's GraphQL API answered no boards for the project: the token may not read it")
	}
	boards := gitlabColumns(p.Project.Boards, "project")
	if p.Project.Group == nil {
		return boards, nil
	}
	var grp struct {
		Group *struct {
			Boards gitlabBoards `json:"boards"`
		} `json:"group"`
	}
	// The group's boards unread, the project's are kept, the group's
	// said: Boards' error beside boards found.
	ok, err = g.graphql(fmt.Sprintf(gitlabBoardsQuery, "group", ""), map[string]any{"p": p.Project.Group.FullPath}, &grp)
	switch {
	case err != nil:
		return boards, fmt.Errorf("the group %s's boards: %s", p.Project.Group.FullPath, Said(err))
	case !ok || grp.Group == nil:
		return boards, fmt.Errorf("the group %s's boards: GitLab's GraphQL API answered none: the token may not read them", p.Project.Group.FullPath)
	}
	return append(boards, gitlabColumns(grp.Group.Boards, "group")...), nil
}

func gitlabColumns(bs gitlabBoards, owner string) []Board {
	var boards []Board
	for _, n := range bs.Nodes {
		b := Board{Forge: "gitlab", Name: n.Name, Owner: owner, URL: n.WebURL, ClosedList: !n.HideClosedList, Columns: []string{}}
		b.Number, _ = strconv.Atoi(n.ID[strings.LastIndex(n.ID, "/")+1:])
		for _, l := range n.Lists.Nodes {
			if l.ListType == "label" && l.Label != nil {
				b.Columns = append(b.Columns, l.Label.Title)
			}
		}
		boards = append(boards, b)
	}
	return boards
}
