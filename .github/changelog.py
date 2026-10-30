#!/usr/bin/env python3
"""Build release notes for a tag from the commits since the previous tag."""

import argparse
import re
import subprocess
from pathlib import Path

REPO_URL = "https://github.com/aliAljaffer/fai"
DEPENDENCY = re.compile(r"^bump\b|dependabot", re.IGNORECASE)


def git(*args: str) -> str:
    return subprocess.run(
        ("git", *args), check=True, capture_output=True, text=True
    ).stdout.strip()


def render(tag: str, previous: str, subjects: list[str]) -> str:
    changes = [s for s in subjects if not DEPENDENCY.search(s)]
    dependencies = [s for s in subjects if DEPENDENCY.search(s)]

    lines = [f"## {tag}", ""]
    if changes:
        lines += ["### Changes", ""] + [f"- {s}" for s in changes] + [""]
    if dependencies:
        lines += ["### Dependencies", ""] + [f"- {s}" for s in dependencies] + [""]
    if previous:
        lines += [f"**Full changelog:** {REPO_URL}/compare/{previous}...{tag}", ""]
    return "\n".join(lines)


def prepend(changelog: Path, entry: str, tag: str) -> bool:
    text = changelog.read_text() if changelog.exists() else "# Changelog\n\n"
    if f"## {tag}\n" in text:
        return False
    head, _, rest = text.partition("\n")
    changelog.write_text(f"{head}\n\n{entry}\n{rest.lstrip()}")
    return True


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--tag", required=True)
    parser.add_argument("--notes", required=True, type=Path)
    parser.add_argument("--changelog", type=Path)
    args = parser.parse_args()

    try:
        previous = git("describe", "--tags", "--abbrev=0", f"{args.tag}^")
        subjects = git(
            "log", "--no-merges", "--pretty=%s", f"{previous}..{args.tag}"
        ).splitlines()
    except subprocess.CalledProcessError:
        print(f"no previous tag before {args.tag}, writing an empty entry")
        previous = ""
        subjects = []

    entry = render(args.tag, previous, subjects)
    args.notes.write_text(entry)
    print(entry)

    if args.changelog and prepend(args.changelog, entry, args.tag):
        print(f"added {args.tag} to {args.changelog}")


if __name__ == "__main__":
    main()
