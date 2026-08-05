#!/usr/bin/env bash
# maintenance-prestage-driver-contract_test.sh — Fail-closed transaction driver contract.
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
DRIVER="$ROOT/scripts/kakao-auth-rollout/maintenance-prestage-driver.sh"
BACKUP_VALIDATOR="$ROOT/scripts/kakao-auth-rollout/maintenance-backup-base-validate.sh"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[[ -x $DRIVER ]] || fail "maintenance pre-stage transaction driver is missing or not executable"
[[ -x $BACKUP_VALIDATOR ]] || fail "maintenance backup base validator is missing or not executable"

TMP=$(mktemp -d "${TMPDIR:-/tmp}/maintenance-prestage-driver-contract.XXXXXX")
trap 'rm -rf "$TMP"' EXIT

run_validator() {
  env \
    MAINTENANCE_BACKUP_VALIDATE_TEST_MODE=1 \
    MAINTENANCE_BACKUP_EXPECTED_UID="${2:-$(id -u)}" \
    "$BACKUP_VALIDATOR" "$1"
}

expect_validator_failure() {
  local label=$1
  local path=$2
  local expected_uid=${3:-$(id -u)}
  local output status
  set +e
  output=$(run_validator "$path" "$expected_uid" 2>&1)
  status=$?
  set -e
  [[ $status -ne 0 && $output == 'ERROR maintenance_backup_base state=blocked reason='* ]] ||
    fail "backup validator accepted $label"
}

ABSENT_BACKUP_BASE="$TMP/absent-backup-base"
[[ $(run_validator "$ABSENT_BACKUP_BASE") == 'PASS maintenance_backup_base state=absent' ]] ||
  fail "backup validator rejected an absent base"

TRUSTED_BACKUP_BASE="$TMP/trusted-backup-base"
mkdir -m 0700 "$TRUSTED_BACKUP_BASE"
mkdir -m 0700 "$TRUSTED_BACKUP_BASE/prepare.Ab12Cd"
mkdir -m 0700 "$TRUSTED_BACKUP_BASE/env-prestage.Z9y8X7"
[[ $(run_validator "$TRUSTED_BACKUP_BASE") == 'PASS maintenance_backup_base state=trusted' ]] ||
  fail "backup validator rejected trusted generations"

WRONG_BASE_MODE="$TMP/wrong-base-mode"
mkdir -m 0755 "$WRONG_BASE_MODE"
expect_validator_failure "wrong base mode" "$WRONG_BASE_MODE"

BASE_SYMLINK="$TMP/base-symlink"
ln -s "$TRUSTED_BACKUP_BASE" "$BASE_SYMLINK"
expect_validator_failure "symlinked base" "$BASE_SYMLINK"

WRONG_OWNER_UID=$(( $(id -u) + 1 ))
expect_validator_failure "wrong base owner" "$TRUSTED_BACKUP_BASE" "$WRONG_OWNER_UID"

for invalid_name in prepare.Ab12C prepare.Ab12Cd7 other.Ab12Cd; do
  invalid_base="$TMP/invalid-name-${invalid_name//./-}"
  mkdir -m 0700 "$invalid_base" "$invalid_base/$invalid_name"
  expect_validator_failure "generation name $invalid_name" "$invalid_base"
done

FILE_GENERATION_BASE="$TMP/file-generation-base"
mkdir -m 0700 "$FILE_GENERATION_BASE"
: > "$FILE_GENERATION_BASE/prepare.Ab12Cd"
expect_validator_failure "non-directory generation" "$FILE_GENERATION_BASE"

SYMLINK_GENERATION_BASE="$TMP/symlink-generation-base"
mkdir -m 0700 "$SYMLINK_GENERATION_BASE"
ln -s "$TRUSTED_BACKUP_BASE/prepare.Ab12Cd" "$SYMLINK_GENERATION_BASE/prepare.Ab12Cd"
expect_validator_failure "symlinked generation" "$SYMLINK_GENERATION_BASE"

WRONG_GENERATION_MODE_BASE="$TMP/wrong-generation-mode-base"
mkdir -m 0700 "$WRONG_GENERATION_MODE_BASE"
mkdir -m 0755 "$WRONG_GENERATION_MODE_BASE/prepare.Ab12Cd"
expect_validator_failure "wrong generation mode" "$WRONG_GENERATION_MODE_BASE"

FAKE_BIN="$TMP/bin"
SSH_LOG="$TMP/ssh.log"
mkdir -m 0755 "$FAKE_BIN"
: > "$SSH_LOG"

cat > "$FAKE_BIN/ssh" <<'FAKE_SSH'
#!/usr/bin/env bash
set -euo pipefail
target=$1
shift
command_text=$*
input=$(cat || true)
printf 'CALL command=%s\n' "$command_text" >> "$SSH_LOG"
if [[ $command_text == *'mktemp -d /var/tmp/alumni-maintenance-prestage.'* ]]; then
  printf '/var/tmp/alumni-maintenance-prestage.Ab12Cd\n'
elif [[ $command_text == *'tar -xf -'* || $command_text == *'sha256sum -c stage.sha256'* ||
        $command_text == *'rm -rf --'* ]]; then
  :
elif [[ $input == *'PRESTAGE_PRECHECK_MARKER=1'* ]]; then
  printf 'PASS maintenance_prestage precheck=stable sentinel=inactive services=active\n'
elif [[ $input == *'PRESTAGE_HTTP_PREPARE_MARKER=1'* ]]; then
  printf 'HTTP_PREPARE\n' >> "$SSH_LOG"
  printf 'PASS maintenance_prepare gate=installed sentinel=inactive backup=/var/backups/alumni-maintenance/prepare.Ab12Cd\n'
elif [[ $input == *'PRESTAGE_ENV_PREPARE_MARKER=1'* ]]; then
  printf 'ENV_PREPARE\n' >> "$SSH_LOG"
  printf 'PASS maintenance_env_prestage state=prepared backup=/var/backups/alumni-maintenance/env-prestage.Z9y8X7\n'
elif [[ $input == *'PRESTAGE_STATE_READ_MARKER=1'* ]]; then
  printf 'httpd_backup=/var/backups/alumni-maintenance/prepare.Ab12Cd\n'
  printf 'env_backup=/var/backups/alumni-maintenance/env-prestage.Z9y8X7\n'
elif [[ $input == *'PRESTAGE_STATE_WRITE_MARKER=1'* ]]; then
  :
elif [[ $input == *'PRESTAGE_POSTCHECK_MARKER=1'* ]]; then
  if [[ ${FAKE_VERIFY_FAIL:-0} == 1 ]]; then
    exit 44
  fi
  printf 'PASS maintenance_prestage sentinel=inactive services=active health=200 legacy=200\n'
elif [[ $input == *'PRESTAGE_ENV_ROLLBACK_MARKER=1'* ]]; then
  printf 'ENV_ROLLBACK\n' >> "$SSH_LOG"
  if [[ ${FAKE_ENV_ROLLBACK_FAIL:-0} == 1 ]]; then
    exit 45
  fi
  printf 'PASS maintenance_env_prestage state=rolled_back backup=/var/backups/alumni-maintenance/env-prestage.Z9y8X7\n'
elif [[ $input == *'PRESTAGE_HTTP_ROLLBACK_MARKER=1'* ]]; then
  printf 'HTTP_ROLLBACK\n' >> "$SSH_LOG"
  printf 'PASS maintenance_prestage rollback=complete services=active\n'
else
  printf 'unexpected fake ssh call target=%s\n' "$target" >&2
  exit 90
fi
FAKE_SSH

cat > "$FAKE_BIN/scp" <<'FAKE_SCP'
#!/usr/bin/env bash
set -euo pipefail
exit 0
FAKE_SCP

cat > "$FAKE_BIN/deploy" <<'FAKE_DEPLOY'
#!/usr/bin/env bash
set -euo pipefail
printf '✓ production EnvironmentFile validation passed\n'
printf '⚠ Migration drift detected — fixture\n' >&2
printf '  Backend was not built, uploaded, or restarted.\n' >&2
exit 1
FAKE_DEPLOY
chmod 0755 "$FAKE_BIN/ssh" "$FAKE_BIN/scp" "$FAKE_BIN/deploy"

COMMON_ENV=(
  MAINTENANCE_PRESTAGE_DRIVER_TEST_MODE=1
  MAINTENANCE_PRESTAGE_SSH_BIN="$FAKE_BIN/ssh"
  MAINTENANCE_PRESTAGE_SCP_BIN="$FAKE_BIN/scp"
  MAINTENANCE_PRESTAGE_DEPLOY_SCRIPT="$FAKE_BIN/deploy"
  SSH_LOG="$SSH_LOG"
)

set +e
UNAPPROVED_OUTPUT=$(env "${COMMON_ENV[@]}" "$DRIVER" prepare test-fixture 2>&1)
UNAPPROVED_STATUS=$?
set -e
[[ $UNAPPROVED_STATUS -ne 0 ]] || fail "unapproved driver did not fail closed"
[[ $UNAPPROVED_OUTPUT == *'reason=approval_required'* ]] || fail "unapproved failure reason is missing"
if grep -Fq 'HTTP_PREPARE' "$SSH_LOG"; then
  fail "unapproved driver reached production mutation"
fi

: > "$SSH_LOG"
set +e
FAILURE_OUTPUT=$(
  env "${COMMON_ENV[@]}" \
    MAINTENANCE_PRESTAGE_COMMAND_APPROVED=1 \
    FAKE_VERIFY_FAIL=1 \
    "$DRIVER" prepare test-fixture 2>&1
)
FAILURE_STATUS=$?
set -e
[[ $FAILURE_STATUS -ne 0 ]] || fail "postcheck failure returned success"
[[ $FAILURE_OUTPUT == *'rollback=complete'* ]] || fail "postcheck failure did not report completed rollback"
ENV_ROLLBACK_LINE=$(grep -n '^ENV_ROLLBACK$' "$SSH_LOG" | cut -d: -f1)
HTTP_ROLLBACK_LINE=$(grep -n '^HTTP_ROLLBACK$' "$SSH_LOG" | cut -d: -f1)
[[ -n $ENV_ROLLBACK_LINE && -n $HTTP_ROLLBACK_LINE && $ENV_ROLLBACK_LINE -lt $HTTP_ROLLBACK_LINE ]] ||
  fail "rollback order is not env then Apache/PHP"

: > "$SSH_LOG"
set +e
ROLLBACK_FAILURE_OUTPUT=$(
  env "${COMMON_ENV[@]}" \
    MAINTENANCE_PRESTAGE_COMMAND_APPROVED=1 \
    FAKE_VERIFY_FAIL=1 \
    FAKE_ENV_ROLLBACK_FAIL=1 \
    "$DRIVER" prepare test-fixture 2>&1
)
ROLLBACK_FAILURE_STATUS=$?
set -e
[[ $ROLLBACK_FAILURE_STATUS -eq 125 ]] || fail "incomplete rollback did not return recovery-required status"
grep -Fxq 'ENV_ROLLBACK' "$SSH_LOG" || fail "incomplete rollback did not attempt EnvironmentFile rollback"
grep -Fxq 'HTTP_ROLLBACK' "$SSH_LOG" || fail "EnvironmentFile rollback failure skipped Apache/PHP rollback"
[[ $ROLLBACK_FAILURE_OUTPUT == *'rollback=failed recovery_stage='* ]] ||
  fail "incomplete rollback did not retain a recovery stage"

: > "$SSH_LOG"
SUCCESS_OUTPUT=$(
  env "${COMMON_ENV[@]}" \
    MAINTENANCE_PRESTAGE_COMMAND_APPROVED=1 \
    FAKE_VERIFY_FAIL=0 \
    "$DRIVER" prepare test-fixture
)
[[ $SUCCESS_OUTPUT == *'PASS maintenance_prestage transaction=prepared'* ]] ||
  fail "successful driver did not return bounded PASS"
if grep -Eq '^(ENV_ROLLBACK|HTTP_ROLLBACK)$' "$SSH_LOG"; then
  fail "successful driver executed rollback"
fi

printf 'PASS: maintenance pre-stage fail-closed transaction driver\n'
