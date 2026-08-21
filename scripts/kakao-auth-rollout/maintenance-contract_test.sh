#!/usr/bin/env bash
# maintenance-contract_test.sh — Local contracts for the production write freeze.
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
GATE="$ROOT/deploy/_maintenance_gate.php"
PREPEND="$ROOT/deploy/_set_docroot.php"
HTTPD="$ROOT/deploy/httpd-alumni.conf"
PREPARE="$ROOT/scripts/kakao-auth-rollout/maintenance-prepare.sh"
ENTER="$ROOT/scripts/kakao-auth-rollout/maintenance-enter.sh"
RELEASE="$ROOT/scripts/kakao-auth-rollout/maintenance-release.sh"
SMOKE="$ROOT/scripts/kakao-auth-rollout/maintenance-smoke.sh"
DEPLOY="$ROOT/deploy.sh"
RECORDER="$ROOT/scripts/kakao-auth-rollout/maintenance-record-deployment.sh"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[[ -f "$GATE" ]] || fail "legacy maintenance gate is missing"
grep -Fq "require __DIR__ . '/_maintenance_gate.php';" "$PREPEND" ||
  fail "legacy maintenance gate is not loaded by auto_prepend"
[[ -x "$PREPARE" ]] || fail "maintenance prepare script is missing or not executable"
grep -Fq 'MAINTENANCE_PREPARE_APPROVED' "$PREPARE" || fail "maintenance prepare does not require explicit approval"
# The literal source contract must retain the production variable reference.
# shellcheck disable=SC2016
grep -Fq 'mktemp -d "$BACKUP_BASE/prepare.XXXXXX"' "$PREPARE" ||
  fail "maintenance prepare backup generation is not unique"
grep -Fq 'MAINTENANCE_PRESTAGE_STATE_FILE' "$PREPARE" ||
  fail "maintenance prepare does not persist its rollback handle"
grep -Fq 'state_write_failed' "$PREPARE" ||
  fail "maintenance prepare does not roll back when state persistence fails"
[[ -x "$ENTER" ]] || fail "maintenance enter script is missing or not executable"
[[ -x "$RELEASE" ]] || fail "maintenance release script is missing or not executable"
[[ -x "$SMOKE" ]] || fail "controlled smoke script is missing or not executable"
grep -Fq 'ALUMNI_MAINTENANCE_GATE=1' "$HTTPD" || fail "Apache maintenance gate marker is missing"
grep -Fq '/run/alumni/maintenance-release-bridge' "$HTTPD" || fail "Apache maintenance gate does not include the release bridge"
grep -Fq 'ALUMNI_MAINTENANCE_RELEASE_BRIDGE' "$GATE" || fail "legacy PHP maintenance gate does not include the release bridge"
grep -Fq 'MAINTENANCE_RELEASE_BRIDGE_PATH' "$RELEASE" || fail "release launcher has no bridge authority"
grep -Fq 'MAINTENANCE_RELEASE_PROOF_FILE' "$RELEASE" || fail "release launcher has no proof custody"
grep -Fq 'MAINTENANCE_RELEASE_APPROVAL_ATTEMPT_ID' "$RELEASE" || fail "release launcher is not attempt-bound"
PREPARED_LINE=$(grep -nF 'PREPARED_RELEASE_BRIDGE_INSTALL' "$RELEASE" | head -n 1 | cut -d: -f1)
DRAIN_LINE=$(grep -nF 'APPLICATION_DRAIN_CONTROL_CALL' "$RELEASE" | head -n 1 | cut -d: -f1)
DRAINED_LINE=$(grep -nF 'DRAINED_RELEASE_BRIDGE_PERSIST' "$RELEASE" | head -n 1 | cut -d: -f1)
OBSERVE_LINE=$(grep -nF 'CANONICAL_SENTINEL_TO_OBSERVATION' "$RELEASE" | head -n 1 | cut -d: -f1)
ARM_LINE=$(grep -nF '/internal/maintenance/arm-open' "$RELEASE" | head -n 1 | cut -d: -f1)
BRIDGE_REMOVE_LINE=$(grep -nF 'FINAL_RELEASE_BRIDGE_UNLINK' "$RELEASE" | head -n 1 | cut -d: -f1)
[[ -n $PREPARED_LINE && -n $DRAIN_LINE && -n $DRAINED_LINE && -n $OBSERVE_LINE && -n $ARM_LINE && -n $BRIDGE_REMOVE_LINE ]] ||
  fail "application-owned release phases are incomplete"
[[ $PREPARED_LINE -lt $DRAIN_LINE && $DRAIN_LINE -lt $DRAINED_LINE && $DRAINED_LINE -lt $OBSERVE_LINE &&
   $OBSERVE_LINE -lt $ARM_LINE && $ARM_LINE -lt $BRIDGE_REMOVE_LINE ]] ||
  fail "application-owned release phases are out of order"
grep -Fq 'api/auth/kakao/callback' "$HTTPD" || fail "Apache gate does not block the mutating OAuth callback"
grep -Fq 'api/(feed|disclosure)/[0-9]+' "$HTTPD" || fail "Apache gate does not block numeric mutating detail GET routes"
if grep -Fq 'api/(feed|disclosure)/[^/]+' "$HTTPD"; then
  fail "Apache gate overblocks non-detail read-only routes"
fi
grep -Eq 'RequestHeader[[:space:]]+unset[[:space:]]+X-Maintenance-Smoke-Proof' "$HTTPD" ||
  fail "Apache does not remove externally supplied smoke proof"
grep -Fq 'BACKEND_DEPLOY_EVIDENCE' "$RELEASE" || fail "release does not require deployment evidence"
grep -Fq 'BACKEND_SMOKE_EVIDENCE' "$RELEASE" || fail "release does not require smoke evidence"
grep -Fq 'X-Maintenance-Smoke-Proof' "$SMOKE" || fail "smoke script does not use the proof header"
grep -Fq 'exec 8<' "$SMOKE" || fail "smoke proof is not pinned to a stable file descriptor"
grep -Fq "curl --disable --config \"\$curl_config\"" "$SMOKE" ||
  fail "controlled smoke does not disable user curl configuration"
grep -Fq 'exec 9<' "$SMOKE" || fail "smoke body is not pinned to a stable file descriptor"
grep -Fq 'file_identity' "$SMOKE" || fail "smoke inputs do not verify descriptor/path identity"
grep -Fq 'MAINTENANCE_LEGACY_PROBE_URL' "$ENTER" || fail "enter does not black-box probe the legacy gate"
grep -Fq 'MAINTENANCE_HTTPD_RESTART_APPROVED' "$ENTER" || fail "enter does not require separate Apache drain approval"
grep -Fq 'MAINTENANCE_BLOCK_PROBE_URL' "$RELEASE" || fail "release does not verify no-proof writes are blocked"
grep -Fq 'PUSH_OBSERVATION_START_OUTPUT' "$RELEASE" || fail "release does not record observation start"

set +e
"$PREPARE" >/dev/null 2>&1
UNAPPROVED_PREPARE_STATUS=$?
set -e
[[ $UNAPPROVED_PREPARE_STATUS -ne 0 ]] || fail "unapproved maintenance prepare did not fail closed"

TMP=$(mktemp -d "${TMPDIR:-/tmp}/maintenance-contract.XXXXXX")
trap 'rm -rf "$TMP"' EXIT
SENTINEL="$TMP/maintenance"
BRIDGE="$TMP/maintenance-release-bridge"
RELEASE_PROOF_FILE="$TMP/maintenance-release-proof"
RELEASE_ATTEMPT=$(printf 'b%.0s' {1..64})
export MAINTENANCE_RELEASE_BRIDGE_PATH="$BRIDGE"
export MAINTENANCE_RELEASE_PROOF_FILE="$RELEASE_PROOF_FILE"
export MAINTENANCE_RELEASE_APPROVAL_ATTEMPT_ID="$RELEASE_ATTEMPT"
export MAINTENANCE_RELEASE_OWNER_UID
MAINTENANCE_RELEASE_OWNER_UID=$(id -u)

touch "$SENTINEL"
# $argv is evaluated by PHP, not by this shell.
# shellcheck disable=SC2016
BLOCKED_OUTPUT=$(
  ALUMNI_MAINTENANCE_SENTINEL="$SENTINEL" \
    ALUMNI_MAINTENANCE_RELEASE_BRIDGE="$BRIDGE" php -r \
    'include $argv[1]; echo "LEGACY_HANDLER_REACHED\n";' "$GATE"
)
[[ "$BLOCKED_OUTPUT" == *'MAINTENANCE_MODE'* ]] || fail "legacy gate did not return maintenance response"
[[ "$BLOCKED_OUTPUT" != *'LEGACY_HANDLER_REACHED'* ]] || fail "legacy handler executed during maintenance"

rm "$SENTINEL"
touch "$BRIDGE"
# $argv is evaluated by PHP, not by this shell.
# shellcheck disable=SC2016
BRIDGE_BLOCKED_OUTPUT=$(
  ALUMNI_MAINTENANCE_SENTINEL="$SENTINEL" \
    ALUMNI_MAINTENANCE_RELEASE_BRIDGE="$BRIDGE" php -r \
    'include $argv[1]; echo "LEGACY_HANDLER_REACHED\n";' "$GATE"
)
[[ "$BRIDGE_BLOCKED_OUTPUT" == *'MAINTENANCE_MODE'* ]] || fail "legacy bridge did not return maintenance response"
[[ "$BRIDGE_BLOCKED_OUTPUT" != *'LEGACY_HANDLER_REACHED'* ]] || fail "legacy handler executed while release bridge was active"
rm "$BRIDGE"
# $argv is evaluated by PHP, not by this shell.
# shellcheck disable=SC2016
OPEN_OUTPUT=$(
  ALUMNI_MAINTENANCE_SENTINEL="$SENTINEL" \
    ALUMNI_MAINTENANCE_RELEASE_BRIDGE="$BRIDGE" php -r \
    'include $argv[1]; echo "LEGACY_HANDLER_REACHED\n";' "$GATE"
)
[[ "$OPEN_OUTPUT" == *'LEGACY_HANDLER_REACHED'* ]] || fail "legacy handler stayed blocked after release"

printf 'PASS: legacy PHP maintenance gate\n'

NON_DIRECTORY_PARENT="$TMP/not-a-directory"
touch "$NON_DIRECTORY_PARENT"
# $argv is evaluated by PHP, not by this shell.
# shellcheck disable=SC2016
BROKEN_SENTINEL_OUTPUT=$(
  ALUMNI_MAINTENANCE_SENTINEL="$NON_DIRECTORY_PARENT/maintenance" \
    ALUMNI_MAINTENANCE_RELEASE_BRIDGE="$BRIDGE" php -r \
    'include $argv[1]; echo "LEGACY_HANDLER_REACHED\n";' "$GATE"
)
[[ "$BROKEN_SENTINEL_OUTPUT" == *'MAINTENANCE_MODE'* ]] || fail "legacy sentinel traversal error did not fail closed"
[[ "$BROKEN_SENTINEL_OUTPUT" != *'LEGACY_HANDLER_REACHED'* ]] || fail "legacy handler ran after sentinel traversal error"

mkdir -p "$TMP/bin"
cat > "$TMP/bin/httpd" <<'FAKE_HTTPD'
#!/usr/bin/env bash
exit 0
FAKE_HTTPD
cat > "$TMP/bin/systemctl" <<'FAKE_SYSTEMCTL'
#!/usr/bin/env bash
[[ -z ${FAKE_SYSTEMCTL_LOG:-} ]] || printf '%s\n' "$*" >> "$FAKE_SYSTEMCTL_LOG"
if [[ ${1:-} == stop ]]; then
  exit 0
fi
if [[ ${1:-} == start ]]; then
  exit 0
fi
if [[ ${1:-} == is-active ]]; then
  [[ ${*: -1} == httpd ]] && exit 0
  [[ ${FAKE_SERVICE_ACTIVE:-0} == 1 ]] && exit 0
  exit 3
fi
if [[ ${1:-} == show && ${2:-} == --property && ${3:-} == MainPID ]]; then
  [[ ${4:-} != --value ]] || exit 64
  if [[ ${FAKE_SENTINEL_MUST_BE_ABSENT:-0} == 1 && -e ${FAKE_SENTINEL_PATH:-} ]]; then
    exit 4
  fi
  if [[ -n ${FAKE_MAIN_PID_FILE:-} ]]; then
    printf 'MainPID=%s\n' "$(<"$FAKE_MAIN_PID_FILE")"
  else
    printf 'MainPID=%s\n' "${FAKE_MAIN_PID:-0}"
  fi
  exit 0
fi
if [[ ${1:-} == show && ${2:-} == --property && ${3:-} == ExecStart ]]; then
  [[ ${4:-} != --value ]] || exit 64
  printf 'ExecStart={ path=%s ; argv[]=%s ; }\n' "${FAKE_EXEC_START:-/unexpected}" "${FAKE_EXEC_START:-/unexpected}"
  exit 0
fi
exit 2
FAKE_SYSTEMCTL
cat > "$TMP/bin/curl" <<'FAKE_CURL'
#!/usr/bin/env bash
[[ ${1:-} == --disable ]] || exit 91
if [[ -n ${FAKE_CURL_COUNT_FILE:-} ]]; then
  count=0
  [[ ! -f $FAKE_CURL_COUNT_FILE ]] || count=$(<"$FAKE_CURL_COUNT_FILE")
  printf '%d\n' "$((count + 1))" > "$FAKE_CURL_COUNT_FILE"
fi
if [[ -n ${FAKE_CURL_SWITCH_MAIN_PID_FILE:-} ]]; then
  printf '%s\n' "${FAKE_CURL_SWITCH_MAIN_PID:-456}" > "$FAKE_CURL_SWITCH_MAIN_PID_FILE"
fi
if [[ " $* " == *'/internal/maintenance/drain '* ]]; then
  [[ ${FAKE_DRAIN_FAIL:-0} != 1 ]] || exit 22
  printf '{"state":"DRAINED"}'
elif [[ " $* " == *'/internal/maintenance/arm-open '* ]]; then
  [[ ${FAKE_ARM_FAIL:-0} != 1 ]] || exit 23
  printf '{"state":"ARMED"}'
elif [[ ${1:-} == --config || ( ${1:-} == --disable && ${2:-} == --config ) ]]; then
  printf '%s' "${FAKE_SMOKE_STATUS:-204}"
elif [[ " $* " == *' --write-out '* ]]; then
  printf '503'
fi
exit 0
FAKE_CURL
chmod +x "$TMP/bin/httpd" "$TMP/bin/systemctl" "$TMP/bin/curl"

HTTPD_FIXTURE="$TMP/alumni.conf"
printf '# ALUMNI_MAINTENANCE_GATE=1\n' > "$HTTPD_FIXTURE"
set +e
PATH="$TMP/bin:$PATH" MAINTENANCE_SENTINEL_PATH="$SENTINEL" HTTPD_CONFIG_PATH="$HTTPD_FIXTURE" \
  "$ENTER" >/dev/null 2>&1
UNAPPROVED_ENTER_STATUS=$?
set -e
[[ $UNAPPROVED_ENTER_STATUS -ne 0 && ! -e $SENTINEL ]] || fail "unapproved maintenance enter did not fail closed"

INJECTED_URL_SENTINEL="$TMP/injected-url-maintenance"
set +e
PATH="$TMP/bin:$PATH" MAINTENANCE_ENTER_APPROVED=1 MAINTENANCE_HTTPD_RESTART_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$INJECTED_URL_SENTINEL" HTTPD_CONFIG_PATH="$HTTPD_FIXTURE" \
  MAINTENANCE_LEGACY_PROBE_URL=$'https://daeilfoundation.or.kr/old/index.php\n--output=/tmp/injected' \
  "$ENTER" >/dev/null 2>&1
INJECTED_URL_STATUS=$?
set -e
[[ $INJECTED_URL_STATUS -ne 0 && ! -e $INJECTED_URL_SENTINEL ]] ||
  fail "injected legacy probe URL was accepted"

INJECTED_RESOLVE_SENTINEL="$TMP/injected-resolve-maintenance"
set +e
PATH="$TMP/bin:$PATH" MAINTENANCE_ENTER_APPROVED=1 MAINTENANCE_HTTPD_RESTART_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$INJECTED_RESOLVE_SENTINEL" HTTPD_CONFIG_PATH="$HTTPD_FIXTURE" \
  MAINTENANCE_LEGACY_PROBE_RESOLVE='-o/tmp/injected' \
  "$ENTER" >/dev/null 2>&1
INJECTED_RESOLVE_STATUS=$?
set -e
[[ $INJECTED_RESOLVE_STATUS -ne 0 && ! -e $INJECTED_RESOLVE_SENTINEL ]] ||
  fail "injected legacy probe resolve tuple was accepted"

DRAIN_ORDER_SENTINEL="$TMP/drain-order-maintenance"
set +e
PATH="$TMP/bin:$PATH" FAKE_SENTINEL_MUST_BE_ABSENT=1 FAKE_SENTINEL_PATH="$DRAIN_ORDER_SENTINEL" \
  MAINTENANCE_ENTER_APPROVED=1 MAINTENANCE_HTTPD_RESTART_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$DRAIN_ORDER_SENTINEL" HTTPD_CONFIG_PATH="$HTTPD_FIXTURE" \
  "$ENTER" >/dev/null 2>&1
DRAIN_ORDER_STATUS=$?
set -e
[[ $DRAIN_ORDER_STATUS -eq 0 && -f $DRAIN_ORDER_SENTINEL ]] ||
  fail "maintenance sentinel activated before backend/Apache process drain"
rm -f "$DRAIN_ORDER_SENTINEL"

LEAKED_PID_SENTINEL="$TMP/leaked-pid-maintenance"
set +e
PATH="$TMP/bin:$PATH" FAKE_MAIN_PID=42 MAINTENANCE_ENTER_APPROVED=1 \
  MAINTENANCE_HTTPD_RESTART_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$LEAKED_PID_SENTINEL" HTTPD_CONFIG_PATH="$HTTPD_FIXTURE" \
  "$ENTER" >/dev/null 2>&1
LEAKED_PID_STATUS=$?
set -e
[[ $LEAKED_PID_STATUS -ne 0 && ! -e $LEAKED_PID_SENTINEL ]] || fail "maintenance enter passed with a live backend PID"

SYSTEMCTL_LOG="$TMP/systemctl.log"
PATH="$TMP/bin:$PATH" FAKE_SYSTEMCTL_LOG="$SYSTEMCTL_LOG" MAINTENANCE_ENTER_APPROVED=1 \
  MAINTENANCE_HTTPD_RESTART_APPROVED=1 MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  HTTPD_CONFIG_PATH="$HTTPD_FIXTURE" "$ENTER" >/dev/null
[[ -f $SENTINEL ]] || fail "approved maintenance enter did not create sentinel"
SENTINEL_MODE=$(stat -f '%Lp' "$SENTINEL" 2>/dev/null || stat -c '%a' "$SENTINEL")
[[ $SENTINEL_MODE == 644 ]] || fail "maintenance enter created a non-canonical sentinel mode"
grep -Fq 'canonical_sentinel_requires_root' "$ENTER" ||
  fail "canonical maintenance enter does not require root ownership"
# The deploy source pattern intentionally matches a literal shell variable.
# shellcheck disable=SC2016
grep -Fq '$mode == 644' "$DEPLOY" || fail "deploy guard rejects the canonical sentinel mode"
# The recorder source pattern intentionally matches a literal shell variable.
# shellcheck disable=SC2016
grep -Fq 'file_mode "$SENTINEL") == 644' "$RECORDER" ||
  fail "deployment recorder rejects the canonical sentinel mode"
grep -Fq 'expected_owner_uid=0' "$RECORDER" ||
  fail "deployment recorder does not require root-owned canonical sentinel"
grep -Fxq 'stop httpd' "$SYSTEMCTL_LOG" || fail "maintenance enter did not stop Apache to drain PHP"
grep -Fxq 'show --property MainPID httpd' "$SYSTEMCTL_LOG" || fail "maintenance enter did not verify Apache process drain"
grep -Fxq 'start httpd' "$SYSTEMCTL_LOG" || fail "maintenance enter did not restart Apache gate"
GENERATION=$(sed -n 's/^generation=//p' "$SENTINEL")
[[ $GENERATION =~ ^[a-f0-9]{32}$ ]] || fail "maintenance enter did not create a generation ID"

DEPLOY_EVIDENCE="$TMP/deploy.pass"
SMOKE_EVIDENCE="$TMP/smoke.pass"
MIGRATION_EVIDENCE="$TMP/migration.pass"
OBSERVATION_OUTPUT="$TMP/observation-start.txt"
printf 'PASS\n' > "$DEPLOY_EVIDENCE"
printf 'FAIL\n' > "$SMOKE_EVIDENCE"
printf 'state=PASS\nkind=migration-postcheck\ngeneration=%s\nrange=036-039\npostcheck_metrics=15\nrecorded_at=2026-08-05T00:00:00Z\n' \
  "$GENERATION" > "$MIGRATION_EVIDENCE"
chmod 0600 "$MIGRATION_EVIDENCE"
set +e
PATH="$TMP/bin:$PATH" FAKE_SERVICE_ACTIVE=1 MAINTENANCE_RELEASE_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" BACKEND_DEPLOY_EVIDENCE="$DEPLOY_EVIDENCE" \
  BACKEND_SMOKE_EVIDENCE="$SMOKE_EVIDENCE" BACKEND_MIGRATION_EVIDENCE="$MIGRATION_EVIDENCE" \
  PUSH_OBSERVATION_START_OUTPUT="$OBSERVATION_OUTPUT" \
  "$RELEASE" >/dev/null 2>&1
FAILED_RELEASE_STATUS=$?
set -e
[[ $FAILED_RELEASE_STATUS -ne 0 && -f $SENTINEL ]] || fail "failed smoke evidence released maintenance"

PROOF_FILE="$TMP/smoke.proof"
SMOKE_OUTPUT="$TMP/smoke-output.pass"
printf 'fixture-proof\n' > "$PROOF_FILE"
chmod 0600 "$PROOF_FILE"
PROOF_LINK="$TMP/smoke-proof.link"
ln -s "$PROOF_FILE" "$PROOF_LINK"
set +e
PATH="$TMP/bin:$PATH" MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  MAINTENANCE_SMOKE_PROOF_FILE="$PROOF_LINK" MAINTENANCE_SMOKE_URL="http://127.0.0.1:8080/api/auth/login" \
  MAINTENANCE_SMOKE_EXPECT_STATUS=204 MAINTENANCE_SMOKE_EVIDENCE_OUTPUT="$SMOKE_OUTPUT" \
  "$SMOKE" >/dev/null 2>&1
PROOF_LINK_STATUS=$?
set -e
[[ $PROOF_LINK_STATUS -ne 0 && ! -e $SMOKE_OUTPUT ]] || fail "symlinked smoke proof was accepted"

BODY_FILE="$TMP/smoke-body.json"
BODY_LINK="$TMP/smoke-body.link"
printf '{}\n' > "$BODY_FILE"
chmod 0600 "$BODY_FILE"
ln -s "$BODY_FILE" "$BODY_LINK"
set +e
PATH="$TMP/bin:$PATH" MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  MAINTENANCE_SMOKE_PROOF_FILE="$PROOF_FILE" MAINTENANCE_SMOKE_BODY_FILE="$BODY_LINK" \
  MAINTENANCE_SMOKE_URL="http://127.0.0.1:8080/api/auth/login" MAINTENANCE_SMOKE_EXPECT_STATUS=204 \
  MAINTENANCE_SMOKE_EVIDENCE_OUTPUT="$SMOKE_OUTPUT" "$SMOKE" >/dev/null 2>&1
BODY_LINK_STATUS=$?
set -e
[[ $BODY_LINK_STATUS -ne 0 && ! -e $SMOKE_OUTPUT ]] || fail "symlinked smoke body was accepted"

set +e
PATH="$TMP/bin:$PATH" MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  MAINTENANCE_SMOKE_PROOF_FILE="$PROOF_FILE" \
  MAINTENANCE_SMOKE_URL=$'http://127.0.0.1:8080/api/auth/login\noutput = "/tmp/injected"' \
  MAINTENANCE_SMOKE_EXPECT_STATUS=204 MAINTENANCE_SMOKE_EVIDENCE_OUTPUT="$SMOKE_OUTPUT" \
  "$SMOKE" >/dev/null 2>&1
INJECTED_SMOKE_STATUS=$?
set -e
[[ $INJECTED_SMOKE_STATUS -ne 0 && ! -e $SMOKE_OUTPUT ]] || fail "smoke curl config accepted injected URL"

REJECTED_STATUS_OUTPUT="$TMP/rejected-status.pass"
set +e
PATH="$TMP/bin:$PATH" FAKE_SMOKE_STATUS=503 MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  MAINTENANCE_SMOKE_PROOF_FILE="$PROOF_FILE" MAINTENANCE_SMOKE_URL="http://127.0.0.1:8080/api/auth/login" \
  MAINTENANCE_SMOKE_EXPECT_STATUS=503 MAINTENANCE_SMOKE_EVIDENCE_OUTPUT="$REJECTED_STATUS_OUTPUT" \
  "$SMOKE" >/dev/null 2>&1
REJECTED_STATUS_RESULT=$?
set -e
[[ $REJECTED_STATUS_RESULT -ne 0 && ! -e $REJECTED_STATUS_OUTPUT ]] || fail "maintenance rejection status produced false smoke PASS"

PROXY_PORT_FILE="$TMP/proxy.port"
PROXY_LOG="$TMP/proxy.log"
python3 - "$PROXY_PORT_FILE" "$PROXY_LOG" <<'PY' &
import http.server
import socketserver
import sys

port_file, log_file = sys.argv[1:]

class Proxy(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        proof_state = "present" if self.headers.get("X-Maintenance-Smoke-Proof") else "absent"
        with open(log_file, "w", encoding="utf-8") as output:
            output.write("proof=" + proof_state + "\n")
        self.send_response(204)
        self.end_headers()

    def log_message(self, _format, *_args):
        return

with socketserver.TCPServer(("127.0.0.1", 0), Proxy) as server:
    with open(port_file, "w", encoding="utf-8") as output:
        output.write(str(server.server_address[1]))
    server.handle_request()
PY
PROXY_PID=$!
for _ in 1 2 3 4 5; do
  [[ -s $PROXY_PORT_FILE ]] && break
  sleep 0.1
done
[[ -s $PROXY_PORT_FILE ]] || fail "proxy fixture did not start"
PROXY_PORT=$(<"$PROXY_PORT_FILE")
PROXY_EVIDENCE="$TMP/proxy-smoke.pass"
set +e
http_proxy="http://127.0.0.1:$PROXY_PORT" HTTP_PROXY="http://127.0.0.1:$PROXY_PORT" \
  all_proxy="http://127.0.0.1:$PROXY_PORT" ALL_PROXY="http://127.0.0.1:$PROXY_PORT" \
  no_proxy='' NO_PROXY='' MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  MAINTENANCE_SMOKE_PROOF_FILE="$PROOF_FILE" MAINTENANCE_SMOKE_URL="http://127.0.0.1:9/api/auth/login" \
  MAINTENANCE_SMOKE_EXPECT_STATUS=204 MAINTENANCE_SMOKE_EVIDENCE_OUTPUT="$PROXY_EVIDENCE" \
  "$SMOKE" >/dev/null 2>&1
PROXIED_SMOKE_STATUS=$?
set -e
kill "$PROXY_PID" 2>/dev/null || true
wait "$PROXY_PID" 2>/dev/null || true
[[ $PROXIED_SMOKE_STATUS -ne 0 && ! -e $PROXY_EVIDENCE ]] || fail "proxy environment produced false smoke PASS"
[[ ! -s $PROXY_LOG ]] || fail "controlled-smoke proof reached proxy environment"

PATH="$TMP/bin:$PATH" MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  MAINTENANCE_SMOKE_PROOF_FILE="$PROOF_FILE" MAINTENANCE_SMOKE_URL="http://127.0.0.1:8080/api/auth/login" \
  MAINTENANCE_SMOKE_EXPECT_STATUS=204 MAINTENANCE_SMOKE_EVIDENCE_OUTPUT="$SMOKE_OUTPUT" "$SMOKE" >/dev/null
grep -Fxq 'state=PASS' "$SMOKE_OUTPUT" || fail "controlled smoke evidence state is missing"
grep -Fxq 'kind=smoke' "$SMOKE_OUTPUT" || fail "controlled smoke evidence kind is missing"
grep -Fxq "generation=$GENERATION" "$SMOKE_OUTPUT" || fail "controlled smoke evidence generation is missing"

set +e
PATH="$TMP/bin:$PATH" FAKE_SERVICE_ACTIVE=1 MAINTENANCE_RELEASE_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" BACKEND_DEPLOY_EVIDENCE="$DEPLOY_EVIDENCE" \
  BACKEND_SMOKE_EVIDENCE="$SMOKE_OUTPUT" BACKEND_MIGRATION_EVIDENCE="$MIGRATION_EVIDENCE" \
  PUSH_OBSERVATION_START_OUTPUT="$OBSERVATION_OUTPUT" \
  "$RELEASE" >/dev/null 2>&1
PLAIN_DEPLOY_RELEASE_STATUS=$?
set -e
[[ $PLAIN_DEPLOY_RELEASE_STATUS -ne 0 && -f $SENTINEL ]] || fail "plain deployment PASS released maintenance"

printf 'state=PASS\nkind=deployment\ngeneration=%s\n' "$GENERATION" > "$DEPLOY_EVIDENCE"
chmod 0600 "$DEPLOY_EVIDENCE"
set +e
PATH="$TMP/bin:$PATH" FAKE_SERVICE_ACTIVE=1 MAINTENANCE_RELEASE_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" BACKEND_DEPLOY_EVIDENCE="$DEPLOY_EVIDENCE" \
  BACKEND_SMOKE_EVIDENCE="$SMOKE_OUTPUT" BACKEND_MIGRATION_EVIDENCE="$MIGRATION_EVIDENCE" \
  PUSH_OBSERVATION_START_OUTPUT="$OBSERVATION_OUTPUT" "$RELEASE" >/dev/null 2>&1
INCOMPLETE_DEPLOY_RELEASE_STATUS=$?
set -e
[[ $INCOMPLETE_DEPLOY_RELEASE_STATUS -ne 0 && -f $SENTINEL ]] ||
  fail "incomplete deployment evidence released maintenance"

ROLLBACK_BINARY="$TMP/server.rollback"
printf 'rollback fixture\n' > "$ROLLBACK_BINARY"
chmod 0755 "$ROLLBACK_BINARY"
ROLLBACK_SHA=$(shasum -a 256 "$ROLLBACK_BINARY" | cut -d ' ' -f 1)
printf 'state=PASS\nkind=deployment\ngeneration=%s\nartifact_sha256=%064d\nrollback_path=%s\nrollback_sha256=%s\nmain_pid=123\nrecorded_at=2026-08-05T00:00:00Z\n' \
  "$GENERATION" 0 "$ROLLBACK_BINARY" "$ROLLBACK_SHA" > "$DEPLOY_EVIDENCE"
chmod 0600 "$DEPLOY_EVIDENCE"
rm -f "$MIGRATION_EVIDENCE"
set +e
PATH="$TMP/bin:$PATH" FAKE_SERVICE_ACTIVE=1 MAINTENANCE_RELEASE_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" BACKEND_DEPLOY_EVIDENCE="$DEPLOY_EVIDENCE" \
  BACKEND_SMOKE_EVIDENCE="$SMOKE_OUTPUT" PUSH_OBSERVATION_START_OUTPUT="$OBSERVATION_OUTPUT" \
  "$RELEASE" >/dev/null 2>&1
MISSING_MIGRATION_RELEASE_STATUS=$?
set -e
[[ $MISSING_MIGRATION_RELEASE_STATUS -ne 0 && -f $SENTINEL ]] ||
  fail "missing migration evidence released maintenance"
printf 'state=PASS\nkind=migration-postcheck\ngeneration=%s\nrange=036-039\npostcheck_metrics=15\nrecorded_at=2026-08-05T00:00:00Z\n' \
  "$GENERATION" > "$MIGRATION_EVIDENCE"
chmod 0600 "$MIGRATION_EVIDENCE"

APPROVED_BINARY="$TMP/server.approved"
UNAPPROVED_BINARY="$TMP/server.unapproved"
printf 'approved runtime fixture\n' > "$APPROVED_BINARY"
printf 'unapproved runtime fixture\n' > "$UNAPPROVED_BINARY"
chmod 0755 "$APPROVED_BINARY" "$UNAPPROVED_BINARY"
APPROVED_SHA=$(shasum -a 256 "$APPROVED_BINARY" | cut -d ' ' -f 1)
PROC_ROOT="$TMP/proc"
mkdir -p "$PROC_ROOT/123"
ln -s "$UNAPPROVED_BINARY" "$PROC_ROOT/123/exe"
printf 'state=PASS\nkind=deployment\ngeneration=%s\nartifact_sha256=%s\nrollback_path=%s\nrollback_sha256=%s\nmain_pid=123\nrecorded_at=2026-08-05T00:00:00Z\n' \
  "$GENERATION" "$APPROVED_SHA" "$ROLLBACK_BINARY" "$ROLLBACK_SHA" > "$DEPLOY_EVIDENCE"
chmod 0600 "$DEPLOY_EVIDENCE"

STALE_RUNTIME_OBSERVATION="$TMP/stale-runtime-observation.txt"
set +e
PATH="$TMP/bin:$PATH" FAKE_SERVICE_ACTIVE=1 FAKE_MAIN_PID=123 FAKE_EXEC_START="$APPROVED_BINARY" \
  BACKEND_BINARY_PATH="$APPROVED_BINARY" BACKEND_PROC_ROOT="$PROC_ROOT" \
  MAINTENANCE_RELEASE_APPROVED=1 MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  BACKEND_DEPLOY_EVIDENCE="$DEPLOY_EVIDENCE" BACKEND_SMOKE_EVIDENCE="$SMOKE_OUTPUT" \
  BACKEND_MIGRATION_EVIDENCE="$MIGRATION_EVIDENCE" \
  PUSH_OBSERVATION_START_OUTPUT="$STALE_RUNTIME_OBSERVATION" "$RELEASE" >/dev/null 2>&1
STALE_RUNTIME_RELEASE_STATUS=$?
set -e
[[ $STALE_RUNTIME_RELEASE_STATUS -ne 0 && -f $SENTINEL && ! -e $STALE_RUNTIME_OBSERVATION ]] ||
  fail "release accepted stale deployment evidence after runtime changed"
rm "$PROC_ROOT/123/exe"
ln -s "$APPROVED_BINARY" "$PROC_ROOT/123/exe"

mkdir -p "$PROC_ROOT/456"
ln -s "$APPROVED_BINARY" "$PROC_ROOT/456/exe"
RUNTIME_PID_FILE="$TMP/runtime-main-pid"
printf '123\n' > "$RUNTIME_PID_FILE"
RUNTIME_SWITCH_OBSERVATION="$TMP/runtime-switch-observation.txt"
set +e
PATH="$TMP/bin:$PATH" FAKE_SERVICE_ACTIVE=1 FAKE_MAIN_PID_FILE="$RUNTIME_PID_FILE" \
  FAKE_CURL_SWITCH_MAIN_PID_FILE="$RUNTIME_PID_FILE" FAKE_CURL_SWITCH_MAIN_PID=456 \
  FAKE_EXEC_START="$APPROVED_BINARY" BACKEND_BINARY_PATH="$APPROVED_BINARY" \
  BACKEND_PROC_ROOT="$PROC_ROOT" MAINTENANCE_RELEASE_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" BACKEND_DEPLOY_EVIDENCE="$DEPLOY_EVIDENCE" \
  BACKEND_SMOKE_EVIDENCE="$SMOKE_OUTPUT" BACKEND_MIGRATION_EVIDENCE="$MIGRATION_EVIDENCE" \
  PUSH_OBSERVATION_START_OUTPUT="$RUNTIME_SWITCH_OBSERVATION" "$RELEASE" >/dev/null 2>&1
RUNTIME_SWITCH_RELEASE_STATUS=$?
set -e
[[ $RUNTIME_SWITCH_RELEASE_STATUS -ne 0 && -f $SENTINEL && ! -e $RUNTIME_SWITCH_OBSERVATION ]] ||
  fail "release accepted a runtime switch during pre-release probes"

ARBITRARY_OBSERVATION="$TMP/arbitrary-observation.txt"
ARBITRARY_GENERATION=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
printf 'state=STARTED\ngeneration=%s\nstarted_at=2026-08-05T00:00:00Z\n' \
  "$ARBITRARY_GENERATION" > "$ARBITRARY_OBSERVATION"
chmod 0600 "$ARBITRARY_OBSERVATION"
set +e
PATH="$TMP/bin:$PATH" FAKE_SERVICE_ACTIVE=1 MAINTENANCE_RELEASE_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$TMP/absent-sentinel" \
  PUSH_OBSERVATION_START_OUTPUT="$ARBITRARY_OBSERVATION" "$RELEASE" >/dev/null 2>&1
ARBITRARY_OBSERVATION_STATUS=$?
set -e
[[ $ARBITRARY_OBSERVATION_STATUS -ne 0 ]] ||
  fail "arbitrary observation record produced idempotent release PASS"

cat > "$TMP/bin/mv" <<'FAKE_MV'
#!/usr/bin/env bash
set -euo pipefail
destination=${*: -1}
/bin/mv "$@"
if [[ -n ${FAKE_RELEASE_RACE_SENTINEL:-} && -n ${FAKE_RELEASE_RACE_OUTPUT:-} &&
      $destination == "$FAKE_RELEASE_RACE_SENTINEL" ]]; then
  mkdir "$FAKE_RELEASE_RACE_OUTPUT"
fi
FAKE_MV
chmod +x "$TMP/bin/mv"
RACE_OBSERVATION_OUTPUT="$TMP/race-observation"
set +e
PATH="$TMP/bin:$PATH" FAKE_SERVICE_ACTIVE=1 FAKE_MAIN_PID=123 FAKE_EXEC_START="$APPROVED_BINARY" \
  BACKEND_BINARY_PATH="$APPROVED_BINARY" BACKEND_PROC_ROOT="$PROC_ROOT" MAINTENANCE_RELEASE_APPROVED=1 \
  FAKE_RELEASE_RACE_SENTINEL="$SENTINEL" FAKE_RELEASE_RACE_OUTPUT="$RACE_OBSERVATION_OUTPUT" \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" BACKEND_DEPLOY_EVIDENCE="$DEPLOY_EVIDENCE" \
  BACKEND_SMOKE_EVIDENCE="$SMOKE_OUTPUT" BACKEND_MIGRATION_EVIDENCE="$MIGRATION_EVIDENCE" \
  PUSH_OBSERVATION_START_OUTPUT="$RACE_OBSERVATION_OUTPUT" "$RELEASE" >/dev/null 2>&1
RACED_RELEASE_STATUS=$?
set -e
[[ $RACED_RELEASE_STATUS -ne 0 && -f $SENTINEL ]] ||
  fail "destination directory race consumed the maintenance sentinel"
rm -rf "$RACE_OBSERVATION_OUTPUT"
printf 'state=active\ngeneration=%s\n' "$GENERATION" > "$SENTINEL"
touch "$DEPLOY_EVIDENCE" "$SMOKE_OUTPUT" "$MIGRATION_EVIDENCE"

RELEASE_CURL_COUNT="$TMP/release-curl.count"
printf '%064d\n\n' 0 > "$RELEASE_PROOF_FILE"
chmod 0600 "$RELEASE_PROOF_FILE"
MALFORMED_PROOF_OBSERVATION="$TMP/malformed-proof-observation.txt"
set +e
PATH="$TMP/bin:$PATH" FAKE_SERVICE_ACTIVE=1 FAKE_MAIN_PID=123 FAKE_EXEC_START="$APPROVED_BINARY" \
  BACKEND_BINARY_PATH="$APPROVED_BINARY" BACKEND_PROC_ROOT="$PROC_ROOT" MAINTENANCE_RELEASE_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" BACKEND_DEPLOY_EVIDENCE="$DEPLOY_EVIDENCE" \
  BACKEND_SMOKE_EVIDENCE="$SMOKE_OUTPUT" BACKEND_MIGRATION_EVIDENCE="$MIGRATION_EVIDENCE" \
  PUSH_OBSERVATION_START_OUTPUT="$MALFORMED_PROOF_OBSERVATION" "$RELEASE" >/dev/null 2>&1
MALFORMED_PROOF_STATUS=$?
set -e
if [[ $MALFORMED_PROOF_STATUS -eq 0 ]]; then
  rm -f "$MALFORMED_PROOF_OBSERVATION" "$BRIDGE"
  printf 'state=active\ngeneration=%s\n' "$GENERATION" > "$SENTINEL"
  chmod 0644 "$SENTINEL"
  touch "$DEPLOY_EVIDENCE" "$SMOKE_OUTPUT" "$MIGRATION_EVIDENCE"
  fail "release accepted noncanonical proof bytes"
fi
[[ -f $SENTINEL && ! -e $BRIDGE && ! -e $MALFORMED_PROOF_OBSERVATION ]] ||
  fail "malformed proof changed release state"

printf '%064d\n' 0 > "$RELEASE_PROOF_FILE"
FAIL_DRAIN_OBSERVATION="$TMP/fail-drain-observation.txt"
set +e
PATH="$TMP/bin:$PATH" FAKE_DRAIN_FAIL=1 FAKE_SERVICE_ACTIVE=1 FAKE_MAIN_PID=123 FAKE_EXEC_START="$APPROVED_BINARY" \
  BACKEND_BINARY_PATH="$APPROVED_BINARY" BACKEND_PROC_ROOT="$PROC_ROOT" MAINTENANCE_RELEASE_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" BACKEND_DEPLOY_EVIDENCE="$DEPLOY_EVIDENCE" \
  BACKEND_SMOKE_EVIDENCE="$SMOKE_OUTPUT" BACKEND_MIGRATION_EVIDENCE="$MIGRATION_EVIDENCE" \
  PUSH_OBSERVATION_START_OUTPUT="$FAIL_DRAIN_OBSERVATION" "$RELEASE" >/dev/null 2>&1
FAIL_DRAIN_STATUS=$?
set -e
[[ $FAIL_DRAIN_STATUS -ne 0 && -f $SENTINEL && ! -e $FAIL_DRAIN_OBSERVATION ]] ||
  fail "drain failure crossed the canonical sentinel boundary"
grep -Fxq 'state=prepared' "$BRIDGE" || fail "drain failure did not retain prepared bridge"
grep -Fxq "generation=$GENERATION" "$BRIDGE" || fail "drain failure bridge lost generation"
grep -Fxq "approval_attempt_id=$RELEASE_ATTEMPT" "$BRIDGE" || fail "drain failure bridge lost attempt"
rm -f "$BRIDGE"

FAIL_ARM_OBSERVATION="$TMP/fail-arm-observation.txt"
set +e
PATH="$TMP/bin:$PATH" FAKE_ARM_FAIL=1 FAKE_SERVICE_ACTIVE=1 FAKE_MAIN_PID=123 FAKE_EXEC_START="$APPROVED_BINARY" \
  BACKEND_BINARY_PATH="$APPROVED_BINARY" BACKEND_PROC_ROOT="$PROC_ROOT" MAINTENANCE_RELEASE_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" BACKEND_DEPLOY_EVIDENCE="$DEPLOY_EVIDENCE" \
  BACKEND_SMOKE_EVIDENCE="$SMOKE_OUTPUT" BACKEND_MIGRATION_EVIDENCE="$MIGRATION_EVIDENCE" \
  PUSH_OBSERVATION_START_OUTPUT="$FAIL_ARM_OBSERVATION" "$RELEASE" >/dev/null 2>&1
FAIL_ARM_STATUS=$?
set -e
[[ $FAIL_ARM_STATUS -ne 0 && ! -e $SENTINEL && -f $FAIL_ARM_OBSERVATION ]] ||
  fail "arm failure did not preserve the post-sentinel observation boundary"
grep -Fxq 'state=drained' "$BRIDGE" || fail "arm failure did not retain drained bridge"
# $argv is evaluated by PHP, not this shell.
# shellcheck disable=SC2016
BRIDGE_FAIL_CLOSED_OUTPUT=$(
  ALUMNI_MAINTENANCE_SENTINEL="$SENTINEL" \
    ALUMNI_MAINTENANCE_RELEASE_BRIDGE="$BRIDGE" php -r \
    'include $argv[1]; echo "LEGACY_HANDLER_REACHED\n";' "$GATE"
)
[[ $BRIDGE_FAIL_CLOSED_OUTPUT == *'MAINTENANCE_MODE'* ]] || fail "arm failure reopened legacy writers"
rm -f "$BRIDGE" "$FAIL_ARM_OBSERVATION"
printf 'state=active\ngeneration=%s\n' "$GENERATION" > "$SENTINEL"
chmod 0644 "$SENTINEL"
touch "$DEPLOY_EVIDENCE" "$SMOKE_OUTPUT" "$MIGRATION_EVIDENCE"

PATH="$TMP/bin:$PATH" FAKE_SERVICE_ACTIVE=1 FAKE_MAIN_PID=123 FAKE_EXEC_START="$APPROVED_BINARY" \
  BACKEND_BINARY_PATH="$APPROVED_BINARY" BACKEND_PROC_ROOT="$PROC_ROOT" \
  FAKE_CURL_COUNT_FILE="$RELEASE_CURL_COUNT" MAINTENANCE_RELEASE_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" BACKEND_DEPLOY_EVIDENCE="$DEPLOY_EVIDENCE" \
  BACKEND_SMOKE_EVIDENCE="$SMOKE_OUTPUT" BACKEND_MIGRATION_EVIDENCE="$MIGRATION_EVIDENCE" \
  PUSH_OBSERVATION_START_OUTPUT="$OBSERVATION_OUTPUT" \
  "$RELEASE" >/dev/null
[[ ! -e $SENTINEL ]] || fail "dual-PASS release did not remove sentinel"
[[ -s $OBSERVATION_OUTPUT ]] || fail "release did not record observation start"
[[ $(<"$RELEASE_CURL_COUNT") == 4 ]] || fail "release control call count is not exact"
grep -Fxq 'state=STARTED' "$OBSERVATION_OUTPUT" || fail "observation record state is missing"
grep -Fxq "generation=$GENERATION" "$OBSERVATION_OUTPUT" || fail "observation record generation is missing"

PATH="$TMP/bin:$PATH" FAKE_SERVICE_ACTIVE=1 FAKE_CURL_COUNT_FILE="$RELEASE_CURL_COUNT" MAINTENANCE_RELEASE_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" BACKEND_DEPLOY_EVIDENCE="$DEPLOY_EVIDENCE" \
  BACKEND_SMOKE_EVIDENCE="$SMOKE_OUTPUT" BACKEND_MIGRATION_EVIDENCE="$MIGRATION_EVIDENCE" \
  PUSH_OBSERVATION_START_OUTPUT="$OBSERVATION_OUTPUT" "$RELEASE" >/dev/null
[[ $(<"$RELEASE_CURL_COUNT") == 4 ]] || fail "idempotent release retry repeated pre-open probes"

printf 'PASS: maintenance enter/smoke/release gates\n'
