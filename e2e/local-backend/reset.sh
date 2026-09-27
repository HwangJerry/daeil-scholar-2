#!/usr/bin/env bash
set -euo pipefail
source "$(dirname -- "${BASH_SOURCE[0]}")/common.sh"
preflight "$@"
"$LOCAL_STACK_DIR/down.sh"
exec "$LOCAL_STACK_DIR/up.sh"
