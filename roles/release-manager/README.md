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
- **Apply**: commits the changelog, then tags. `flow: direct` only; the
  merge-request flow needs a forge.

## Proposed: several packages in one repository

Worked from DomoticsCore, a library of twelve components released together.
Its rules, and how each would be met:

| DomoticsCore rule | How |
|---|---|
| Each component has its own semver, in its `library.json` | `packages`: each folder with a version file is a package, bumped from the commits that touched it |
| The version is repeated in the sources (`metadata.version = "X.Y.Z";`) and must match | `version-patterns`: every match is rewritten by the same patch |
| Changes to tests and examples do not move a component | `counts`: only these paths make a package move |
| The root version says what the release is *for* (2.6.1 was a patch while two components took a minor) | the root's bump is proposed as the highest component bump; **a person** may lower it, with a reason that goes into the notes. Never the AI: the version is not its decision |
| The tag is the root version | `tag: root` |
| Versions move once, when a series ships | already true: only the release manager bumps |
| `tools/check_versions.py` must pass | a check in the release gate, calling the project's own script |
| No release while an open MEDIUM is unarbitrated | a check in the release gate |
| Keep a Changelog format, with written notes in the entry | `changelog-format: keep-a-changelog`, notes written into the entry, not only on the tag |

```yaml
roles:
  release-manager:
    settings:
      packages:
        glob: "DomoticsCore-*"
        version-file: library.json        # the "version" key
        version-patterns: ['metadata.version = "{version}";']
        counts: ["include/**", "src/**"]
      root: {version-file: library.json, bump: proposed}   # a person confirms or lowers it
      tag: root
      changelog-format: keep-a-changelog
```
