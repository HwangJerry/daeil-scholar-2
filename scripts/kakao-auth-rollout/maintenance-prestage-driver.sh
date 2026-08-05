#!/usr/bin/env bash
# maintenance-prestage-driver.sh — Execute the inactive production pre-stage as one rollback-bound transaction.
set -euo pipefail

ACTION=${1:-}
TARGET=${2:-}
RECOVERY_STAGE=${3:-}
ROOT=$(git rev-parse --show-toplevel)
TEST_MODE=${MAINTENANCE_PRESTAGE_DRIVER_TEST_MODE:-0}
transaction_committed=0
mutation_started=0
rolling_back=0
stage=
manifest=
httpd_backup=
env_backup=

fail() {
  printf 'ERROR maintenance_prestage transaction=blocked reason=%s\n' "$1" >&2
  return 1
}

if [[ $TEST_MODE == 1 ]]; then
  [[ $TARGET == test-fixture ]] || fail test_target_invalid
  SSH_BIN=${MAINTENANCE_PRESTAGE_SSH_BIN:?}
  SCP_BIN=${MAINTENANCE_PRESTAGE_SCP_BIN:?}
  TAR_BIN=${MAINTENANCE_PRESTAGE_TAR_BIN:-/usr/bin/tar}
  DEPLOY_SCRIPT=${MAINTENANCE_PRESTAGE_DEPLOY_SCRIPT:?}
else
  [[ $TARGET == daeil-prod ]] || fail production_target_invalid
  SSH_BIN=/usr/bin/ssh
  SCP_BIN=/usr/bin/scp
  TAR_BIN=/usr/bin/tar
  DEPLOY_SCRIPT="$ROOT/deploy.sh"
fi

validate_stage() {
  [[ $1 =~ ^/var/tmp/alumni-maintenance-prestage\.[A-Za-z0-9]{6}$ ]]
}

validate_httpd_backup() {
  [[ $1 =~ ^/var/backups/alumni-maintenance/prepare\.[A-Za-z0-9]{6}$ ]]
}

validate_env_backup() {
  [[ $1 =~ ^/var/backups/alumni-maintenance/env-prestage\.[A-Za-z0-9]{6}$ ]]
}

read_remote_state() {
  local output
  output=$(
    "$SSH_BIN" "$TARGET" 'bash -s' -- "$stage/transaction.state" <<'REMOTE_STATE_READ'
set -euo pipefail
PRESTAGE_STATE_READ_MARKER=1
STATE_FILE=$1
[[ -f $STATE_FILE && ! -L $STATE_FILE ]]
[[ $(stat -c '%u:%a' "$STATE_FILE") == 0:600 ]]
[[ $(grep -c '^httpd_backup=' "$STATE_FILE") == 1 ]]
[[ $(grep -c '^env_backup=' "$STATE_FILE") == 1 ]]
printf 'httpd_backup=%s\n' "$(sed -n 's/^httpd_backup=//p' "$STATE_FILE")"
printf 'env_backup=%s\n' "$(sed -n 's/^env_backup=//p' "$STATE_FILE")"
REMOTE_STATE_READ
  ) || return 1
  [[ $(grep -c '^httpd_backup=' <<< "$output") == 1 ]] || return 1
  [[ $(grep -c '^env_backup=' <<< "$output") == 1 ]] || return 1
  httpd_backup=$(sed -n 's/^httpd_backup=//p' <<< "$output")
  env_backup=$(sed -n 's/^env_backup=//p' <<< "$output")
  validate_httpd_backup "$httpd_backup" || return 1
  if [[ -n $env_backup ]]; then
    validate_env_backup "$env_backup" || return 1
  fi
}

rollback_env() {
  [[ -n $env_backup ]] || return 0
  local output
  output=$(
    "$SSH_BIN" "$TARGET" 'bash -s' -- "$stage" "$env_backup" <<'REMOTE_ENV_ROLLBACK'
set -euo pipefail
PRESTAGE_ENV_ROLLBACK_MARKER=1
STAGE=$1
ENV_BACKUP=$2
[[ $STAGE =~ ^/var/tmp/alumni-maintenance-prestage\.[A-Za-z0-9]{6}$ ]]
[[ $ENV_BACKUP =~ ^/var/backups/alumni-maintenance/env-prestage\.[A-Za-z0-9]{6}$ ]]
env \
  MAINTENANCE_ENV_PRESTAGE_APPROVED=1 \
  MAINTENANCE_ENV_FILE_PATH=/etc/sysconfig/alumni-backend \
  MAINTENANCE_SENTINEL_PATH=/run/alumni/maintenance \
  MAINTENANCE_SMOKE_PROOF_FILE=/run/alumni/maintenance-smoke-proof \
  MAINTENANCE_ENV_BACKUP_BASE=/var/backups/alumni-maintenance \
  "$STAGE/scripts/kakao-auth-rollout/maintenance-env-prestage.sh" rollback "$ENV_BACKUP"
REMOTE_ENV_ROLLBACK
  ) || return 1
  [[ $output == 'PASS maintenance_env_prestage state=rolled_back backup='* ]]
}

rollback_httpd() {
  [[ -n $httpd_backup ]] || return 1
  local output
  output=$(
    "$SSH_BIN" "$TARGET" 'bash -s' -- "$httpd_backup" <<'REMOTE_HTTPD_ROLLBACK'
set -euo pipefail
PRESTAGE_HTTP_ROLLBACK_MARKER=1
BACKUP=$1
[[ $BACKUP =~ ^/var/backups/alumni-maintenance/prepare\.[A-Za-z0-9]{6}$ ]]
[[ -d $BACKUP && ! -L $BACKUP ]]
[[ $(stat -c '%u:%a' "$BACKUP") == 0:700 ]]
for entry in \
  '_set_docroot.php:/var/www/html/_set_docroot.php' \
  '_maintenance_gate.php:/var/www/html/_maintenance_gate.php' \
  '_legacy_docroot.php:/var/www/html/_legacy_docroot.php' \
  '_legacy_url_rewriter.php:/var/www/html/_legacy_url_rewriter.php' \
  'httpd-alumni.conf:/etc/httpd/conf.d/alumni.conf'; do
  name=${entry%%:*}
  target=${entry#*:}
  if [[ -f $BACKUP/$name.present ]]; then
    [[ -f $BACKUP/$name && ! -L $BACKUP/$name ]]
    install -m 0644 "$BACKUP/$name" "$target"
  else
    [[ -f $BACKUP/$name.absent ]]
    rm -f "$target"
  fi
done
rm -f /run/alumni/maintenance-smoke-proof
rmdir /run/alumni
[[ ! -e /run/alumni && ! -L /run/alumni ]]
[[ $(sha256sum /etc/httpd/conf.d/alumni.conf | cut -d' ' -f1) == 47c07a773072c01ca5bd5d4688f44c2691a50010ab4ce7e81e4ad73b80913705 ]]
[[ $(sha256sum /var/www/html/_set_docroot.php | cut -d' ' -f1) == 74fbc635a3495f9fe1b0768a8a6da664c2241474adc6c87c97feee25c794db9b ]]
[[ ! -e /var/www/html/_maintenance_gate.php && ! -L /var/www/html/_maintenance_gate.php ]]
[[ $(sha256sum /var/www/html/_legacy_docroot.php | cut -d' ' -f1) == e5b5c915a0ab98a5129efdc5e182f10960dbd0dd65a00d9255ead4c0c0067886 ]]
[[ $(sha256sum /var/www/html/_legacy_url_rewriter.php | cut -d' ' -f1) == b10d17dc754f5f22e570376f8378af73d07ce6ba7c0a7e200da0893d71abc43b ]]
for key in MAINTENANCE_SENTINEL_PATH MAINTENANCE_SMOKE_PROOF_SHA256 MAINTENANCE_SMOKE_ALLOWED_PATHS; do
  [[ $(grep -c "^${key}=" /etc/sysconfig/alumni-backend || true) == 0 ]]
done
httpd -t >/dev/null
systemctl reload httpd
systemctl is-active --quiet httpd
systemctl is-active --quiet alumni-backend
[[ $(curl --disable --noproxy '*' --silent --show-error --max-time 10 --output /dev/null --write-out '%{http_code}' http://127.0.0.1:8080/api/health) == 200 ]]
[[ $(curl --disable --noproxy '*' --silent --show-error --max-time 10 --output /dev/null --write-out '%{http_code}' --resolve daeilfoundation.or.kr:443:127.0.0.1 https://daeilfoundation.or.kr/old/index.php) == 200 ]]
printf 'PASS maintenance_prestage rollback=complete services=active\n'
REMOTE_HTTPD_ROLLBACK
  ) || return 1
  [[ $output == 'PASS maintenance_prestage rollback=complete services=active' ]]
}

cleanup_stage() {
  [[ -n $stage ]] || return 0
  validate_stage "$stage" || return 1
  "$SSH_BIN" "$TARGET" "rm -rf -- '$stage'" </dev/null
}

rollback_transaction() {
  local failed=0
  [[ $rolling_back == 0 ]] || return 1
  rolling_back=1
  if ! read_remote_state; then
    validate_httpd_backup "$httpd_backup" || failed=1
    if [[ -n $env_backup ]]; then
      validate_env_backup "$env_backup" || failed=1
    fi
  fi
  if [[ $failed == 0 && -n $env_backup ]]; then
    rollback_env || failed=1
  fi
  rollback_httpd || failed=1
  if [[ $failed == 0 ]]; then
    cleanup_stage || failed=1
  fi
  if [[ $failed == 0 ]]; then
    printf 'PASS maintenance_prestage rollback=complete\n' >&2
    return 0
  fi
  printf 'ERROR maintenance_prestage rollback=failed recovery_stage=%s\n' "$stage" >&2
  return 1
}

on_exit() {
  local status=$?
  trap - EXIT INT TERM
  [[ -z $manifest ]] || rm -f "$manifest"
  if [[ $status -ne 0 && $mutation_started == 1 && $transaction_committed == 0 ]]; then
    if rollback_transaction; then
      exit "$status"
    fi
    exit 125
  fi
  if [[ $status -ne 0 && $mutation_started == 0 && -n $stage ]]; then
    cleanup_stage || true
  fi
  exit "$status"
}
trap on_exit EXIT
trap 'exit 130' INT TERM

[[ ${MAINTENANCE_PRESTAGE_COMMAND_APPROVED:-0} == 1 ]] || fail approval_required
[[ $ACTION == prepare || $ACTION == rollback ]] || fail invalid_action

if [[ $ACTION == rollback ]]; then
  stage=$RECOVERY_STAGE
  validate_stage "$stage" || fail recovery_stage_invalid
  mutation_started=1
  rollback_transaction
  transaction_committed=1
  trap - EXIT INT TERM
  exit 0
fi

files=(
  scripts/kakao-auth-rollout/maintenance-backup-base-validate.sh
  scripts/kakao-auth-rollout/maintenance-env-prestage.sh
  scripts/kakao-auth-rollout/maintenance-prepare.sh
  deploy/httpd-alumni.conf
  deploy/_maintenance_gate.php
  deploy/_legacy_docroot.php
  deploy/_legacy_url_rewriter.php
  deploy/_set_docroot.php
)
stage=$("$SSH_BIN" "$TARGET" 'umask 077; mktemp -d /var/tmp/alumni-maintenance-prestage.XXXXXX' </dev/null)
validate_stage "$stage" || fail stage_path_invalid
manifest=$(mktemp "${TMPDIR:-/tmp}/maintenance-prestage-manifest.XXXXXX")
chmod 0600 "$manifest"
(
  cd "$ROOT"
  shasum -a 256 "${files[@]}"
) > "$manifest"
"$TAR_BIN" -C "$ROOT" -cf - "${files[@]}" | "$SSH_BIN" "$TARGET" "tar -xf - -C '$stage'"
"$SCP_BIN" -q "$manifest" "$TARGET:$stage/stage.sha256"
"$SSH_BIN" "$TARGET" "cd '$stage' && sha256sum -c stage.sha256 >/dev/null" </dev/null

"$SSH_BIN" "$TARGET" 'bash -s' -- "$stage" <<'REMOTE_PRECHECK'
set -euo pipefail
PRESTAGE_PRECHECK_MARKER=1
STAGE=$1
[[ $STAGE =~ ^/var/tmp/alumni-maintenance-prestage\.[A-Za-z0-9]{6}$ ]]
[[ ! -e /run/alumni && ! -L /run/alumni ]]
[[ ! -e /run/alumni/maintenance && ! -L /run/alumni/maintenance ]]
[[ ! -e /run/alumni/maintenance-smoke-proof && ! -L /run/alumni/maintenance-smoke-proof ]]
MAINTENANCE_BACKUP_VALIDATE_APPROVED=1 \
  "$STAGE/scripts/kakao-auth-rollout/maintenance-backup-base-validate.sh" \
  /var/backups/alumni-maintenance >/dev/null
[[ -f /etc/sysconfig/alumni-backend && ! -L /etc/sysconfig/alumni-backend ]]
[[ $(stat -c '%u:%g:%a' /etc/sysconfig/alumni-backend) == 0:991:640 ]]
for key in MAINTENANCE_SENTINEL_PATH MAINTENANCE_SMOKE_PROOF_SHA256 MAINTENANCE_SMOKE_ALLOWED_PATHS; do
  [[ $(grep -c "^${key}=" /etc/sysconfig/alumni-backend || true) == 0 ]]
done
[[ $(sha256sum /etc/httpd/conf.d/alumni.conf | cut -d' ' -f1) == 47c07a773072c01ca5bd5d4688f44c2691a50010ab4ce7e81e4ad73b80913705 ]]
[[ $(sha256sum /var/www/html/_set_docroot.php | cut -d' ' -f1) == 74fbc635a3495f9fe1b0768a8a6da664c2241474adc6c87c97feee25c794db9b ]]
[[ ! -e /var/www/html/_maintenance_gate.php && ! -L /var/www/html/_maintenance_gate.php ]]
[[ $(sha256sum /var/www/html/_legacy_docroot.php | cut -d' ' -f1) == e5b5c915a0ab98a5129efdc5e182f10960dbd0dd65a00d9255ead4c0c0067886 ]]
[[ $(sha256sum /var/www/html/_legacy_url_rewriter.php | cut -d' ' -f1) == b10d17dc754f5f22e570376f8378af73d07ce6ba7c0a7e200da0893d71abc43b ]]
httpd -t >/dev/null
systemctl is-active --quiet httpd
systemctl is-active --quiet alumni-backend
[[ $(curl --disable --noproxy '*' --silent --show-error --max-time 10 --output /dev/null --write-out '%{http_code}' http://127.0.0.1:8080/api/health) == 200 ]]
[[ $(curl --disable --noproxy '*' --silent --show-error --max-time 10 --output /dev/null --write-out '%{http_code}' --resolve daeilfoundation.or.kr:443:127.0.0.1 https://daeilfoundation.or.kr/old/index.php) == 200 ]]
printf 'PASS maintenance_prestage precheck=stable sentinel=inactive services=active\n'
REMOTE_PRECHECK

mutation_started=1
httpd_output=$(
  "$SSH_BIN" "$TARGET" 'bash -s' -- "$stage" <<'REMOTE_HTTPD_PREPARE'
set -euo pipefail
PRESTAGE_HTTP_PREPARE_MARKER=1
STAGE=$1
[[ $STAGE =~ ^/var/tmp/alumni-maintenance-prestage\.[A-Za-z0-9]{6}$ ]]
env \
  MAINTENANCE_PREPARE_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH=/run/alumni/maintenance \
  MAINTENANCE_HTTPD_SOURCE="$STAGE/deploy/httpd-alumni.conf" \
  MAINTENANCE_HTTPD_TARGET=/etc/httpd/conf.d/alumni.conf \
  MAINTENANCE_SHIM_SOURCE_DIR="$STAGE/deploy" \
  MAINTENANCE_SHIM_TARGET_DIR=/var/www/html \
  MAINTENANCE_BACKUP_BASE=/var/backups/alumni-maintenance \
  MAINTENANCE_PRESTAGE_STATE_FILE="$STAGE/transaction.state" \
  bash "$STAGE/scripts/kakao-auth-rollout/maintenance-prepare.sh"
REMOTE_HTTPD_PREPARE
)
[[ $httpd_output == 'PASS maintenance_prepare gate=installed sentinel=inactive backup='* ]] || fail httpd_prepare_output_invalid
httpd_backup=${httpd_output##*backup=}
validate_httpd_backup "$httpd_backup" || fail httpd_backup_invalid

env_output=$(
  "$SSH_BIN" "$TARGET" 'bash -s' -- "$stage" "$httpd_backup" <<'REMOTE_ENV_PREPARE'
set -euo pipefail
PRESTAGE_ENV_PREPARE_MARKER=1
STAGE=$1
HTTPD_BACKUP=$2
[[ $STAGE =~ ^/var/tmp/alumni-maintenance-prestage\.[A-Za-z0-9]{6}$ ]]
[[ $HTTPD_BACKUP =~ ^/var/backups/alumni-maintenance/prepare\.[A-Za-z0-9]{6}$ ]]
env \
  MAINTENANCE_ENV_PRESTAGE_APPROVED=1 \
  MAINTENANCE_ENV_FILE_PATH=/etc/sysconfig/alumni-backend \
  MAINTENANCE_SENTINEL_PATH=/run/alumni/maintenance \
  MAINTENANCE_SMOKE_PROOF_FILE=/run/alumni/maintenance-smoke-proof \
  MAINTENANCE_ENV_BACKUP_BASE=/var/backups/alumni-maintenance \
  MAINTENANCE_PRESTAGE_STATE_FILE="$STAGE/transaction.state" \
  MAINTENANCE_PRESTAGE_HTTPD_BACKUP="$HTTPD_BACKUP" \
  "$STAGE/scripts/kakao-auth-rollout/maintenance-env-prestage.sh" prepare
REMOTE_ENV_PREPARE
)
[[ $env_output == 'PASS maintenance_env_prestage state=prepared backup='* ]] || fail env_prepare_output_invalid
env_backup=${env_output##*backup=}
validate_env_backup "$env_backup" || fail env_backup_invalid

"$SSH_BIN" "$TARGET" 'bash -s' <<'REMOTE_POSTCHECK'
set -euo pipefail
PRESTAGE_POSTCHECK_MARKER=1
[[ ! -e /run/alumni/maintenance && ! -L /run/alumni/maintenance ]]
[[ $(stat -c '%u:%a' /run/alumni) == 0:755 ]]
[[ -f /run/alumni/maintenance-smoke-proof && ! -L /run/alumni/maintenance-smoke-proof ]]
[[ $(stat -c '%u:%a' /run/alumni/maintenance-smoke-proof) == 0:600 ]]
for shim in _maintenance_gate.php _legacy_docroot.php _legacy_url_rewriter.php _set_docroot.php; do
  php -l "/var/www/html/$shim" >/dev/null
done
httpd -t >/dev/null
systemctl is-active --quiet httpd
systemctl is-active --quiet alumni-backend
[[ $(curl --disable --noproxy '*' --silent --show-error --max-time 10 --output /dev/null --write-out '%{http_code}' http://127.0.0.1:8080/api/health) == 200 ]]
[[ $(curl --disable --noproxy '*' --silent --show-error --max-time 10 --output /dev/null --write-out '%{http_code}' --resolve daeilfoundation.or.kr:443:127.0.0.1 https://daeilfoundation.or.kr/old/index.php) == 200 ]]
printf 'PASS maintenance_prestage sentinel=inactive services=active health=200 legacy=200\n'
REMOTE_POSTCHECK

set +e
deploy_preflight_output=$("$DEPLOY_SCRIPT" "$TARGET" --backend=true --frontend=false --preflight-only 2>&1)
deploy_preflight_status=$?
set -e
[[ $deploy_preflight_status -eq 1 ]] || fail deploy_preflight_status_unexpected
grep -Fq 'production EnvironmentFile validation passed' <<< "$deploy_preflight_output" || fail environment_preflight_marker_missing
grep -Fq 'Migration drift detected' <<< "$deploy_preflight_output" || fail migration_drift_marker_missing
grep -Fq 'Backend was not built, uploaded, or restarted.' <<< "$deploy_preflight_output" || fail mutation_boundary_marker_missing
if grep -Fq 'Production EnvironmentFile validation failed' <<< "$deploy_preflight_output"; then
  fail environment_preflight_failed
fi

transaction_committed=1
cleanup_stage || fail stage_cleanup_failed
trap - EXIT INT TERM
rm -f "$manifest"
printf 'PASS maintenance_prestage transaction=prepared httpd_backup=%s env_backup=%s\n' "$httpd_backup" "$env_backup"
