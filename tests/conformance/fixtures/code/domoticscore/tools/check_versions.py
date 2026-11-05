#!/usr/bin/env python3
"""Version consistency checker for DomoticsCore.

Checks that:
- Each DomoticsCore-*/library.json version matches all metadata.version assignments
  in that component's C++ sources (include/ and src/).
- Optionally, the root library.json version matches the Git tag vX.Y.Z
  when running in CI on a tag or when --check-tag is requested.

Intended to be called from GitHub Actions and locally.
"""

import argparse
import json
import os
import re
import sys
from pathlib import Path
from typing import Iterable, Set, Tuple, Optional


REPO_ROOT = Path(__file__).resolve().parents[1]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Check DomoticsCore versions")
    parser.add_argument(
        "--repo-root",
        type=str,
        default=str(REPO_ROOT),
        help="Repository root (default: script parent)",
    )
    parser.add_argument(
        "--check-tag",
        action="store_true",
        help="Also verify that root library.json version matches the Git tag vX.Y.Z",
    )
    parser.add_argument(
        "--tag",
        type=str,
        default=None,
        help="Explicit tag name to use for --check-tag (e.g. v1.2.3). Overrides GITHUB_REF.",
    )
    parser.add_argument(
        "--verbose",
        action="store_true",
        help="Print per-component success messages",
    )
    return parser.parse_args()


def parse_semver(version: str) -> Tuple[int, int, int]:
    match = re.match(r"^(\d+)\.(\d+)\.(\d+)$", version.strip())
    if not match:
        raise ValueError(f"Invalid semantic version: {version!r}")
    return int(match.group(1)), int(match.group(2)), int(match.group(3))


def load_json(path: Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8"))


def iter_source_files(component_dir: Path) -> Iterable[Path]:
    """Yield relevant C++ source/header files under include/ and src/.

    Skips examples, tests, and .pio directories defensively.
    """

    exts = {".h", ".hpp", ".hh", ".hxx", ".cpp", ".cxx", ".cc", ".ino"}

    for sub in ("include", "src"):
        base = component_dir / sub
        if not base.is_dir():
            continue
        for path in base.rglob("*"):
            if not path.is_file():
                continue
            if path.suffix.lower() not in exts:
                continue
            lower_parts = {p.lower() for p in path.parts}
            if {".pio", "tests", "test", "examples"} & lower_parts:
                continue
            yield path


def find_metadata_versions(component_dir: Path) -> Set[str]:
    """Return all distinct metadata.version strings found in this component.

    Only scans include/ and src/ to avoid picking up example-specific versions.
    """

    versions: Set[str] = set()
    pattern = re.compile(r'metadata\.version\s*=\s*"([^"]+)"')

    for src in iter_source_files(component_dir):
        try:
            text = src.read_text(encoding="utf-8", errors="ignore")
        except OSError:
            continue
        for match in pattern.finditer(text):
            versions.add(match.group(1))
    return versions


def check_component_versions(root: Path, verbose: bool = False) -> Tuple[bool, list]:
    """Check all DomoticsCore-*/library.json files against metadata.version.

    Returns (ok, errors).
    """

    errors: list = []
    ok = True

    for lib_json in sorted(root.glob("DomoticsCore-*/library.json")):
        data = load_json(lib_json)
        name = data.get("name") or lib_json.parent.name
        json_version = data.get("version")

        if not json_version:
            errors.append(f"{name}: library.json has no 'version' field")
            ok = False
            continue

        try:
            parse_semver(json_version)
        except ValueError as e:
            errors.append(f"{name}: {e}")
            ok = False
            continue

        component_dir = lib_json.parent
        versions_in_code = find_metadata_versions(component_dir)

        if not versions_in_code:
            # Some packages (e.g., System or Core) are orchestration-only and
            # may not have a dedicated IComponent with metadata.version.
            # Treat this as a warning, not a hard failure.
            if verbose:
                print(f"[warn] {name}: no metadata.version found under include/src, skipping")
            continue

        # Enforce that all metadata.version values inside the component agree
        if len(versions_in_code) > 1:
            values = ", ".join(sorted(versions_in_code))
            errors.append(
                f"{name}: multiple metadata.version values in code ({values}) "
                f"but library.json is {json_version}"
            )
            ok = False
            continue

        (code_version,) = tuple(versions_in_code)
        if code_version != json_version:
            errors.append(
                f"{name}: version mismatch - library.json={json_version} "
                f"!= metadata.version={code_version}"
            )
            ok = False
        elif verbose:
            print(f"[ok] {name}: {json_version}")

    return ok, errors


def extract_tag_from_env_or_arg(args: argparse.Namespace) -> Optional[str]:
    """Return tag name from --tag or GITHUB_REF if it points to a tag."""

    if args.tag:
        return args.tag

    ref = os.getenv("GITHUB_REF", "")
    if ref.startswith("refs/tags/"):
        return ref.split("/", 2)[-1]
    return None


def check_root_against_tag(root: Path, args: argparse.Namespace, errors: list) -> bool:
    """Check that root library.json version matches vX.Y.Z tag.

    Appends any error messages to 'errors' and returns bool ok.
    """

    tag_name = extract_tag_from_env_or_arg(args)
    should_check = args.check_tag or tag_name is not None

    if not should_check:
        return True

    if not tag_name:
        errors.append("Tag check requested but no tag provided and GITHUB_REF is not a tag")
        return False

    m = re.match(r"^v?(\d+\.\d+\.\d+)$", tag_name)
    if not m:
        errors.append(f"Tag '{tag_name}' does not look like vX.Y.Z")
        return False

    tag_version = m.group(1)

    root_lib = root / "library.json"
    if not root_lib.is_file():
        errors.append("Root library.json not found for tag check")
        return False

    data = load_json(root_lib)
    root_version = data.get("version")
    if not root_version:
        errors.append("Root library.json is missing 'version' field")
        return False

    if root_version != tag_version:
        errors.append(
            f"Root version mismatch for tag {tag_name}: library.json={root_version} "
            f"!= tag={tag_version}"
        )
        return False

    return True


def check_root_vs_components(root: Path, errors: list) -> bool:
    """Ensure root library.json version is >= max DomoticsCore-* component version."""

    root_lib = root / "library.json"
    if not root_lib.is_file():
        errors.append("Root library.json not found for root vs components check")
        return False

    data = load_json(root_lib)
    root_version = data.get("version")
    if not root_version:
        errors.append("Root library.json is missing 'version' field")
        return False

    try:
        root_tuple = parse_semver(root_version)
    except ValueError as e:
        errors.append(f"Root library.json has invalid version: {e}")
        return False

    max_comp_version = None
    max_comp_name = None

    for lib_json in sorted(root.glob("DomoticsCore-*/library.json")):
        comp_data = load_json(lib_json)
        comp_name = comp_data.get("name") or lib_json.parent.name
        comp_version = comp_data.get("version")
        if not comp_version:
            continue
        try:
            comp_tuple = parse_semver(comp_version)
        except ValueError:
            # Component version problems are already reported elsewhere.
            continue

        if max_comp_version is None or comp_tuple > max_comp_version:
            max_comp_version = comp_tuple
            max_comp_name = comp_name

    if max_comp_version is None:
        # No components found; nothing to compare against.
        return True

    if root_tuple < max_comp_version:
        max_version_str = ".".join(str(x) for x in max_comp_version)
        errors.append(
            f"Root version {root_version} is lower than component {max_comp_name} version {max_version_str}"
        )
        return False

    return True


def check_properties_vs_root(root: Path, errors: list) -> bool:
    """Ensure library.properties (the Arduino registry) carries the root version."""
    props = root / "library.properties"
    if not props.is_file():
        return True
    m = re.search(r"^version=(.*)$", props.read_text(encoding="utf-8"), flags=re.M)
    root_version = load_json(root / "library.json").get("version")
    if not m or m.group(1).strip() != root_version:
        found = m.group(1).strip() if m else "none"
        errors.append(f"library.properties version {found} does not match root library.json {root_version}")
        return False
    return True


def main() -> None:
    args = parse_args()
    root = Path(args.repo_root).resolve()

    if not root.is_dir():
        print(f"Repository root does not exist: {root}", file=sys.stderr)
        sys.exit(1)

    ok_components, errors = check_component_versions(root, verbose=args.verbose)
    ok_tag = check_root_against_tag(root, args, errors)
    ok_root_vs_components = check_root_vs_components(root, errors)
    ok_properties = check_properties_vs_root(root, errors)

    if errors:
        print("Version inconsistencies detected:")
        for msg in errors:
            print(f" - {msg}")
        sys.exit(1)

    if ok_components and ok_tag and ok_root_vs_components and ok_properties:
        print("All version checks passed.")
    else:
        # Defensive fallback; in practice if there were problems, 'errors' would be non-empty.
        sys.exit(1)


if __name__ == "__main__":
    main()
