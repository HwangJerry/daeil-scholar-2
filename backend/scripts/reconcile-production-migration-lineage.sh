#!/usr/bin/env bash
# Reconcile the production-applied 001-039 migration filenames with exact source digests.
set -euo pipefail

fail() {
  printf 'LINEAGE_RECONCILIATION_FAIL [REDACTED] %s\n' "$1" >&2
  exit 1
}

[ "$#" -eq 2 ] || fail "usage: reconcile-production-migration-lineage.sh <mysql-options-file> <database>"
MYSQL_OPTIONS_FILE="$1"
DB_NAME="$2"
EXTERNAL_LINEAGE_MANIFEST_SHA256="${CANONICAL_PRODUCTION_LINEAGE_MANIFEST_SHA256:-}"
readonly EXTERNAL_LINEAGE_MANIFEST_SHA256
unset CANONICAL_PRODUCTION_LINEAGE_MANIFEST_SHA256 || true
readonly MYSQL_OPTIONS_FILE DB_NAME

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd -P)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/.." && pwd -P)"
MIGRATIONS_DIR="${BACKEND_DIR}/migrations"
MANIFEST="${MIGRATIONS_DIR}/testdata/canonical_identity_production_lineage_001_039.sha256"
readonly SCRIPT_DIR BACKEND_DIR MIGRATIONS_DIR MANIFEST
TEMP_DIR=""

cleanup() {
  local status=$?
  trap - EXIT
  if [ -n "$TEMP_DIR" ]; then
    rm -rf "$TEMP_DIR" || status=125
  fi
  exit "$status"
}
trap cleanup EXIT

[[ "$DB_NAME" =~ ^[A-Za-z0-9_]+$ ]] || fail "invalid database identifier"
[ -f "$MYSQL_OPTIONS_FILE" ] && [ ! -L "$MYSQL_OPTIONS_FILE" ] || fail "mysql options file missing or symlinked"
case "$(stat -c '%a' "$MYSQL_OPTIONS_FILE" 2>/dev/null || true)" in
  600|400) ;;
  *) fail "mysql options file mode must be 0600 or 0400" ;;
esac
[ -f "$MANIFEST" ] && [ ! -L "$MANIFEST" ] || fail "production lineage manifest missing or symlinked"
[[ "$EXTERNAL_LINEAGE_MANIFEST_SHA256" =~ ^[0-9a-f]{64}$ ]] || fail "external production lineage approval missing or malformed"

TEMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/dflh-production-lineage.XXXXXX")"
chmod 0700 "$TEMP_DIR"
MANIFEST_SNAPSHOT="${TEMP_DIR}/lineage.sha256"
cp "$MANIFEST" "$MANIFEST_SNAPSHOT"
chmod 0600 "$MANIFEST_SNAPSHOT"
[ "$(shasum -a 256 "$MANIFEST_SNAPSHOT" | cut -d ' ' -f 1)" = "$EXTERNAL_LINEAGE_MANIFEST_SHA256" ] || \
  fail "production lineage manifest approval mismatch"

mysql_exec() {
  mysql --defaults-extra-file="$MYSQL_OPTIONS_FILE" --database="$DB_NAME" --batch --skip-column-names "$@"
}

declare -a FILENAMES=()
declare -a DIGESTS=()
declare -A SEEN_FILENAMES=()
declare -A SEEN_NUMBERS=()
entry_count=0
while read -r digest filename extra; do
  case "$digest" in \#*|'') continue ;; esac
  [ -z "${extra:-}" ] || fail "production lineage manifest entry has extra fields"
  [[ "$digest" =~ ^[0-9a-f]{64}$ ]] || fail "production lineage manifest digest malformed"
  [[ "$filename" =~ ^([0-9]{3})_[a-z0-9][a-z0-9_]*\.sql$ ]] || fail "production lineage filename malformed"
  number="${BASH_REMATCH[1]}"
  expected_number="$(printf '%03d' $((entry_count + 1)))"
  [ "$number" = "$expected_number" ] || fail "production lineage numbering is not exact 001-039"
  [ -z "${SEEN_FILENAMES[$filename]:-}" ] || fail "duplicate production lineage filename"
  [ -z "${SEEN_NUMBERS[$number]:-}" ] || fail "duplicate production lineage number"
  source_file="${MIGRATIONS_DIR}/${filename}"
  [ -f "$source_file" ] && [ ! -L "$source_file" ] || fail "approved production lineage source missing or symlinked"
  [ "$(cd "$(dirname "$source_file")" && pwd -P)" = "$MIGRATIONS_DIR" ] || fail "production lineage source is not a direct child"
  [ "$(shasum -a 256 "$source_file" | cut -d ' ' -f 1)" = "$digest" ] || fail "approved production lineage source digest mismatch"
  FILENAMES+=("$filename")
  DIGESTS+=("$digest")
  SEEN_FILENAMES["$filename"]=1
  SEEN_NUMBERS["$number"]=1
  entry_count=$((entry_count + 1))
done < "$MANIFEST_SNAPSHOT"
[ "$entry_count" -eq 39 ] || fail "production lineage manifest must contain exactly 39 entries"

source_count=0
for source_file in "${MIGRATIONS_DIR}"/[0-9][0-9][0-9]_*.sql; do
  [ -f "$source_file" ] || continue
  filename="$(basename "$source_file")"
  number="${filename%%_*}"
  if [ "$((10#$number))" -le 39 ]; then
    source_count=$((source_count + 1))
    [ -n "${SEEN_FILENAMES[$filename]:-}" ] || fail "unapproved source exists in production lineage range"
  fi
done
[ "$source_count" -eq 39 ] || fail "active production lineage source cardinality mismatch"

server_version="$(mysql_exec -e 'SELECT VERSION();' 2>/dev/null | tr -d '\r')" || fail "database version query failed"
case "$server_version" in
  10.1.38-MariaDB*) ;;
  *) fail "database version is not exact MariaDB 10.1.38" ;;
esac

expected_names="$(printf '%s\n' "${FILENAMES[@]}")"
actual_names="$(mysql_exec -e 'SELECT filename FROM _migration_history ORDER BY filename;' 2>/dev/null)" || \
  fail "migration history filename query failed"
[ "$actual_names" = "$expected_names" ] || fail "migration history filenames do not match approved 001-039 lineage"

index_shape_sql="
  (SELECT COUNT(*) FROM information_schema.STATISTICS
   WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history' AND INDEX_NAME='PRIMARY'
     AND NON_UNIQUE=0 AND SEQ_IN_INDEX=1 AND COLUMN_NAME='filename' AND SUB_PART IS NULL
     AND INDEX_TYPE='BTREE' AND COLLATION='A'), ':',
  (SELECT COUNT(*) FROM information_schema.STATISTICS
   WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history' AND INDEX_NAME='PRIMARY'), ':',
  (SELECT COUNT(*) FROM information_schema.STATISTICS
   WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history')"
legacy_shape="$(mysql_exec -e "SELECT CONCAT(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history' AND ENGINE='InnoDB'), ':',
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history' AND (
    (COLUMN_NAME='filename' AND COLUMN_TYPE='varchar(255)' AND CHARACTER_SET_NAME='utf8' AND COLLATION_NAME='utf8_general_ci' AND IS_NULLABLE='NO' AND COLUMN_DEFAULT IS NULL AND EXTRA='' AND ORDINAL_POSITION=1) OR
    (COLUMN_NAME='applied_at' AND COLUMN_TYPE='timestamp' AND CHARACTER_SET_NAME IS NULL AND COLLATION_NAME IS NULL AND IS_NULLABLE='NO' AND EXTRA='' AND ORDINAL_POSITION=2 AND UPPER(COALESCE(COLUMN_DEFAULT,'')) IN ('CURRENT_TIMESTAMP','CURRENT_TIMESTAMP()'))
  )), ':',
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history'), ':',
  ${index_shape_sql});" 2>/dev/null)" || fail "legacy migration history shape query failed"

target_shape() {
  mysql_exec -e "SELECT CONCAT(
    (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history' AND ENGINE='InnoDB'), ':',
    (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history' AND (
      (COLUMN_NAME='filename' AND COLUMN_TYPE='varchar(255)' AND CHARACTER_SET_NAME='ascii' AND COLLATION_NAME='ascii_general_ci' AND IS_NULLABLE='NO' AND COLUMN_DEFAULT IS NULL AND EXTRA='' AND ORDINAL_POSITION=1) OR
      (COLUMN_NAME='sha256' AND COLUMN_TYPE='char(64)' AND CHARACTER_SET_NAME='ascii' AND COLLATION_NAME='ascii_general_ci' AND IS_NULLABLE='NO' AND COLUMN_DEFAULT IS NULL AND EXTRA='' AND ORDINAL_POSITION=2) OR
      (COLUMN_NAME='applied_at' AND COLUMN_TYPE='timestamp' AND CHARACTER_SET_NAME IS NULL AND COLLATION_NAME IS NULL AND IS_NULLABLE='NO' AND EXTRA='' AND ORDINAL_POSITION=3 AND UPPER(COALESCE(COLUMN_DEFAULT,'')) IN ('CURRENT_TIMESTAMP','CURRENT_TIMESTAMP()'))
    )), ':',
    (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history'), ':',
    ${index_shape_sql});" 2>/dev/null
}

partial_shape() {
  mysql_exec -e "SELECT CONCAT(
    (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history' AND ENGINE='InnoDB'), ':',
    (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history' AND (
      (COLUMN_NAME='filename' AND COLUMN_TYPE='varchar(255)' AND CHARACTER_SET_NAME IN ('utf8','ascii') AND COLLATION_NAME IN ('utf8_general_ci','ascii_general_ci') AND IS_NULLABLE='NO' AND COLUMN_DEFAULT IS NULL AND EXTRA='' AND ORDINAL_POSITION=1) OR
      (COLUMN_NAME='sha256' AND COLUMN_TYPE='char(64)' AND CHARACTER_SET_NAME='ascii' AND COLLATION_NAME='ascii_general_ci' AND IS_NULLABLE='YES' AND COLUMN_DEFAULT IS NULL AND EXTRA='' AND ORDINAL_POSITION=2) OR
      (COLUMN_NAME='applied_at' AND COLUMN_TYPE='timestamp' AND CHARACTER_SET_NAME IS NULL AND COLLATION_NAME IS NULL AND IS_NULLABLE='NO' AND EXTRA='' AND ORDINAL_POSITION=3 AND UPPER(COALESCE(COLUMN_DEFAULT,'')) IN ('CURRENT_TIMESTAMP','CURRENT_TIMESTAMP()'))
    )), ':',
    (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history'), ':',
    ${index_shape_sql});" 2>/dev/null
}

if [ "$(target_shape)" != "1:3:3:1:1:1" ]; then
  if [ "$legacy_shape" = "1:2:2:1:1:1" ]; then
    mysql_exec -e "ALTER TABLE _migration_history ADD COLUMN sha256 CHAR(64) CHARACTER SET ascii COLLATE ascii_general_ci NULL AFTER filename;" \
      >/dev/null 2>&1 || fail "adding migration history digest column failed"
  fi

  [ "$(partial_shape)" = "1:3:3:1:1:1" ] || fail "migration history is neither approved legacy, partial, nor target shape"
  digest_case="CASE filename"
  for index in "${!FILENAMES[@]}"; do
    digest_case+=" WHEN '${FILENAMES[$index]}' THEN '${DIGESTS[$index]}'"
  done
  digest_case+=" ELSE NULL END"
  mismatch_count="$(mysql_exec -e "SELECT COUNT(*) FROM _migration_history WHERE sha256 IS NOT NULL AND sha256 <> ${digest_case};" 2>/dev/null)" || \
    fail "partial migration history digest query failed"
  [ "$mismatch_count" = "0" ] || fail "partial migration history contains an unapproved digest"
  mysql_exec -e "START TRANSACTION; UPDATE _migration_history SET sha256=${digest_case} WHERE sha256 IS NULL; COMMIT;" \
    >/dev/null 2>&1 || fail "migration history digest population failed"
  unresolved_count="$(mysql_exec -e "SELECT COUNT(*) FROM _migration_history WHERE sha256 IS NULL OR sha256 <> ${digest_case};" 2>/dev/null)" || \
    fail "populated migration history digest query failed"
  [ "$unresolved_count" = "0" ] || fail "migration history digest population is incomplete"
  mysql_exec -e "ALTER TABLE _migration_history
    MODIFY filename VARCHAR(255) CHARACTER SET ascii COLLATE ascii_general_ci NOT NULL FIRST,
    MODIFY sha256 CHAR(64) CHARACTER SET ascii COLLATE ascii_general_ci NOT NULL AFTER filename,
    MODIFY applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP AFTER sha256,
    DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;" \
    >/dev/null 2>&1 || fail "final migration history shape conversion failed"
fi

[ "$(target_shape)" = "1:3:3:1:1:1" ] || fail "final migration history shape mismatch"
digest_case="CASE filename"
for index in "${!FILENAMES[@]}"; do
  digest_case+=" WHEN '${FILENAMES[$index]}' THEN '${DIGESTS[$index]}'"
done
digest_case+=" ELSE NULL END"
final_mismatch_count="$(mysql_exec -e "SELECT COUNT(*) FROM _migration_history WHERE sha256 IS NULL OR sha256 <> ${digest_case};" 2>/dev/null)" || \
  fail "final migration history digest query failed"
[ "$final_mismatch_count" = "0" ] || fail "final migration history digest mismatch"

printf 'PRODUCTION_LINEAGE_HISTORY_RECONCILIATION=PASS state=target rows=39\n'
