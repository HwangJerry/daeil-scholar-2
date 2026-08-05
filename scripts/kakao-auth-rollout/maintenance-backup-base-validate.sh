#!/usr/bin/env bash
# maintenance-backup-base-validate.sh — Validate retained rollback backup generations without mutation.
set -euo pipefail

BACKUP_BASE=${1:-}
TEST_MODE=${MAINTENANCE_BACKUP_VALIDATE_TEST_MODE:-0}

fail() {
  printf 'ERROR maintenance_backup_base state=blocked reason=%s\n' "$1" >&2
  exit 1
}

file_mode() {
  stat -c '%a' "$1" 2>/dev/null || stat -f '%Lp' "$1"
}

file_uid() {
  stat -c '%u' "$1" 2>/dev/null || stat -f '%u' "$1"
}

[[ $BACKUP_BASE == /* ]] || fail path_invalid
if [[ $TEST_MODE == 1 ]]; then
  EXPECTED_UID=${MAINTENANCE_BACKUP_EXPECTED_UID:?}
  [[ $EXPECTED_UID =~ ^[0-9]+$ ]] || fail expected_uid_invalid
else
  [[ ${MAINTENANCE_BACKUP_VALIDATE_APPROVED:-0} == 1 ]] || fail approval_required
  [[ $BACKUP_BASE == /var/backups/alumni-maintenance ]] || fail canonical_path_required
  [[ $(id -u) == 0 ]] || fail root_required
  EXPECTED_UID=0
fi

if [[ ! -e $BACKUP_BASE && ! -L $BACKUP_BASE ]]; then
  printf 'PASS maintenance_backup_base state=absent\n'
  exit 0
fi

[[ -d $BACKUP_BASE && ! -L $BACKUP_BASE ]] || fail base_type_invalid
[[ $(file_uid "$BACKUP_BASE") == "$EXPECTED_UID" ]] || fail base_owner_invalid
[[ $(file_mode "$BACKUP_BASE") == 700 ]] || fail base_mode_invalid

shopt -s nullglob
for generation in "$BACKUP_BASE"/*; do
  generation_name=${generation##*/}
  [[ $generation_name =~ ^(prepare|env-prestage)\.[A-Za-z0-9]{6}$ ]] || fail generation_name_invalid
  [[ -d $generation && ! -L $generation ]] || fail generation_type_invalid
  [[ $(file_uid "$generation") == "$EXPECTED_UID" ]] || fail generation_owner_invalid
  [[ $(file_mode "$generation") == 700 ]] || fail generation_mode_invalid
done

printf 'PASS maintenance_backup_base state=trusted\n'
