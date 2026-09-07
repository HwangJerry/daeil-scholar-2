#!/bin/bash
# deploy.sh — Prepare immutable artifacts locally; deploy only a reviewed bundle.
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
exec python3 "${SCRIPT_DIR}/deploy/release.py" "$@"
