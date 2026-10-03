package forge

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Local is a forge kept in the clone, never committed: for a project with no
// forge, or one whose person works alone. Each issue is a Markdown file,
// .git/workline/issues/<n>.md; each merge request is a local branch, recorded
// the same way in .git/workline/merge-requests/<n>.md. Nothing is pushed: `workline issues`
// reads them. Writes are idempotent, as on any forge.
type Local struct{ Repo string }

// LocalItem is an issue or a merge request of the local forge.
type LocalItem struct {
	ID       int      `yaml:"-"`
	Title    string   `yaml:"title"`
	State    string   `yaml:"state"` // open or closed, as written; see Local.StateOf
	Labels   []string `yaml:"labels,flow"`
	Branch   string   `yaml:"branch,omitempty"` // a merge request's local branch
	Base     string   `yaml:"base,omitempty"`
	Body     string   `yaml:"-"`
	Comments []string `yaml:"-"`
}

// commentLine starts each comment in an item's file. It is not a marker
// (Marker), so no comment's own marker is taken for it.
const commentLine = "<!-- workline-comment -->"

func kindDir(kind string) string {
	if kind == "merge-request" {
		return "merge-requests"
	}
	return "issues"
}

// dir is .git/workline in the repository's main clone, which its worktrees share.
func (l *Local) dir() (string, error) {
	out, err := exec.Command("git", "-C", l.Repo, "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		return "", fmt.Errorf("the local forge needs a git repository: %v", err)
	}
	return filepath.Join(strings.TrimSpace(string(out)), "workline"), nil
}

func (l *Local) path(kind string, id int) (string, error) {
	d, err := l.dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, kindDir(kind), fmt.Sprintf("%d.md", id)), nil
}

// Item reads one issue or merge request.
func (l *Local) Item(kind string, id int) (*LocalItem, error) {
	p, err := l.path(kind, id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("no %s", Target{Kind: kind, ID: id})
	}
	it, err := parseLocal(string(data))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p, err)
	}
	it.ID = id
	return it, nil
}

// Items lists the issues, or the merge requests, by number.
func (l *Local) Items(kind string) ([]*LocalItem, error) {
	d, err := l.dir()
	if err != nil {
		return nil, err
	}
	files, _ := filepath.Glob(filepath.Join(d, kindDir(kind), "*.md"))
	var out []*LocalItem
	for _, f := range files {
		id, err := strconv.Atoi(strings.TrimSuffix(filepath.Base(f), ".md"))
		if err != nil {
			continue
		}
		it, err := l.Item(kind, id)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// StateOf is an item's state as it stands: a merge request whose branch is
// gone is closed, one whose branch its base holds is merged — the person
// merges it with git.
func (l *Local) StateOf(it *LocalItem) string {
	if it.State != "open" || it.Branch == "" {
		return it.State
	}
	if exec.Command("git", "-C", l.Repo, "rev-parse", "-q", "--verify", "refs/heads/"+it.Branch).Run() != nil {
		return "closed"
	}
	if it.Base != "" && it.Base != it.Branch &&
		exec.Command("git", "-C", l.Repo, "merge-base", "--is-ancestor", "refs/heads/"+it.Branch, "refs/heads/"+it.Base).Run() == nil {
		return "merged"
	}
	return "open"
}

// InCI names the variable that says the run is in CI, or "" outside it.
func InCI() string {
	for _, v := range []string{"GITHUB_ACTIONS", "GITLAB_CI", "CI"} {
		if val := os.Getenv(v); val != "" && val != "false" && val != "0" {
			return v
		}
	}
	return ""
}

// writable refuses a write in CI: the job's clone is thrown away after it,
// and what the local forge holds with it. Reads stay allowed, so a project
// whose config says `local` for its laptops still runs in CI what writes
// nothing.
func (l *Local) writable() error {
	if v := InCI(); v != "" {
		return fmt.Errorf("the local forge in CI (%s is set): this clone is thrown away after the job, and what it holds with it; "+
			"pass --forge github, gitlab or cmd:<command> on this job, or --forge none to refuse the writes", v)
	}
	return nil
}

func (l *Local) save(kind string, it *LocalItem) error {
	if err := l.writable(); err != nil {
		return err
	}
	p, err := l.path(kind, it.ID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if it.Labels == nil {
		it.Labels = []string{}
	}
	head, err := yaml.Marshal(it)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("---\n")
	b.Write(head)
	b.WriteString("---\n\n")
	if body := strings.Trim(it.Body, "\n"); body != "" {
		b.WriteString(body + "\n")
	}
	for _, c := range it.Comments {
		b.WriteString("\n" + commentLine + "\n" + strings.Trim(c, "\n") + "\n")
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func parseLocal(s string) (*LocalItem, error) {
	rest, ok := strings.CutPrefix(s, "---\n")
	if !ok {
		return nil, fmt.Errorf("no front matter")
	}
	head, rest, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		head, ok = strings.CutSuffix(rest, "\n---")
		if !ok {
			return nil, fmt.Errorf("front matter not closed")
		}
		rest = ""
	}
	it := &LocalItem{}
	if err := yaml.Unmarshal([]byte(head), it); err != nil {
		return nil, err
	}
	if it.State == "" {
		it.State = "open"
	}
	parts := []string{""}
	for _, line := range strings.Split(rest, "\n") {
		if line == commentLine {
			parts = append(parts, "")
			continue
		}
		parts[len(parts)-1] += line + "\n"
	}
	it.Body = strings.Trim(parts[0], "\n")
	for _, c := range parts[1:] {
		it.Comments = append(it.Comments, strings.Trim(c, "\n"))
	}
	return it, nil
}

// change reads an item, lets change edit it, and saves it.
func (l *Local) change(t Target, change func(*LocalItem)) error {
	it, err := l.Item(t.Kind, t.ID)
	if err != nil {
		return err
	}
	change(it)
	return l.save(t.Kind, it)
}

// next is the number a new item of this kind takes.
func (l *Local) next(kind string) (int, error) {
	items, err := l.Items(kind)
	if err != nil {
		return 0, err
	}
	id := 1
	for _, it := range items {
		id = max(id, it.ID+1)
	}
	return id, nil
}

func (l *Local) Issue(id int) (*Issue, error) {
	it, err := l.Item("issue", id)
	if err != nil {
		return nil, err
	}
	return &Issue{ID: it.ID, Title: it.Title, Body: it.Body, Labels: it.Labels}, nil
}

func (l *Local) Comment(t Target, body, marker string) error {
	return l.change(t, func(it *LocalItem) {
		for _, c := range it.Comments {
			if strings.Contains(c, marker) {
				return
			}
		}
		it.Comments = append(it.Comments, body+"\n\n"+marker)
	})
}

func (l *Local) Sticky(t Target, body, marker string, create bool) error {
	return l.change(t, func(it *LocalItem) {
		for i, c := range it.Comments {
			if strings.Contains(c, marker) {
				it.Comments[i] = body + "\n\n" + marker
				return
			}
		}
		if create {
			it.Comments = append(it.Comments, body+"\n\n"+marker)
		}
	})
}

func (l *Local) Label(t Target, add, remove []string) error {
	return l.change(t, func(it *LocalItem) {
		for _, a := range add {
			if !slices.Contains(it.Labels, a) {
				it.Labels = append(it.Labels, a)
			}
		}
		it.Labels = slices.DeleteFunc(it.Labels, func(x string) bool { return slices.Contains(remove, x) })
	})
}

// openTitled is the open issue with this title, if any.
func (l *Local) openTitled(title string) (*LocalItem, error) {
	items, err := l.Items("issue")
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		if it.Title == title && it.State == "open" {
			return it, nil
		}
	}
	return nil, nil
}

func (l *Local) OpenIssue(title, body, marker string) (int, error) {
	it, err := l.openTitled(title)
	if err != nil {
		return 0, err
	}
	if it != nil {
		return it.ID, l.Comment(Target{Kind: "issue", ID: it.ID}, body, marker)
	}
	id, err := l.next("issue")
	if err != nil {
		return 0, err
	}
	return id, l.save("issue", &LocalItem{ID: id, Title: title, State: "open", Body: body + "\n\n" + marker})
}

func (l *Local) KeepIssue(title, body string, create bool) (int, error) {
	it, err := l.openTitled(title)
	if err != nil {
		return 0, err
	}
	if it != nil {
		it.Body = body
		return it.ID, l.save("issue", it)
	}
	if !create {
		return 0, nil
	}
	id, err := l.next("issue")
	if err != nil {
		return 0, err
	}
	return id, l.save("issue", &LocalItem{ID: id, Title: title, State: "open", Body: body})
}

func (l *Local) OpenMergeRequest(branch, base, title, body string) (int, error) {
	items, err := l.Items("merge-request")
	if err != nil {
		return 0, err
	}
	for _, it := range items {
		if it.Branch == branch && l.StateOf(it) == "open" {
			it.Title, it.Body = title, body
			return it.ID, l.save("merge-request", it)
		}
	}
	id, err := l.next("merge-request")
	if err != nil {
		return 0, err
	}
	return id, l.save("merge-request", &LocalItem{ID: id, Title: title, State: "open", Branch: branch, Base: base, Body: body})
}

func (l *Local) OpenMergeRequests(prefix string) ([]string, error) {
	items, err := l.Items("merge-request")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, it := range items {
		if strings.HasPrefix(it.Branch, prefix) && l.StateOf(it) == "open" {
			out = append(out, it.Branch)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (l *Local) MergeRequest(id int) (MergeRequest, error) {
	it, err := l.Item("merge-request", id)
	if err != nil {
		return MergeRequest{}, err
	}
	return MergeRequest{Branch: it.Branch, Base: it.Base, Here: true}, nil
}

func (l *Local) keepsBranches() {}
