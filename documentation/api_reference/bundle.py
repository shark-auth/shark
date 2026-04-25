#!/usr/bin/env python3
"""
Bundle SharkAuth OpenAPI sections into a single openapi.bundled.yaml.
Usage: python bundle.py
"""
import sys
import copy
import os
import shutil

try:
    import yaml
except ImportError:
    import subprocess
    subprocess.check_call([sys.executable, "-m", "pip", "install", "pyyaml", "-q"])
    import yaml

BASE_DIR = __file__.replace("\\", "/").rsplit("/", 1)[0]

MASTER = f"{BASE_DIR}/openapi.yaml"
SECTIONS = [
    f"{BASE_DIR}/sections/oauth.yaml",
    f"{BASE_DIR}/sections/auth.yaml",
    f"{BASE_DIR}/sections/platform.yaml",
    f"{BASE_DIR}/sections/admin.yaml",
    f"{BASE_DIR}/sections/system.yaml",
]
OUTPUT = f"{BASE_DIR}/openapi.bundled.yaml"


def deep_merge(target: dict, source: dict) -> dict:
    """Recursively merge source into target (target wins on key collision)."""
    for k, v in source.items():
        if k in target and isinstance(target[k], dict) and isinstance(v, dict):
            deep_merge(target[k], v)
        elif k not in target:
            target[k] = copy.deepcopy(v)
    return target


def load_yaml(path: str) -> dict:
    with open(path, "r", encoding="utf-8") as f:
        return yaml.safe_load(f) or {}


def main():
    master = load_yaml(MASTER)

    # Ensure top-level paths and components exist
    if "paths" not in master or master["paths"] is None:
        master["paths"] = {}
    if "components" not in master or master["components"] is None:
        master["components"] = {}

    total_paths = 0
    skipped = []

    for section_path in SECTIONS:
        section_name = section_path.split("/")[-1]
        try:
            section = load_yaml(section_path)
        except Exception as e:
            skipped.append(f"{section_name}: {e}")
            print(f"  SKIP {section_name}: {e}", file=sys.stderr)
            continue

        # Merge paths
        section_paths = section.get("paths") or {}
        for path_key, path_item in section_paths.items():
            if path_key in master["paths"]:
                print(f"  WARN: duplicate path {path_key} in {section_name}, skipping", file=sys.stderr)
            else:
                master["paths"][path_key] = copy.deepcopy(path_item)
                total_paths += 1

        # Merge components (master wins on collision)
        section_components = section.get("components") or {}
        deep_merge(master["components"], section_components)

        print(f"  Merged {section_name}: {len(section_paths)} paths")

    # Remove externalDocs references in tags that point to section files (cosmetic)
    # Keep them as-is — they are harmless for validators

    # Write output
    with open(OUTPUT, "w", encoding="utf-8") as f:
        yaml.dump(master, f, allow_unicode=True, sort_keys=False, default_flow_style=False, width=120)

    print(f"\nBundled {total_paths} paths -> {OUTPUT}")
    if skipped:
        print(f"Skipped sections (TODO): {skipped}")

    # Mirror to internal/api/docs/ so go:embed picks it up without ../ traversal
    repo_root = BASE_DIR.rsplit("/documentation/", 1)[0]
    embed_target = f"{repo_root}/internal/api/docs/openapi.bundled.yaml"
    os.makedirs(os.path.dirname(embed_target), exist_ok=True)
    shutil.copy2(OUTPUT, embed_target)
    print(f"Copied -> {embed_target}")

    return 0


if __name__ == "__main__":
    sys.exit(main())
