#!/usr/bin/env bash
# maintenance-release-env-amend-contract_test.sh — Active-maintenance release proof/env transaction contract.
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
SCRIPT="$ROOT/scripts/kakao-auth-rollout/maintenance-release-env-amend.sh"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[[ -x $SCRIPT ]] || fail "active-maintenance release env amendment script is missing"

TMP=$(mktemp -d "${TMPDIR:-/tmp}/maintenance-release-env-amend.XXXXXX")
trap 'rm -rf "$TMP"' EXIT
RUNTIME="$TMP/run-alumni"
ENV_FILE="$TMP/alumni-backend.env"
SENTINEL="$RUNTIME/maintenance"
BRIDGE="$RUNTIME/maintenance-release-bridge"
PROOF_FILE="$RUNTIME/maintenance-release-proof"
BACKUP_BASE="$TMP/backups"
GENERATION=0123456789abcdef0123456789abcdef
mkdir -m 0755 "$RUNTIME"
printf 'state=active\ngeneration=%s\n' "$GENERATION" > "$SENTINEL"
chmod 0644 "$SENTINEL"
{
  printf 'DB_PASSWORD=fixture-password\n'
  printf 'JWT_SECRET=fixture-jwt-secret\n'
  printf 'ENV=prod\n'
} > "$ENV_FILE"
chmod 0600 "$ENV_FILE"
ORIGINAL_SHA=$(shasum -a 256 "$ENV_FILE" | cut -d' ' -f1)
SENTINEL_SHA=$(shasum -a 256 "$SENTINEL" | cut -d' ' -f1)

set +e
UNAPPROVED_OUTPUT=$(
  MAINTENANCE_ENV_FILE_PATH="$ENV_FILE" \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  MAINTENANCE_RELEASE_BRIDGE_PATH="$BRIDGE" \
  MAINTENANCE_RELEASE_PROOF_FILE="$PROOF_FILE" \
  MAINTENANCE_RELEASE_ENV_BACKUP_BASE="$BACKUP_BASE" \
  "$SCRIPT" prepare 2>&1
)
UNAPPROVED_STATUS=$?
set -e
[[ $UNAPPROVED_STATUS -ne 0 ]] || fail "unapproved amendment passed"
[[ ! -e $PROOF_FILE && $(shasum -a 256 "$ENV_FILE" | cut -d' ' -f1) == "$ORIGINAL_SHA" ]] ||
  fail "unapproved amendment changed state"
[[ $UNAPPROVED_OUTPUT != *fixture-password* && $UNAPPROVED_OUTPUT != *fixture-jwt-secret* ]] ||
  fail "unapproved output exposed secrets"

PREPARE_OUTPUT=$(
  MAINTENANCE_RELEASE_ENV_AMEND_APPROVED=1 \
  MAINTENANCE_ENV_FILE_PATH="$ENV_FILE" \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  MAINTENANCE_RELEASE_BRIDGE_PATH="$BRIDGE" \
  MAINTENANCE_RELEASE_PROOF_FILE="$PROOF_FILE" \
  MAINTENANCE_RELEASE_ENV_BACKUP_BASE="$BACKUP_BASE" \
  "$SCRIPT" prepare
)
[[ $PREPARE_OUTPUT == 'PASS maintenance_release_env_amend state=prepared backup='* ]] ||
  fail "approved amendment did not return bounded metadata"
[[ $PREPARE_OUTPUT != *fixture-password* && $PREPARE_OUTPUT != *fixture-jwt-secret* ]] ||
  fail "approved output exposed secrets"
BACKUP_DIR=${PREPARE_OUTPUT##*backup=}
[[ $BACKUP_DIR == "$BACKUP_BASE"/* && -d $BACKUP_DIR ]] || fail "backup handle is invalid"
[[ -f $PROOF_FILE && ! -L $PROOF_FILE ]] || fail "release proof is not regular"
PROOF_MODE=$(stat -f '%Lp' "$PROOF_FILE" 2>/dev/null || stat -c '%a' "$PROOF_FILE")
[[ $PROOF_MODE == 600 ]] || fail "release proof mode is not 0600"
[[ $(wc -c < "$PROOF_FILE") -eq 65 ]] || fail "release proof byte size is not canonical"
PROOF=$(<"$PROOF_FILE")
[[ $PROOF =~ ^[a-f0-9]{64}$ ]] || fail "release proof format is invalid"
PROOF_SHA=$(printf '%s' "$PROOF" | shasum -a 256 | cut -d' ' -f1)
unset PROOF
[[ $(grep -c '^MAINTENANCE_RELEASE_BRIDGE_PATH=' "$ENV_FILE") == 1 ]] || fail "bridge key count is not one"
[[ $(grep -c '^MAINTENANCE_RELEASE_PROOF_SHA256=' "$ENV_FILE") == 1 ]] || fail "proof digest key count is not one"
[[ $(grep -c '^MAINTENANCE_RELEASE_OWNER_UID=' "$ENV_FILE") == 1 ]] || fail "owner key count is not one"
[[ $(grep -c '^MAINTENANCE_RELEASE_DRAIN_TIMEOUT=' "$ENV_FILE") == 1 ]] || fail "timeout key count is not one"
grep -Fxq "MAINTENANCE_RELEASE_BRIDGE_PATH=$BRIDGE" "$ENV_FILE" || fail "bridge path was not installed"
grep -Fxq "MAINTENANCE_RELEASE_PROOF_SHA256=$PROOF_SHA" "$ENV_FILE" || fail "proof digest was not installed"
grep -Fxq "MAINTENANCE_RELEASE_OWNER_UID=$(id -u)" "$ENV_FILE" || fail "owner UID was not installed"
grep -Fxq 'MAINTENANCE_RELEASE_DRAIN_TIMEOUT=90s' "$ENV_FILE" || fail "drain timeout was not installed"
grep -Fxq 'DB_PASSWORD=fixture-password' "$ENV_FILE" || fail "DB secret was not preserved"
grep -Fxq 'JWT_SECRET=fixture-jwt-secret' "$ENV_FILE" || fail "JWT secret was not preserved"
[[ $(shasum -a 256 "$SENTINEL" | cut -d' ' -f1) == "$SENTINEL_SHA" ]] || fail "prepare changed active sentinel"
[[ ! -e $BRIDGE ]] || fail "prepare created the release bridge"

ROLLBACK_OUTPUT=$(
  MAINTENANCE_RELEASE_ENV_AMEND_APPROVED=1 \
  MAINTENANCE_ENV_FILE_PATH="$ENV_FILE" \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  MAINTENANCE_RELEASE_BRIDGE_PATH="$BRIDGE" \
  MAINTENANCE_RELEASE_PROOF_FILE="$PROOF_FILE" \
  MAINTENANCE_RELEASE_ENV_BACKUP_BASE="$BACKUP_BASE" \
  "$SCRIPT" rollback "$BACKUP_DIR"
)
[[ $ROLLBACK_OUTPUT == 'PASS maintenance_release_env_amend state=rolled_back backup='* ]] ||
  fail "rollback did not return bounded metadata"
[[ $(shasum -a 256 "$ENV_FILE" | cut -d' ' -f1) == "$ORIGINAL_SHA" ]] || fail "rollback did not restore env bytes"
[[ ! -e $PROOF_FILE && ! -L $PROOF_FILE ]] || fail "rollback left release proof"
[[ $(shasum -a 256 "$SENTINEL" | cut -d' ' -f1) == "$SENTINEL_SHA" ]] || fail "rollback changed active sentinel"
[[ ! -e $BRIDGE ]] || fail "rollback created the release bridge"

printf 'state=prepared\ngeneration=%s\napproval_attempt_id=%064d\n' "$GENERATION" 0 > "$BRIDGE"
chmod 0644 "$BRIDGE"
set +e
BRIDGE_OUTPUT=$(
  MAINTENANCE_RELEASE_ENV_AMEND_APPROVED=1 \
  MAINTENANCE_ENV_FILE_PATH="$ENV_FILE" \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  MAINTENANCE_RELEASE_BRIDGE_PATH="$BRIDGE" \
  MAINTENANCE_RELEASE_PROOF_FILE="$PROOF_FILE" \
  MAINTENANCE_RELEASE_ENV_BACKUP_BASE="$BACKUP_BASE" \
  "$SCRIPT" prepare 2>&1
)
BRIDGE_STATUS=$?
set -e
[[ $BRIDGE_STATUS -ne 0 && $BRIDGE_OUTPUT == *'reason=release_bridge_must_be_absent'* ]] ||
  fail "existing bridge did not fail closed"
[[ $(shasum -a 256 "$ENV_FILE" | cut -d' ' -f1) == "$ORIGINAL_SHA" && ! -e $PROOF_FILE ]] ||
  fail "existing bridge changed env/proof state"
rm -f "$BRIDGE"

FAIL_BIN="$TMP/fail-bin"
mkdir -m 0755 "$FAIL_BIN"
REAL_PHP=$(command -v php)
PHP_COUNT_FILE="$TMP/php-count"
cat > "$FAIL_BIN/php" <<'FAKE_PHP'
#!/usr/bin/env bash
set -euo pipefail
count=0
[[ ! -f $PHP_COUNT_FILE ]] || count=$(<"$PHP_COUNT_FILE")
count=$((count + 1))
printf '%s\n' "$count" > "$PHP_COUNT_FILE"
[[ $count != 2 ]] || exit 42
exec "$REAL_PHP" "$@"
FAKE_PHP
chmod 0755 "$FAIL_BIN/php"
set +e
INJECTED_OUTPUT=$(
  PATH="$FAIL_BIN:$PATH" REAL_PHP="$REAL_PHP" PHP_COUNT_FILE="$PHP_COUNT_FILE" \
  MAINTENANCE_RELEASE_ENV_AMEND_APPROVED=1 \
  MAINTENANCE_ENV_FILE_PATH="$ENV_FILE" \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  MAINTENANCE_RELEASE_BRIDGE_PATH="$BRIDGE" \
  MAINTENANCE_RELEASE_PROOF_FILE="$PROOF_FILE" \
  MAINTENANCE_RELEASE_ENV_BACKUP_BASE="$BACKUP_BASE" \
  "$SCRIPT" prepare 2>&1
)
INJECTED_STATUS=$?
set -e
[[ $INJECTED_STATUS -ne 0 && $INJECTED_OUTPUT == *'reason=environment_validation_failed'* ]] ||
  fail "injected post-install validation failure was not reported"
[[ $(shasum -a 256 "$ENV_FILE" | cut -d' ' -f1) == "$ORIGINAL_SHA" ]] ||
  fail "injected failure did not restore env bytes"
[[ ! -e $PROOF_FILE && ! -L $PROOF_FILE ]] || fail "injected failure left release proof"
[[ $(shasum -a 256 "$SENTINEL" | cut -d' ' -f1) == "$SENTINEL_SHA" ]] ||
  fail "injected failure changed active sentinel"
[[ $INJECTED_OUTPUT != *fixture-password* && $INJECTED_OUTPUT != *fixture-jwt-secret* ]] ||
  fail "injected failure exposed secrets"

printf 'PASS: active-maintenance release env amendment\n'
