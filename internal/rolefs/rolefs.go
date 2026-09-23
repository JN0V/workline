// Package rolefs extracts the built-in roles to a cache folder, because their
// pre and post scripts must exist as executable files to be run.
package rolefs

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/JN0V/workline"
)

// Dir returns a folder holding the built-in roles, extracting them the first
// time. The folder name carries a digest of the content, so a new binary never
// runs an old copy.
func Dir() (string, error) {
	h := sha256.New()
	err := fs.WalkDir(workline.Roles, "roles", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := workline.Roles.ReadFile(p)
		if err != nil {
			return err
		}
		h.Write([]byte(p))
		h.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cache, "workline", "roles-"+hex.EncodeToString(h.Sum(nil))[:16])
	if _, err := os.Stat(filepath.Join(dir, ".complete")); err == nil {
		return dir, nil
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dir), ".extract-")
	if err != nil {
		if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
			return "", err
		}
		if tmp, err = os.MkdirTemp(filepath.Dir(dir), ".extract-"); err != nil {
			return "", err
		}
	}
	err = fs.WalkDir(workline.Roles, "roles", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel("roles", filepath.FromSlash(p))
		target := filepath.Join(tmp, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := workline.Roles.ReadFile(p)
		if err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if name := path.Base(p); name == "pre" || name == "post" {
			mode = 0o755
		}
		return os.WriteFile(target, data, mode)
	})
	if err != nil {
		os.RemoveAll(tmp)
		return "", err
	}
	if err := os.WriteFile(filepath.Join(tmp, ".complete"), nil, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, dir); err != nil {
		os.RemoveAll(tmp)
		if _, statErr := os.Stat(filepath.Join(dir, ".complete")); statErr == nil {
			return dir, nil // another process extracted it first
		}
		return "", err
	}
	return dir, nil
}
