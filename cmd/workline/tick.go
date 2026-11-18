package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/JN0V/workline/internal/clock"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/tick"
)

// tickCmd prints the line's jobs due now, each with why and the name its
// run is started under (ADR-0069 §2): read from the forge and the routing,
// no agent, nothing written. What did not list a job, a read that failed
// or a run going, is said on stderr; it exits 0 all the same.
func tickCmd(args []string) int {
	fs := newFlags("tick")
	repo := fs.String("repo", ".", "repository")
	forgeSpec := fs.String("forge", "", "github, gitlab, local, cmd:<command>, fake:<file>, none (default: the project's `forge` setting)")
	asJSON := fs.Bool("json", false, `print the jobs due as JSON: [{"job", "why", "key", …}], [] when none`)
	issues := fs.Bool("issues", false, "list the reconcile's issues, review rounds and answers due, each its own job, in the place of the reconcile")
	at := fs.String("now", "", "the time the tick runs at, RFC 3339 (tests; default: now)")
	if code, done := parseOnly(fs, args); done {
		return code
	}
	root, err := gitRoot(*repo)
	if err != nil {
		return notInRepo()
	}
	now := clock.Now()
	if *at != "" {
		if now, err = time.Parse(time.RFC3339, *at); err != nil {
			return misuse(fs, fmt.Sprintf("--now %q: an RFC 3339 time, as 2026-11-15T02:20:00Z", *at))
		}
		fixed := now
		clock.Now = func() time.Time { return fixed }
	}
	pc, err := role.LoadProjectConfig(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 64
	}
	cfg, err := routing.Load(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "workline:", err)
		return 64
	}
	if *forgeSpec == "" {
		*forgeSpec = pc.Forge
	}
	// Each listing read again sends its last ETag: unchanged, it costs
	// no quota (#1762).
	forge.ReadConditionally(true)
	var f forge.Forge
	if *forgeSpec != "" && *forgeSpec != "none" {
		if f, err = forge.Open(*forgeSpec, root); err != nil {
			fmt.Fprintln(os.Stderr, "workline:", err)
			return 64
		}
	}
	r := tick.Due(tick.Options{Repo: root, Config: cfg, Forge: f, ForgeSpec: *forgeSpec, Now: now, Issues: *issues})
	for _, s := range r.Unread {
		fmt.Fprintln(os.Stderr, "workline: tick: not read:", s)
	}
	for _, s := range r.Said {
		fmt.Fprintln(os.Stderr, "workline: tick:", s)
	}
	if r.Calls > 0 {
		fmt.Fprintf(os.Stderr, "workline: tick: %d GitHub calls, %d of them cost quota (%d answered 304 Not Modified)\n", r.Calls, r.Calls-r.Free, r.Free)
	}
	if *asJSON {
		items := r.Items
		if items == nil {
			items = []tick.Item{}
		}
		out, _ := json.Marshal(items)
		fmt.Println(string(out))
		return 0
	}
	if len(r.Items) == 0 {
		fmt.Println("nothing due")
	}
	for _, it := range r.Items {
		target := ""
		if it.Target != "" {
			target = " " + it.Target
		}
		fmt.Printf("%s%s: %s\n", it.Job, target, it.Why)
	}
	return 0
}
