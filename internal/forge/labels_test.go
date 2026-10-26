package forge

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// GitLab lists the project's labels across pages, and creates a label
// ranked only when given a rank; one there already is left (ADR-0040).
func TestGitLabLabels(t *testing.T) {
	var posted []url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/group%2Fproj")
		switch {
		case r.Method == "GET" && p == "/labels" && r.URL.Query().Get("page") != "2":
			w.Header().Set("X-Next-Page", "2")
			json.NewEncoder(w).Encode([]map[string]any{{"name": "bug"}, {"name": "P0"}})
		case r.Method == "GET" && p == "/labels":
			json.NewEncoder(w).Encode([]map[string]any{{"name": "P1"}})
		case r.Method == "GET" && p == "/labels/priority::2":
			json.NewEncoder(w).Encode(map[string]any{"name": "priority::2"})
		case r.Method == "POST" && p == "/labels":
			body, _ := io.ReadAll(r.Body)
			form, _ := url.ParseQuery(string(body))
			posted = append(posted, form)
			w.Write([]byte(`{}`))
		default:
			http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
		}
	}))
	defer srv.Close()
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "group/proj")
	t.Setenv("GITLAB_TOKEN", "secret")
	g := &gitlab{repo: t.TempDir()}

	names, err := ProjectLabels(g, nil)
	if err != nil || !slices.Equal(names, []string{"bug", "P0", "P1"}) {
		t.Fatalf("labels = %v, %v; want both pages", names, err)
	}
	for _, c := range []struct {
		name string
		rank int
	}{{"priority::1", 1}, {"priority::2", 2}, {"effort::low", 0}} {
		if err := EnsureRankedLabel(g, c.name, "b60205", "about", c.rank); err != nil {
			t.Fatal(err)
		}
	}
	if len(posted) != 2 || posted[0].Get("name") != "priority::1" || posted[0].Get("priority") != "1" || posted[0].Get("color") != "#b60205" {
		t.Fatalf("posted = %v; want priority::1 ranked, priority::2 left", posted)
	}
	if posted[1].Get("name") != "effort::low" || posted[1].Has("priority") {
		t.Errorf("a rank of 0 is no rank: %v", posted[1])
	}
}

// A forge that does not list its labels answers those on the open issues
// given; one that does not rank creates the label unranked.
func TestLabelsFallBack(t *testing.T) {
	open := []Issue{{ID: 1, Labels: []string{"bug", "P1"}}, {ID: 2, Labels: []string{"P1", "P3"}}}
	names, err := ProjectLabels(struct{ Forge }{}, open)
	if err != nil || !slices.Equal(names, []string{"bug", "P1", "P3"}) {
		t.Errorf("labels = %v, %v", names, err)
	}
	f := &Fake{Path: filepath.Join(t.TempDir(), "forge.json")}
	if err := os.WriteFile(f.Path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureRankedLabel(f, "priority/urgent", "b60205", "about", 1); err != nil {
		t.Fatal(err)
	}
	s, err := f.load()
	if err != nil || len(s.Labels) != 1 || s.Labels[0].Title != "priority/urgent" || s.Labels[0].Rank != 0 {
		t.Errorf("labels = %+v, %v; want one, unranked on a forge whose labels are not scoped", s.Labels, err)
	}
}
