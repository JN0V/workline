// Package ci ships the CI templates inside the binary, so `workline doctor`
// can compare a copied workflow with the template of its own release.
package ci

import "embed"

// Templates holds defaults/ci/github and defaults/ci/gitlab.
//
//go:embed github/*.yml gitlab/*.gitlab-ci.yml
var Templates embed.FS
