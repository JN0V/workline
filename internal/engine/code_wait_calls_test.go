package engine

import (
	"os"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/devmark"
)

// postsSeen is the local forge, its issues said to be a person's, that notes
// each comment written anew on a merge request: a comment created, or
// posted anew (Renewer); an edit in place is none.
type postsSeen struct {
	*forge.Local
	author string
	fresh  []string // the bodies of the comments posted anew on merge requests
}

func (p *postsSeen) Issue(id int) (*forge.Issue, error) {
	is, err := p.Local.Issue(id)
	if is != nil {
		is.Author, is.Insider = p.author, true
	}
	return is, err
}

// wrote runs a write on t and notes the comment it added, if it did.
func (p *postsSeen) wrote(t forge.Target, write func() error) error {
	before, _ := p.Comments(t)
	err := write()
	after, _ := p.Comments(t)
	if t.Kind == "merge-request" && len(after) > len(before) {
		p.fresh = append(p.fresh, after[len(after)-1])
	}
	return err
}

func (p *postsSeen) Comment(t forge.Target, body, marker string) error {
	return p.wrote(t, func() error { return p.Local.Comment(t, body, marker) })
}

func (p *postsSeen) Sticky(t forge.Target, body, marker string, create bool) error {
	return p.wrote(t, func() error { return p.Local.Sticky(t, body, marker, create) })
}

// Renew posts anew: the local forge keeps one comment a marker, so the
// comment is edited and the post noted.
func (p *postsSeen) Renew(t forge.Target, body, marker string) error {
	if t.Kind == "merge-request" {
		p.fresh = append(p.fresh, body+"\n\n"+marker)
	}
	return p.Local.Sticky(t, body, marker, true)
}

// The yes awaited still calls
// Given a pull request of the developer's code for a feature waits for its reporter's yes,
// When the line weighs it, then weighs it again,
// Then exactly one new comment on the pull request mentions the reporter, the second weighing posting none.
func TestWeighWaitForAYesCallsThePersonOnce(t *testing.T) {
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "T")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "T")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@example.invalid")
	repo, run, write := gitRepo(t)
	write("a.txt", "a\n")
	run("add", "a.txt")
	run("commit", "-q", "-m", "first")
	run("checkout", "-q", "-b", "workline/developer/1")
	write("b.txt", "b\n")
	run("add", "b.txt")
	run("commit", "-q", "-m", "feat: b", "-m", OwnTrailer+": developer")
	run("checkout", "-q", "main")

	l := &forge.Local{Repo: repo}
	if _, err := l.Open("a feature", "A feature.", []string{"enhancement"}); err != nil {
		t.Fatal(err)
	}
	id, err := l.OpenMergeRequest("workline/developer/1", "main", "feat: b", "Closes #1", false)
	if err != nil {
		t.Fatal(err)
	}
	target := forge.Target{Kind: "merge-request", ID: id}
	record := "**The developer's record**\n\n" + devmark.Act{Act: "code", Issue: 1}.Line()
	if err := l.Comment(target, record, forge.Marker("sticky=developer/"+devmark.RecordKey)); err != nil {
		t.Fatal(err)
	}
	mr, err := l.MergeRequest(id)
	if err != nil {
		t.Fatal(err)
	}

	p := &postsSeen{Local: l, author: "zed"}
	n := &normal{local: true, app: forge.LocalApp, steers: func(string) string { return "" }}
	o := FollowOptions{Repo: repo, Base: "main", Merge: MergeNormal}
	hand := map[string]kept{}
	for i := 0; i < 2; i++ {
		if _, err := weigh(p, p, o, true, mr, &Result{}, hand, n); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			mentions := 0
			for _, c := range p.fresh {
				if strings.Contains(c, "@zed") {
					mentions++
				}
			}
			if mentions != 1 {
				t.Fatalf("after the first weighing, %d new comments on the pull request mention the reporter, want 1; new comments: %q", mentions, p.fresh)
			}
		}
	}
	mentions := 0
	for _, c := range p.fresh {
		if strings.Contains(c, "@zed") {
			mentions++
		}
	}
	if mentions != 1 {
		t.Errorf("after the second weighing, %d new comments mention the reporter, want exactly 1; new comments: %q", mentions, p.fresh)
	}
}
