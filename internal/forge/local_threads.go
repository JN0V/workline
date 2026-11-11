package forge

// Conversations on a line, kept in the clone (#1265): the local forge has
// no changes page, so each comment on a line is a comment of its own on
// the merge request, its place hidden in it, and a reply in one is a
// comment naming it. The tester answers an objection there, as on a
// forge's review thread (ADR-0055).

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// localLine hides a comment's place on a line: its key, its file, its
// line. localReply names the conversation a comment answers, by the
// place of its first comment among the merge request's comments.
var (
	localLine  = regexp.MustCompile(`<!-- workline:on-line key=(\S*) path=(\S+) line=(\d+) -->`)
	localReply = regexp.MustCompile(`<!-- workline:reply-to=(\d+) -->`)
)

// localWhere opens the line a person reads under a comment on a line.
const localWhere = "\n\n_On `"

// Review keeps each comment on a line as a comment of its own: one whose
// key a comment holds already is open, never posted again.
func (l *Local) Review(id int, head, body string, comments []LineComment) (Reviewed, error) {
	var r Reviewed
	err := l.change(Target{Kind: "merge-request", ID: id}, func(it *LocalItem) {
		for _, c := range comments {
			p := Placed{LineComment: c, State: LinePosted}
			if c.Key != "" && slices.ContainsFunc(it.Comments, func(s string) bool {
				return strings.Contains(s, "on-line key="+c.Key+" ") && !strings.Contains(s, localResolved)
			}) {
				p.State = LineOpen
			} else {
				// Its key's mark as on a forge (withKey), read back by the
				// reviewer resolving or weighing its own, after the place.
				key := ""
				if c.Key != "" {
					key = "\n" + lineMarker(c.Key)
				}
				it.Comments = append(it.Comments, fmt.Sprintf("%s%s%s:%d`_\n\n<!-- workline:on-line key=%s path=%s line=%d -->%s",
					strings.TrimSpace(c.Body), localWhere, c.Path, c.Line, c.Key, c.Path, c.Line, key))
			}
			r.Placed = append(r.Placed, p)
		}
	})
	return r, err
}

// Threads are the merge request's comments on a line, each with the
// replies naming it, oldest first. A thread's id is its first comment's
// place, counting from 1.
func (l *Local) Threads(id int) ([]Thread, error) {
	notes, err := l.Notes(Target{Kind: "merge-request", ID: id})
	if err != nil {
		return nil, err
	}
	var out []Thread
	at := map[string]int{}
	for i, n := range notes {
		if m := localLine.FindStringSubmatch(n.Body); m != nil {
			line, _ := strconv.Atoi(m[3])
			first := n
			body, rest, _ := strings.Cut(n.Body, localWhere)
			first.Body = body
			if k := lineKey.FindString(rest); k != "" {
				first.Body += "\n\n" + k // its key, as a forge's comment holds it
			}
			tid := strconv.Itoa(i + 1)
			at[tid] = len(out)
			// Where a person reads it: the merge request's file in the clone.
			url := fmt.Sprintf(".git/workline/%s/%d.md", kindDir("merge-request"), id)
			out = append(out, Thread{ID: tid, Path: m[2], Line: line, URL: url, Resolved: strings.Contains(n.Body, localResolved), Comments: []Note{first}})
			continue
		}
		if m := localReply.FindStringSubmatch(n.Body); m != nil {
			if k, ok := at[m[1]]; ok {
				reply := n
				reply.Body = strings.TrimSpace(localReply.ReplaceAllString(n.Body, ""))
				out[k].Comments = append(out[k].Comments, reply)
			}
		}
	}
	return out, nil
}

// localResolved marks the first comment of a conversation resolved.
const localResolved = "<!-- workline:resolved -->"

// Resolve marks the threads resolved; those it does not hold, the rest
// resolved still, are said in the error.
func (l *Local) Resolve(id int, threads []string) error {
	var missing []string
	err := l.change(Target{Kind: "merge-request", ID: id}, func(it *LocalItem) {
		for _, thread := range threads {
			i, err := strconv.Atoi(thread)
			if err != nil || i < 1 || i > len(it.Comments) || !localLine.MatchString(it.Comments[i-1]) {
				missing = append(missing, thread)
				continue
			}
			if !strings.Contains(it.Comments[i-1], localResolved) {
				it.Comments[i-1] += "\n" + localResolved
			}
		}
	})
	if err != nil {
		return err
	}
	return noThreads(id, missing)
}

// Reply adds a comment answering the thread.
func (l *Local) Reply(id int, thread, body string) error {
	threads, err := l.Threads(id)
	if err != nil {
		return err
	}
	for _, t := range threads {
		if t.ID == thread {
			return l.change(Target{Kind: "merge-request", ID: id}, func(it *LocalItem) {
				it.Comments = append(it.Comments, strings.TrimSpace(body)+"\n\n<!-- workline:reply-to="+thread+" -->")
			})
		}
	}
	return fmt.Errorf("merge request %d holds no thread %s", id, thread)
}
