#!/usr/bin/env bash
# maintenance-env-prestage.sh — Atomically pre-stage or roll back maintenance env/proof state.
set -euo pipefail

ACTION=${1:-}
ROLLBACK_DIR=${2:-}
ENV_FILE=${MAINTENANCE_ENV_FILE_PATH:-/etc/sysconfig/alumni-backend}
SENTINEL=${MAINTENANCE_SENTINEL_PATH:-/run/alumni/maintenance}
PROOF_FILE=${MAINTENANCE_SMOKE_PROOF_FILE:-/run/alumni/maintenance-smoke-proof}
BACKUP_BASE=${MAINTENANCE_ENV_BACKUP_BASE:-/var/backups/alumni-maintenance}
STATE_FILE=${MAINTENANCE_PRESTAGE_STATE_FILE:-}
HTTPD_BACKUP=${MAINTENANCE_PRESTAGE_HTTPD_BACKUP:-}
CANONICAL_ENV_FILE=/etc/sysconfig/alumni-backend
CANONICAL_SENTINEL=/run/alumni/maintenance
CANONICAL_PROOF_FILE=/run/alumni/maintenance-smoke-proof
ALLOWED_PATHS=/api/auth/login,/api/auth/logout
current_uid=$(id -u)
backup_dir=
proof_created=0
env_replaced=0

fail() {
  local reason=$1
  local rollback_failed=0
  if [[ $ACTION == prepare && -n $backup_dir ]]; then
    if [[ $env_replaced == 1 && -f $backup_dir/alumni-backend.env ]]; then
      restore_environment_from_backup "$backup_dir/alumni-backend.env" "$before_sha" || rollback_failed=1
    fi
    if [[ $proof_created == 1 ]]; then
      remove_proof_safely || rollback_failed=1
    fi
    if [[ $rollback_failed == 1 ]]; then
      printf 'ERROR maintenance_env_prestage rollback=failed backup=%s\n' "$backup_dir" >&2
    fi
  fi
  printf 'ERROR maintenance_env_prestage state=blocked reason=%s\n' "$reason" >&2
  exit 1
}

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

remove_proof_safely() {
  [[ ! -d $PROOF_FILE || -L $PROOF_FILE ]] || return 1
  rm -f -- "$PROOF_FILE" || return 1
  [[ ! -e $PROOF_FILE && ! -L $PROOF_FILE ]]
}

restore_environment_from_backup() {
  local backup=$1
  local expected_sha=$2
  local backup_uid backup_gid backup_mode env_parent env_name restore_tmp

  [[ -f $backup && ! -L $backup ]] || return 1
  [[ $expected_sha =~ ^[a-f0-9]{64}$ ]] || return 1
  [[ $(sha256_file "$backup") == "$expected_sha" ]] || return 1
  backup_uid=$(file_uid "$backup") || return 1
  backup_gid=$(file_gid "$backup") || return 1
  backup_mode=$(file_mode "$backup") || return 1
  [[ $backup_uid == "$current_uid" ]] || return 1
  [[ $backup_mode == 600 || $backup_mode == 640 ]] || return 1

  env_parent=$(dirname "$ENV_FILE")
  env_name=$(basename "$ENV_FILE")
  [[ -d $env_parent && ! -L $env_parent ]] || return 1
  [[ $(file_uid "$env_parent") == "$current_uid" ]] || return 1
  restore_tmp=$(mktemp "$env_parent/.${env_name}.rollback.XXXXXX") || return 1
  rm -f "$restore_tmp" || return 1
  if ! cp -p "$backup" "$restore_tmp"; then
    rm -f "$restore_tmp"
    return 1
  fi
  if [[ -d $ENV_FILE && ! -L $ENV_FILE ]]; then
    if ! rmdir "$ENV_FILE"; then
      rm -f "$restore_tmp"
      return 1
    fi
  fi
  if ! mv -f "$restore_tmp" "$ENV_FILE"; then
    rm -f "$restore_tmp"
    return 1
  fi
  [[ -f $ENV_FILE && ! -L $ENV_FILE ]] || return 1
  [[ $(file_uid "$ENV_FILE") == "$backup_uid" ]] || return 1
  [[ $(file_gid "$ENV_FILE") == "$backup_gid" ]] || return 1
  [[ $(file_mode "$ENV_FILE") == "$backup_mode" ]] || return 1
  [[ $(sha256_file "$ENV_FILE") == "$expected_sha" ]]
}

validate_path() {
  [[ $1 =~ ^/[A-Za-z0-9._/-]+$ ]] || fail invalid_path
}

validate_common() {
  [[ ${MAINTENANCE_ENV_PRESTAGE_APPROVED:-0} == 1 ]] || fail approval_required
  [[ $ACTION == prepare || $ACTION == rollback ]] || fail invalid_action
  validate_path "$ENV_FILE"
  validate_path "$SENTINEL"
  validate_path "$PROOF_FILE"
  validate_path "$BACKUP_BASE"
  if [[ -n $STATE_FILE ]]; then
    validate_path "$STATE_FILE"
    validate_path "$HTTPD_BACKUP"
    [[ $HTTPD_BACKUP == /var/backups/alumni-maintenance/prepare.* ]] || fail httpd_backup_path_invalid
    state_parent=$(dirname "$STATE_FILE")
    [[ -d $state_parent && ! -L $state_parent ]] || fail state_parent_invalid
    [[ $(file_uid "$state_parent") == "$current_uid" ]] || fail state_parent_owner_invalid
    [[ $(file_mode "$state_parent") == 700 ]] || fail state_parent_mode_invalid
  fi

  if [[ $ENV_FILE == "$CANONICAL_ENV_FILE" || $SENTINEL == "$CANONICAL_SENTINEL" ||
        $PROOF_FILE == "$CANONICAL_PROOF_FILE" ]]; then
    [[ $ENV_FILE == "$CANONICAL_ENV_FILE" && $SENTINEL == "$CANONICAL_SENTINEL" &&
       $PROOF_FILE == "$CANONICAL_PROOF_FILE" ]] || fail canonical_paths_must_be_bound
    [[ $current_uid == 0 ]] || fail canonical_paths_require_root
  fi

  sentinel_parent=$(dirname "$SENTINEL")
  proof_parent=$(dirname "$PROOF_FILE")
  [[ $sentinel_parent == "$proof_parent" ]] || fail proof_parent_mismatch
  [[ -d $sentinel_parent && ! -L $sentinel_parent ]] || fail runtime_parent_invalid
  [[ $(file_uid "$sentinel_parent") == "$current_uid" ]] || fail runtime_parent_owner_invalid
  [[ $(file_mode "$sentinel_parent") == 755 ]] || fail runtime_parent_mode_invalid
  [[ ! -e $SENTINEL && ! -L $SENTINEL ]] || fail sentinel_must_be_inactive
}

validate_prepare_environment() {
  [[ -f $ENV_FILE && ! -L $ENV_FILE ]] || fail environment_file_invalid
  env_mode=$(file_mode "$ENV_FILE")
  env_uid=$(file_uid "$ENV_FILE")
  env_gid=$(file_gid "$ENV_FILE")
  [[ $env_uid == "$current_uid" ]] || fail environment_file_owner_invalid
  if [[ $env_mode == 640 ]]; then
    [[ $(file_group "$ENV_FILE") == alumni-backend ]] || fail environment_file_group_invalid
  else
    [[ $env_mode == 600 ]] || fail environment_file_mode_invalid
  fi
}

prepare() {
  validate_prepare_environment
  [[ ! -e $PROOF_FILE && ! -L $PROOF_FILE ]] || fail proof_must_not_exist

  if [[ -e $BACKUP_BASE ]]; then
    [[ -d $BACKUP_BASE && ! -L $BACKUP_BASE ]] || fail backup_base_invalid
    [[ $(file_uid "$BACKUP_BASE") == "$current_uid" ]] || fail backup_base_owner_invalid
    [[ $(file_mode "$BACKUP_BASE") == 700 ]] || fail backup_base_mode_invalid
  else
    install -d -m 0700 "$BACKUP_BASE" || fail backup_base_create_failed
  fi

  backup_dir=$(mktemp -d "$BACKUP_BASE/env-prestage.XXXXXX") || fail backup_create_failed
  chmod 0700 "$backup_dir" || fail backup_mode_failed
  cp -p "$ENV_FILE" "$backup_dir/alumni-backend.env" || fail backup_copy_failed
  before_sha=$(sha256_file "$backup_dir/alumni-backend.env") || fail backup_hash_failed
  [[ $before_sha =~ ^[a-f0-9]{64}$ ]] || fail backup_hash_invalid

  proof_tmp=$(mktemp "${PROOF_FILE}.tmp.XXXXXX") || fail proof_temp_create_failed
  chmod 0600 "$proof_tmp" || fail proof_temp_mode_failed
  proof=$(openssl rand -hex 32) || fail proof_generation_failed
  [[ $proof =~ ^[a-f0-9]{64}$ ]] || fail proof_format_invalid
  printf '%s\n' "$proof" > "$proof_tmp" || fail proof_write_failed
  proof_sha=$(printf '%s' "$proof" | sha256_text) || fail proof_hash_failed
  unset proof
  [[ $proof_sha =~ ^[a-f0-9]{64}$ ]] || fail proof_hash_invalid
  [[ ! -L $proof_tmp && -f $proof_tmp ]] || fail proof_temp_invalid
  mv -f "$proof_tmp" "$PROOF_FILE" || fail proof_install_failed
  proof_created=1
  [[ $(file_mode "$PROOF_FILE") == 600 && $(file_uid "$PROOF_FILE") == "$current_uid" ]] ||
    fail proof_metadata_invalid

  env_tmp=$(mktemp "${ENV_FILE}.tmp.XXXXXX") || fail environment_temp_create_failed
  chmod 0600 "$env_tmp" || fail environment_temp_mode_failed
  # PHP variables are intentionally protected from shell expansion.
  # shellcheck disable=SC2016
  php -r '
$source = $argv[1];
$target = $argv[2];
$updates = array(
    "MAINTENANCE_SENTINEL_PATH" => $argv[3],
    "MAINTENANCE_SMOKE_PROOF_SHA256" => $argv[4],
    "MAINTENANCE_SMOKE_ALLOWED_PATHS" => $argv[5]
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
' "$ENV_FILE" "$env_tmp" "$SENTINEL" "$proof_sha" "$ALLOWED_PATHS" || fail environment_update_failed

  if [[ $current_uid == 0 ]]; then
    chown "$env_uid:$env_gid" "$env_tmp" || fail environment_temp_owner_failed
  fi
  chmod "$env_mode" "$env_tmp" || fail environment_temp_restore_mode_failed
  mv -f "$env_tmp" "$ENV_FILE" || fail environment_install_failed
  env_replaced=1

  installed_proof_sha=$(tr -d '\r\n' < "$PROOF_FILE" | sha256_text) || fail installed_proof_hash_failed
  [[ $installed_proof_sha == "$proof_sha" ]] || fail installed_proof_hash_mismatch
  # PHP variables are intentionally protected from shell expansion.
  # shellcheck disable=SC2016
  php -r '
$env = @parse_ini_file($argv[1], false, INI_SCANNER_RAW);
if ($env === false) { exit(20); }
$expected = array(
    "MAINTENANCE_SENTINEL_PATH" => $argv[2],
    "MAINTENANCE_SMOKE_PROOF_SHA256" => $argv[3],
    "MAINTENANCE_SMOKE_ALLOWED_PATHS" => $argv[4]
);
foreach ($expected as $key => $value) {
    if (!isset($env[$key]) || !hash_equals($value, (string)$env[$key])) { exit(21); }
}
' "$ENV_FILE" "$SENTINEL" "$proof_sha" "$ALLOWED_PATHS" || fail environment_validation_failed

  manifest="$backup_dir/manifest"
  umask 077
  {
    printf 'state=prepared\n'
    printf 'environment_file=%s\n' "$ENV_FILE"
    printf 'sentinel=%s\n' "$SENTINEL"
    printf 'proof_file=%s\n' "$PROOF_FILE"
    printf 'before_sha256=%s\n' "$before_sha"
    printf 'proof_sha256=%s\n' "$proof_sha"
  } > "$manifest" || fail manifest_write_failed
  chmod 0600 "$manifest" || fail manifest_mode_failed

  if [[ -n $STATE_FILE ]]; then
    state_tmp=$(mktemp "${STATE_FILE}.tmp.XXXXXX") || fail state_temp_create_failed
    chmod 0600 "$state_tmp" || fail state_temp_mode_failed
    {
      printf 'httpd_backup=%s\n' "$HTTPD_BACKUP"
      printf 'env_backup=%s\n' "$backup_dir"
    } > "$state_tmp" || fail state_write_failed
    mv -f "$state_tmp" "$STATE_FILE" || fail state_install_failed
    [[ -f $STATE_FILE && ! -L $STATE_FILE ]] || fail state_validation_failed
    [[ $(file_uid "$STATE_FILE") == "$current_uid" && $(file_mode "$STATE_FILE") == 600 ]] ||
      fail state_metadata_invalid
  fi

  proof_created=0
  env_replaced=0
  printf 'PASS maintenance_env_prestage state=prepared backup=%s\n' "$backup_dir"
}

rollback() {
  [[ -n $ROLLBACK_DIR ]] || fail rollback_backup_required
  validate_path "$ROLLBACK_DIR"
  [[ $ROLLBACK_DIR == "$BACKUP_BASE"/env-prestage.* ]] || fail rollback_backup_outside_base
  [[ -d $ROLLBACK_DIR && ! -L $ROLLBACK_DIR ]] || fail rollback_backup_invalid
  [[ $(file_uid "$ROLLBACK_DIR") == "$current_uid" && $(file_mode "$ROLLBACK_DIR") == 700 ]] ||
    fail rollback_backup_metadata_invalid
  manifest="$ROLLBACK_DIR/manifest"
  backup_env="$ROLLBACK_DIR/alumni-backend.env"
  [[ -f $manifest && ! -L $manifest && $(file_mode "$manifest") == 600 ]] || fail rollback_manifest_invalid
  [[ -f $backup_env && ! -L $backup_env ]] || fail rollback_environment_backup_invalid

  manifest_value() {
    local key=$1
    local count value
    count=$(grep -c "^${key}=" "$manifest" || true)
    [[ $count == 1 ]] || fail rollback_manifest_key_invalid
    value=$(sed -n "s/^${key}=//p" "$manifest")
    printf '%s' "$value"
  }

  [[ $(manifest_value state) == prepared ]] || fail rollback_state_invalid
  [[ $(manifest_value environment_file) == "$ENV_FILE" ]] || fail rollback_environment_path_mismatch
  [[ $(manifest_value sentinel) == "$SENTINEL" ]] || fail rollback_sentinel_path_mismatch
  [[ $(manifest_value proof_file) == "$PROOF_FILE" ]] || fail rollback_proof_path_mismatch
  before_sha=$(manifest_value before_sha256)
  proof_sha=$(manifest_value proof_sha256)
  [[ $before_sha =~ ^[a-f0-9]{64}$ && $proof_sha =~ ^[a-f0-9]{64}$ ]] || fail rollback_hash_invalid
  [[ $(sha256_file "$backup_env") == "$before_sha" ]] || fail rollback_environment_backup_hash_mismatch
  restore_environment_from_backup "$backup_env" "$before_sha" || fail rollback_environment_restore_failed
  remove_proof_safely || fail rollback_proof_remove_failed
  sed 's/^state=prepared$/state=rolled_back/' "$manifest" > "$manifest.tmp" || fail rollback_manifest_update_failed
  chmod 0600 "$manifest.tmp" || fail rollback_manifest_temp_mode_failed
  mv -f "$manifest.tmp" "$manifest" || fail rollback_manifest_install_failed

  printf 'PASS maintenance_env_prestage state=rolled_back backup=%s\n' "$ROLLBACK_DIR"
}

validate_common
case "$ACTION" in
  prepare) prepare ;;
  rollback) rollback ;;
esac
