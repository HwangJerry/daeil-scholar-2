#!/usr/bin/env bash
set -euo pipefail
source "$(dirname -- "${BASH_SOURCE[0]}")/common.sh"
preflight "$@"
python3 "$LOCAL_STACK_DIR/stage-build.py"
if ! compose up --build --detach --wait --wait-timeout 240; then
    echo 'Local stack did not become healthy. Inspect with the README logs command; down.sh cleans up.' >&2
    exit 1
fi
curl -q --silent --show-error --fail --noproxy '*' --proto '=http' \
    --connect-timeout 5 --max-time 15 --retry 5 --retry-connrefused --retry-delay 1 \
    --output /dev/null http://127.0.0.1:18080/api/health
cat <<'INFO'
Local E2E stack is healthy.
iOS simulator:    http://localhost:18080
Android emulator: http://10.0.2.2:18080
Host API:         http://127.0.0.1:18080
MariaDB:          127.0.0.1:13306 (alumni_e2e / e2e / ts09-synthetic-db)
Approved: e2e_member  / Synthetic-password-09!
Approved: e2e_friend  / Synthetic-password-09!
Pending:  e2e_pending / Synthetic-password-09!
Signup:   01000000002 / SMS code 654321 (synthetic; no SMS sent)
INFO
