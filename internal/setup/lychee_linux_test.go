package setup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// No brew, no cargo:
// Given a Linux machine with neither,
// When I run `workline setup`,
// Then lychee is installed from a pinned release binary.
//
// The release server is a fake one, named by WORKLINE_LYCHEE_RELEASE_URL;
// the test checks the pinned download, its checksum and the install path.
func TestNoBrewNoCargo(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("a Linux scenario")
	}
	work := t.TempDir()
	bin := filepath.Join(work, "workline")
	if out, err := exec.Command("go", "build", "-o", bin, "../../cmd/workline").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	// The release: a tarball holding a lychee, and its checksum.
	lychee := "#!/bin/sh\necho lychee 0.0.0\n"
	var tgz bytes.Buffer
	gz := gzip.NewWriter(&tgz)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "lychee", Mode: 0o755, Size: int64(len(lychee))})
	_, _ = tw.Write([]byte(lychee))
	tw.Close()
	gz.Close()
	sum := sha256.Sum256(tgz.Bytes())

	var mu sync.Mutex
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		p := strings.ToLower(r.URL.Path)
		if strings.Contains(p, "sha256") || strings.Contains(p, "checksum") {
			w.Write([]byte(hex.EncodeToString(sum[:]) + "  " + filepath.Base(r.URL.Path) + "\n"))
			return
		}
		if strings.HasSuffix(p, ".tar.gz") {
			w.Write(tgz.Bytes())
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	// A PATH with the basic tools, and neither brew nor cargo.
	tools := filepath.Join(work, "path")
	if err := os.Mkdir(tools, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sh", "curl", "tar", "gzip", "sha256sum", "git", "mkdir", "mv", "cp", "chmod", "cat", "rm", "uname", "install", "mktemp", "sed", "grep", "tr", "cut", "awk", "head", "dirname", "basename", "wget"} {
		if p, err := exec.LookPath(name); err == nil {
			_ = os.Symlink(p, filepath.Join(tools, name))
		}
	}
	home := filepath.Join(work, "home")
	proj := filepath.Join(work, "project")
	for _, d := range []string{home, proj} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(bin, "setup")
	cmd.Dir = proj
	cmd.Env = []string{"PATH=" + tools, "HOME=" + home, "WORKLINE_LYCHEE_RELEASE_URL=" + srv.URL}
	out, _ := cmd.CombinedOutput()

	mu.Lock()
	got := strings.Join(paths, "\n")
	mu.Unlock()
	if !regexp.MustCompile(`(?m)^.*lychee-v\d+\.\d+\.\d+.*\.tar\.gz$`).MatchString(got) {
		t.Errorf("no pinned release binary downloaded; requests:\n%s\nsetup said:\n%s", got, out)
	}
	if !regexp.MustCompile(`(?mi)sha256|checksum`).MatchString(got) {
		t.Errorf("the checksum not fetched; requests:\n%s", got)
	}
	installed := filepath.Join(home, ".local", "bin", "lychee")
	b, err := os.ReadFile(installed)
	if err != nil || string(b) != lychee {
		t.Fatalf("lychee not installed at %s: %v\nsetup said:\n%s", installed, err, out)
	}
	if st, _ := os.Stat(installed); st == nil || st.Mode()&0o111 == 0 {
		t.Errorf("the installed lychee is not executable")
	}
}
