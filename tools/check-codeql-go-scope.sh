#!/usr/bin/env bash
set -euo pipefail

# CodeQL's Go file-coverage report is based on files that enter extraction.
# Keep the build scope explicit so a new build tag, platform file, or module
# cannot silently fall outside the traced CodeQL build.
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

GOOS=linux GOARCH=amd64 go list -json -tags cloudsigma_guard_audit ./... > "$tmpdir/sdk.json"
(cd tools && GOOS=linux GOARCH=amd64 go list -json ./tools.go) > "$tmpdir/tools.json"

python3 - "$tmpdir/sdk.json" "$tmpdir/tools.json" <<'PY'
import json
import pathlib
import subprocess
import sys

root = pathlib.Path.cwd()
found = set()

def add_packages(path):
    text = pathlib.Path(path).read_text()
    decoder = json.JSONDecoder()
    offset = 0
    while offset < len(text):
        while offset < len(text) and text[offset].isspace():
            offset += 1
        if offset == len(text):
            break
        package, end = decoder.raw_decode(text, offset)
        offset = end
        package_dir = pathlib.Path(package["Dir"])
        for field in ("GoFiles", "CgoFiles", "TestGoFiles", "XTestGoFiles"):
            for name in package.get(field, []):
                found.add((package_dir / name).resolve())

add_packages(sys.argv[1])
add_packages(sys.argv[2])
tracked = {
    (root / name).resolve()
    for name in subprocess.check_output(
        ["git", "ls-files", "--", "*.go"], text=True
    ).splitlines()
}
missing = sorted(tracked - found)
if missing:
    print("Go files outside the CodeQL build scope:", file=sys.stderr)
    for path in missing:
        print(f"  {path.relative_to(root)}", file=sys.stderr)
    raise SystemExit(1)
print(f"CodeQL Go build scope covers all {len(tracked)} tracked Go files.")
PY
