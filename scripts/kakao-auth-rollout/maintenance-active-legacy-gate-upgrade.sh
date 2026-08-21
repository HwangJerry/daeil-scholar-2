#!/usr/bin/env bash
# maintenance-active-legacy-gate-upgrade.sh — Upgrade/rollback bridge-aware Apache/PHP gates while maintenance is active.
set -euo pipefail

ACTION=${1:-}
ROLLBACK_DIR=${2:-}
TEST_MODE=${MAINTENANCE_ACTIVE_LEGACY_TEST_MODE:-0}
SENTINEL=${MAINTENANCE_SENTINEL_PATH:-/run/alumni/maintenance}
BRIDGE=${MAINTENANCE_RELEASE_BRIDGE_PATH:-/run/alumni/maintenance-release-bridge}
HTTPD_SOURCE=${MAINTENANCE_HTTPD_SOURCE:-}
GATE_SOURCE=${MAINTENANCE_GATE_SOURCE:-}
HTTPD_TARGET=${MAINTENANCE_HTTPD_TARGET:-/etc/httpd/conf.d/alumni.conf}
GATE_TARGET=${MAINTENANCE_GATE_TARGET:-/var/www/html/_maintenance_gate.php}
BACKUP_BASE=${MAINTENANCE_ACTIVE_LEGACY_BACKUP_BASE:-/var/backups/alumni-maintenance-release}
PHP_BIN=${MAINTENANCE_PHP_BIN:-php}
HTTPD_BIN=${MAINTENANCE_HTTPD_BIN:-httpd}
SYSTEMCTL_BIN=${MAINTENANCE_SYSTEMCTL_BIN:-systemctl}
CANONICAL_SENTINEL=/run/alumni/maintenance
CANONICAL_BRIDGE=/run/alumni/maintenance-release-bridge
CANONICAL_HTTPD_TARGET=/etc/httpd/conf.d/alumni.conf
CANONICAL_GATE_TARGET=/var/www/html/_maintenance_gate.php
CANONICAL_BACKUP_BASE=/var/backups/alumni-maintenance-release
current_uid=$(id -u)
backup_dir=
transaction_active=0
before_httpd_sha=
before_gate_sha=
sentinel_sha=

file_mode() {
  stat -c '%a' "$1" 2>/dev/null || stat -f '%Lp' "$1"
}

file_uid() {
  stat -c '%u' "$1" 2>/dev/null || stat -f '%u' "$1"
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d' ' -f1
  else
    shasum -a 256 "$1" | cut -d' ' -f1
  fi
}

fail() {
  printf 'ERROR maintenance_active_legacy_gate state=blocked reason=%s\n' "$1" >&2
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

validate_runtime_authority() {
  local runtime_parent
  runtime_parent=$(dirname "$SENTINEL")
  [[ $(dirname "$BRIDGE") == "$runtime_parent" ]] || fail runtime_parent_mismatch
  [[ -d $runtime_parent && ! -L $runtime_parent ]] || fail runtime_parent_invalid
  [[ $(file_uid "$runtime_parent") == "$current_uid" ]] || fail runtime_parent_owner_invalid
  [[ $(file_mode "$runtime_parent") == 755 ]] || fail runtime_parent_mode_invalid
  [[ ! -e $BRIDGE && ! -L $BRIDGE ]] || fail release_bridge_must_be_absent
  validate_active_sentinel
}

validate_target() {
  local target=$1
  [[ -f $target && ! -L $target ]] || fail target_invalid
  [[ $(file_uid "$target") == "$current_uid" ]] || fail target_owner_invalid
  [[ $(file_mode "$target") == 644 ]] || fail target_mode_invalid
  [[ -d $(dirname "$target") && ! -L $(dirname "$target") ]] || fail target_parent_invalid
  [[ $(file_uid "$(dirname "$target")") == "$current_uid" ]] || fail target_parent_owner_invalid
}

validate_common() {
  [[ ${MAINTENANCE_ACTIVE_LEGACY_APPROVED:-0} == 1 ]] || fail approval_required
  [[ $ACTION == prepare || $ACTION == rollback ]] || fail invalid_action
  for path in "$SENTINEL" "$BRIDGE" "$HTTPD_TARGET" "$GATE_TARGET" "$BACKUP_BASE"; do
    validate_path "$path"
  done
  if [[ $TEST_MODE != 1 ]]; then
    [[ $current_uid == 0 ]] || fail root_required
    [[ $SENTINEL == "$CANONICAL_SENTINEL" && $BRIDGE == "$CANONICAL_BRIDGE" &&
       $HTTPD_TARGET == "$CANONICAL_HTTPD_TARGET" && $GATE_TARGET == "$CANONICAL_GATE_TARGET" &&
       $BACKUP_BASE == "$CANONICAL_BACKUP_BASE" ]] || fail canonical_paths_required
  fi
  validate_runtime_authority
}

validate_sources() {
  validate_path "$HTTPD_SOURCE"
  validate_path "$GATE_SOURCE"
  [[ -f $HTTPD_SOURCE && ! -L $HTTPD_SOURCE ]] || fail httpd_source_invalid
  [[ -f $GATE_SOURCE && ! -L $GATE_SOURCE ]] || fail gate_source_invalid
  grep -Fq '/run/alumni/maintenance-release-bridge' "$HTTPD_SOURCE" || fail httpd_source_missing_bridge
  grep -Fq '/run/alumni/maintenance-release-bridge' "$GATE_SOURCE" || fail gate_source_missing_bridge
  "$PHP_BIN" -l "$GATE_SOURCE" >/dev/null || fail gate_source_syntax_invalid
}

install_atomic() {
  local source=$1
  local target=$2
  local temp
  temp=$(mktemp "$(dirname "$target")/.$(basename "$target").upgrade.XXXXXX") || return 1
  rm -f "$temp" || return 1
  if ! install -m 0644 "$source" "$temp"; then
    rm -f "$temp"
    return 1
  fi
  if [[ $current_uid == 0 ]]; then
    chown 0:0 "$temp" || { rm -f "$temp"; return 1; }
  fi
  mv -f "$temp" "$target" || { rm -f "$temp"; return 1; }
}

restore_from_backup() {
  local restore_httpd restore_gate
  [[ -n $backup_dir && -d $backup_dir && ! -L $backup_dir ]] || return 1
  [[ $(sha256_file "$backup_dir/alumni.conf") == "$before_httpd_sha" ]] || return 1
  [[ $(sha256_file "$backup_dir/_maintenance_gate.php") == "$before_gate_sha" ]] || return 1
  install_atomic "$backup_dir/_maintenance_gate.php" "$GATE_TARGET" || return 1
  install_atomic "$backup_dir/alumni.conf" "$HTTPD_TARGET" || return 1
  restore_gate=$(sha256_file "$GATE_TARGET") || return 1
  restore_httpd=$(sha256_file "$HTTPD_TARGET") || return 1
  [[ $restore_gate == "$before_gate_sha" && $restore_httpd == "$before_httpd_sha" ]] || return 1
  "$PHP_BIN" -l "$GATE_TARGET" >/dev/null || return 1
  "$HTTPD_BIN" -t >/dev/null || return 1
  "$SYSTEMCTL_BIN" reload httpd || return 1
  "$SYSTEMCTL_BIN" is-active --quiet httpd || return 1
  validate_runtime_authority
  [[ $(sha256_file "$SENTINEL") == "$sentinel_sha" ]]
}

on_exit() {
  local status=$?
  trap - EXIT HUP INT TERM
  if [[ $transaction_active == 1 && $status -ne 0 ]]; then
    if ! restore_from_backup; then
      printf 'ERROR maintenance_active_legacy_gate rollback=failed backup=%s\n' "$backup_dir" >&2
    fi
  fi
  exit "$status"
}
trap on_exit EXIT
trap 'exit 1' HUP INT TERM

prepare() {
  local source_httpd_sha source_gate_sha manifest
  validate_sources
  validate_target "$HTTPD_TARGET"
  validate_target "$GATE_TARGET"
  "$HTTPD_BIN" -t >/dev/null || fail current_httpd_config_invalid
  "$SYSTEMCTL_BIN" is-active --quiet httpd || fail httpd_not_active

  if [[ -e $BACKUP_BASE ]]; then
    [[ -d $BACKUP_BASE && ! -L $BACKUP_BASE ]] || fail backup_base_invalid
    [[ $(file_uid "$BACKUP_BASE") == "$current_uid" && $(file_mode "$BACKUP_BASE") == 700 ]] ||
      fail backup_base_metadata_invalid
  else
    install -d -m 0700 "$BACKUP_BASE" || fail backup_base_create_failed
  fi
  backup_dir=$(mktemp -d "$BACKUP_BASE/active-legacy.XXXXXX") || fail backup_create_failed
  chmod 0700 "$backup_dir" || fail backup_mode_failed
  cp -p "$HTTPD_TARGET" "$backup_dir/alumni.conf" || fail backup_copy_failed
  cp -p "$GATE_TARGET" "$backup_dir/_maintenance_gate.php" || fail backup_copy_failed
  before_httpd_sha=$(sha256_file "$backup_dir/alumni.conf") || fail backup_hash_failed
  before_gate_sha=$(sha256_file "$backup_dir/_maintenance_gate.php") || fail backup_hash_failed
  source_httpd_sha=$(sha256_file "$HTTPD_SOURCE") || fail source_hash_failed
  source_gate_sha=$(sha256_file "$GATE_SOURCE") || fail source_hash_failed
  sentinel_sha=$(sha256_file "$SENTINEL") || fail sentinel_hash_failed
  for digest in "$before_httpd_sha" "$before_gate_sha" "$source_httpd_sha" "$source_gate_sha" "$sentinel_sha"; do
    [[ $digest =~ ^[a-f0-9]{64}$ ]] || fail hash_invalid
  done
  manifest="$backup_dir/manifest"
  {
    printf 'state=prepared\n'
    printf 'httpd_target=%s\n' "$HTTPD_TARGET"
    printf 'gate_target=%s\n' "$GATE_TARGET"
    printf 'sentinel=%s\n' "$SENTINEL"
    printf 'bridge=%s\n' "$BRIDGE"
    printf 'before_httpd_sha256=%s\n' "$before_httpd_sha"
    printf 'before_gate_sha256=%s\n' "$before_gate_sha"
    printf 'source_httpd_sha256=%s\n' "$source_httpd_sha"
    printf 'source_gate_sha256=%s\n' "$source_gate_sha"
    printf 'sentinel_sha256=%s\n' "$sentinel_sha"
  } > "$manifest" || fail manifest_write_failed
  chmod 0600 "$manifest" || fail manifest_mode_failed
  transaction_active=1

  install_atomic "$GATE_SOURCE" "$GATE_TARGET" || fail gate_install_failed
  install_atomic "$HTTPD_SOURCE" "$HTTPD_TARGET" || fail httpd_install_failed
  [[ $(sha256_file "$GATE_TARGET") == "$source_gate_sha" ]] || fail gate_install_hash_mismatch
  [[ $(sha256_file "$HTTPD_TARGET") == "$source_httpd_sha" ]] || fail httpd_install_hash_mismatch
  "$PHP_BIN" -l "$GATE_TARGET" >/dev/null || fail installed_gate_syntax_invalid
  "$HTTPD_BIN" -t >/dev/null || fail installed_httpd_config_invalid
  "$SYSTEMCTL_BIN" reload httpd || fail httpd_reload_failed
  "$SYSTEMCTL_BIN" is-active --quiet httpd || fail httpd_not_active_after_reload
  validate_runtime_authority
  [[ $(sha256_file "$SENTINEL") == "$sentinel_sha" ]] || fail sentinel_changed
  transaction_active=0
  printf 'PASS maintenance_active_legacy_gate state=prepared backup=%s\n' "$backup_dir"
}

manifest_value() {
  local manifest=$1
  local key=$2
  local count value
  count=$(grep -c "^${key}=" "$manifest" || true)
  [[ $count == 1 ]] || fail rollback_manifest_key_invalid
  value=$(sed -n "s/^${key}=//p" "$manifest")
  printf '%s' "$value"
}

rollback() {
  local manifest
  [[ -n $ROLLBACK_DIR ]] || fail rollback_backup_required
  validate_path "$ROLLBACK_DIR"
  [[ $ROLLBACK_DIR == "$BACKUP_BASE"/active-legacy.* ]] || fail rollback_backup_outside_base
  [[ -d $ROLLBACK_DIR && ! -L $ROLLBACK_DIR ]] || fail rollback_backup_invalid
  [[ $(file_uid "$ROLLBACK_DIR") == "$current_uid" && $(file_mode "$ROLLBACK_DIR") == 700 ]] ||
    fail rollback_backup_metadata_invalid
  backup_dir=$ROLLBACK_DIR
  manifest="$backup_dir/manifest"
  [[ -f $manifest && ! -L $manifest && $(file_mode "$manifest") == 600 ]] || fail rollback_manifest_invalid
  [[ $(manifest_value "$manifest" state) == prepared ]] || fail rollback_state_invalid
  [[ $(manifest_value "$manifest" httpd_target) == "$HTTPD_TARGET" ]] || fail rollback_httpd_target_mismatch
  [[ $(manifest_value "$manifest" gate_target) == "$GATE_TARGET" ]] || fail rollback_gate_target_mismatch
  [[ $(manifest_value "$manifest" sentinel) == "$SENTINEL" ]] || fail rollback_sentinel_mismatch
  [[ $(manifest_value "$manifest" bridge) == "$BRIDGE" ]] || fail rollback_bridge_mismatch
  before_httpd_sha=$(manifest_value "$manifest" before_httpd_sha256)
  before_gate_sha=$(manifest_value "$manifest" before_gate_sha256)
  sentinel_sha=$(manifest_value "$manifest" sentinel_sha256)
  [[ $before_httpd_sha =~ ^[a-f0-9]{64}$ && $before_gate_sha =~ ^[a-f0-9]{64}$ &&
     $sentinel_sha =~ ^[a-f0-9]{64}$ ]] || fail rollback_hash_invalid
  restore_from_backup || fail rollback_restore_failed
  sed 's/^state=prepared$/state=rolled_back/' "$manifest" > "$manifest.tmp" || fail rollback_manifest_update_failed
  chmod 0600 "$manifest.tmp" || fail rollback_manifest_mode_failed
  mv -f "$manifest.tmp" "$manifest" || fail rollback_manifest_install_failed
  printf 'PASS maintenance_active_legacy_gate state=rolled_back backup=%s\n' "$backup_dir"
}

validate_common
case "$ACTION" in
  prepare) prepare ;;
  rollback) rollback ;;
esac
