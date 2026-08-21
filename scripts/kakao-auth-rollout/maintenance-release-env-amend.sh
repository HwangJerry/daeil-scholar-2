#!/usr/bin/env bash
# maintenance-release-env-amend.sh — Provision/rollback release proof while canonical maintenance is active.
set -euo pipefail

ACTION=${1:-}
ROLLBACK_DIR=${2:-}
ENV_FILE=${MAINTENANCE_ENV_FILE_PATH:-/etc/sysconfig/alumni-backend}
SENTINEL=${MAINTENANCE_SENTINEL_PATH:-/run/alumni/maintenance}
BRIDGE=${MAINTENANCE_RELEASE_BRIDGE_PATH:-/run/alumni/maintenance-release-bridge}
PROOF_FILE=${MAINTENANCE_RELEASE_PROOF_FILE:-/run/alumni/maintenance-release-proof}
BACKUP_BASE=${MAINTENANCE_RELEASE_ENV_BACKUP_BASE:-/var/backups/alumni-maintenance-release}
DRAIN_TIMEOUT=90s
CANONICAL_ENV_FILE=/etc/sysconfig/alumni-backend
CANONICAL_SENTINEL=/run/alumni/maintenance
CANONICAL_BRIDGE=/run/alumni/maintenance-release-bridge
CANONICAL_PROOF_FILE=/run/alumni/maintenance-release-proof
current_uid=$(id -u)
backup_dir=
before_sha=
env_mode=
env_uid=
env_gid=
proof_created=0
env_replaced=0
transaction_active=0

file_mode() {
  stat -c '%a' "$1" 2>/dev/null || stat -f '%Lp' "$1"
}

file_uid() {
  stat -c '%u' "$1" 2>/dev/null || stat -f '%u' "$1"
}

file_gid() {
  stat -c '%g' "$1" 2>/dev/null || stat -f '%g' "$1"
}

file_group() {
  stat -c '%G' "$1" 2>/dev/null || stat -f '%Sg' "$1"
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d' ' -f1
  else
    shasum -a 256 "$1" | cut -d' ' -f1
  fi
}

sha256_text() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum | cut -d' ' -f1
  else
    shasum -a 256 | cut -d' ' -f1
  fi
}

fail() {
  printf 'ERROR maintenance_release_env_amend state=blocked reason=%s\n' "$1" >&2
  exit 1
}

validate_path() {
  [[ $1 =~ ^/[A-Za-z0-9._/-]+$ ]] || fail invalid_path
}

validate_active_sentinel() {
  local generation
  [[ -f $SENTINEL && ! -L $SENTINEL ]] || fail sentinel_not_active
  [[ $(file_uid "$SENTINEL") == "$current_uid" ]] || fail sentinel_owner_invalid
  [[ $(file_mode "$SENTINEL") == 644 ]] || fail sentinel_mode_invalid
  [[ $(wc -l < "$SENTINEL") -eq 2 ]] || fail sentinel_schema_invalid
  grep -Fxq 'state=active' "$SENTINEL" || fail sentinel_state_invalid
  [[ $(grep -c '^generation=' "$SENTINEL" || true) == 1 ]] || fail sentinel_generation_invalid
  generation=$(sed -n 's/^generation=//p' "$SENTINEL")
  [[ $generation =~ ^[a-f0-9]{32}$ ]] || fail sentinel_generation_invalid
  printf 'state=active\ngeneration=%s\n' "$generation" | cmp -s - "$SENTINEL" || fail sentinel_bytes_invalid
}

validate_common() {
  local runtime_parent
  [[ ${MAINTENANCE_RELEASE_ENV_AMEND_APPROVED:-0} == 1 ]] || fail approval_required
  [[ $ACTION == prepare || $ACTION == rollback ]] || fail invalid_action
  validate_path "$ENV_FILE"
  validate_path "$SENTINEL"
  validate_path "$BRIDGE"
  validate_path "$PROOF_FILE"
  validate_path "$BACKUP_BASE"
  if [[ $ENV_FILE == "$CANONICAL_ENV_FILE" || $SENTINEL == "$CANONICAL_SENTINEL" ||
        $BRIDGE == "$CANONICAL_BRIDGE" || $PROOF_FILE == "$CANONICAL_PROOF_FILE" ]]; then
    [[ $ENV_FILE == "$CANONICAL_ENV_FILE" && $SENTINEL == "$CANONICAL_SENTINEL" &&
       $BRIDGE == "$CANONICAL_BRIDGE" && $PROOF_FILE == "$CANONICAL_PROOF_FILE" ]] ||
      fail canonical_paths_must_be_bound
    [[ $current_uid == 0 ]] || fail canonical_paths_require_root
  fi
  runtime_parent=$(dirname "$SENTINEL")
  [[ $(dirname "$BRIDGE") == "$runtime_parent" && $(dirname "$PROOF_FILE") == "$runtime_parent" ]] ||
    fail runtime_parent_mismatch
  [[ -d $runtime_parent && ! -L $runtime_parent ]] || fail runtime_parent_invalid
  [[ $(file_uid "$runtime_parent") == "$current_uid" ]] || fail runtime_parent_owner_invalid
  [[ $(file_mode "$runtime_parent") == 755 ]] || fail runtime_parent_mode_invalid
  [[ ! -e $BRIDGE && ! -L $BRIDGE ]] || fail release_bridge_must_be_absent
  validate_active_sentinel
}

validate_environment() {
  [[ -f $ENV_FILE && ! -L $ENV_FILE ]] || fail environment_file_invalid
  env_mode=$(file_mode "$ENV_FILE")
  env_uid=$(file_uid "$ENV_FILE")
  env_gid=$(file_gid "$ENV_FILE")
  [[ $env_uid == "$current_uid" ]] || fail environment_owner_invalid
  if [[ $env_mode == 640 ]]; then
    [[ $(file_group "$ENV_FILE") == alumni-backend ]] || fail environment_group_invalid
  else
    [[ $env_mode == 600 ]] || fail environment_mode_invalid
  fi
}

remove_owned_proof() {
  [[ ! -d $PROOF_FILE || -L $PROOF_FILE ]] || return 1
  rm -f -- "$PROOF_FILE" || return 1
  [[ ! -e $PROOF_FILE && ! -L $PROOF_FILE ]]
}

restore_prepare_state() {
  local restore_tmp
  if [[ $env_replaced == 1 && -f $backup_dir/alumni-backend.env && ! -L $backup_dir/alumni-backend.env ]]; then
    if [[ $(sha256_file "$backup_dir/alumni-backend.env") != "$before_sha" ]]; then
      return 1
    fi
    restore_tmp=$(mktemp "$(dirname "$ENV_FILE")/.alumni-backend.env.rollback.XXXXXX") || return 1
    rm -f "$restore_tmp" || return 1
    cp -p "$backup_dir/alumni-backend.env" "$restore_tmp" || { rm -f "$restore_tmp"; return 1; }
    mv -f "$restore_tmp" "$ENV_FILE" || { rm -f "$restore_tmp"; return 1; }
  fi
  if [[ $proof_created == 1 ]]; then
    remove_owned_proof || return 1
  fi
}

on_exit() {
  local status=$?
  if [[ $transaction_active == 1 && $status -ne 0 ]]; then
    if ! restore_prepare_state; then
      printf 'ERROR maintenance_release_env_amend rollback=failed backup=%s\n' "$backup_dir" >&2
    fi
  fi
  exit "$status"
}
trap on_exit EXIT
trap 'exit 1' HUP INT TERM

prepare() {
  local proof_tmp proof proof_sha env_tmp manifest
  validate_environment
  [[ ! -e $PROOF_FILE && ! -L $PROOF_FILE ]] || fail release_proof_must_be_absent
  if [[ -e $BACKUP_BASE ]]; then
    [[ -d $BACKUP_BASE && ! -L $BACKUP_BASE ]] || fail backup_base_invalid
    [[ $(file_uid "$BACKUP_BASE") == "$current_uid" ]] || fail backup_base_owner_invalid
    [[ $(file_mode "$BACKUP_BASE") == 700 ]] || fail backup_base_mode_invalid
  else
    install -d -m 0700 "$BACKUP_BASE" || fail backup_base_create_failed
  fi
  backup_dir=$(mktemp -d "$BACKUP_BASE/release-env-amend.XXXXXX") || fail backup_create_failed
  chmod 0700 "$backup_dir" || fail backup_mode_failed
  cp -p "$ENV_FILE" "$backup_dir/alumni-backend.env" || fail backup_copy_failed
  before_sha=$(sha256_file "$backup_dir/alumni-backend.env") || fail backup_hash_failed
  [[ $before_sha =~ ^[a-f0-9]{64}$ ]] || fail backup_hash_invalid
  transaction_active=1

  proof_tmp=$(mktemp "${PROOF_FILE}.tmp.XXXXXX") || fail proof_temp_create_failed
  chmod 0600 "$proof_tmp" || fail proof_temp_mode_failed
  proof=$(openssl rand -hex 32) || fail proof_generation_failed
  [[ $proof =~ ^[a-f0-9]{64}$ ]] || fail proof_format_invalid
  printf '%s\n' "$proof" > "$proof_tmp" || fail proof_write_failed
  proof_sha=$(printf '%s' "$proof" | sha256_text) || fail proof_hash_failed
  unset proof
  [[ $proof_sha =~ ^[a-f0-9]{64}$ ]] || fail proof_hash_invalid
  ln "$proof_tmp" "$PROOF_FILE" || fail proof_install_failed
  rm -f "$proof_tmp" || fail proof_temp_remove_failed
  proof_created=1
  [[ -f $PROOF_FILE && ! -L $PROOF_FILE && $(file_mode "$PROOF_FILE") == 600 ]] || fail proof_metadata_invalid
  [[ $(file_uid "$PROOF_FILE") == "$current_uid" && $(wc -c < "$PROOF_FILE") -eq 65 ]] || fail proof_custody_invalid

  env_tmp=$(mktemp "$(dirname "$ENV_FILE")/.alumni-backend.env.release.XXXXXX") || fail environment_temp_create_failed
  chmod 0600 "$env_tmp" || fail environment_temp_mode_failed
  # PHP variables are intentionally protected from shell expansion.
  # shellcheck disable=SC2016
  php -r '
$source = $argv[1];
$target = $argv[2];
$updates = array(
    "MAINTENANCE_RELEASE_BRIDGE_PATH" => $argv[3],
    "MAINTENANCE_RELEASE_PROOF_SHA256" => $argv[4],
    "MAINTENANCE_RELEASE_OWNER_UID" => $argv[5],
    "MAINTENANCE_RELEASE_DRAIN_TIMEOUT" => $argv[6]
);
$lines = @file($source);
if ($lines === false) { exit(10); }
$seen = array();
foreach ($updates as $key => $_) { $seen[$key] = 0; }
$out = array();
foreach ($lines as $line) {
    $matched = false;
    foreach ($updates as $key => $value) {
        if (preg_match("/^" . preg_quote($key, "/") . "=/", $line) === 1) {
            $seen[$key]++;
            if ($seen[$key] > 1) { exit(11); }
            $out[] = $key . "=" . $value . "\n";
            $matched = true;
            break;
        }
    }
    if (!$matched) { $out[] = $line; }
}
if (count($out) > 0 && substr($out[count($out) - 1], -1) !== "\n") { $out[] = "\n"; }
foreach ($updates as $key => $value) {
    if ($seen[$key] === 0) { $out[] = $key . "=" . $value . "\n"; }
}
if (@file_put_contents($target, implode("", $out), LOCK_EX) === false) { exit(12); }
' "$ENV_FILE" "$env_tmp" "$BRIDGE" "$proof_sha" "$current_uid" "$DRAIN_TIMEOUT" || fail environment_update_failed
  if [[ $current_uid == 0 ]]; then
    chown "$env_uid:$env_gid" "$env_tmp" || fail environment_temp_owner_failed
  fi
  chmod "$env_mode" "$env_tmp" || fail environment_temp_restore_mode_failed
  mv -f "$env_tmp" "$ENV_FILE" || fail environment_install_failed
  env_replaced=1

  # shellcheck disable=SC2016
  php -r '
$env = @parse_ini_file($argv[1], false, INI_SCANNER_RAW);
if ($env === false) { exit(20); }
$expected = array(
    "MAINTENANCE_RELEASE_BRIDGE_PATH" => $argv[2],
    "MAINTENANCE_RELEASE_PROOF_SHA256" => $argv[3],
    "MAINTENANCE_RELEASE_OWNER_UID" => $argv[4],
    "MAINTENANCE_RELEASE_DRAIN_TIMEOUT" => $argv[5]
);
foreach ($expected as $key => $value) {
    if (!isset($env[$key]) || (string)$env[$key] !== $value) { exit(21); }
}
' "$ENV_FILE" "$BRIDGE" "$proof_sha" "$current_uid" "$DRAIN_TIMEOUT" || fail environment_validation_failed
  validate_active_sentinel
  [[ ! -e $BRIDGE && ! -L $BRIDGE ]] || fail release_bridge_created_during_prepare

  manifest="$backup_dir/manifest"
  umask 077
  {
    printf 'state=prepared\n'
    printf 'environment_file=%s\n' "$ENV_FILE"
    printf 'sentinel=%s\n' "$SENTINEL"
    printf 'bridge=%s\n' "$BRIDGE"
    printf 'proof_file=%s\n' "$PROOF_FILE"
    printf 'before_sha256=%s\n' "$before_sha"
    printf 'proof_sha256=%s\n' "$proof_sha"
  } > "$manifest" || fail manifest_write_failed
  chmod 0600 "$manifest" || fail manifest_mode_failed
  transaction_active=0
  printf 'PASS maintenance_release_env_amend state=prepared backup=%s\n' "$backup_dir"
}

rollback() {
  local manifest backup before proof_sha restore_tmp current_proof
  [[ $ROLLBACK_DIR == "$BACKUP_BASE"/release-env-amend.* ]] || fail rollback_path_invalid
  [[ -d $ROLLBACK_DIR && ! -L $ROLLBACK_DIR ]] || fail rollback_directory_invalid
  [[ $(file_uid "$ROLLBACK_DIR") == "$current_uid" && $(file_mode "$ROLLBACK_DIR") == 700 ]] ||
    fail rollback_directory_custody_invalid
  manifest="$ROLLBACK_DIR/manifest"
  backup="$ROLLBACK_DIR/alumni-backend.env"
  [[ -f $manifest && ! -L $manifest && $(file_mode "$manifest") == 600 ]] || fail manifest_invalid
  [[ -f $backup && ! -L $backup ]] || fail backup_invalid
  [[ $(grep -c '^before_sha256=' "$manifest" || true) == 1 ]] || fail manifest_before_hash_invalid
  [[ $(grep -c '^proof_sha256=' "$manifest" || true) == 1 ]] || fail manifest_proof_hash_invalid
  before=$(sed -n 's/^before_sha256=//p' "$manifest")
  proof_sha=$(sed -n 's/^proof_sha256=//p' "$manifest")
  [[ $before =~ ^[a-f0-9]{64}$ && $proof_sha =~ ^[a-f0-9]{64}$ ]] || fail manifest_hash_invalid
  [[ $(sha256_file "$backup") == "$before" ]] || fail backup_hash_mismatch
  validate_environment
  restore_tmp=$(mktemp "$(dirname "$ENV_FILE")/.alumni-backend.env.rollback.XXXXXX") || fail rollback_temp_create_failed
  rm -f "$restore_tmp" || fail rollback_temp_reserve_failed
  cp -p "$backup" "$restore_tmp" || fail rollback_copy_failed
  mv -f "$restore_tmp" "$ENV_FILE" || fail rollback_install_failed
  if [[ -e $PROOF_FILE || -L $PROOF_FILE ]]; then
    [[ -f $PROOF_FILE && ! -L $PROOF_FILE && $(file_mode "$PROOF_FILE") == 600 ]] || fail rollback_proof_invalid
    current_proof=$(<"$PROOF_FILE") || fail rollback_proof_read_failed
    [[ $current_proof =~ ^[a-f0-9]{64}$ ]] || fail rollback_proof_format_invalid
    [[ $(printf '%s' "$current_proof" | sha256_text) == "$proof_sha" ]] || fail rollback_proof_hash_mismatch
    unset current_proof
    remove_owned_proof || fail rollback_proof_remove_failed
  fi
  [[ $(sha256_file "$ENV_FILE") == "$before" ]] || fail rollback_environment_hash_mismatch
  validate_active_sentinel
  [[ ! -e $BRIDGE && ! -L $BRIDGE ]] || fail rollback_bridge_created
  printf 'PASS maintenance_release_env_amend state=rolled_back backup=%s\n' "$ROLLBACK_DIR"
}

validate_common
case "$ACTION" in
  prepare) prepare ;;
  rollback) rollback ;;
esac
