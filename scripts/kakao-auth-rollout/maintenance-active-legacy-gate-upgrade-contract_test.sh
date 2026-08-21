#!/usr/bin/env bash
# maintenance-active-legacy-gate-upgrade-contract_test.sh — Active maintenance legacy gate transaction.
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
SCRIPT="$ROOT/scripts/kakao-auth-rollout/maintenance-active-legacy-gate-upgrade.sh"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[[ -x $SCRIPT ]] || fail "active legacy gate upgrade script is missing"

TMP=$(mktemp -d "${TMPDIR:-/tmp}/maintenance-active-legacy.XXXXXX")
trap 'rm -rf "$TMP"' EXIT
BIN="$TMP/bin"
RUNTIME="$TMP/run-alumni"
TARGET_DIR="$TMP/target"
BACKUP_BASE="$TMP/backups"
SENTINEL="$RUNTIME/maintenance"
BRIDGE="$RUNTIME/maintenance-release-bridge"
HTTPD_SOURCE="$ROOT/deploy/httpd-alumni.conf"
GATE_SOURCE="$ROOT/deploy/_maintenance_gate.php"
HTTPD_TARGET="$TARGET_DIR/alumni.conf"
GATE_TARGET="$TARGET_DIR/_maintenance_gate.php"
EVENTS="$TMP/events"
GENERATION=0123456789abcdef0123456789abcdef
mkdir -m 0755 "$BIN" "$RUNTIME" "$TARGET_DIR"
printf 'state=active\ngeneration=%s\n' "$GENERATION" > "$SENTINEL"
chmod 0644 "$SENTINEL"
printf 'OLD_HTTPD_BYTES\n' > "$HTTPD_TARGET"
printf '<?php /* OLD_GATE_BYTES */\n' > "$GATE_TARGET"
chmod 0644 "$HTTPD_TARGET" "$GATE_TARGET"
: > "$EVENTS"
ORIGINAL_HTTPD_SHA=$(shasum -a 256 "$HTTPD_TARGET" | cut -d' ' -f1)
ORIGINAL_GATE_SHA=$(shasum -a 256 "$GATE_TARGET" | cut -d' ' -f1)
SENTINEL_SHA=$(shasum -a 256 "$SENTINEL" | cut -d' ' -f1)
SOURCE_HTTPD_SHA=$(shasum -a 256 "$HTTPD_SOURCE" | cut -d' ' -f1)
SOURCE_GATE_SHA=$(shasum -a 256 "$GATE_SOURCE" | cut -d' ' -f1)
REAL_PHP=$(command -v php)

cat > "$BIN/php" <<'FAKE_PHP'
#!/usr/bin/env bash
set -euo pipefail
printf 'PHP_LINT\n' >> "$EVENTS"
exec "$REAL_PHP" "$@"
FAKE_PHP
cat > "$BIN/httpd" <<'FAKE_HTTPD'
#!/usr/bin/env bash
set -euo pipefail
printf 'HTTPD_TEST\n' >> "$EVENTS"
[[ ${1:-} == -t ]]
grep -Eq '(/run/alumni/maintenance-release-bridge|OLD_HTTPD_BYTES)' "$HTTPD_TARGET"
FAKE_HTTPD
cat > "$BIN/systemctl" <<'FAKE_SYSTEMCTL'
#!/usr/bin/env bash
set -euo pipefail
printf 'SYSTEMCTL %s\n' "$*" >> "$EVENTS"
case "${1:-} ${2:-}" in
  'reload httpd') [[ ${FAKE_RELOAD_FAIL:-0} != 1 ]] ;;
  'is-active --quiet') [[ ${3:-} == httpd ]] ;;
  *) exit 90 ;;
esac
FAKE_SYSTEMCTL
chmod 0755 "$BIN/php" "$BIN/httpd" "$BIN/systemctl"

common_env=(
  MAINTENANCE_ACTIVE_LEGACY_TEST_MODE=1
  MAINTENANCE_SENTINEL_PATH="$SENTINEL"
  MAINTENANCE_RELEASE_BRIDGE_PATH="$BRIDGE"
  MAINTENANCE_HTTPD_SOURCE="$HTTPD_SOURCE"
  MAINTENANCE_GATE_SOURCE="$GATE_SOURCE"
  MAINTENANCE_HTTPD_TARGET="$HTTPD_TARGET"
  MAINTENANCE_GATE_TARGET="$GATE_TARGET"
  MAINTENANCE_ACTIVE_LEGACY_BACKUP_BASE="$BACKUP_BASE"
  MAINTENANCE_PHP_BIN="$BIN/php"
  MAINTENANCE_HTTPD_BIN="$BIN/httpd"
  MAINTENANCE_SYSTEMCTL_BIN="$BIN/systemctl"
  REAL_PHP="$REAL_PHP"
  EVENTS="$EVENTS"
  HTTPD_TARGET="$HTTPD_TARGET"
)

set +e
UNAPPROVED_OUTPUT=$(env "${common_env[@]}" "$SCRIPT" prepare 2>&1)
UNAPPROVED_STATUS=$?
set -e
[[ $UNAPPROVED_STATUS -ne 0 && $UNAPPROVED_OUTPUT == *'reason=approval_required'* ]] ||
  fail "unapproved upgrade did not fail closed"
[[ $(shasum -a 256 "$HTTPD_TARGET" | cut -d' ' -f1) == "$ORIGINAL_HTTPD_SHA" ]] || fail "unapproved upgrade changed httpd"
[[ $(shasum -a 256 "$GATE_TARGET" | cut -d' ' -f1) == "$ORIGINAL_GATE_SHA" ]] || fail "unapproved upgrade changed gate"

PREPARE_OUTPUT=$(env "${common_env[@]}" MAINTENANCE_ACTIVE_LEGACY_APPROVED=1 "$SCRIPT" prepare)
[[ $PREPARE_OUTPUT == 'PASS maintenance_active_legacy_gate state=prepared backup='* ]] ||
  fail "approved upgrade did not return bounded PASS"
BACKUP_DIR=${PREPARE_OUTPUT##*backup=}
[[ $BACKUP_DIR == "$BACKUP_BASE"/active-legacy.* && -d $BACKUP_DIR ]] || fail "backup handle is invalid"
[[ $(shasum -a 256 "$HTTPD_TARGET" | cut -d' ' -f1) == "$SOURCE_HTTPD_SHA" ]] || fail "httpd source was not installed"
[[ $(shasum -a 256 "$GATE_TARGET" | cut -d' ' -f1) == "$SOURCE_GATE_SHA" ]] || fail "PHP gate source was not installed"
[[ $(shasum -a 256 "$SENTINEL" | cut -d' ' -f1) == "$SENTINEL_SHA" ]] || fail "prepare changed active sentinel"
[[ ! -e $BRIDGE && ! -L $BRIDGE ]] || fail "prepare created bridge"
grep -Fxq 'SYSTEMCTL reload httpd' "$EVENTS" || fail "prepare did not reload httpd"
grep -Fxq 'SYSTEMCTL is-active --quiet httpd' "$EVENTS" || fail "prepare did not recheck httpd"

ROLLBACK_OUTPUT=$(env "${common_env[@]}" MAINTENANCE_ACTIVE_LEGACY_APPROVED=1 "$SCRIPT" rollback "$BACKUP_DIR")
[[ $ROLLBACK_OUTPUT == 'PASS maintenance_active_legacy_gate state=rolled_back backup='* ]] ||
  fail "rollback did not return bounded PASS"
[[ $(shasum -a 256 "$HTTPD_TARGET" | cut -d' ' -f1) == "$ORIGINAL_HTTPD_SHA" ]] || fail "rollback did not restore httpd"
[[ $(shasum -a 256 "$GATE_TARGET" | cut -d' ' -f1) == "$ORIGINAL_GATE_SHA" ]] || fail "rollback did not restore gate"
[[ $(shasum -a 256 "$SENTINEL" | cut -d' ' -f1) == "$SENTINEL_SHA" ]] || fail "rollback changed active sentinel"
[[ ! -e $BRIDGE && ! -L $BRIDGE ]] || fail "rollback created bridge"

: > "$EVENTS"
set +e
FAIL_OUTPUT=$(env "${common_env[@]}" MAINTENANCE_ACTIVE_LEGACY_APPROVED=1 FAKE_RELOAD_FAIL=1 "$SCRIPT" prepare 2>&1)
FAIL_STATUS=$?
set -e
[[ $FAIL_STATUS -ne 0 && $FAIL_OUTPUT == *'reason=httpd_reload_failed'* ]] || fail "reload failure was not reported"
[[ $(shasum -a 256 "$HTTPD_TARGET" | cut -d' ' -f1) == "$ORIGINAL_HTTPD_SHA" ]] || fail "reload failure did not restore httpd"
[[ $(shasum -a 256 "$GATE_TARGET" | cut -d' ' -f1) == "$ORIGINAL_GATE_SHA" ]] || fail "reload failure did not restore gate"
[[ $(shasum -a 256 "$SENTINEL" | cut -d' ' -f1) == "$SENTINEL_SHA" ]] || fail "reload failure changed active sentinel"
[[ ! -e $BRIDGE && ! -L $BRIDGE ]] || fail "reload failure created bridge"
[[ $FAIL_OUTPUT != *OLD_GATE_BYTES* ]] || fail "failure output exposed target contents"

printf 'PASS: active-maintenance legacy bridge gate transaction\n'
