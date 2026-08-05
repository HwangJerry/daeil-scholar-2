#!/usr/bin/env bash
# maintenance-record-deployment-contract_test.sh — Verify deployment evidence provenance.
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
RECORDER="$ROOT/scripts/kakao-auth-rollout/maintenance-record-deployment.sh"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[[ -x $RECORDER ]] || fail "deployment evidence recorder is missing or not executable"

TMP=$(mktemp -d "${TMPDIR:-/tmp}/maintenance-deploy-evidence.XXXXXX")
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$TMP/bin"

cat > "$TMP/bin/systemctl" <<'FAKE_SYSTEMCTL'
#!/usr/bin/env bash
if [[ ${1:-} == is-active && ${2:-} == --quiet ]]; then
  [[ ${FAKE_SERVICE_ACTIVE:-1} == 1 ]]
  exit
fi
if [[ ${1:-} == show && ${2:-} == --property && ${3:-} == MainPID && ${4:-} == --value ]]; then
  printf '%s\n' "${FAKE_MAIN_PID:-731}"
  exit 0
fi
if [[ ${1:-} == show && ${2:-} == --property && ${3:-} == ExecStart && ${4:-} == --value ]]; then
  printf '{ path=%s ; argv[]=%s ; }\n' "${FAKE_EXEC_START:-/unexpected}" "${FAKE_EXEC_START:-/unexpected}"
  exit 0
fi
exit 2
FAKE_SYSTEMCTL

cat > "$TMP/bin/curl" <<'FAKE_CURL'
#!/usr/bin/env bash
[[ ${1:-} == --disable ]] || exit 91
[[ " $* " == *" --noproxy * "* ]] || exit 9
[[ ${FAKE_HEALTH_OK:-1} == 1 ]]
FAKE_CURL
chmod +x "$TMP/bin/systemctl" "$TMP/bin/curl"

SENTINEL="$TMP/maintenance"
BINARY="$TMP/server"
ROLLBACK_BINARY="$TMP/server.rollback"
EVIDENCE="$TMP/deployment.pass"
GENERATION=0123456789abcdef0123456789abcdef
printf 'state=active\ngeneration=%s\n' "$GENERATION" > "$SENTINEL"
chmod 0644 "$SENTINEL"
printf 'fixture backend artifact\n' > "$BINARY"
printf 'fixture rollback artifact\n' > "$ROLLBACK_BINARY"
chmod 0755 "$BINARY"
chmod 0755 "$ROLLBACK_BINARY"
EXPECTED_SHA=$(shasum -a 256 "$BINARY" | cut -d ' ' -f 1)
ROLLBACK_SHA=$(shasum -a 256 "$ROLLBACK_BINARY" | cut -d ' ' -f 1)
PROC_ROOT="$TMP/proc"
mkdir -p "$PROC_ROOT/731"

set +e
PATH="$TMP/bin:$PATH" MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  BACKEND_BINARY_PATH="$BINARY" BACKEND_EXPECTED_SHA256="$EXPECTED_SHA" \
  BACKEND_DEPLOY_EVIDENCE_OUTPUT="$EVIDENCE" "$RECORDER" >/dev/null 2>&1
UNAPPROVED_STATUS=$?
set -e
[[ $UNAPPROVED_STATUS -ne 0 && ! -e $EVIDENCE ]] || fail "unapproved deployment evidence was recorded"

set +e
PATH="$TMP/bin:$PATH" MAINTENANCE_DEPLOY_EVIDENCE_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" BACKEND_BINARY_PATH="$BINARY" \
  BACKEND_EXPECTED_SHA256=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
  BACKEND_ROLLBACK_PATH="$ROLLBACK_BINARY" \
  BACKEND_DEPLOY_EVIDENCE_OUTPUT="$EVIDENCE" "$RECORDER" >/dev/null 2>&1
WRONG_HASH_STATUS=$?
set -e
[[ $WRONG_HASH_STATUS -ne 0 && ! -e $EVIDENCE ]] || fail "mismatched backend artifact produced deployment evidence"

set +e
PATH="$TMP/bin:$PATH" MAINTENANCE_DEPLOY_EVIDENCE_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" BACKEND_BINARY_PATH="$BINARY" \
  BACKEND_EXPECTED_SHA256="$EXPECTED_SHA" BACKEND_DEPLOY_EVIDENCE_OUTPUT="$EVIDENCE" \
  BACKEND_HEALTH_URL=http://127.0.0.1:8080/api/health "$RECORDER" >/dev/null 2>&1
MISSING_ROLLBACK_STATUS=$?
set -e
[[ $MISSING_ROLLBACK_STATUS -ne 0 && ! -e $EVIDENCE ]] || fail "missing rollback artifact produced deployment evidence"

ln -s "$ROLLBACK_BINARY" "$PROC_ROOT/731/exe"
set +e
PATH="$TMP/bin:$PATH" MAINTENANCE_DEPLOY_EVIDENCE_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" BACKEND_BINARY_PATH="$BINARY" \
  BACKEND_EXPECTED_SHA256="$EXPECTED_SHA" BACKEND_DEPLOY_EVIDENCE_OUTPUT="$EVIDENCE" \
  BACKEND_ROLLBACK_PATH="$ROLLBACK_BINARY" BACKEND_PROC_ROOT="$PROC_ROOT" \
  BACKEND_HEALTH_URL=http://127.0.0.1:8080/api/health "$RECORDER" >/dev/null 2>&1
WRONG_EXECUTABLE_STATUS=$?
set -e
[[ $WRONG_EXECUTABLE_STATUS -ne 0 && ! -e $EVIDENCE ]] ||
  fail "MainPID executable mismatch produced deployment evidence"
rm "$PROC_ROOT/731/exe"
ln -s "$BINARY" "$PROC_ROOT/731/exe"

printf 'state=STARTED\ngeneration=%s\n' "$GENERATION" > "$SENTINEL"
chmod 0644 "$SENTINEL"
set +e
PATH="$TMP/bin:$PATH" FAKE_EXEC_START="$BINARY" \
  MAINTENANCE_DEPLOY_EVIDENCE_APPROVED=1 MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  BACKEND_BINARY_PATH="$BINARY" BACKEND_EXPECTED_SHA256="$EXPECTED_SHA" \
  BACKEND_DEPLOY_EVIDENCE_OUTPUT="$EVIDENCE" BACKEND_ROLLBACK_PATH="$ROLLBACK_BINARY" \
  BACKEND_PROC_ROOT="$PROC_ROOT" BACKEND_HEALTH_URL=http://127.0.0.1:8080/api/health \
  "$RECORDER" >/dev/null 2>&1
WRONG_SENTINEL_STATE_STATUS=$?
set -e
[[ $WRONG_SENTINEL_STATE_STATUS -ne 0 && ! -e $EVIDENCE ]] ||
  fail "non-active sentinel state produced deployment evidence"
printf 'state=active\ngeneration=%s\n' "$GENERATION" > "$SENTINEL"
chmod 0644 "$SENTINEL"

set +e
PATH="$TMP/bin:$PATH" FAKE_EXEC_START="$ROLLBACK_BINARY" \
  MAINTENANCE_DEPLOY_EVIDENCE_APPROVED=1 MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  BACKEND_BINARY_PATH="$BINARY" BACKEND_EXPECTED_SHA256="$EXPECTED_SHA" \
  BACKEND_DEPLOY_EVIDENCE_OUTPUT="$EVIDENCE" BACKEND_ROLLBACK_PATH="$ROLLBACK_BINARY" \
  BACKEND_PROC_ROOT="$PROC_ROOT" BACKEND_HEALTH_URL=http://127.0.0.1:8080/api/health \
  "$RECORDER" >/dev/null 2>&1
WRONG_EXEC_START_STATUS=$?
set -e
[[ $WRONG_EXEC_START_STATUS -ne 0 && ! -e $EVIDENCE ]] ||
  fail "effective ExecStart mismatch produced deployment evidence"

PATH="$TMP/bin:$PATH" FAKE_EXEC_START="$BINARY" MAINTENANCE_DEPLOY_EVIDENCE_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" BACKEND_BINARY_PATH="$BINARY" \
  BACKEND_EXPECTED_SHA256="$EXPECTED_SHA" BACKEND_DEPLOY_EVIDENCE_OUTPUT="$EVIDENCE" \
  BACKEND_ROLLBACK_PATH="$ROLLBACK_BINARY" \
  BACKEND_PROC_ROOT="$PROC_ROOT" \
  BACKEND_HEALTH_URL=http://127.0.0.1:8080/api/health "$RECORDER" >/dev/null

grep -Fxq 'state=PASS' "$EVIDENCE" || fail "deployment evidence state is missing"
grep -Fxq 'kind=deployment' "$EVIDENCE" || fail "deployment evidence kind is missing"
grep -Fxq "generation=$GENERATION" "$EVIDENCE" || fail "deployment evidence generation is missing"
grep -Fxq "artifact_sha256=$EXPECTED_SHA" "$EVIDENCE" || fail "deployment evidence artifact digest is missing"
grep -Fxq "rollback_path=$ROLLBACK_BINARY" "$EVIDENCE" || fail "deployment evidence rollback path is missing"
grep -Fxq "rollback_sha256=$ROLLBACK_SHA" "$EVIDENCE" || fail "deployment evidence rollback digest is missing"
MODE=$(stat -f '%Lp' "$EVIDENCE" 2>/dev/null || stat -c '%a' "$EVIDENCE")
[[ $MODE == 600 ]] || fail "deployment evidence mode is not 0600"

printf 'PASS: generation-bound deployment evidence producer\n'
