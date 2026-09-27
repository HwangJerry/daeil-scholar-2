#!/usr/bin/env python3
"""Stage a source-only context for the unchanged backend Dockerfile."""

from pathlib import Path
import shutil
import subprocess

ROOT = Path(__file__).resolve().parent
REPO = ROOT.parent.parent
DEST = ROOT / ".build/backend"
if DEST.exists():
    shutil.rmtree(DEST)
files = subprocess.check_output(["git", "ls-files", "-z", "backend/"], cwd=REPO).decode().split("\0")
for name in filter(None, files):
    relative = Path(name).relative_to("backend")
    source = REPO / name
    is_go = relative.suffix == ".go" and not relative.name.endswith("_test.go")
    is_template = relative.parent == Path("internal/handler/templates") and relative.suffix == ".html"
    if not (is_go or is_template or str(relative) in {"Dockerfile", "go.mod", "go.sum"}):
        continue
    if source.is_symlink():
        raise SystemExit("Refusing symlink in build context: " + name)
    destination = DEST / relative
    destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(source, destination)
