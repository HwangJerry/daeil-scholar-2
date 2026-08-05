#!/usr/bin/env bash
# secret-scan.sh — Report credential findings without printing secret values.
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: secret-scan.sh [--repo PATH] [--current] [--history] [--artifacts]

Scans Git-tracked/untracked source, reachable Git history, and local
build/package artifacts. Findings contain only scope, path, key/category,
state, and count. Secret values are never printed.

Exit 0: no non-placeholder findings
Exit 1: one or more findings
Exit 2: invalid usage or an unavailable scan prerequisite
EOF
}

REPO=.
SCAN_CURRENT=false
SCAN_HISTORY=false
SCAN_ARTIFACTS=false

while (($#)); do
  case "$1" in
    --repo)
      [[ $# -ge 2 ]] || { usage >&2; exit 2; }
      REPO=$2
      shift 2
      ;;
    --current) SCAN_CURRENT=true; shift ;;
    --history) SCAN_HISTORY=true; shift ;;
    --artifacts) SCAN_ARTIFACTS=true; shift ;;
    --help|-h) usage; exit 0 ;;
    *) usage >&2; exit 2 ;;
  esac
done

if ! $SCAN_CURRENT && ! $SCAN_HISTORY && ! $SCAN_ARTIFACTS; then
  SCAN_CURRENT=true
  SCAN_HISTORY=true
  SCAN_ARTIFACTS=true
fi

command -v python3 >/dev/null 2>&1 || {
  printf 'ERROR: python3 is required\n' >&2
  exit 2
}

ROOT=$(git -C "$REPO" rev-parse --show-toplevel) || exit 2

python3 - "$ROOT" "$SCAN_CURRENT" "$SCAN_HISTORY" "$SCAN_ARTIFACTS" <<'PY'
import os
import re
import subprocess
import sys
import tarfile
import zipfile
from collections import Counter
from pathlib import Path

root = Path(sys.argv[1]).resolve()
scan_current = sys.argv[2] == "true"
scan_history = sys.argv[3] == "true"
scan_artifacts = sys.argv[4] == "true"

credential_keys = (
    "DB_PASSWORD",
    "JWT_SECRET",
    "KAKAO_CLIENT_SECRET",
    "SMTP_PASSWORD",
    "DEBUG_AGENT_SECRET",
    "VISIT_IP_SALT",
    "APNS_PRIVATE_KEY",
    "APNS_AUTH_KEY",
    "APNS_KEY_VALUE",
    "FCM_CREDENTIALS_JSON",
    "EASYPAY_SECRET",
    "APPLE_CLIENT_SECRET",
    "KEYSTORE_PASSWORD",
    "SIGNING_PASSWORD",
)
key_pattern = "|".join(re.escape(key) for key in credential_keys)
assignment = re.compile(
    rb"^[ \t]*(?:Environment[ \t]*=[ \t]*[\"']?)?[\"']?"
    + rb"(?P<key>" + key_pattern.encode() + rb")"
    + rb"[\"']?[ \t]*[:=][ \t]*(?P<value>[^\r\n]*)[ \t]*$",
    re.IGNORECASE | re.MULTILINE,
)
placeholder = re.compile(
    rb"^(?:\s*[\"']?)?(?:|change[-_ ]?me(?:[-_ ]?in[-_ ]?production)?|"
    rb"replace(?:[-_ ].*)?|example|placeholder|redacted|your[_-].*|<.*>|"
    rb"\$\{.*\}|\$\(.*\))(?:[\"']?\s*)$",
    re.IGNORECASE,
)
private_key = re.compile(rb"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----")
compact_jwt = re.compile(
    rb"eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}"
)

findings = Counter()


def safe_path(path):
    return re.sub(r"[\x00-\x20\x7f]+", "?", str(path))[:500]


def inspect(scope, path, data):
    if len(data) > 200_000_000:
        return
    for match in assignment.finditer(data):
        value = match.group("value").strip()
        if value.startswith(b">"):
            value = value[1:].strip()
        value = value.rstrip(b",").strip()
        if len(value) >= 2 and value[:1] == value[-1:] and value[:1] in (b"\"", b"'"):
            value = value[1:-1]
        else:
            value = value.rstrip(b"\"'")
        is_shell_case = (
            b"|" in value
            and value.rstrip().endswith(b")")
            and any((key.encode() + b":") in value for key in credential_keys)
        )
        if is_shell_case:
            continue
        if placeholder.fullmatch(value):
            continue
        key = match.group("key").decode("ascii", "replace").upper()
        findings[(scope, safe_path(path), key, "literal")] += 1
    if private_key.search(data):
        findings[(scope, safe_path(path), "PRIVATE_KEY_MATERIAL", "material")] += 1
    if compact_jwt.search(data):
        findings[(scope, safe_path(path), "JWT_COMPACT_TOKEN", "material")] += 1


def git_bytes(*args, input_data=None):
    return subprocess.run(
        ["git", "-C", str(root), *args],
        input=input_data,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=True,
    ).stdout


def scan_worktree():
    paths = git_bytes("ls-files", "-co", "--exclude-standard", "-z").split(b"\0")
    for raw_path in paths:
        if not raw_path:
            continue
        relative = raw_path.decode("utf-8", "surrogateescape")
        path = root / relative
        try:
            if path.is_file() and not path.is_symlink() and path.stat().st_size <= 20_000_000:
                inspect("current", relative, path.read_bytes())
        except (OSError, PermissionError):
            continue


def history_objects():
    objects = {}
    for line in git_bytes("rev-list", "--objects", "--all").splitlines():
        object_id, _, raw_path = line.partition(b" ")
        objects.setdefault(
            object_id.decode("ascii"),
            raw_path.decode("utf-8", "surrogateescape") if raw_path else "<unknown>",
        )
    if not objects:
        return
    object_ids = list(objects)
    checks = git_bytes(
        "cat-file",
        "--batch-check=%(objectname) %(objecttype) %(objectsize)",
        input_data=("\n".join(object_ids) + "\n").encode(),
    ).decode().splitlines()
    blobs = []
    for line in checks:
        object_id, object_type, object_size = line.split()
        if object_type == "blob" and int(object_size) <= 20_000_000:
            blobs.append((object_id, int(object_size)))
    process = subprocess.Popen(
        ["git", "-C", str(root), "cat-file", "--batch"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    assert process.stdin is not None and process.stdout is not None
    for object_id, expected_size in blobs:
        process.stdin.write((object_id + "\n").encode())
        process.stdin.flush()
        header = process.stdout.readline().decode().strip().split()
        if len(header) != 3 or header[1] != "blob":
            continue
        size = int(header[2])
        data = process.stdout.read(size)
        process.stdout.read(1)
        if size == expected_size:
            inspect("history", objects[object_id], data)
    process.stdin.close()
    process.wait()
    if process.returncode:
        raise RuntimeError("git cat-file --batch failed")


artifact_directories = {"dist", "build", "release", "artifacts", "test-results"}
archive_extensions = {".zip", ".jar", ".war", ".apk", ".aab", ".ipa"}
tar_extensions = {".tar", ".tgz", ".gz"}
excluded_directories = {".git", "node_modules", "vendor", "Pods", ".gradle", ".build"}


def inspect_archive(relative, path):
    try:
        if path.suffix.lower() in archive_extensions and zipfile.is_zipfile(path):
            with zipfile.ZipFile(path) as archive:
                for entry in archive.infolist():
                    if entry.is_dir() or entry.file_size > 20_000_000:
                        continue
                    inspect("artifact", f"{relative}!{entry.filename}", archive.read(entry))
        elif path.suffix.lower() in tar_extensions and tarfile.is_tarfile(path):
            with tarfile.open(path) as archive:
                for entry in archive.getmembers():
                    if not entry.isfile() or entry.size > 20_000_000:
                        continue
                    stream = archive.extractfile(entry)
                    if stream is not None:
                        inspect("artifact", f"{relative}!{entry.name}", stream.read())
    except (OSError, EOFError, tarfile.TarError, zipfile.BadZipFile):
        return


def scan_artifact_files():
    for directory, child_dirs, files in os.walk(root):
        directory_path = Path(directory)
        relative_dir = directory_path.relative_to(root)
        child_dirs[:] = [name for name in child_dirs if name not in excluded_directories]
        in_artifact_dir = bool(artifact_directories.intersection(relative_dir.parts))
        for filename in files:
            path = directory_path / filename
            relative = path.relative_to(root)
            suffix = path.suffix.lower()
            candidate = (
                in_artifact_dir
                or suffix in archive_extensions
                or suffix in tar_extensions
                or filename in {"server", "alumni-backend"}
            )
            if not candidate or path.is_symlink():
                continue
            try:
                size = path.stat().st_size
                if size <= 200_000_000:
                    inspect("artifact", relative, path.read_bytes())
                if size <= 500_000_000:
                    inspect_archive(relative, path)
            except (OSError, PermissionError):
                continue


try:
    if scan_current:
        scan_worktree()
    if scan_history:
        history_objects()
    if scan_artifacts:
        scan_artifact_files()
except (subprocess.CalledProcessError, RuntimeError) as error:
    print(f"ERROR: scan prerequisite failed ({type(error).__name__})", file=sys.stderr)
    sys.exit(2)

for (scope, path, key, state), count in sorted(findings.items()):
    print(
        f"FINDING scope={scope} path={path} key={key} "
        f"state={state} count={count}"
    )

sys.exit(1 if findings else 0)
PY
