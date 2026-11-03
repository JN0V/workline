package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runDoctor(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	var code int
	var out string
	errOut, _ := captureStderr(t, func() int {
		out, code = captureStdout(t, func() int { return dispatch(append([]string{"doctor"}, args...)) })
		return code
	})
	return out, errOut, code
}

func TestDoctorMissingFolderJSON(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	out, _, code := runDoctor(t, "--repo", missing, "--json")
	if code != 64 {
		t.Errorf("exit %d, want 64", code)
	}
	var r struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("not JSON: %v: %q", err, out)
	}
	if r.Status == "" || r.Status == "pass" {
		t.Errorf("status %q, want a failing one", r.Status)
	}
}

func TestDoctorRepoIsAFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{f, filepath.Join(f, "sub")} {
		out, errOut, code := runDoctor(t, "--repo", p)
		if code != 64 {
			t.Errorf("%s: exit %d, want 64", p, code)
		}
		if strings.Contains(out, "everything workline uses is set up") {
			t.Errorf("%s: says all is set up: %q", p, out)
		}
		if !strings.Contains(errOut, "workline:") {
			t.Errorf("%s: no error line, got %q", p, errOut)
		}
	}
}

func TestDoctorOutsideRepositoryChecksMachineOnly(t *testing.T) {
	t.Chdir(t.TempDir())
	out, _, _ := runDoctor(t)
	if strings.Contains(out, "This repository") {
		t.Errorf("checked a repository outside one: %q", out)
	}
	if !strings.Contains(out, "This machine") {
		t.Errorf("no machine checks: %q", out)
	}
}

func TestDoctorExistingFolderIsNotMissing(t *testing.T) {
	out, errOut, code := runDoctor(t, "--repo", t.TempDir())
	if code == 64 || strings.Contains(errOut+out, "does not exist") {
		t.Errorf("existing folder reported missing: exit %d, %q %q", code, out, errOut)
	}
}
