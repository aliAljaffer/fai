#!/usr/bin/env python3
"""Collect release artifacts and merge per-platform update metadata."""

import argparse
import shutil
from pathlib import Path

import yaml


def merge_update_info(paths: list[Path]) -> dict:
    docs = [yaml.safe_load(path.read_text()) for path in paths]
    merged = docs[0]
    files = list(merged.get("files") or [])
    seen = {entry["url"] for entry in files}

    for doc in docs[1:]:
        for entry in doc.get("files") or []:
            if entry["url"] in seen:
                continue
            files.append(entry)
            seen.add(entry["url"])

    if files:
        merged["files"] = files
        zip_entry = next((entry for entry in files if entry["url"].endswith(".zip")), None)
        if zip_entry:
            merged["path"] = zip_entry["url"]
            merged["sha512"] = zip_entry["sha512"]

    return merged


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--artifacts", required=True, type=Path)
    parser.add_argument("--dist", required=True, type=Path)
    args = parser.parse_args()

    args.dist.mkdir(parents=True, exist_ok=True)
    metadata: dict[str, list[Path]] = {}

    for path in args.artifacts.rglob("*"):
        if not path.is_file():
            continue
        if path.name.startswith("latest") and path.suffix in {".yml", ".yaml"}:
            metadata.setdefault(path.name, []).append(path)
            continue
        target = args.dist / path.name
        if not target.exists():
            shutil.copy2(path, target)

    for name, paths in metadata.items():
        merged = merge_update_info(paths)
        (args.dist / name).write_text(yaml.safe_dump(merged, sort_keys=False))
        print(f"merged {len(paths)} {name} file(s)")


if __name__ == "__main__":
    main()
