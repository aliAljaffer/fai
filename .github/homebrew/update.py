#!/usr/bin/env python3
"""Rewrite the fai cask in the Homebrew tap from release artifacts."""

import argparse
import hashlib
import pathlib


def sha256(path: pathlib.Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as fh:
        for chunk in iter(lambda: fh.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--tap", required=True, type=pathlib.Path)
    parser.add_argument("--dist", required=True, type=pathlib.Path)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--repo", default="alialjaffer/fai")
    args = parser.parse_args()

    version = args.tag.removeprefix("v")
    arm = args.dist / f"fai-{version}-arm64-mac.zip"
    intel = args.dist / f"fai-{version}-x64-mac.zip"
    for artifact in (arm, intel):
        if not artifact.is_file():
            raise SystemExit(f"missing release artifact: {artifact}")

    base = f"https://github.com/{args.repo}/releases/download/{args.tag}"
    cask = f'''cask "fai" do
  version "{version}"
  sha256 arm:   "{sha256(arm)}",
         intel: "{sha256(intel)}"

  on_arm do
    url "{base}/fai-#{{version}}-arm64-mac.zip"
  end
  on_intel do
    url "{base}/fai-#{{version}}-x64-mac.zip"
  end

  name "fai"
  desc "Self-hosted task tracker, Kanban, and notes"
  homepage "https://github.com/{args.repo}"

  depends_on :macos

  app "fai.app"
end
'''

    cask_dir = args.tap / "Casks"
    cask_dir.mkdir(parents=True, exist_ok=True)
    (cask_dir / "fai.rb").write_text(cask)
    print(f"wrote {cask_dir / 'fai.rb'} for {version}")


if __name__ == "__main__":
    main()
