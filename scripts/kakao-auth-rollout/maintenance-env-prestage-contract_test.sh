#!/usr/bin/env bash
# maintenance-env-prestage-contract_test.sh — Contract for inactive maintenance env/proof pre-stage.
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
SCRIPT="$ROOT/scripts/kakao-auth-rollout/maintenance-env-prestage.sh"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[[ -x $SCRIPT ]] || fail "maintenance env pre-stage script is missing or not executable"
grep -Fq "stat -c '%u' \"\$1\" 2>/dev/null || stat -f '%u' \"\$1\"" "$SCRIPT" ||
  fail "EnvironmentFile helper is not GNU/Linux stat compatible"
grep -Fq "stat -c '%a' \"\$1\" 2>/dev/null || stat -f '%Lp' \"\$1\"" "$SCRIPT" ||
  fail "EnvironmentFile mode helper is not GNU/Linux stat compatible"
grep -Fq "stat -c '%g' \"\$1\" 2>/dev/null || stat -f '%g' \"\$1\"" "$SCRIPT" ||
  fail "EnvironmentFile gid helper is not GNU/Linux stat compatible"
grep -Fq "stat -c '%G' \"\$1\" 2>/dev/null || stat -f '%Sg' \"\$1\"" "$SCRIPT" ||
  fail "EnvironmentFile group helper is not GNU/Linux stat compatible"

TMP=$(mktemp -d "${TMPDIR:-/tmp}/maintenance-env-prestage-contract.XXXXXX")
trap 'rm -rf "$TMP"' EXIT
STAT_BIN="$TMP/gnu-stat-bin"
ENV_FILE="$TMP/alumni-backend.env"
RUNTIME_DIR="$TMP/run-alumni"
SENTINEL="$RUNTIME_DIR/maintenance"
PROOF_FILE="$RUNTIME_DIR/maintenance-smoke-proof"
BACKUP_BASE="$TMP/backups"
STATE_FILE="$TMP/transaction.state"
mkdir -m 0755 "$RUNTIME_DIR"
mkdir -m 0755 "$STAT_BIN"
REAL_STAT=$(command -v stat)
cat > "$STAT_BIN/stat" <<'FAKE_GNU_STAT'
#!/usr/bin/env bash
set -euo pipefail
if [[ ${1:-} == -c ]]; then
  format=$2
  path=$3
  case "$format" in
    '%a') exec "$REAL_STAT" -f '%Lp' "$path" ;;
    '%u') exec "$REAL_STAT" -f '%u' "$path" ;;
    '%g') exec "$REAL_STAT" -f '%g' "$path" ;;
    '%G') printf 'alumni-backend\n'; exit 0 ;;
    *) exit 91 ;;
  esac
fi
if [[ ${1:-} == -f ]]; then
  printf 'contaminating GNU filesystem output\n'
  exit 1
fi
exit 92
FAKE_GNU_STAT
chmod 0755 "$STAT_BIN/stat"
DB_PASSWORD_KEY=DB_PASSWORD
JWT_SECRET_KEY=JWT_SECRET
{
  printf 'DB_USER=fixture-user\n'
  printf '%s=fixture-password\n' "$DB_PASSWORD_KEY"
  printf '%s=fixture-jwt-secret\n' "$JWT_SECRET_KEY"
  printf 'ENV=prod\n'
} > "$ENV_FILE"
chmod 0640 "$ENV_FILE"
ORIGINAL_SHA=$(shasum -a 256 "$ENV_FILE" | cut -d' ' -f1)

set +e
UNAPPROVED_OUTPUT=$(
  MAINTENANCE_ENV_FILE_PATH="$ENV_FILE" \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  MAINTENANCE_SMOKE_PROOF_FILE="$PROOF_FILE" \
  MAINTENANCE_ENV_BACKUP_BASE="$BACKUP_BASE" \
  "$SCRIPT" prepare 2>&1
)
UNAPPROVED_STATUS=$?
set -e
[[ $UNAPPROVED_STATUS -ne 0 ]] || fail "unapproved pre-stage did not fail closed"
[[ ! -e $PROOF_FILE ]] || fail "unapproved pre-stage created a proof file"
[[ $(shasum -a 256 "$ENV_FILE" | cut -d' ' -f1) == "$ORIGINAL_SHA" ]] ||
  fail "unapproved pre-stage changed the EnvironmentFile"
[[ $UNAPPROVED_OUTPUT != *fixture-password* && $UNAPPROVED_OUTPUT != *fixture-jwt-secret* ]] ||
  fail "unapproved output exposed a credential"

PREPARE_OUTPUT=$(
  PATH="$STAT_BIN:$PATH" \
  REAL_STAT="$REAL_STAT" \
  MAINTENANCE_ENV_PRESTAGE_APPROVED=1 \
  MAINTENANCE_ENV_FILE_PATH="$ENV_FILE" \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  MAINTENANCE_SMOKE_PROOF_FILE="$PROOF_FILE" \
  MAINTENANCE_ENV_BACKUP_BASE="$BACKUP_BASE" \
  MAINTENANCE_PRESTAGE_STATE_FILE="$STATE_FILE" \
  MAINTENANCE_PRESTAGE_HTTPD_BACKUP=/var/backups/alumni-maintenance/prepare.fixture \
  "$SCRIPT" prepare
)
[[ $PREPARE_OUTPUT == 'PASS maintenance_env_prestage state=prepared backup='* ]] ||
  fail "approved pre-stage did not return bounded metadata"
[[ $PREPARE_OUTPUT != *fixture-password* && $PREPARE_OUTPUT != *fixture-jwt-secret* ]] ||
  fail "approved output exposed a credential"
BACKUP_DIR=${PREPARE_OUTPUT##*backup=}
[[ $BACKUP_DIR == "$BACKUP_BASE"/* && -d $BACKUP_DIR ]] || fail "pre-stage backup path is invalid"
[[ -f $STATE_FILE && ! -L $STATE_FILE ]] || fail "pre-stage state file was not persisted"
[[ $(stat -f '%Lp' "$STATE_FILE" 2>/dev/null || stat -c '%a' "$STATE_FILE") == 600 ]] ||
  fail "pre-stage state file mode is not 0600"
grep -Fxq 'httpd_backup=/var/backups/alumni-maintenance/prepare.fixture' "$STATE_FILE" ||
  fail "pre-stage state lost the Apache/PHP backup handle"
grep -Fxq "env_backup=$BACKUP_DIR" "$STATE_FILE" || fail "pre-stage state lost the EnvironmentFile backup handle"
[[ ! -e $SENTINEL ]] || fail "inactive pre-stage activated maintenance"
[[ -f $PROOF_FILE && ! -L $PROOF_FILE ]] || fail "proof is not a regular non-symlink"
PROOF_MODE=$(stat -f '%Lp' "$PROOF_FILE" 2>/dev/null || stat -c '%a' "$PROOF_FILE")
[[ $PROOF_MODE == 600 ]] || fail "proof mode is not 0600"
[[ $(stat -f '%u' "$PROOF_FILE" 2>/dev/null || stat -c '%u' "$PROOF_FILE") == "$(id -u)" ]] ||
  fail "proof owner is not the executing user"
PROOF=$(tr -d '\r\n' < "$PROOF_FILE")
[[ $PROOF =~ ^[a-f0-9]{64}$ ]] || fail "proof format is invalid"
PROOF_SHA=$(printf '%s' "$PROOF" | shasum -a 256 | cut -d' ' -f1)
unset PROOF

ENV_PROOF_SHA=$(sed -n 's/^MAINTENANCE_SMOKE_PROOF_SHA256=//p' "$ENV_FILE")
[[ $ENV_PROOF_SHA == "$PROOF_SHA" ]] || fail "EnvironmentFile proof digest is not bound to the proof"
[[ $(grep -c '^MAINTENANCE_SENTINEL_PATH=' "$ENV_FILE") == 1 ]] || fail "sentinel key count is not exactly one"
[[ $(grep -c '^MAINTENANCE_SMOKE_PROOF_SHA256=' "$ENV_FILE") == 1 ]] || fail "proof hash key count is not exactly one"
[[ $(grep -c '^MAINTENANCE_SMOKE_ALLOWED_PATHS=' "$ENV_FILE") == 1 ]] || fail "allowlist key count is not exactly one"
grep -Fxq "MAINTENANCE_SENTINEL_PATH=$SENTINEL" "$ENV_FILE" ||
  fail "canonical sentinel value was not installed"
grep -Fxq 'MAINTENANCE_SMOKE_ALLOWED_PATHS=/api/auth/login,/api/auth/logout' "$ENV_FILE" ||
  fail "exact smoke allowlist was not installed"
grep -Fxq "${DB_PASSWORD_KEY}=fixture-password" "$ENV_FILE" || fail "existing credential was not preserved"
grep -Fxq "${JWT_SECRET_KEY}=fixture-jwt-secret" "$ENV_FILE" || fail "existing secret was not preserved"

rm -f "$PROOF_FILE"
rm -f "$ENV_FILE"
mkdir "$ENV_FILE"
ROLLBACK_OUTPUT=$(
  PATH="$STAT_BIN:$PATH" \
  REAL_STAT="$REAL_STAT" \
  MAINTENANCE_ENV_PRESTAGE_APPROVED=1 \
  MAINTENANCE_ENV_FILE_PATH="$ENV_FILE" \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  MAINTENANCE_SMOKE_PROOF_FILE="$PROOF_FILE" \
  MAINTENANCE_ENV_BACKUP_BASE="$BACKUP_BASE" \
  "$SCRIPT" rollback "$BACKUP_DIR"
)
[[ $ROLLBACK_OUTPUT == 'PASS maintenance_env_prestage state=rolled_back backup='* ]] ||
  fail "rollback did not return bounded metadata"
[[ ! -e $PROOF_FILE ]] || fail "rollback left the proof file behind"
[[ -f $ENV_FILE && ! -L $ENV_FILE ]] || fail "rollback did not recover a non-regular EnvironmentFile path"
[[ $(shasum -a 256 "$ENV_FILE" | cut -d' ' -f1) == "$ORIGINAL_SHA" ]] ||
  fail "rollback did not restore the EnvironmentFile byte-for-byte"
[[ ! -e $SENTINEL ]] || fail "rollback activated maintenance"

FAIL_RUNTIME_DIR="$TMP/fail-run-alumni"
FAIL_ENV_FILE="$TMP/fail-alumni-backend.env"
FAIL_SENTINEL="$FAIL_RUNTIME_DIR/maintenance"
FAIL_PROOF_FILE="$FAIL_RUNTIME_DIR/maintenance-smoke-proof"
FAIL_BACKUP_BASE="$TMP/fail-backups"
FAIL_BIN="$TMP/fail-bin"
mkdir -m 0755 "$FAIL_RUNTIME_DIR"
mkdir -m 0755 "$FAIL_BIN"
cp "$ENV_FILE" "$FAIL_ENV_FILE"
chmod 0600 "$FAIL_ENV_FILE"
REAL_PHP=$(command -v php)
REAL_MV=$(command -v mv)
PHP_COUNT_FILE="$TMP/php-count"
cat > "$FAIL_BIN/php" <<'FAKE_PHP'
#!/usr/bin/env bash
set -euo pipefail
count=0
[[ ! -f $PHP_COUNT_FILE ]] || count=$(<"$PHP_COUNT_FILE")
count=$((count + 1))
printf '%s\n' "$count" > "$PHP_COUNT_FILE"
if [[ $count == 2 ]]; then
  exit 42
fi
exec "$REAL_PHP" "$@"
FAKE_PHP
cat > "$FAIL_BIN/mv" <<'FAKE_MV'
#!/usr/bin/env bash
set -euo pipefail
for arg in "$@"; do
  if [[ $arg == *.rollback.* ]]; then
    exit 43
  fi
done
exec "$REAL_MV" "$@"
FAKE_MV
chmod 0755 "$FAIL_BIN/php" "$FAIL_BIN/mv"
set +e
INTERNAL_FAILURE_OUTPUT=$(
  PATH="$FAIL_BIN:$PATH" \
  REAL_PHP="$REAL_PHP" \
  REAL_MV="$REAL_MV" \
  PHP_COUNT_FILE="$PHP_COUNT_FILE" \
  MAINTENANCE_ENV_PRESTAGE_APPROVED=1 \
  MAINTENANCE_ENV_FILE_PATH="$FAIL_ENV_FILE" \
  MAINTENANCE_SENTINEL_PATH="$FAIL_SENTINEL" \
  MAINTENANCE_SMOKE_PROOF_FILE="$FAIL_PROOF_FILE" \
  MAINTENANCE_ENV_BACKUP_BASE="$FAIL_BACKUP_BASE" \
  "$SCRIPT" prepare 2>&1
)
INTERNAL_FAILURE_STATUS=$?
set -e
[[ $INTERNAL_FAILURE_STATUS -ne 0 ]] || fail "injected internal failure unexpectedly passed"
[[ $INTERNAL_FAILURE_OUTPUT == *'ERROR maintenance_env_prestage rollback=failed backup='* ]] ||
  fail "internal restore failure was not reported separately"
[[ $INTERNAL_FAILURE_OUTPUT == *'reason=environment_validation_failed'* ]] ||
  fail "internal failure lost its primary reason"
[[ $INTERNAL_FAILURE_OUTPUT != *fixture-password* && $INTERNAL_FAILURE_OUTPUT != *fixture-jwt-secret* ]] ||
  fail "internal rollback failure exposed a credential"

SYMLINK_ENV="$TMP/symlink.env"
ln -s "$ENV_FILE" "$SYMLINK_ENV"
set +e
SYMLINK_OUTPUT=$(
  MAINTENANCE_ENV_PRESTAGE_APPROVED=1 \
  MAINTENANCE_ENV_FILE_PATH="$SYMLINK_ENV" \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  MAINTENANCE_SMOKE_PROOF_FILE="$PROOF_FILE" \
  MAINTENANCE_ENV_BACKUP_BASE="$BACKUP_BASE" \
  "$SCRIPT" prepare 2>&1
)
SYMLINK_STATUS=$?
set -e
[[ $SYMLINK_STATUS -ne 0 && ! -e $PROOF_FILE ]] || fail "symlinked EnvironmentFile was accepted"
[[ $SYMLINK_OUTPUT != *fixture-password* && $SYMLINK_OUTPUT != *fixture-jwt-secret* ]] ||
  fail "symlink rejection exposed a credential"

DUPLICATE_ENV="$TMP/duplicate.env"
cp "$ENV_FILE" "$DUPLICATE_ENV"
printf 'MAINTENANCE_SENTINEL_PATH=%s\nMAINTENANCE_SENTINEL_PATH=%s\n' "$SENTINEL" "$SENTINEL" >> "$DUPLICATE_ENV"
chmod 0600 "$DUPLICATE_ENV"
set +e
DUPLICATE_OUTPUT=$(
  MAINTENANCE_ENV_PRESTAGE_APPROVED=1 \
  MAINTENANCE_ENV_FILE_PATH="$DUPLICATE_ENV" \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  MAINTENANCE_SMOKE_PROOF_FILE="$PROOF_FILE" \
  MAINTENANCE_ENV_BACKUP_BASE="$BACKUP_BASE" \
  "$SCRIPT" prepare 2>&1
)
DUPLICATE_STATUS=$?
set -e
[[ $DUPLICATE_STATUS -ne 0 && ! -e $PROOF_FILE ]] || fail "duplicate maintenance keys were accepted"
[[ $DUPLICATE_OUTPUT != *fixture-password* && $DUPLICATE_OUTPUT != *fixture-jwt-secret* ]] ||
  fail "duplicate-key rejection exposed a credential"

set +e
CANONICAL_MIX_OUTPUT=$(
  MAINTENANCE_ENV_PRESTAGE_APPROVED=1 \
  MAINTENANCE_ENV_FILE_PATH="$ENV_FILE" \
  MAINTENANCE_SENTINEL_PATH=/run/alumni/maintenance \
  MAINTENANCE_SMOKE_PROOF_FILE="$PROOF_FILE" \
  MAINTENANCE_ENV_BACKUP_BASE="$BACKUP_BASE" \
  "$SCRIPT" prepare 2>&1
)
CANONICAL_MIX_STATUS=$?
set -e
[[ $CANONICAL_MIX_STATUS -ne 0 && ! -e $PROOF_FILE ]] || fail "partially canonical paths were accepted"
[[ $CANONICAL_MIX_OUTPUT != *fixture-password* && $CANONICAL_MIX_OUTPUT != *fixture-jwt-secret* ]] ||
  fail "canonical-path rejection exposed a credential"

printf 'PASS: maintenance env/proof inactive pre-stage transaction\n'
