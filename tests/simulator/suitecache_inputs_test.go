package simulator

import (
	"os"
	"path/filepath"
	"testing"
)

// Changed source, a new build: a changed component header or Unity file
// is a new key too, so the suite is built again, not run stale.
func TestChangedHeaderOrUnityNewBuild(t *testing.T) {
	for name, dir := range map[string]string{"header": "DomoticsCore-Core/include", "unity": "tools/unity"} {
		t.Run(name, func(t *testing.T) {
			repo, shim, cache, log := suiteWorld(t)
			if n, _, err := suiteRun(t, repo, shim, cache, log); err != nil || n != 1 {
				t.Fatalf("first run: %d g++, %v", n, err)
			}
			var file string
			_ = filepath.Walk(filepath.Join(repo, dir), func(p string, i os.FileInfo, err error) error {
				if err == nil && !i.IsDir() && file == "" {
					file = p
				}
				return nil
			})
			if file == "" {
				t.Fatalf("no file under %s", dir)
			}
			f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0)
			if err != nil {
				t.Fatal(err)
			}
			_, err = f.WriteString("\n// changed\n")
			f.Close()
			if err != nil {
				t.Fatal(err)
			}
			if n, _, err := suiteRun(t, repo, shim, cache, log); err != nil || n != 1 {
				t.Errorf("after a changed %s: %d g++, want 1 (%v)", name, n, err)
			}
		})
	}
}
