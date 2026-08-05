#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
MIGRATIONS_DIR="${ROOT_DIR}/backend/migrations"
MANIFEST="${MIGRATIONS_DIR}/kakao-auth-036-039.sha256"
PREFLIGHT="${ROOT_DIR}/scripts/kakao-auth-rollout/preflight.sql"
POSTCHECK="${ROOT_DIR}/scripts/kakao-auth-rollout/postcheck.sql"
MODE="${1:-}"
PREFLIGHT_METRICS=$'missing_required_tables\nduplicate_user_provider_groups\nconflicting_provider_subject_groups\nunsupported_social_status_rows\ncohort_source_overflow_rows\ndepartment_source_overflow_rows'
POSTCHECK_METRICS=$'social_engine_not_innodb\nmissing_canonical_social_columns\ninvalid_social_status_column_shape\ninvalid_social_email_column_shape\nlegacy_active_status_rows\nduplicate_user_provider_groups\nconflicting_provider_subject_groups\nmissing_unique_user_provider_index\nmissing_unique_provider_subject_index\nmissing_refresh_rotation_columns\nmissing_refresh_sid_index\nmissing_auth_tables\ninvalid_auth_table_engines\nlegacy_verification_projection_mismatch\nlegacy_root_projection_missing'

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

file_mode() {
  stat -c '%a' "$1" 2>/dev/null || stat -f '%Lp' "$1"
}

file_owner_uid() {
  stat -c '%u' "$1" 2>/dev/null || stat -f '%u' "$1"
}

file_mtime() {
  stat -c '%Y' "$1" 2>/dev/null || stat -f '%m' "$1"
}

has_exact_line_once() {
  local path=$1 expected=$2 count
  count=$(grep -Fxc "$expected" "$path" || true)
  [[ $count == 1 ]]
}

database_drain_evidence_valid() {
  local sentinel=$1 evidence=$2 generation generation_count
  [[ $sentinel == /* && $evidence == /* ]] || return 1
  [[ -f $sentinel && ! -L $sentinel && -f $evidence && ! -L $evidence ]] || return 1
  [[ $(file_mode "$evidence") == 600 ]] || return 1
  [[ $(file_owner_uid "$evidence") == $(id -u) ]] || return 1
  [[ $(file_mtime "$evidence") -ge $(file_mtime "$sentinel") ]] || return 1
  generation_count=$(grep -c '^generation=' "$sentinel" || true)
  generation=$(sed -n 's/^generation=//p' "$sentinel")
  [[ $generation_count == 1 && $generation =~ ^[a-f0-9]{32}$ ]] || return 1
  has_exact_line_once "$evidence" 'state=PASS' || return 1
  has_exact_line_once "$evidence" 'kind=db-drain' || return 1
  has_exact_line_once "$evidence" "generation=$generation" || return 1
  has_exact_line_once "$evidence" 'open_transactions=0' || return 1
  has_exact_line_once "$evidence" 'other_connections=0' || return 1
  has_exact_line_once "$evidence" 'samples=3' || return 1
  [[ $(grep -c '^recorded_at=' "$evidence" || true) == 1 ]]
}

[[ "${MODE}" == "--preflight-only" || "${MODE}" == "--apply" ]] ||
  fail "usage: apply-migrations.sh --preflight-only|--apply"
[[ -n "${MYSQL_DEFAULTS_FILE:-}" && -r "${MYSQL_DEFAULTS_FILE}" ]] ||
  fail "MYSQL_DEFAULTS_FILE must reference a readable client option file"
[[ -n "${DB_NAME:-}" ]] || fail "DB_NAME is required"
[[ -s "${MANIFEST}" && -s "${PREFLIGHT}" && -s "${POSTCHECK}" ]] ||
  fail "migration manifest or validation SQL is missing"

(
  cd "${ROOT_DIR}"
  shasum -a 256 -c "${MANIFEST}" >/dev/null
) || fail "migration checksum verification failed"

MYSQL=(mysql --defaults-extra-file="${MYSQL_DEFAULTS_FILE}" --skip-ssl --batch --skip-column-names "${DB_NAME}")

query() {
  "${MYSQL[@]}" -e "$1"
}

apply_file() {
  "${MYSQL[@]}" < "$1"
}

assert_scalar() {
  local expected=$1
  local sql=$2
  local actual
  actual="$(query "${sql}")"
  [[ "${actual}" == "${expected}" ]] || fail "postcondition mismatch"
}

assert_live_database_drained() {
  local open_transactions other_connections
  open_transactions=$(query "SELECT COUNT(*) FROM information_schema.INNODB_TRX t JOIN information_schema.PROCESSLIST p ON p.ID=t.trx_mysql_thread_id WHERE p.DB=DATABASE()") ||
    fail "live database writer drain check failed"
  other_connections=$(query "SELECT COUNT(*) FROM information_schema.PROCESSLIST WHERE ID<>CONNECTION_ID()") ||
    fail "live database writer drain check failed"
  [[ $open_transactions == 0 && $other_connections == 0 ]] ||
    fail "live database writer drain check failed"
}

assert_zero_metrics() {
  local file=$1
  local expected_metrics=$2
  local output metric violations extra seen expected_metric
  local actual_count=0 expected_count
  output="$("${MYSQL[@]}" < "${file}")"
  [[ -n $output ]] || fail "validation metrics are incomplete"
  seen=$'\n'
  while IFS=$'\t' read -r metric violations extra; do
    [[ -n $metric && -n $violations && -z $extra ]] || fail "validation metrics are incomplete"
    grep -Fxq "$metric" <<< "$expected_metrics" || fail "validation metrics are incomplete"
    [[ $seen != *$'\n'"$metric"$'\n'* ]] || fail "validation metrics are incomplete"
    [[ "${violations}" == "0" ]] || fail "${metric} has ${violations} violation(s)"
    seen+="$metric"$'\n'
    actual_count=$((actual_count + 1))
  done <<< "${output}"
  expected_count=$(grep -c . <<< "$expected_metrics")
  [[ $actual_count -eq $expected_count ]] || fail "validation metrics are incomplete"
  while IFS= read -r expected_metric; do
    [[ $seen == *$'\n'"$expected_metric"$'\n'* ]] || fail "validation metrics are incomplete"
  done <<< "$expected_metrics"
}

VERSION="$(query 'SELECT VERSION()')"
[[ "${VERSION}" == 10.1.38-MariaDB* ]] || fail "MariaDB 10.1.38 is required"
assert_zero_metrics "${PREFLIGHT}" "$PREFLIGHT_METRICS"
printf 'PREFLIGHT PASS mariadb=10.1.38 checksums=verified\n'

if [[ "${MODE}" == "--preflight-only" ]]; then
  printf 'PASS: no migration was applied\n'
  exit 0
fi

if [[ "${KAKAO_AUTH_MIGRATION_APPROVED:-0}" != "1" ||
      "${MAINTENANCE_WRITES_FROZEN:-0}" != "1" ||
      "${VERIFIED_BACKUP_RESTORE:-0}" != "1" ]]; then
  fail "approval gates are incomplete"
fi

SENTINEL="${MAINTENANCE_SENTINEL_PATH:-/run/alumni/maintenance}"
DB_DRAIN_EVIDENCE="${MAINTENANCE_DB_DRAIN_EVIDENCE:-}"
MIGRATION_EVIDENCE_OUTPUT="${MAINTENANCE_MIGRATION_EVIDENCE_OUTPUT:-/run/alumni/migration-postcheck.pass}"
database_drain_evidence_valid "$SENTINEL" "$DB_DRAIN_EVIDENCE" ||
  fail "database drain evidence is invalid"
[[ $MIGRATION_EVIDENCE_OUTPUT == /* ]] || fail "migration evidence path is invalid"
[[ $MIGRATION_EVIDENCE_OUTPUT != "$SENTINEL" && $MIGRATION_EVIDENCE_OUTPUT != "$DB_DRAIN_EVIDENCE" ]] ||
  fail "migration evidence path is invalid"
migration_evidence_dir=$(dirname "$MIGRATION_EVIDENCE_OUTPUT")
[[ -d $migration_evidence_dir && ! -L $migration_evidence_dir ]] ||
  fail "migration evidence directory is invalid"
assert_live_database_drained

assert_scalar "1" "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history'"
assert_scalar "1" "SELECT COUNT(*) FROM _migration_history WHERE filename='035_create_push_preference.sql'"
assert_scalar "0" "SELECT COUNT(*) FROM _migration_history WHERE filename IN ('036_extend_mobile_refresh_token_for_rotation.sql','037_harden_member_social_links.sql','038_create_auth_principal_tables.sql','039_create_social_link_continuation.sql')"

apply_and_record() {
  local filename=$1
  apply_file "${MIGRATIONS_DIR}/${filename}"
  case "${filename}" in
    036_*)
      assert_scalar "3" "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_MOBILE_REFRESH_TOKEN' AND COLUMN_NAME IN ('CONSUMED_AT','REVOKED_AT','ROTATED_TO_JTI')"
      assert_scalar "0" "SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE MRT_REVOKED_AT IS NOT NULL AND REVOKED_AT IS NULL"
      ;;
    037_*)
      assert_scalar "InnoDB" "SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL'"
      assert_scalar "0" "SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL WHERE NMS_STATUS='Y'"
      assert_scalar "1" "SELECT COUNT(DISTINCT INDEX_NAME) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL' AND INDEX_NAME='UK_USR_PROVIDER'"
      assert_scalar "1" "SELECT COUNT(DISTINCT INDEX_NAME) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL' AND INDEX_NAME='UK_PROVIDER_SUBJECT'"
      ;;
    038_*)
      assert_scalar "2" "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('ALUMNI_ADMIN_ROLE','ALUMNI_VERIFICATION') AND ENGINE='InnoDB'"
      assert_scalar "0" "SELECT COUNT(*) FROM WEO_MEMBER m LEFT JOIN ALUMNI_VERIFICATION v ON v.USR_SEQ=m.USR_SEQ WHERE m.USR_STATUS IN ('BAA','BBB','CCC','ZZZ') AND v.USR_SEQ IS NULL"
      ;;
    039_*)
      assert_scalar "2" "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('ALUMNI_SOCIAL_LINK_REAUTH_GUARD','ALUMNI_SOCIAL_LINK_CONTINUATION') AND ENGINE='InnoDB'"
      ;;
  esac
  query "INSERT INTO _migration_history (filename) VALUES ('${filename}')" >/dev/null
  printf 'APPLY PASS %s\n' "${filename}"
}

apply_and_record 036_extend_mobile_refresh_token_for_rotation.sql
apply_and_record 037_harden_member_social_links.sql
apply_and_record 038_create_auth_principal_tables.sql
apply_and_record 039_create_social_link_continuation.sql
assert_zero_metrics "${POSTCHECK}" "$POSTCHECK_METRICS"
generation=$(sed -n 's/^generation=//p' "$SENTINEL")
recorded_at=$(date -u '+%Y-%m-%dT%H:%M:%SZ') || fail "migration evidence clock failed"
migration_evidence_tmp=$(mktemp "${MIGRATION_EVIDENCE_OUTPUT}.tmp.XXXXXX") ||
  fail "migration evidence create failed"
trap 'rm -f "$migration_evidence_tmp"' EXIT
printf 'state=PASS\nkind=migration-postcheck\ngeneration=%s\nrange=036-039\npostcheck_metrics=15\nrecorded_at=%s\n' \
  "$generation" "$recorded_at" > "$migration_evidence_tmp"
chmod 0600 "$migration_evidence_tmp"
mv -f "$migration_evidence_tmp" "$MIGRATION_EVIDENCE_OUTPUT"
trap - EXIT
printf 'MIGRATION PASS range=036-039 postcheck=zero\n'
