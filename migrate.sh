#!/bin/bash
# migrate.sh — Apply SQL migration files using DB credentials from .env
# Usage: ./migrate.sh [migration_file]
#   No args:         apply all unapplied migrations in order
#   With arg:        apply a single specified migration file
#   --seed N:        mark migrations 001..N as applied without executing SQL

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/backend/.env"
MIGRATIONS_DIR="${SCRIPT_DIR}/backend/migrations"
EXTERNAL_CANDIDATE_MANIFEST_SHA256="${CANONICAL_CANDIDATE_MANIFEST_SHA256:-}"
T03_AUTH_ENGINE_BOUND_APPLY="${T03_AUTH_ENGINE_BOUND_APPLY:-0}"
readonly EXTERNAL_CANDIDATE_MANIFEST_SHA256
readonly T03_AUTH_ENGINE_BOUND_APPLY
case "$T03_AUTH_ENGINE_BOUND_APPLY" in 0|1) ;; *) echo "Error: invalid T03 bound-apply mode" >&2; exit 1 ;; esac
if [ "$T03_AUTH_ENGINE_BOUND_APPLY" = "1" ] && [ "$(id -u)" -ne 0 ]; then
  echo "Error: T03 bound apply requires root authority" >&2
  exit 1
fi

# Load .env
if [ ! -f "$ENV_FILE" ]; then
  echo "Error: ${ENV_FILE} not found"
  exit 1
fi

while IFS='=' read -r key value; do
  # Skip comments and blank lines
  [[ -z "$key" || "$key" =~ ^# ]] && continue
  # Trim surrounding whitespace without spawning a process over secret values.
  key="${key#"${key%%[![:space:]]*}"}"
  key="${key%"${key##*[![:space:]]}"}"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  export "$key=$value"
done < "$ENV_FILE"

DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-3306}"
DB_USER="${DB_USER:-root}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_NAME="${DB_NAME:-alumni}"
DB_SOCKET="${DB_SOCKET:-}"

for value in "$DB_HOST" "$DB_PORT" "$DB_USER" "$DB_PASSWORD" "$DB_NAME"; do
  case "$value" in
    *$'\n'*|*$'\r'*) echo "Error: database option contains a line break" >&2; exit 1 ;;
  esac
done
if [ "$T03_AUTH_ENGINE_BOUND_APPLY" = "1" ]; then
  case "$DB_SOCKET" in /*) ;; *) echo "Error: T03 bound apply requires an absolute DB_SOCKET" >&2; exit 1 ;; esac
  [ -S "$DB_SOCKET" ] || { echo "Error: T03 bound apply DB_SOCKET is unavailable" >&2; exit 1; }
fi

option_value() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  printf '"%s"' "$value"
}

if [ "$T03_AUTH_ENGINE_BOUND_APPLY" = "1" ]; then
  MYSQL_OPTION_FILE="$(mktemp /run/dflh-migrate.cnf.XXXXXX)"
else
  MYSQL_OPTION_FILE="$(mktemp "${TMPDIR:-/tmp}/dflh-migrate.cnf.XXXXXX")"
fi
RUNNER_LOCK_ACQUIRED=0
RUNNER_SAFE_TO_RELEASE=0
cleanup() {
  local original_status=$?
  local cleanup_status=0
  trap - EXIT
  if [ "$RUNNER_LOCK_ACQUIRED" = "1" ] && [ "$RUNNER_SAFE_TO_RELEASE" = "1" ]; then
    mysql "${MYSQL_ARGS[@]}" -e "DELETE FROM _migration_runner_lock WHERE lock_name='global';" >/dev/null 2>&1 || cleanup_status=125
  fi
  rm -f "$MYSQL_OPTION_FILE" || cleanup_status=125
  [ ! -e "$MYSQL_OPTION_FILE" ] || cleanup_status=125
  if [ "$cleanup_status" -ne 0 ]; then
    exit "$cleanup_status"
  fi
  exit "$original_status"
}
trap cleanup EXIT

{
  printf '[client]\n'
  printf 'user=%s\n' "$(option_value "$DB_USER")"
  printf 'password=%s\n' "$(option_value "$DB_PASSWORD")"
  printf 'database=%s\n' "$(option_value "$DB_NAME")"
  if [ "$T03_AUTH_ENGINE_BOUND_APPLY" = "1" ]; then
    printf 'host=localhost\n'
    printf 'socket=%s\n' "$(option_value "$DB_SOCKET")"
  else
    printf 'host=%s\n' "$(option_value "$DB_HOST")"
    printf 'port=%s\n' "$(option_value "$DB_PORT")"
    printf 'protocol=tcp\n'
    printf 'skip-ssl\n'
  fi
} > "$MYSQL_OPTION_FILE"
chmod 600 "$MYSQL_OPTION_FILE"
MYSQL_ARGS=(--defaults-extra-file="$MYSQL_OPTION_FILE")
[ "$T03_AUTH_ENGINE_BOUND_APPLY" = "0" ] || echo "  T03_BOUND_STAGE=connection-configured"

mysql_exec() {
  mysql "${MYSQL_ARGS[@]}" "$@"
}

preflight_migration_source_set() {
  local file filename parent_dir number
  local seen_numbers=" "
  for file in "$MIGRATIONS_DIR"/[0-9][0-9][0-9]_*.sql; do
    [ -f "$file" ] || continue
    filename="$(basename "$file")"
    if [[ ! "$filename" =~ ^([0-9]{3})_[a-z0-9][a-z0-9_]*\.sql$ ]]; then
      echo "Error: invalid migration filename: ${filename}" >&2
      return 1
    fi
    parent_dir="$(cd "$(dirname "$file")" && pwd -P)"
    if [ "$parent_dir" != "$(cd "$MIGRATIONS_DIR" && pwd -P)" ] || [ -L "$file" ]; then
      echo "Error: migration must be a direct non-symlink child of ${MIGRATIONS_DIR}" >&2
      return 1
    fi
    number="${filename%%_*}"
    case "$seen_numbers" in
      *" ${number} "*)
        echo "Error: duplicate migration number in source set: ${number}" >&2
        return 1
        ;;
    esac
    seen_numbers="${seen_numbers}${number} "
  done
}

preflight_candidate_approval() {
  local manifest="${MIGRATIONS_DIR}/testdata/canonical_identity_candidate_lineage.sha256"
  local approved_digest="$EXTERNAL_CANDIDATE_MANIFEST_SHA256"
  local actual_manifest_digest expected_digest filename extra file source_digest number
  local future_count=0 entry_count=0
  local approved_filenames=" " approved_numbers=" "

  for file in "$MIGRATIONS_DIR"/[0-9][0-9][0-9]_*.sql; do
    [ -f "$file" ] || continue
    filename="$(basename "$file")"
    number="${filename%%_*}"
    if (( 10#$number >= 40 )); then
      future_count=$((future_count + 1))
    fi
  done
  [ "$future_count" -gt 0 ] || return 0

  [ -f "$manifest" ] && [ ! -L "$manifest" ] || {
    echo "Error: canonical candidate manifest missing" >&2
    return 1
  }
  [ "${#approved_digest}" -eq 64 ] || {
    echo "Error: canonical candidate manifest external approval missing" >&2
    return 1
  }
  case "$approved_digest" in
    *[!0-9a-f]*) echo "Error: canonical candidate manifest external approval malformed" >&2; return 1 ;;
  esac
  actual_manifest_digest="$(shasum -a 256 "$manifest" | cut -d ' ' -f 1)"
  [ "$actual_manifest_digest" = "$approved_digest" ] || {
    echo "Error: canonical candidate manifest external approval mismatch" >&2
    return 1
  }

  while read -r expected_digest filename extra; do
    case "$expected_digest" in
      \#*|'') continue ;;
    esac
    [ -z "${extra:-}" ] || {
      echo "Error: canonical candidate manifest malformed entry" >&2
      return 1
    }
    [ "${#expected_digest}" -eq 64 ] || {
      echo "Error: canonical candidate manifest malformed digest" >&2
      return 1
    }
    case "$expected_digest" in
      *[!0-9a-f]*) echo "Error: canonical candidate manifest malformed digest" >&2; return 1 ;;
    esac
    if [[ ! "$filename" =~ ^([0-9]{3})_[a-z0-9][a-z0-9_]*\.sql$ ]]; then
      echo "Error: canonical candidate manifest filename invalid" >&2
      return 1
    fi
    number="${filename%%_*}"
    if (( 10#$number < 40 )); then
      echo "Error: canonical candidate manifest contains a historical migration" >&2
      return 1
    fi
    case "$approved_filenames" in
      *" ${filename} "*) echo "Error: canonical candidate manifest duplicate filename" >&2; return 1 ;;
    esac
    case "$approved_numbers" in
      *" ${number} "*) echo "Error: canonical candidate manifest duplicate number" >&2; return 1 ;;
    esac
    file="${MIGRATIONS_DIR}/${filename}"
    [ -f "$file" ] && [ ! -L "$file" ] || {
      echo "Error: canonical candidate manifest source missing" >&2
      return 1
    }
    source_digest="$(shasum -a 256 "$file" | cut -d ' ' -f 1)"
    [ "$source_digest" = "$expected_digest" ] || {
      echo "Error: canonical candidate source digest mismatch" >&2
      return 1
    }
    approved_filenames="${approved_filenames}${filename} "
    approved_numbers="${approved_numbers}${number} "
    entry_count=$((entry_count + 1))
  done < "$manifest"

  [ "$entry_count" -gt 0 ] && [ "$entry_count" -eq "$future_count" ] || {
    echo "Error: canonical candidate manifest/source cardinality mismatch" >&2
    return 1
  }
  for file in "$MIGRATIONS_DIR"/[0-9][0-9][0-9]_*.sql; do
    [ -f "$file" ] || continue
    filename="$(basename "$file")"
    number="${filename%%_*}"
    if (( 10#$number >= 40 )); then
      case "$approved_filenames" in
        *" ${filename} "*) ;;
        *) echo "Error: unapproved future migration source: ${filename}" >&2; return 1 ;;
      esac
    fi
  done
}

# Reject source ambiguity before the first database side effect.
preflight_migration_source_set
preflight_candidate_approval

# Ensure migration tracking table exists
mysql_exec -e "
CREATE TABLE IF NOT EXISTS _migration_history (
  filename VARCHAR(255) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  sha256 CHAR(64) CHARACTER SET ascii NOT NULL,
  applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS _migration_journal (
  filename VARCHAR(255) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  sha256 CHAR(64) CHARACTER SET ascii NOT NULL,
  state VARCHAR(16) CHARACTER SET ascii NOT NULL,
  started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  completed_at TIMESTAMP NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS _migration_runner_lock (
  lock_name VARCHAR(32) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  acquired_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
" 2>/dev/null

history_table_shape="$(mysql_exec -N -e "
SELECT CONCAT(
  (SELECT COUNT(*) FROM information_schema.TABLES
   WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history' AND ENGINE='InnoDB'), ':',
  (SELECT COUNT(*) FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history' AND (
     (COLUMN_NAME='filename' AND DATA_TYPE='varchar' AND COLUMN_TYPE='varchar(255)' AND
      CHARACTER_SET_NAME='ascii' AND COLLATION_NAME='ascii_general_ci' AND IS_NULLABLE='NO' AND
      COLUMN_DEFAULT IS NULL AND EXTRA='' AND ORDINAL_POSITION=1) OR
     (COLUMN_NAME='sha256' AND DATA_TYPE='char' AND COLUMN_TYPE='char(64)' AND
      CHARACTER_SET_NAME='ascii' AND COLLATION_NAME='ascii_general_ci' AND IS_NULLABLE='NO' AND
      COLUMN_DEFAULT IS NULL AND EXTRA='' AND ORDINAL_POSITION=2) OR
     (COLUMN_NAME='applied_at' AND DATA_TYPE='timestamp' AND COLUMN_TYPE='timestamp' AND
      CHARACTER_SET_NAME IS NULL AND COLLATION_NAME IS NULL AND IS_NULLABLE='NO' AND EXTRA='' AND
      ORDINAL_POSITION=3 AND UPPER(COALESCE(COLUMN_DEFAULT, '')) IN ('CURRENT_TIMESTAMP', 'CURRENT_TIMESTAMP()'))
   )), ':',
  (SELECT COUNT(*) FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history'), ':',
  (SELECT COUNT(*) FROM information_schema.STATISTICS
   WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history' AND INDEX_NAME='PRIMARY' AND
     NON_UNIQUE=0 AND SEQ_IN_INDEX=1 AND COLUMN_NAME='filename' AND SUB_PART IS NULL AND
     INDEX_TYPE='BTREE' AND COLLATION='A'), ':',
  (SELECT COUNT(*) FROM information_schema.STATISTICS
   WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history' AND INDEX_NAME='PRIMARY'), ':',
  (SELECT COUNT(*) FROM information_schema.STATISTICS
   WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history')
);" 2>/dev/null)"
if [ "$history_table_shape" != "1:3:3:1:1:1" ]; then
  echo "Error: _migration_history requires approved exact-shape reconciliation" >&2
  exit 1
fi

invalid_history_digest_count="$(mysql_exec -N -e "
SELECT COUNT(*) FROM _migration_history
WHERE sha256 IS NULL OR CHAR_LENGTH(sha256) <> 64 OR sha256 REGEXP '[^0-9A-Fa-f]';" 2>/dev/null)"
if [ "$invalid_history_digest_count" != "0" ]; then
  echo "Error: _migration_history contains unreconciled content digests" >&2
  exit 1
fi

journal_table_shape="$(mysql_exec -N -e "
SELECT CONCAT(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_journal' AND ENGINE='InnoDB'), ':',
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_journal' AND (
    (COLUMN_NAME='filename' AND COLUMN_TYPE='varchar(255)' AND CHARACTER_SET_NAME='ascii' AND COLLATION_NAME='ascii_general_ci' AND IS_NULLABLE='NO' AND COLUMN_DEFAULT IS NULL AND EXTRA='' AND ORDINAL_POSITION=1) OR
    (COLUMN_NAME='sha256' AND COLUMN_TYPE='char(64)' AND CHARACTER_SET_NAME='ascii' AND COLLATION_NAME='ascii_general_ci' AND IS_NULLABLE='NO' AND COLUMN_DEFAULT IS NULL AND EXTRA='' AND ORDINAL_POSITION=2) OR
    (COLUMN_NAME='state' AND COLUMN_TYPE='varchar(16)' AND CHARACTER_SET_NAME='ascii' AND COLLATION_NAME='ascii_general_ci' AND IS_NULLABLE='NO' AND COLUMN_DEFAULT IS NULL AND EXTRA='' AND ORDINAL_POSITION=3) OR
    (COLUMN_NAME='started_at' AND COLUMN_TYPE='timestamp' AND IS_NULLABLE='NO' AND EXTRA='' AND ORDINAL_POSITION=4 AND UPPER(COALESCE(COLUMN_DEFAULT,'')) IN ('CURRENT_TIMESTAMP','CURRENT_TIMESTAMP()')) OR
    (COLUMN_NAME='completed_at' AND COLUMN_TYPE='timestamp' AND IS_NULLABLE='YES' AND COLUMN_DEFAULT IS NULL AND EXTRA='' AND ORDINAL_POSITION=5)
  )), ':',
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_journal'), ':',
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_journal' AND INDEX_NAME='PRIMARY' AND NON_UNIQUE=0 AND SEQ_IN_INDEX=1 AND COLUMN_NAME='filename' AND SUB_PART IS NULL AND INDEX_TYPE='BTREE' AND COLLATION='A'), ':',
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_journal' AND INDEX_NAME='PRIMARY'), ':',
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_journal')
);" 2>/dev/null)"
[ "$journal_table_shape" = "1:5:5:1:1:1" ] || { echo "Error: _migration_journal requires approved exact-shape reconciliation" >&2; exit 1; }

runner_lock_table_shape="$(mysql_exec -N -e "
SELECT CONCAT(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_runner_lock' AND ENGINE='InnoDB'), ':',
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_runner_lock' AND (
    (COLUMN_NAME='lock_name' AND COLUMN_TYPE='varchar(32)' AND CHARACTER_SET_NAME='ascii' AND COLLATION_NAME='ascii_general_ci' AND IS_NULLABLE='NO' AND COLUMN_DEFAULT IS NULL AND EXTRA='' AND ORDINAL_POSITION=1) OR
    (COLUMN_NAME='acquired_at' AND COLUMN_TYPE='timestamp' AND IS_NULLABLE='NO' AND EXTRA='' AND ORDINAL_POSITION=2 AND UPPER(COALESCE(COLUMN_DEFAULT,'')) IN ('CURRENT_TIMESTAMP','CURRENT_TIMESTAMP()'))
  )), ':',
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_runner_lock'), ':',
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_runner_lock' AND INDEX_NAME='PRIMARY' AND NON_UNIQUE=0 AND SEQ_IN_INDEX=1 AND COLUMN_NAME='lock_name' AND SUB_PART IS NULL AND INDEX_TYPE='BTREE' AND COLLATION='A'), ':',
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_runner_lock' AND INDEX_NAME='PRIMARY'), ':',
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_runner_lock')
);" 2>/dev/null)"
[ "$runner_lock_table_shape" = "1:2:2:1:1:1" ] || { echo "Error: _migration_runner_lock requires approved exact-shape reconciliation" >&2; exit 1; }

mysql_exec -e "
INSERT IGNORE INTO _migration_journal (filename, sha256, state, started_at, completed_at)
SELECT filename, sha256, 'APPLIED', applied_at, applied_at FROM _migration_history;
" 2>/dev/null

if ! mysql_exec -e "INSERT INTO _migration_runner_lock (lock_name) VALUES ('global');" 2>/dev/null; then
  echo "Error: migration runner is locked pending explicit reconciliation" >&2
  exit 1
fi
RUNNER_LOCK_ACQUIRED=1
[ "$T03_AUTH_ENGINE_BOUND_APPLY" = "0" ] || echo "  T03_BOUND_STAGE=runner-locked"

journal_integrity_count="$(mysql_exec -N -e "
SELECT
  (SELECT COUNT(*) FROM _migration_journal j LEFT JOIN _migration_history h ON h.filename=j.filename
   WHERE j.state NOT IN ('STARTED','APPLIED') OR j.sha256 IS NULL OR CHAR_LENGTH(j.sha256)<>64 OR j.sha256 REGEXP '[^0-9A-Fa-f]' OR
     j.state='STARTED' OR (j.state='APPLIED' AND (h.filename IS NULL OR h.sha256<>j.sha256)))
  +
  (SELECT COUNT(*) FROM _migration_history h LEFT JOIN _migration_journal j ON j.filename=h.filename
   WHERE j.filename IS NULL OR j.state<>'APPLIED' OR j.sha256<>h.sha256);
" 2>/dev/null)"
if [ "$journal_integrity_count" != "0" ]; then
  echo "Error: migration journal/history divergence requires explicit reconciliation" >&2
  exit 1
fi

validate_migration_file() {
  local file="$1"
  local filename parent_dir
  filename="$(basename "$file")"
  if [[ ! "$filename" =~ ^([0-9]{3})_[a-z0-9][a-z0-9_]*\.sql$ ]]; then
    echo "Error: invalid migration filename: ${filename}" >&2
    return 1
  fi
  parent_dir="$(cd "$(dirname "$file")" && pwd -P)"
  if [ "$parent_dir" != "$(cd "$MIGRATIONS_DIR" && pwd -P)" ] || [ -L "$file" ]; then
    echo "Error: migration must be a direct non-symlink child of ${MIGRATIONS_DIR}" >&2
    return 1
  fi
}

apply_migration() {
  local file="$1"
  local filename number digest stored_sha256 collision_count history_row_count journal_row_count
  local t03_preflight_helper t03_preflight_output t03_postflight_output
  local t03_before_rows t03_before_checksum t03_before_history_total t03_before_applied_total
  local t03_after_rows t03_after_checksum t03_expected_history_total t03_expected_applied_total
  validate_migration_file "$file" || exit 1
  filename="$(basename "$file")"
  number="${filename%%_*}"
  digest="$(shasum -a 256 "$file" | cut -d ' ' -f 1)"

  collision_count="$(mysql_exec -N -e "SELECT COUNT(*) FROM _migration_history WHERE LEFT(filename, 4) = '${number}_' AND filename <> '${filename}';" 2>/dev/null)"
  if [ "$collision_count" -ne 0 ]; then
    echo "  FAIL  ${filename} (migration number collision)" >&2
    exit 1
  fi

  # Check if already applied
  history_row_count="$(mysql_exec -N -e "SELECT COUNT(*) FROM _migration_history WHERE filename='${filename}';" 2>/dev/null)"
  if [ "$history_row_count" = "1" ]; then
    stored_sha256="$(mysql_exec -N -e "SELECT sha256 FROM _migration_history WHERE filename='${filename}';" 2>/dev/null)"
    if [ "$stored_sha256" != "$digest" ]; then
      echo "  FAIL  ${filename} (migration content digest mismatch)" >&2
      exit 1
    fi
    echo "  SKIP  ${filename} (already applied)"
    return 0
  elif [ "$history_row_count" != "0" ]; then
    echo "  FAIL  ${filename} (migration history cardinality mismatch)" >&2
    exit 1
  fi

  journal_row_count="$(mysql_exec -N -e "SELECT COUNT(*) FROM _migration_journal WHERE filename='${filename}';" 2>/dev/null)"
  if [ "$journal_row_count" != "0" ]; then
    echo "  FAIL  ${filename} (incomplete or divergent migration journal)" >&2
    exit 1
  fi

  if [ "$filename" = "040_convert_auth_transaction_boundary_to_innodb.sql" ]; then
    [ "$T03_AUTH_ENGINE_BOUND_APPLY" = "1" ] || {
      RUNNER_SAFE_TO_RELEASE=1
      echo "  FAIL  ${filename} (bound operational preflight required)" >&2
      exit 1
    }
    t03_preflight_helper="${SCRIPT_DIR}/backend/scripts/preflight-auth-transaction-boundary.sh"
    [ -f "$t03_preflight_helper" ] && [ ! -L "$t03_preflight_helper" ] && [ -x "$t03_preflight_helper" ] || {
      RUNNER_SAFE_TO_RELEASE=1
      echo "  FAIL  ${filename} (bound operational preflight unavailable)" >&2
      exit 1
    }
    echo "  T03_BOUND_STAGE=preflight-started"
    if ! t03_preflight_output="$("$t03_preflight_helper" "$MYSQL_OPTION_FILE" "$DB_NAME" 2>&1)"; then
      RUNNER_SAFE_TO_RELEASE=1
      echo "  FAIL  ${filename} (bound operational preflight failed)" >&2
      exit 1
    fi
    case "$t03_preflight_output" in *$'\n'*) RUNNER_SAFE_TO_RELEASE=1; echo "  FAIL  ${filename} (bound operational preflight malformed)" >&2; exit 1 ;; esac
    for required_token in T03_AUTH_ENGINE_PREFLIGHT=PASS writer_masks=verified database_locality=unix-socket execution_binding=pending-controller; do
      case " ${t03_preflight_output} " in
        *" ${required_token} "*) ;;
        *) RUNNER_SAFE_TO_RELEASE=1; echo "  FAIL  ${filename} (bound operational preflight malformed)" >&2; exit 1 ;;
      esac
    done
    echo "  T03_BOUND_PREFLIGHT=PASS connection=unix-socket masks=verified runner_lock=held"
    t03_before_rows="$(mysql_exec -N -B -e "SELECT COUNT(*) FROM WEO_MEMBER;" 2>/dev/null)"
    t03_before_checksum="$(mysql_exec -N -B -e "CHECKSUM TABLE WEO_MEMBER EXTENDED;" 2>/dev/null | cut -f2)"
    t03_before_history_total="$(mysql_exec -N -B -e "SELECT COUNT(*) FROM _migration_history;" 2>/dev/null)"
    t03_before_applied_total="$(mysql_exec -N -B -e "SELECT COUNT(*) FROM _migration_journal WHERE state='APPLIED';" 2>/dev/null)"
    for value in "$t03_before_rows" "$t03_before_checksum" "$t03_before_history_total" "$t03_before_applied_total"; do
      case "$value" in ''|*[!0-9]*) echo "  FAIL  ${filename} (bound baseline malformed)" >&2; exit 1 ;; esac
    done
  fi

  echo "  APPLY ${filename} ..."
  mysql_exec -e "INSERT INTO _migration_journal (filename, sha256, state) VALUES ('${filename}', '${digest}', 'STARTED');" 2>/dev/null
  if mysql_exec < "$file" 2>&1; then
    if [ "$filename" = "040_convert_auth_transaction_boundary_to_innodb.sql" ]; then
      if ! t03_postflight_output="$("$t03_preflight_helper" "$MYSQL_OPTION_FILE" "$DB_NAME" 2>&1)"; then
        echo "  FAIL  ${filename} (bound postflight failed)" >&2
        exit 1
      fi
      case "$t03_postflight_output" in *$'\n'*) echo "  FAIL  ${filename} (bound postflight malformed)" >&2; exit 1 ;; esac
      for required_token in T03_AUTH_ENGINE_PREFLIGHT=PASS engine=InnoDB state=target writer_masks=verified database_locality=unix-socket; do
        case " ${t03_postflight_output} " in
          *" ${required_token} "*) ;;
          *) echo "  FAIL  ${filename} (bound postflight malformed)" >&2; exit 1 ;;
        esac
      done
      t03_after_rows="$(mysql_exec -N -B -e "SELECT COUNT(*) FROM WEO_MEMBER;" 2>/dev/null)"
      t03_after_checksum="$(mysql_exec -N -B -e "CHECKSUM TABLE WEO_MEMBER EXTENDED;" 2>/dev/null | cut -f2)"
      [ "$t03_after_rows" = "$t03_before_rows" ] || { echo "  FAIL  ${filename} (row count changed)" >&2; exit 1; }
      [ "$t03_after_checksum" = "$t03_before_checksum" ] || { echo "  FAIL  ${filename} (row checksum changed)" >&2; exit 1; }
      [ "$(mysql_exec -N -B -e "SELECT COUNT(*) FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA=DATABASE() AND ROUTINE_TYPE='PROCEDURE' AND LEFT(ROUTINE_NAME,5)='_040_';" 2>/dev/null)" = "0" ] || { echo "  FAIL  ${filename} (procedure residue)" >&2; exit 1; }
      [ "$(mysql_exec -N -B -e "SELECT COUNT(*) FROM _migration_history;" 2>/dev/null)" = "$t03_before_history_total" ] || { echo "  FAIL  ${filename} (history changed before finalize)" >&2; exit 1; }
      [ "$(mysql_exec -N -B -e "SELECT COUNT(*) FROM _migration_journal WHERE state='APPLIED';" 2>/dev/null)" = "$t03_before_applied_total" ] || { echo "  FAIL  ${filename} (journal changed before finalize)" >&2; exit 1; }
      [ "$(mysql_exec -N -B -e "SELECT COUNT(*) FROM _migration_journal WHERE filename='${filename}' AND sha256='${digest}' AND state='STARTED';" 2>/dev/null)" = "1" ] || { echo "  FAIL  ${filename} (started journal missing)" >&2; exit 1; }
      [ "$(mysql_exec -N -B -e "SELECT COUNT(*) FROM _migration_runner_lock WHERE lock_name='global';" 2>/dev/null)" = "1" ] || { echo "  FAIL  ${filename} (runner lock missing)" >&2; exit 1; }
      echo "  T03_BOUND_ACCEPTANCE=PASS engine=InnoDB rows=${t03_after_rows} checksum=preserved history=pending journal=STARTED runner_lock=held"
    fi
    mysql_exec -e "START TRANSACTION;
      INSERT INTO _migration_history (filename, sha256) VALUES ('${filename}', '${digest}');
      UPDATE _migration_journal SET state='APPLIED', completed_at=CURRENT_TIMESTAMP
      WHERE filename='${filename}' AND sha256='${digest}' AND state='STARTED';
      COMMIT;" 2>/dev/null
    [ "$(mysql_exec -N -e "SELECT COUNT(*) FROM _migration_journal j JOIN _migration_history h ON h.filename=j.filename AND h.sha256=j.sha256 WHERE j.filename='${filename}' AND j.state='APPLIED';" 2>/dev/null)" = "1" ] || {
      echo "  FAIL  ${filename} (history/journal commit mismatch)" >&2
      exit 1
    }
    if [ "$filename" = "040_convert_auth_transaction_boundary_to_innodb.sql" ]; then
      t03_expected_history_total=$((t03_before_history_total + 1))
      t03_expected_applied_total=$((t03_before_applied_total + 1))
      [ "$(mysql_exec -N -B -e "SELECT COUNT(*) FROM _migration_history;" 2>/dev/null)" = "$t03_expected_history_total" ] || { echo "  FAIL  ${filename} (history finalize cardinality mismatch)" >&2; exit 1; }
      [ "$(mysql_exec -N -B -e "SELECT COUNT(*) FROM _migration_journal WHERE state='APPLIED';" 2>/dev/null)" = "$t03_expected_applied_total" ] || { echo "  FAIL  ${filename} (journal finalize cardinality mismatch)" >&2; exit 1; }
      echo "  T03_BOUND_FINALIZE=PASS history=${t03_expected_history_total} journal=${t03_expected_applied_total} runner_lock=held"
    fi
    echo "  OK    ${filename}"
  else
    echo "  FAIL  ${filename}"
    exit 1
  fi
}

# --seed mode: register migrations as applied without executing SQL
if [ "${1:-}" = "--seed" ]; then
  if [ -z "${2:-}" ]; then
    echo "Usage: ./migrate.sh --seed <up_to_number>"
    echo "  e.g. ./migrate.sh --seed 006  — marks 001~006 as applied"
    exit 1
  fi
  UP_TO="$2"
  if [[ ! "$UP_TO" =~ ^[0-9]{3}$ ]]; then
    echo "Error: seed upper bound must be exactly three digits" >&2
    exit 1
  fi
  echo "=== DB Migration (seed mode) ==="
  echo "  Target: ${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
  echo "  Seeding migrations up to: ${UP_TO}"
  echo ""
  for file in "$MIGRATIONS_DIR"/[0-9][0-9][0-9]_*.sql; do
    [ -f "$file" ] || continue
    validate_migration_file "$file" || exit 1
    filename="$(basename "$file")"
    file_num="${filename%%_*}"
    if [ "$file_num" -le "$UP_TO" ] 2>/dev/null; then
      digest="$(shasum -a 256 "$file" | cut -d ' ' -f 1)"
      history_row_count="$(mysql_exec -N -e "SELECT COUNT(*) FROM _migration_history WHERE filename='${filename}';" 2>/dev/null)"
      stored_sha256=""
      if [ "$history_row_count" = "1" ]; then
        stored_sha256="$(mysql_exec -N -e "SELECT sha256 FROM _migration_history WHERE filename='${filename}';" 2>/dev/null)"
      elif [ "$history_row_count" != "0" ]; then
        echo "  FAIL  ${filename} (migration history cardinality mismatch)" >&2
        exit 1
      fi
      collision_count="$(mysql_exec -N -e "SELECT COUNT(*) FROM _migration_history WHERE LEFT(filename, 4) = '${file_num}_' AND filename <> '${filename}';" 2>/dev/null)"
      if [ "$collision_count" -ne 0 ]; then
        echo "  FAIL  ${filename} (migration number collision)" >&2
        exit 1
      elif [ -n "$stored_sha256" ] && [ "$stored_sha256" != "$digest" ]; then
        echo "  FAIL  ${filename} (migration content digest mismatch)" >&2
        exit 1
      elif [ -n "$stored_sha256" ]; then
        echo "  SKIP  ${filename} (already in history)"
      else
        mysql_exec -e "START TRANSACTION;
          INSERT INTO _migration_history (filename, sha256) VALUES ('${filename}', '${digest}');
          INSERT INTO _migration_journal (filename, sha256, state, completed_at) VALUES ('${filename}', '${digest}', 'APPLIED', CURRENT_TIMESTAMP);
          COMMIT;" 2>/dev/null
        echo "  SEED  ${filename}"
      fi
    fi
  done
  echo ""
  echo "=== Seed complete ==="
  RUNNER_SAFE_TO_RELEASE=1
  exit 0
fi

echo "=== DB Migration ==="
echo "  Target: ${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
echo ""

if [ $# -ge 1 ]; then
  # Apply single file
  if [ ! -f "$1" ]; then
    echo "Error: file not found: $1"
    exit 1
  fi
  apply_migration "$1"
else
  # Apply all migrations in order
  found=0
  for file in "$MIGRATIONS_DIR"/[0-9][0-9][0-9]_*.sql; do
    [ -f "$file" ] || continue
    found=1
    apply_migration "$file"
  done
  if [ "$found" -eq 0 ]; then
    echo "  No migration files found in ${MIGRATIONS_DIR}"
  fi
fi

echo ""
echo "=== Done ==="
RUNNER_SAFE_TO_RELEASE=1
