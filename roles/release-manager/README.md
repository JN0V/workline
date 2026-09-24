---
sources: [internal/builtin/releasemanager, roles/release-manager/role.yaml]
checked: d30d22a
---
# Release manager

What runs, for humans. The AI never reads this file.

## Today

- **Prepare** (`pre`, no AI): reads the conventional commits since the last tag,
  computes the next version (semver: breaking → major, feat → minor, fix/perf →
  patch; or calver), writes the changelog section, and proposes the changelog
  patch and the release as fallback proposals.
- **Propose** (AI): writes the release notes only.
- **Judge** (`post`, no AI): refuses a version the commits did not give, and any
  changelog other than the generated one.
- **Apply**: commits the changelog, then tags, and publishes the release on
  the forge when there is one. `flow: direct`, the default, is the only flow
  built; `merge-request` is refused until it is.

## Several packages in one repository

Worked from DomoticsCore, a library of twelve components released together,
and tried on a copy of it: the release it computes is the one its own rules
give, and its `tools/check_versions.py --check-tag` passes on the result.

| DomoticsCore rule | How |
|---|---|
| Each component has its own semver, in its `library.json` | `packages`: each folder matching `glob` with a `version-file` is a package, bumped from the commits that touched it |
| The version is repeated in the sources (`metadata.version = "X.Y.Z";`) and must match | `version-patterns`: every match under the package is rewritten with the same release |
| Changes to tests and examples do not move a component | `counts`: only these paths, relative to the package, make it move |
| The root version says what the release is *for* | the root moves by the largest package move, unless **a person** passes `--input root-bump=patch --input root-reason="…"`; the reason goes into the changelog. Never the AI |
| The tag is the root version | the root's `version-file` must match the last tag, or the release stops |
| `tools/check_versions.py` must pass | a gate calling the project's own script |
| Keep a Changelog format | `changelog-format: keep-a-changelog` |

```yaml
roles:
  release-manager:
    settings:
      changelog-format: keep-a-changelog
      version-files: ["library.json", "DomoticsCore-*/library.json",
                      "DomoticsCore-*/include/**", "DomoticsCore-*/src/**"]   # what the role may write
      packages:
        glob: "DomoticsCore-*"
        version-file: library.json
        version-patterns: ['metadata.version = "{version}";']
        counts: ["include/**", "src/**"]
```

Not yet: build metadata (`+<build>`), the merge-request flow, dependency ranges
between packages (a package whose dependency takes a major).
