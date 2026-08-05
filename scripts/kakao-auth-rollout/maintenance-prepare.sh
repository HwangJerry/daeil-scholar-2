#!/usr/bin/env bash
# maintenance-prepare.sh — Pre-stage inactive Apache/PHP maintenance gates with rollback.
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
SENTINEL=${MAINTENANCE_SENTINEL_PATH:-/run/alumni/maintenance}
HTTPD_SOURCE=${MAINTENANCE_HTTPD_SOURCE:-$ROOT/deploy/httpd-alumni.conf}
HTTPD_TARGET=${MAINTENANCE_HTTPD_TARGET:-/etc/httpd/conf.d/alumni.conf}
SHIM_SOURCE_DIR=${MAINTENANCE_SHIM_SOURCE_DIR:-$ROOT/deploy}
SHIM_TARGET_DIR=${MAINTENANCE_SHIM_TARGET_DIR:-/var/www/html}
BACKUP_BASE=${MAINTENANCE_BACKUP_BASE:-/var/backups/alumni-maintenance}
STATE_FILE=${MAINTENANCE_PRESTAGE_STATE_FILE:-}
sentinel_parent=$(dirname "$SENTINEL")
sentinel_parent_existed=0
sentinel_parent_mode=
mutated=0
backup_dir=

files=(
  "httpd-alumni.conf:$HTTPD_TARGET"
  "_maintenance_gate.php:$SHIM_TARGET_DIR/_maintenance_gate.php"
  "_legacy_docroot.php:$SHIM_TARGET_DIR/_legacy_docroot.php"
  "_legacy_url_rewriter.php:$SHIM_TARGET_DIR/_legacy_url_rewriter.php"
  "_set_docroot.php:$SHIM_TARGET_DIR/_set_docroot.php"
)
rollback_files=(
  "_set_docroot.php:$SHIM_TARGET_DIR/_set_docroot.php"
  "_maintenance_gate.php:$SHIM_TARGET_DIR/_maintenance_gate.php"
  "_legacy_docroot.php:$SHIM_TARGET_DIR/_legacy_docroot.php"
  "_legacy_url_rewriter.php:$SHIM_TARGET_DIR/_legacy_url_rewriter.php"
  "httpd-alumni.conf:$HTTPD_TARGET"
)

rollback() {
  local entry name target
  [[ -n $backup_dir && -d $backup_dir ]] || return 0
  for entry in "${rollback_files[@]}"; do
    name=${entry%%:*}
    target=${entry#*:}
    if [[ -f $backup_dir/$name.present ]]; then
      install -m 0644 "$backup_dir/$name" "$target" || return 1
    else
      rm -f "$target" || return 1
    fi
  done
  if [[ $sentinel_parent_existed == 1 ]]; then
    chmod "$sentinel_parent_mode" "$sentinel_parent" || return 1
  else
    rmdir "$sentinel_parent" 2>/dev/null || true
  fi
  httpd -t >/dev/null || return 1
  systemctl reload httpd || return 1
}

fail() {
  local reason=$1
  if [[ $mutated == 1 ]]; then
    rollback || printf 'ERROR maintenance_prepare rollback=failed\n' >&2
  fi
  printf 'ERROR maintenance_prepare state=blocked reason=%s\n' "$reason" >&2
  exit 1
}

[[ ${MAINTENANCE_PREPARE_APPROVED:-0} == 1 ]] || fail approval_required
[[ ${EUID:-$(id -u)} -eq 0 ]] || fail root_required
[[ $SENTINEL == /* ]] || fail sentinel_path_must_be_absolute
if [[ -n $STATE_FILE ]]; then
  [[ $STATE_FILE =~ ^/[A-Za-z0-9._/-]+$ ]] || fail state_path_invalid
  state_parent=$(dirname "$STATE_FILE")
  [[ -d $state_parent && ! -L $state_parent ]] || fail state_parent_invalid
  [[ $(stat -c '%u:%a' "$state_parent") == 0:700 ]] || fail state_parent_metadata_invalid
fi
[[ ! -e $SENTINEL ]] || fail sentinel_must_be_inactive
[[ ! -e $sentinel_parent || -d $sentinel_parent ]] || fail sentinel_parent_not_directory
[[ -r $HTTPD_SOURCE ]] || fail httpd_source_unreadable
grep -Fq 'ALUMNI_MAINTENANCE_GATE=1' "$HTTPD_SOURCE" || fail httpd_source_missing_gate
for shim in _maintenance_gate.php _legacy_docroot.php _legacy_url_rewriter.php _set_docroot.php; do
  [[ -r $SHIM_SOURCE_DIR/$shim ]] || fail shim_source_unreadable
  php -l "$SHIM_SOURCE_DIR/$shim" >/dev/null || fail shim_source_invalid
 done
httpd -t >/dev/null || fail current_httpd_config_invalid
systemctl is-active --quiet httpd || fail httpd_not_active

install -d -m 0700 "$BACKUP_BASE" || fail backup_base_create_failed
backup_dir=$(mktemp -d "$BACKUP_BASE/prepare.XXXXXX") || fail backup_create_failed
chmod 0700 "$backup_dir" || fail backup_mode_failed
for entry in "${files[@]}"; do
  name=${entry%%:*}
  target=${entry#*:}
  if [[ -f $target ]]; then
    cp -p "$target" "$backup_dir/$name" || fail backup_copy_failed
    : > "$backup_dir/$name.present"
  else
    : > "$backup_dir/$name.absent"
  fi
done

if [[ -d $sentinel_parent ]]; then
  sentinel_parent_mode=$(stat -c '%a' "$sentinel_parent") || fail sentinel_parent_mode_read_failed
  sentinel_parent_existed=1
fi
mutated=1
install -d -m 0755 "$sentinel_parent" || fail sentinel_parent_prepare_failed
install -m 0644 "$SHIM_SOURCE_DIR/_maintenance_gate.php" "$SHIM_TARGET_DIR/_maintenance_gate.php" || fail gate_install_failed
install -m 0644 "$SHIM_SOURCE_DIR/_legacy_docroot.php" "$SHIM_TARGET_DIR/_legacy_docroot.php" || fail legacy_docroot_install_failed
install -m 0644 "$SHIM_SOURCE_DIR/_legacy_url_rewriter.php" "$SHIM_TARGET_DIR/_legacy_url_rewriter.php" || fail legacy_rewriter_install_failed
install -m 0644 "$SHIM_SOURCE_DIR/_set_docroot.php" "$SHIM_TARGET_DIR/_set_docroot.php" || fail prepend_install_failed
install -m 0644 "$HTTPD_SOURCE" "$HTTPD_TARGET" || fail httpd_install_failed
php -l "$SHIM_TARGET_DIR/_maintenance_gate.php" >/dev/null || fail installed_gate_invalid
php -l "$SHIM_TARGET_DIR/_set_docroot.php" >/dev/null || fail installed_prepend_invalid
httpd -t >/dev/null || fail installed_httpd_config_invalid
systemctl reload httpd || fail httpd_reload_failed
systemctl is-active --quiet httpd || fail httpd_not_active_after_reload
if [[ -n $STATE_FILE ]]; then
  state_tmp=$(mktemp "${STATE_FILE}.tmp.XXXXXX") || fail state_temp_create_failed
  chmod 0600 "$state_tmp" || fail state_temp_mode_failed
  {
    printf 'httpd_backup=%s\n' "$backup_dir"
    printf 'env_backup=\n'
  } > "$state_tmp" || fail state_write_failed
  mv -f "$state_tmp" "$STATE_FILE" || fail state_install_failed
  [[ -f $STATE_FILE && ! -L $STATE_FILE ]] || fail state_validation_failed
  [[ $(stat -c '%u:%a' "$STATE_FILE") == 0:600 ]] || fail state_metadata_invalid
fi
mutated=0

printf 'PASS maintenance_prepare gate=installed sentinel=inactive backup=%s\n' "$backup_dir"
