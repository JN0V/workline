package linters

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// trackedFiles are the files git tracks, as it names them: neither a
// symbolic link nor a submodule, which hold no code of their own.
func trackedFiles(repo string) ([]string, error) {
	out, err := exec.Command("git", "-C", repo, "ls-files", "-z", "--stage").Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	var files []string
	for _, entry := range strings.Split(string(out), "\x00") {
		info, name, ok := strings.Cut(entry, "\t")
		if !ok {
			continue
		}
		if mode, _, _ := strings.Cut(info, " "); mode == "120000" || mode == "160000" {
			continue
		}
		files = append(files, name)
	}
	return files, nil
}

// vendoredDirs are folders of others' code, left out unless
// .gitattributes says `-linguist-vendored` (or `=false`) of a file.
var vendoredDirs = []string{"vendor", "node_modules", ".pio", "third_party"}

// notVendored are the files neither vendored nor generated: by
// .gitattributes (linguist-vendored, linguist-generated, as GitHub
// Linguist reads them), else by the folder they are in.
func notVendored(repo string, files []string) ([]string, error) {
	if len(files) == 0 {
		return nil, nil
	}
	cmd := exec.Command("git", "-C", repo, "check-attr", "-z", "--stdin", "linguist-vendored", "linguist-generated")
	cmd.Stdin = strings.NewReader(strings.Join(files, "\x00") + "\x00")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git check-attr: %v %s", err, strings.TrimSpace(stderr.String()))
	}
	// Each answer is three fields: the file, the attribute, its value.
	type said struct{ vendored, generated string }
	attrs := map[string]*said{}
	fields := strings.Split(string(out), "\x00")
	for i := 0; i+2 < len(fields); i += 3 {
		f, attr, value := fields[i], fields[i+1], fields[i+2]
		s := attrs[f]
		if s == nil {
			s = &said{}
			attrs[f] = s
		}
		if attr == "linguist-vendored" {
			s.vendored = value
		} else {
			s.generated = value
		}
	}
	set := func(v string) bool { return v == "set" || v == "true" }
	unset := func(v string) bool { return v == "unset" || v == "false" }
	var kept []string
	for _, f := range files {
		s := attrs[f]
		if s == nil {
			s = &said{}
		}
		if set(s.vendored) || set(s.generated) {
			continue
		}
		if !unset(s.vendored) && inVendoredDir(f) {
			continue
		}
		kept = append(kept, f)
	}
	return kept, nil
}

func inVendoredDir(f string) bool {
	parts := strings.Split(f, "/")
	for _, p := range parts[:len(parts)-1] {
		for _, d := range vendoredDirs {
			if p == d {
				return true
			}
		}
	}
	return false
}
