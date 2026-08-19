#!/usr/bin/env bash
# test-canonical-identity-migrations.sh — Disposable MariaDB 10.1.38 migration contract runner.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
MIGRATIONS_DIR="${BACKEND_DIR}/migrations"
TESTDATA_DIR="${MIGRATIONS_DIR}/testdata"
IMAGE="$(tr -d '[:space:]' < "${TESTDATA_DIR}/mariadb-10.1.38.image")"
APPROVED_IMAGE="mariadb@sha256:e4e5e5e2fb7c089688ddb55cc5ef38c9acff4aeb0aa25375f92f0708795b7a1c"
APPROVED_AUTHORITATIVE_MANIFEST_SHA256="a3d6035ec4804c4b06ea3419f045c71ed2cb26c997260b7496a9ca88cf5c4302"
APPROVED_HARNESS_FIXTURE_MANIFEST_SHA256="9d08eaef34c43abfaca76995de38e3826269de12d0f8b9117ca8234da46e92b4"
MODE="${1:-}"
CONTAINER=""
TEMP_DIR=""
VOLUME_IDS=""
READY_TIMEOUT_SECONDS="${HARNESS_READY_TIMEOUT_SECONDS:-120}"

case "$READY_TIMEOUT_SECONDS" in
  ''|*[!0-9]*|0) printf 'HARNESS_FAIL [REDACTED] invalid readiness timeout\n' >&2; exit 2 ;;
esac

usage() {
  printf 'Usage: %s --check-candidate-range|--check-fixture-lineage|--check-source-lineage|--expect-current-red|--self-test-migration-runner|--self-test-postconditions|--self-test-production-lineage-reconciliation|--self-test-runner-sql-mode|--self-test-t03-bound-apply|--self-test-t03-preflight|--self-test-t03-preflight-negative-controls|--self-test-t03-preservation|--self-test-t03-target-resume|--self-test-t03-transaction-boundary|--self-test-t03-unexpected-engine|--self-test-t04-additive-preparation|--self-test-t04-consent-session-outbox|--self-test-t04-credential-token-boundaries|--self-test-t04-identity-cardinality|--self-test-t04-maintenance-finalization|--verify\n' "$0"
}

fail() {
  printf 'HARNESS_FAIL [REDACTED] %s\n' "$1" >&2
  exit 1
}

[ "$IMAGE" = "$APPROVED_IMAGE" ] || fail "MariaDB image pin mismatch"

cleanup_resources() {
  local status=0
  local container="$CONTAINER"
  local temp_dir="$TEMP_DIR"
  local volume

  if [ -n "$CONTAINER" ]; then
    if [ "${HERMES_TEST_CLEANUP_COMMAND_FAILURE:-0}" = "1" ]; then
      docker rm -fv "$CONTAINER" >/dev/null 2>&1 || true
      status=125
    elif ! docker rm -fv "$CONTAINER" >/dev/null 2>&1; then
      status=125
    fi
    if docker inspect "$container" >/dev/null 2>&1; then
      status=125
    fi
    for volume in $VOLUME_IDS; do
      if docker volume inspect "$volume" >/dev/null 2>&1; then
        status=125
      fi
    done
    CONTAINER=""
  fi
  if [ -n "$TEMP_DIR" ]; then
    if ! rm -rf "$TEMP_DIR"; then
      status=125
    fi
    if [ -e "$temp_dir" ]; then
      status=125
    fi
    TEMP_DIR=""
  fi
  VOLUME_IDS=""
  return "$status"
}

cleanup_trap() {
  local original_status=$?
  trap - EXIT
  if ! cleanup_resources; then
    exit 125
  fi
  exit "$original_status"
}
trap cleanup_trap EXIT

case "$MODE" in
  --check-candidate-range|--check-fixture-lineage|--check-source-lineage|--expect-current-red|--self-test-migration-runner|--self-test-postconditions|--self-test-production-lineage-reconciliation|--self-test-runner-sql-mode|--self-test-t03-bound-apply|--self-test-t03-preflight|--self-test-t03-preflight-negative-controls|--self-test-t03-preservation|--self-test-t03-target-resume|--self-test-t03-transaction-boundary|--self-test-t03-unexpected-engine|--self-test-t04-additive-preparation|--self-test-t04-consent-session-outbox|--self-test-t04-credential-token-boundaries|--self-test-t04-identity-cardinality|--self-test-t04-maintenance-finalization|--verify) ;;
  *) usage; exit 2 ;;
esac

check_source_lineage() {
  local manifest="${TESTDATA_DIR}/canonical_identity_authoritative_lineage.sha256"
  local expected_digest filename number candidate actual_digest
  local candidate_count exact_candidate conflicts=0 missing=0

  while read -r expected_digest filename; do
    case "$expected_digest" in
      \#*|'') continue ;;
    esac

    number="${filename%%_*}"
    candidate_count=0
    exact_candidate=""
    for candidate in "${MIGRATIONS_DIR}/${number}_"*.sql; do
      [ -f "$candidate" ] || continue
      candidate_count=$((candidate_count + 1))
      if [ "$(basename "$candidate")" = "$filename" ]; then
        exact_candidate="$candidate"
      fi
    done

    if [ "$candidate_count" -eq 0 ]; then
      missing=$((missing + 1))
      continue
    fi
    if [ "$candidate_count" -ne 1 ] || [ -z "$exact_candidate" ]; then
      conflicts=$((conflicts + 1))
      continue
    fi

    actual_digest="$(shasum -a 256 "$exact_candidate" | cut -d ' ' -f 1)"
    if [ "$actual_digest" != "$expected_digest" ]; then
      conflicts=$((conflicts + 1))
    fi
  done < "$manifest"

  if [ "$conflicts" -ne 0 ] || [ "$missing" -ne 0 ]; then
    if [ "$MODE" = "--verify" ]; then
      fail "authoritative source lineage drift"
    fi
    printf 'AUTHORITATIVE_LINEAGE_DRIFT=EXPECTED_RED conflicting_numbers=%s missing_numbers=%s\n' \
      "$conflicts" "$missing"
    return 0
  fi

  printf 'AUTHORITATIVE_SOURCE_LINEAGE=PASS\n'
}

verify_authoritative_testdata_lineage() {
  local manifest="${TESTDATA_DIR}/canonical_identity_authoritative_lineage.sha256"
  local predecessor_root="${TESTDATA_DIR}/authoritative-164788c"
  local expected_digest filename number root candidate candidate_path actual_digest extra
  local candidate_count=0 entry_count=0 seen_numbers=" "

  [ -f "$manifest" ] && [ ! -L "$manifest" ] || fail "authoritative manifest missing or symlinked"
  [ "$(shasum -a 256 "$manifest" | cut -d ' ' -f 1)" = "$APPROVED_AUTHORITATIVE_MANIFEST_SHA256" ] || \
    fail "authoritative manifest authority mismatch"
  while read -r expected_digest filename extra; do
    case "$expected_digest" in
      \#*|'') continue ;;
    esac
    [ -z "${extra:-}" ] || fail "authoritative manifest malformed entry"
    [ "${#expected_digest}" -eq 64 ] || fail "authoritative manifest malformed digest"
    case "$expected_digest" in *[!0-9a-f]*) fail "authoritative manifest malformed digest" ;; esac
    [[ "$filename" =~ ^0(2[8-9]|3[0-9])_[a-z0-9][a-z0-9_]*\.sql$ ]] || \
      fail "authoritative manifest malformed filename"
    number="${filename%%_*}"
    case "$seen_numbers" in
      *" ${number} "*) fail "authoritative manifest duplicate migration number" ;;
    esac
    seen_numbers="${seen_numbers}${number} "

    case "$number" in
      028|029|030|031|032|033|034|035) root="$predecessor_root" ;;
      036|037|038|039) root="$MIGRATIONS_DIR" ;;
      *) fail "authoritative manifest unapproved migration number" ;;
    esac
    candidate_count=0
    candidate=""
    for candidate_path in "${root}/${number}_"*.sql; do
      [ -f "$candidate_path" ] || continue
      [ ! -L "$candidate_path" ] || fail "authoritative migration symlink rejected"
      candidate_count=$((candidate_count + 1))
      candidate="$candidate_path"
    done
    [ "$candidate_count" -eq 1 ] || fail "authoritative migration number cardinality mismatch"
    [ "$(basename "$candidate")" = "$filename" ] || fail "authoritative migration filename mismatch"
    actual_digest="$(shasum -a 256 "$candidate" | cut -d ' ' -f 1)"
    [ "$actual_digest" = "$expected_digest" ] || fail "authoritative migration digest mismatch"
    entry_count=$((entry_count + 1))
  done < "$manifest"

  [ "$entry_count" -eq 12 ] || fail "authoritative manifest cardinality mismatch"
  printf 'AUTHORITATIVE_TESTDATA_LINEAGE=PASS entries=12\n'
}

verify_harness_fixture_lineage() {
  local manifest="${TESTDATA_DIR}/canonical_identity_harness_fixtures.sha256"
  local expected_digest filename extra actual_digest entry_count=0 seen_paths=" "
  [ -f "$manifest" ] && [ ! -L "$manifest" ] || fail "harness fixture manifest missing or symlinked"
  [ "$(shasum -a 256 "$manifest" | cut -d ' ' -f 1)" = "$APPROVED_HARNESS_FIXTURE_MANIFEST_SHA256" ] || \
    fail "harness fixture manifest authority mismatch"
  while read -r expected_digest filename extra; do
    case "$expected_digest" in
      \#*|'') continue ;;
    esac
    [ -z "${extra:-}" ] || fail "harness fixture manifest malformed entry"
    [ "${#expected_digest}" -eq 64 ] || fail "harness fixture manifest malformed digest"
    case "$expected_digest" in *[!0-9a-f]*) fail "harness fixture manifest malformed digest" ;; esac
    case "$filename" in
      current-branch-8dcba0b/028_social_auth_security.sql|\
      canonical_identity_current_branch_pre_028_fixture.sql|\
      authoritative-164788c/kakao_auth_028_035_fixture.sql|\
      authoritative-164788c/kakao_auth_edge_cases.sql|\
      canonical_identity_fresh_fixture.sql|\
      canonical_identity_conflict_duplicate_provider_subject.sql|\
      canonical_identity_conflict_duplicate_normalized_email.sql|\
      canonical_identity_conflict_orphan_social_row.sql|\
      canonical_identity_conflict_malformed_identity.sql|\
      canonical_identity_conflict_unreadable_algorithm.sql|\
      canonical_identity_runner_fixture.sql|\
      canonical_identity_runner_started_failure_fixture.sql) ;;
      *) fail "harness fixture manifest contains an unapproved path" ;;
    esac
    case "$seen_paths" in
      *" ${filename} "*) fail "harness fixture manifest duplicate path" ;;
    esac
    seen_paths="${seen_paths}${filename} "
    [ -f "${TESTDATA_DIR}/${filename}" ] && [ ! -L "${TESTDATA_DIR}/${filename}" ] || \
      fail "harness fixture missing or symlinked"
    actual_digest="$(shasum -a 256 "${TESTDATA_DIR}/${filename}" | cut -d ' ' -f 1)"
    [ "$actual_digest" = "$expected_digest" ] || fail "harness fixture digest mismatch"
    entry_count=$((entry_count + 1))
  done < "$manifest"
  [ "$entry_count" -eq 12 ] || fail "harness fixture manifest cardinality mismatch"
  for filename in \
    current-branch-8dcba0b/028_social_auth_security.sql \
    canonical_identity_current_branch_pre_028_fixture.sql \
    authoritative-164788c/kakao_auth_028_035_fixture.sql \
    authoritative-164788c/kakao_auth_edge_cases.sql \
    canonical_identity_fresh_fixture.sql \
    canonical_identity_conflict_duplicate_provider_subject.sql \
    canonical_identity_conflict_duplicate_normalized_email.sql \
    canonical_identity_conflict_orphan_social_row.sql \
    canonical_identity_conflict_malformed_identity.sql \
    canonical_identity_conflict_unreadable_algorithm.sql \
    canonical_identity_runner_fixture.sql \
    canonical_identity_runner_started_failure_fixture.sql; do
    case "$seen_paths" in
      *" ${filename} "*) ;;
      *) fail "harness fixture manifest missing approved path" ;;
    esac
  done
}

candidate_manifest_is_valid() {
  local manifest="${TESTDATA_DIR}/canonical_identity_candidate_lineage.sha256"
  local expected_digest filename number candidate candidate_path actual_digest extra
  local candidate_count=0 source_count=0 entry_count=0 seen_numbers=" "
  [ -f "$manifest" ] && [ ! -L "$manifest" ] || fail "canonical candidate digest manifest missing"
  local approved_manifest_digest="${CANONICAL_CANDIDATE_MANIFEST_SHA256:-}"
  [ "${#approved_manifest_digest}" -eq 64 ] || fail "canonical candidate manifest approval missing"
  case "$approved_manifest_digest" in *[!0-9a-f]*) fail "canonical candidate manifest approval malformed" ;; esac
  [ "$(shasum -a 256 "$manifest" | cut -d ' ' -f 1)" = "$approved_manifest_digest" ] || \
    fail "canonical candidate manifest approval mismatch"

  while read -r expected_digest filename extra; do
    case "$expected_digest" in
      \#*|'') continue ;;
    esac
    [ -z "${extra:-}" ] || fail "candidate manifest malformed entry"
    [ "${#expected_digest}" -eq 64 ] || fail "candidate manifest malformed digest"
    case "$expected_digest" in *[!0-9a-f]*) fail "candidate manifest malformed digest" ;; esac
    if [[ ! "$filename" =~ ^([0-9]{3})_[a-z0-9][a-z0-9_]*\.sql$ ]]; then
      fail "candidate manifest filename invalid"
    fi
    number="${filename%%_*}"
    (( 10#$number >= 40 )) || fail "candidate manifest contains a historical migration"
    case "$seen_numbers" in
      *" ${number} "*) fail "candidate manifest duplicate migration number" ;;
    esac
    seen_numbers="${seen_numbers}${number} "

    candidate_count=0
    candidate=""
    for candidate_path in "${MIGRATIONS_DIR}/${number}_"*.sql; do
      [ -f "$candidate_path" ] || continue
      [ ! -L "$candidate_path" ] || fail "candidate migration symlink rejected"
      candidate_count=$((candidate_count + 1))
      candidate="$candidate_path"
    done
    [ "$candidate_count" -eq 1 ] || fail "candidate source migration number cardinality mismatch"
    [ "$(basename "$candidate")" = "$filename" ] || fail "candidate source migration filename mismatch"
    actual_digest="$(shasum -a 256 "$candidate" | cut -d ' ' -f 1)"
    [ "$actual_digest" = "$expected_digest" ] || fail "canonical candidate digest mismatch"
    entry_count=$((entry_count + 1))
  done < "$manifest"

  for candidate_path in "${MIGRATIONS_DIR}"/[0-9][0-9][0-9]_*.sql; do
    [ -f "$candidate_path" ] || continue
    filename="$(basename "$candidate_path")"
    number="${filename%%_*}"
    if (( 10#$number >= 40 )); then
      source_count=$((source_count + 1))
    fi
  done
  [ "$entry_count" -eq "$source_count" ] || fail "candidate manifest cardinality mismatch"
}

check_candidate_range() {
  local missing=0
  local filename
  for filename in \
    040_convert_auth_transaction_boundary_to_innodb.sql \
    041_create_canonical_identity_schema.sql \
    042_prepare_canonical_auth_cutover.sql; do
    if [ ! -f "${MIGRATIONS_DIR}/${filename}" ]; then
      missing=$((missing + 1))
    fi
  done

  if [ "$missing" -ne 0 ]; then
    if [ "$MODE" = "--verify" ]; then
      fail "canonical candidate migration range incomplete"
    fi
    printf 'CANONICAL_CANDIDATE_RANGE=EXPECTED_RED missing_numbers=%s maintenance_excluded=043\n' "$missing"
    return 0
  fi

  candidate_manifest_is_valid
  schema_contract_is_pinned || fail "canonical schema contract missing or unpinned"
  printf 'CANONICAL_CANDIDATE_RANGE=PASS maintenance_excluded=043\n'
}

start_container() {
  local scenario="$1"
  cleanup_resources || exit 125
  TEMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/canonical-identity-mariadb.XXXXXX")"
  CONTAINER="canonical-identity-${scenario}-$$-${RANDOM}"
  local env_file="${TEMP_DIR}/container.env"
  local password
  password="$(openssl rand -hex 24)"
  umask 077
  printf 'MYSQL_ROOT_PASSWORD=%s\n' "$password" > "$env_file"
  unset password

  if ! docker run -d --name "$CONTAINER" --env-file "$env_file" \
      "$IMAGE" --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci \
      >"${TEMP_DIR}/container.id" 2>"${TEMP_DIR}/docker.log"; then
    fail "container start failed"
  fi
  rm -f "$env_file"
  [ ! -e "$env_file" ] || fail "temporary container credential cleanup failed"

  VOLUME_IDS="$(docker inspect --format '{{range .Mounts}}{{if eq .Type "volume"}}{{println .Name}}{{end}}{{end}}' "$CONTAINER" 2>/dev/null)" || \
    fail "container volume inventory failed"
  [ -n "$VOLUME_IDS" ] || fail "container-owned volume missing"

  local deadline init_ready=0 authenticated_ready=0 server_version
  deadline=$((SECONDS + READY_TIMEOUT_SECONDS))
  while [ "$SECONDS" -lt "$deadline" ]; do
    if docker logs "$CONTAINER" 2>&1 | grep -Fq 'MySQL init process done. Ready for start up.'; then
      init_ready=1
    fi
    if [ "$init_ready" = "1" ] && docker exec "$CONTAINER" sh -c \
        'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot --protocol=tcp -h127.0.0.1 --batch --skip-column-names -e "SELECT 1"' \
        >/dev/null 2>&1; then
      authenticated_ready=1
      break
    fi
    sleep 1
  done
  if [ "$init_ready" = "1" ] && [ "$authenticated_ready" = "1" ]; then
    server_version="$(docker exec "$CONTAINER" sh -c \
      'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot --protocol=tcp -h127.0.0.1 --batch --skip-column-names -e "SELECT VERSION()"' \
      2>/dev/null | tr -d '\r')" || fail "database version query failed"
    case "$server_version" in
      10.1.38-MariaDB*) ;;
      *) fail "database version mismatch" ;;
    esac
    return 0
  fi
  fail "database readiness timeout"
}

mysql_without_database() {
  docker exec -i "$CONTAINER" sh -c \
    'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysql -uroot --batch --skip-column-names'
}

mysql_database() {
  docker exec -i "$CONTAINER" sh -c \
    'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysql -uroot --batch --skip-column-names canonical_identity_test'
}

mysql_error_code() {
  local error_file="$1"
  local code
  code="$(grep -Eo '^ERROR [0-9]+' "$error_file" 2>/dev/null | head -n 1 | tr ' ' '_' || true)"
  printf '%s' "${code:-UNKNOWN}"
}

load_fixture() {
  local fixture="$1"
  if ! mysql_without_database < "$fixture" >"${TEMP_DIR}/mysql.out" 2>"${TEMP_DIR}/mysql.err"; then
    fail "fixture load failed code=$(mysql_error_code "${TEMP_DIR}/mysql.err")"
  fi
  : >"${TEMP_DIR}/mysql.out"
  : >"${TEMP_DIR}/mysql.err"
}

create_test_database() {
  if ! mysql_without_database >"${TEMP_DIR}/mysql.out" 2>"${TEMP_DIR}/mysql.err" <<'SQL'
DROP DATABASE IF EXISTS canonical_identity_test;
CREATE DATABASE canonical_identity_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE canonical_identity_test;
CREATE TABLE _migration_history (
    filename VARCHAR(255) CHARACTER SET ascii NOT NULL PRIMARY KEY,
    sha256 CHAR(64) CHARACTER SET ascii NOT NULL,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
SQL
  then
    fail "test database creation failed code=$(mysql_error_code "${TEMP_DIR}/mysql.err")"
  fi
  : >"${TEMP_DIR}/mysql.out"
  : >"${TEMP_DIR}/mysql.err"
}

load_database_fixture() {
  local fixture="$1"
  if ! mysql_database < "$fixture" >"${TEMP_DIR}/mysql.out" 2>"${TEMP_DIR}/mysql.err"; then
    fail "database fixture load failed code=$(mysql_error_code "${TEMP_DIR}/mysql.err")"
  fi
  : >"${TEMP_DIR}/mysql.out"
  : >"${TEMP_DIR}/mysql.err"
}

query_scalar() {
  local sql="$1"
  printf '%s\n' "$sql" | mysql_database 2>"${TEMP_DIR}/mysql.err" | tr -d '[:space:]'
}

apply_migration() {
  local migration="$1"
  local filename digest
  filename="$(basename "$migration")"
  digest="$(shasum -a 256 "$migration" | cut -d ' ' -f 1)"
  if ! mysql_database < "$migration" >"${TEMP_DIR}/migration.out" 2>"${TEMP_DIR}/migration.err"; then
    : >"${TEMP_DIR}/migration.out"
    return 1
  fi
  printf "INSERT INTO _migration_history (filename, sha256) VALUES ('%s', '%s');\n" "$filename" "$digest" | \
    mysql_database >/dev/null 2>"${TEMP_DIR}/mysql.err" || fail "migration history write failed"
  : >"${TEMP_DIR}/migration.out"
  : >"${TEMP_DIR}/migration.err"
}

run_current_branch_lineage_probe() {
  start_container "current-028-to-036"
  load_fixture "${TESTDATA_DIR}/canonical_identity_current_branch_pre_028_fixture.sql"

  apply_migration "${TESTDATA_DIR}/current-branch-8dcba0b/028_social_auth_security.sql" || \
    fail "current branch migration 028 setup failed"

  if apply_migration "${MIGRATIONS_DIR}/036_extend_mobile_refresh_token_for_rotation.sql"; then
    fail "current branch 028 to immutable 036 lineage unexpectedly succeeded"
  fi

  local code residual_procedures migration_036_history
  code="$(mysql_error_code "${TEMP_DIR}/migration.err")"
  residual_procedures="$(query_scalar "SELECT COUNT(*) FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA=DATABASE() AND ROUTINE_TYPE='PROCEDURE' AND LEFT(ROUTINE_NAME, 5)='_036_';")"
  migration_036_history="$(query_scalar "SELECT COUNT(*) FROM _migration_history WHERE filename='036_extend_mobile_refresh_token_for_rotation.sql';")"

  [ "$code" = "ERROR_1054" ] || fail "current branch lineage returned an unexpected error class"
  [ "$residual_procedures" = "2" ] || fail "current branch lineage residual DDL mismatch"
  [ "$migration_036_history" = "0" ] || fail "failed migration was recorded in history"

  printf 'CURRENT_BRANCH_028_TO_036=EXPECTED_RED code=ERROR_1054 residual_procedures=2\n'
}

apply_historical_migrations() {
  local number file
  for number in 036 037 038 039; do
    file="$(find "$MIGRATIONS_DIR" -maxdepth 1 -type f -name "${number}_*.sql" -print)"
    [ -n "$file" ] || fail "historical migration missing"
    apply_migration "$file" || fail "historical migration ${number} failed"
  done
}

apply_authoritative_predecessors() {
  local root="${TESTDATA_DIR}/authoritative-164788c"
  local number file
  for number in 028 029 030 031 032 033 034 035; do
    file="$(find "$root" -maxdepth 1 -type f -name "${number}_*.sql" -print)"
    [ -n "$file" ] || fail "authoritative predecessor migration missing"
    apply_migration "$file" || fail "authoritative predecessor migration ${number} failed"
  done
}

run_authoritative_upgrade_probe() {
  start_container "authoritative-upgrade"
  create_test_database
  load_database_fixture "${TESTDATA_DIR}/authoritative-164788c/kakao_auth_028_035_fixture.sql"
  apply_authoritative_predecessors
  apply_historical_migrations

  local history_count social_engine refresh_columns admin_rows verification_rows trigger_count
  history_count="$(query_scalar "SELECT COUNT(*) FROM _migration_history WHERE filename REGEXP '^0(2[8-9]|3[0-9])_';")"
  social_engine="$(query_scalar "SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL';")"
  refresh_columns="$(query_scalar "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_MOBILE_REFRESH_TOKEN' AND COLUMN_NAME IN ('MRT_REVOKED_AT','CONSUMED_AT','REVOKED_AT','ROTATED_TO_JTI');")"
  admin_rows="$(query_scalar "SELECT COUNT(*) FROM ALUMNI_ADMIN_ROLE;")"
  verification_rows="$(query_scalar "SELECT COUNT(*) FROM ALUMNI_VERIFICATION;")"
  trigger_count="$(query_scalar "SELECT COUNT(*) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=DATABASE() AND TRIGGER_NAME IN ('TRG_WEO_MEMBER_AUTH_PRINCIPAL_INSERT','TRG_WEO_MEMBER_AUTH_PRINCIPAL_UPDATE');")"

  [ "$history_count" = "12" ] || fail "authoritative migration history mismatch"
  [ "$social_engine" = "InnoDB" ] || fail "authoritative social engine mismatch"
  [ "$refresh_columns" = "4" ] || fail "authoritative refresh-token shape mismatch"
  [ "$admin_rows" = "1" ] || fail "authoritative admin backfill mismatch"
  [ "$verification_rows" = "4" ] || fail "authoritative verification backfill mismatch"
  [ "$trigger_count" = "2" ] || fail "authoritative trigger shape mismatch"

  if [ "$MODE" = "--verify" ]; then
    apply_canonical_migrations
    assert_canonical_metadata || fail "canonical metadata postcondition failed"
    if ! mysql_database >/dev/null 2>"${TEMP_DIR}/mysql.err" <<'SQL'
ALTER TABLE AUTH_PASSWORD_CREDENTIAL
    DROP FOREIGN KEY FK_AUTH_PASSWORD_CREDENTIAL_IDENTITY;
ALTER TABLE AUTH_PASSWORD_CREDENTIAL
    MODIFY PASSWORD_HASH TEXT NULL;
DROP TRIGGER TRG_AUTH_PROVIDER_CREDENTIAL_OWNER_UPDATE;
UPDATE _migration_history
SET sha256 = REPEAT('0', 64)
WHERE filename = '041_create_canonical_identity_schema.sql';
SQL
    then
      fail "canonical metadata mutation negative-control setup failed"
    fi
    if assert_canonical_metadata; then
      fail "mutated canonical metadata passed exact contract"
    fi
    printf 'AUTHORITATIVE_GIT_UPGRADE=VERIFIED history=12 exact_metadata_negative=reject\n'
  else
    [ "$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='AUTH_IDENTITY';")" = "0" ] || \
      fail "current-red expectation is stale"
    printf 'AUTHORITATIVE_GIT_UPGRADE=EXPECTED_RED missing canonical schema history=12\n'
  fi
}

apply_canonical_migrations() {
  local manifest="${TESTDATA_DIR}/canonical_identity_candidate_lineage.sha256"
  local expected_digest filename actual_digest applied=0

  candidate_manifest_is_valid
  while read -r expected_digest filename; do
    case "$expected_digest" in
      \#*|'') continue ;;
    esac
    case "$filename" in
      040_convert_auth_transaction_boundary_to_innodb.sql|\
      041_create_canonical_identity_schema.sql|\
      042_prepare_canonical_auth_cutover.sql) ;;
      *) continue ;;
    esac

    [ -f "${MIGRATIONS_DIR}/${filename}" ] || fail "canonical candidate migration missing"
    actual_digest="$(shasum -a 256 "${MIGRATIONS_DIR}/${filename}" | cut -d ' ' -f 1)"
    [ "$actual_digest" = "$expected_digest" ] || fail "canonical candidate digest mismatch"
    apply_migration "${MIGRATIONS_DIR}/${filename}" || fail "canonical candidate migration failed"
    applied=$((applied + 1))
  done < "$manifest"

  [ "$applied" -eq 3 ] || fail "canonical candidate manifest cardinality mismatch"
}

schema_contract_is_pinned() {
  local contract="${TESTDATA_DIR}/canonical_identity_schema_contract.sql"
  local manifest="${TESTDATA_DIR}/canonical_identity_schema_contract.sha256"
  local expected_digest filename extra actual_digest pinned_digest="" entry_count=0
  local approved_manifest_digest="${CANONICAL_SCHEMA_CONTRACT_MANIFEST_SHA256:-}"
  [ -f "$contract" ] && [ ! -L "$contract" ] && [ -f "$manifest" ] && [ ! -L "$manifest" ] || return 1
  [ "${#approved_manifest_digest}" -eq 64 ] || return 1
  case "$approved_manifest_digest" in *[!0-9a-f]*) return 1 ;; esac
  [ "$(shasum -a 256 "$manifest" | cut -d ' ' -f 1)" = "$approved_manifest_digest" ] || return 1
  while read -r expected_digest filename extra || [ -n "${expected_digest:-}${filename:-}${extra:-}" ]; do
    [ -n "${expected_digest:-}" ] && [ -n "${filename:-}" ] && [ -z "${extra:-}" ] || return 1
    [ "$filename" = "canonical_identity_schema_contract.sql" ] || return 1
    [ "${#expected_digest}" -eq 64 ] || return 1
    case "$expected_digest" in *[!0-9a-f]*) return 1 ;; esac
    pinned_digest="$expected_digest"
    entry_count=$((entry_count + 1))
  done < "$manifest"
  [ "$entry_count" -eq 1 ] || return 1
  actual_digest="$(shasum -a 256 "$contract" | cut -d ' ' -f 1)"
  [ "$actual_digest" = "$pinned_digest" ] || return 1
  ! grep -Fxq "SELECT 'CANONICAL_SCHEMA_CONTRACT_PASS';" "$contract" || return 1
  for marker in \
    canonical_identity_schema_contract_v1 \
    information_schema.COLUMNS \
    information_schema.STATISTICS \
    information_schema.TABLE_CONSTRAINTS \
    information_schema.TRIGGERS \
    _migration_history \
    CANONICAL_SCHEMA_CONTRACT_FAIL; do
    grep -Fq "$marker" "$contract" || return 1
  done
}

assert_canonical_metadata() {
  local contract="${TESTDATA_DIR}/canonical_identity_schema_contract.sql"
  local result
  schema_contract_is_pinned || return 1
  result="$(mysql_database < "$contract" 2>"${TEMP_DIR}/mysql.err" | tr -d '\r')" || return 1
  [ "$result" = "CANONICAL_SCHEMA_CONTRACT_PASS" ]
}

run_postcondition_negative_control() {
  start_container "postcondition-negative"
  create_test_database
  if ! mysql_database >/dev/null 2>"${TEMP_DIR}/mysql.err" <<'SQL'
CREATE TABLE AUTH_IDENTITY (
    ID BIGINT NOT NULL PRIMARY KEY
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE AUTH_ACCOUNT_STATE (ID BIGINT NOT NULL PRIMARY KEY) ENGINE=InnoDB;
CREATE TABLE AUTH_PASSWORD_CREDENTIAL (ID BIGINT NOT NULL PRIMARY KEY) ENGINE=InnoDB;
CREATE TABLE AUTH_PROVIDER_CREDENTIAL (ID BIGINT NOT NULL PRIMARY KEY) ENGINE=InnoDB;
CREATE TABLE AUTH_CONSENT (ID BIGINT NOT NULL PRIMARY KEY) ENGINE=InnoDB;
CREATE TABLE AUTH_EMAIL_VERIFICATION (ID BIGINT NOT NULL PRIMARY KEY) ENGINE=InnoDB;
CREATE TABLE AUTH_SIGNUP_CONTINUATION (ID BIGINT NOT NULL PRIMARY KEY) ENGINE=InnoDB;
CREATE TABLE AUTH_SESSION_FAMILY (ID BIGINT NOT NULL PRIMARY KEY) ENGINE=InnoDB;
CREATE TABLE AUTH_PROVIDER_REVOKE_OUTBOX (ID BIGINT NOT NULL PRIMARY KEY) ENGINE=InnoDB;
CREATE TABLE AUTH_IDENTITY_MIGRATION_RUN (ID BIGINT NOT NULL PRIMARY KEY) ENGINE=InnoDB;
CREATE TABLE WEO_MEMBER (USR_SEQ INT NOT NULL PRIMARY KEY) ENGINE=InnoDB;
CREATE TABLE WEO_MEMBER_SOCIAL (NMS_SEQ INT NOT NULL PRIMARY KEY) ENGINE=InnoDB;
INSERT INTO _migration_history (filename, sha256) VALUES
  ('040_convert_auth_transaction_boundary_to_innodb.sql', REPEAT('0', 64)),
  ('041_create_canonical_identity_schema.sql', REPEAT('1', 64)),
  ('042_prepare_canonical_auth_cutover.sql', REPEAT('2', 64));
SQL
  then
    fail "postcondition negative fixture failed"
  fi

  if assert_canonical_metadata; then
    fail "incomplete canonical schema passed postconditions"
  fi
  cleanup_resources || exit 125
  trap - EXIT
  printf 'POSTCONDITION_NEGATIVE_CONTROL=PASS\n'
}

run_runner_sql_mode_probe() {
  start_container "runner-sql-mode"
  create_test_database
  query_scalar "INSERT INTO _migration_history (filename, sha256) VALUES ('036_conflict.sql', REPEAT('0', 64)); SELECT 1;" >/dev/null

  local default_result no_backslash_result
  default_result="$(query_scalar "SELECT COUNT(*) FROM _migration_history WHERE LEFT(filename, 4) = '036_';")"
  no_backslash_result="$(query_scalar "SET SESSION sql_mode='NO_BACKSLASH_ESCAPES'; SELECT COUNT(*) FROM _migration_history WHERE LEFT(filename, 4) = '036_';")"
  [ "$default_result" = "1" ] || fail "default SQL-mode collision expression mismatch"
  [ "$no_backslash_result" = "1" ] || fail "NO_BACKSLASH_ESCAPES collision expression mismatch"

  cleanup_resources || exit 125
  trap - EXIT
  printf 'RUNNER_COLLISION_SQL_MODE=PASS default=1 no_backslash_escapes=1 paths=normal,seed\n'
}

prepare_production_runner_fixture() {
  local runner_root="/tmp/canonical-migration-runner"
  docker exec "$CONTAINER" sh -c \
    'rm -rf /tmp/canonical-migration-runner && mkdir -p /tmp/canonical-migration-runner/backend/migrations && umask 077 && printf "DB_HOST=127.0.0.1\nDB_PORT=3306\nDB_USER=root\nDB_PASSWORD=%s\nDB_NAME=canonical_identity_test\n" "$MYSQL_ROOT_PASSWORD" > /tmp/canonical-migration-runner/backend/.env' \
    >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "migration runner credential fixture setup failed"
  docker cp "${BACKEND_DIR}/../migrate.sh" "${CONTAINER}:${runner_root}/migrate.sh" \
    >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "migration runner copy failed"
  docker cp "${TESTDATA_DIR}/canonical_identity_runner_fixture.sql" \
    "${CONTAINER}:${runner_root}/backend/migrations/001_fixture.sql" \
    >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "migration runner SQL fixture copy failed"
  docker exec "$CONTAINER" chmod 700 "${runner_root}/migrate.sh" \
    >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "migration runner permission setup failed"
}

reset_runner_database() {
  local shape="$1"
  create_test_database
  case "$shape" in
    none)
      query_scalar "DROP TABLE _migration_history; SELECT 1;" >/dev/null
      ;;
    exact) ;;
    filename_only)
      query_scalar "DROP TABLE _migration_history; CREATE TABLE _migration_history (filename VARCHAR(255) CHARACTER SET ascii NOT NULL PRIMARY KEY, applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4; SELECT 1;" >/dev/null
      ;;
    wrong_shape)
      query_scalar "DROP TABLE _migration_history; CREATE TABLE _migration_history (filename VARCHAR(255) CHARACTER SET ascii NOT NULL PRIMARY KEY, sha256 VARCHAR(64) CHARACTER SET utf8mb4 NULL, applied_at TIMESTAMP NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4; SELECT 1;" >/dev/null
      ;;
    prefix_primary_key)
      query_scalar "ALTER TABLE _migration_history DROP PRIMARY KEY, ADD PRIMARY KEY (filename(4)); SELECT 1;" >/dev/null
      ;;
    extra_column)
      query_scalar "ALTER TABLE _migration_history ADD COLUMN unexpected_state INT NULL; SELECT 1;" >/dev/null
      ;;
    extra_index)
      query_scalar "CREATE INDEX idx_unexpected_applied_at ON _migration_history (applied_at); SELECT 1;" >/dev/null
      ;;
    malformed_digest)
      query_scalar "INSERT INTO _migration_history (filename, sha256) VALUES ('999_legacy.sql', REPEAT('z', 64)); SELECT 1;" >/dev/null
      ;;
    collision)
      query_scalar "INSERT INTO _migration_history (filename, sha256) VALUES ('001_other.sql', REPEAT('0', 64)); SELECT 1;" >/dev/null
      ;;
    *) fail "unknown migration runner database shape" ;;
  esac
}

run_production_runner() {
  local path="$1"
  case "$path" in
    normal)
      docker exec "$CONTAINER" /tmp/canonical-migration-runner/migrate.sh \
        /tmp/canonical-migration-runner/backend/migrations/001_fixture.sql
      ;;
    all)
      docker exec "$CONTAINER" /tmp/canonical-migration-runner/migrate.sh
      ;;
    seed)
      docker exec "$CONTAINER" /tmp/canonical-migration-runner/migrate.sh --seed 001
      ;;
    *) return 2 ;;
  esac
}

run_production_candidate_runner() {
  local approved_manifest_digest="${1:-}"
  local runner=/tmp/canonical-migration-runner/migrate.sh
  local candidate=/tmp/canonical-migration-runner/backend/migrations/040_candidate.sql
  if [ -n "$approved_manifest_digest" ]; then
    docker exec -e CANONICAL_CANDIDATE_MANIFEST_SHA256="$approved_manifest_digest" \
      "$CONTAINER" "$runner" "$candidate"
  else
    docker exec "$CONTAINER" "$runner" "$candidate"
  fi
}

expect_production_runner_failure() {
  local path="$1"
  if run_production_runner "$path" >"${TEMP_DIR}/runner.out" 2>"${TEMP_DIR}/runner.err"; then
    fail "migration runner scenario unexpectedly succeeded"
  fi
  : >"${TEMP_DIR}/runner.out"
  : >"${TEMP_DIR}/runner.err"
}

run_exact_migration_runner_probe() {
  start_container "exact-migration-runner"
  prepare_production_runner_fixture
  local fixture_digest failure_digest path sql_mode fixture_count history_count
  fixture_digest="$(shasum -a 256 "${TESTDATA_DIR}/canonical_identity_runner_fixture.sql" | cut -d ' ' -f 1)"

  reset_runner_database none
  run_production_runner normal >"${TEMP_DIR}/runner.out" 2>"${TEMP_DIR}/runner.err" || \
    fail "fresh migration runner apply failed"
  fixture_count="$(query_scalar "SELECT COUNT(*) FROM RUNNER_FIXTURE;")"
  history_count="$(query_scalar "SELECT COUNT(*) FROM _migration_history WHERE filename='001_fixture.sql' AND sha256='${fixture_digest}';")"
  [ "$fixture_count" = "0" ] && [ "$history_count" = "1" ] || fail "fresh migration runner postcondition mismatch"

  run_production_runner normal >"${TEMP_DIR}/runner.out" 2>"${TEMP_DIR}/runner.err" || \
    fail "reconciled migration runner rerun failed"
  grep -Fq 'SKIP  001_fixture.sql' "${TEMP_DIR}/runner.out" || fail "reconciled migration runner did not skip"

  [ "$(query_scalar "SELECT COUNT(*) FROM RUNNER_REPLAY_WITNESS;")" = "1" ] || fail "initial replay witness mismatch"
  query_scalar "DELETE FROM _migration_history WHERE filename='001_fixture.sql'; SELECT 1;" >/dev/null
  expect_production_runner_failure normal
  [ "$(query_scalar "SELECT COUNT(*) FROM RUNNER_REPLAY_WITNESS;")" = "1" ] || \
    fail "schema-present/history-absent state replayed migration SQL"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_journal WHERE filename='001_fixture.sql' AND sha256='${fixture_digest}' AND state='APPLIED';")" = "1" ] || \
    fail "schema-present/history-absent journal evidence missing"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_runner_lock WHERE lock_name='global';")" = "1" ] || \
    fail "unsafe replay state did not retain the global runner lock"

  docker cp "${TESTDATA_DIR}/canonical_identity_runner_started_failure_fixture.sql" \
    "${CONTAINER}:/tmp/canonical-migration-runner/backend/migrations/001_fixture.sql" \
    >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "STARTED failure fixture copy failed"
  failure_digest="$(shasum -a 256 "${TESTDATA_DIR}/canonical_identity_runner_started_failure_fixture.sql" | cut -d ' ' -f 1)"
  reset_runner_database none
  expect_production_runner_failure normal
  [ "$(query_scalar "SELECT COUNT(*) FROM RUNNER_STARTED_FAILURE_WITNESS;")" = "1" ] || \
    fail "failed migration did not retain its durable SQL witness"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_history WHERE filename='001_fixture.sql';")" = "0" ] || \
    fail "failed migration was recorded in history"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_journal WHERE filename='001_fixture.sql' AND sha256='${failure_digest}' AND state='STARTED';")" = "1" ] || \
    fail "failed migration did not retain exact STARTED journal evidence"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_runner_lock WHERE lock_name='global';")" = "1" ] || \
    fail "failed migration did not retain the global runner lock"
  expect_production_runner_failure normal
  [ "$(query_scalar "SELECT COUNT(*) FROM RUNNER_STARTED_FAILURE_WITNESS;")" = "1" ] || \
    fail "STARTED migration replayed SQL"
  docker cp "${TESTDATA_DIR}/canonical_identity_runner_fixture.sql" \
    "${CONTAINER}:/tmp/canonical-migration-runner/backend/migrations/001_fixture.sql" \
    >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "normal runner fixture restore failed"

  reset_runner_database filename_only
  expect_production_runner_failure normal
  [ "$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='RUNNER_FIXTURE';")" = "0" ] || \
    fail "filename-only history executed migration SQL"

  reset_runner_database wrong_shape
  expect_production_runner_failure normal
  [ "$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='RUNNER_FIXTURE';")" = "0" ] || \
    fail "wrong-shape history executed migration SQL"

  reset_runner_database prefix_primary_key
  expect_production_runner_failure normal
  [ "$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='RUNNER_FIXTURE';")" = "0" ] || \
    fail "prefix-primary-key history executed migration SQL"

  reset_runner_database extra_column
  expect_production_runner_failure normal
  [ "$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='RUNNER_FIXTURE';")" = "0" ] || \
    fail "extra-column history executed migration SQL"

  reset_runner_database extra_index
  expect_production_runner_failure normal
  [ "$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='RUNNER_FIXTURE';")" = "0" ] || \
    fail "extra-index history executed migration SQL"

  reset_runner_database malformed_digest
  expect_production_runner_failure normal
  [ "$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='RUNNER_FIXTURE';")" = "0" ] || \
    fail "malformed history digest executed migration SQL"

  docker exec "$CONTAINER" cp \
    /tmp/canonical-migration-runner/backend/migrations/001_fixture.sql \
    /tmp/canonical-migration-runner/backend/migrations/001_other.sql \
    >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "duplicate-number source fixture setup failed"
  reset_runner_database none
  expect_production_runner_failure all
  [ "$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('_migration_history','RUNNER_FIXTURE');")" = "0" ] || \
    fail "duplicate-number source was rejected after database side effects"
  docker exec "$CONTAINER" rm -f /tmp/canonical-migration-runner/backend/migrations/001_other.sql \
    >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "duplicate-number source fixture cleanup failed"
  for sql_mode in default no_backslash_escapes; do
    if [ "$sql_mode" = "default" ]; then
      query_scalar "SET GLOBAL sql_mode=''; SELECT 1;" >/dev/null
    else
      query_scalar "SET GLOBAL sql_mode='NO_BACKSLASH_ESCAPES'; SELECT 1;" >/dev/null
    fi
    for path in normal seed; do
      reset_runner_database collision
      expect_production_runner_failure "$path"
      history_count="$(query_scalar "SELECT COUNT(*) FROM _migration_history;")"
      [ "$history_count" = "1" ] || fail "migration-number collision changed history"
      if [ "$path" = "normal" ]; then
        [ "$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='RUNNER_FIXTURE';")" = "0" ] || \
          fail "migration-number collision executed migration SQL"
      fi
    done
  done
  query_scalar "SET GLOBAL sql_mode=''; SELECT 1;" >/dev/null

  docker exec "$CONTAINER" sh -c '
    set -eu
    migrations=/tmp/canonical-migration-runner/backend/migrations
    mkdir -p "$migrations/testdata"
    cp "$migrations/001_fixture.sql" "$migrations/040_candidate.sql"
    digest=$(sha256sum "$migrations/040_candidate.sql" | cut -d " " -f 1)
    printf "%s  040_candidate.sql\n" "$digest" > "$migrations/testdata/canonical_identity_candidate_lineage.sha256"
  ' >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "candidate approval fixture setup failed"
  local candidate_manifest_digest candidate_digest
  candidate_manifest_digest="$(docker exec "$CONTAINER" sh -c \
    'sha256sum /tmp/canonical-migration-runner/backend/migrations/testdata/canonical_identity_candidate_lineage.sha256 | cut -d " " -f 1')"
  candidate_digest="$(docker exec "$CONTAINER" sh -c \
    'sha256sum /tmp/canonical-migration-runner/backend/migrations/040_candidate.sql | cut -d " " -f 1')"

  reset_runner_database none
  if run_production_candidate_runner "" >"${TEMP_DIR}/runner.out" 2>"${TEMP_DIR}/runner.err"; then
    fail "candidate runner accepted missing external approval"
  fi
  [ "$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('_migration_history','RUNNER_FIXTURE');")" = "0" ] || \
    fail "missing candidate approval was rejected after database side effects"

  docker exec "$CONTAINER" sh -c '
    set -eu
    root=/tmp/canonical-migration-runner/backend
    cp "$root/.env" "$root/.env.before-self-authorization"
    printf "EXTERNAL_CANDIDATE_MANIFEST_SHA256=%s\n" "$1" >> "$root/.env"
  ' sh "$candidate_manifest_digest" >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || \
    fail "candidate env self-authorization setup failed"
  reset_runner_database none
  if run_production_candidate_runner "" >"${TEMP_DIR}/runner.out" 2>"${TEMP_DIR}/runner.err"; then
    fail "candidate runner accepted .env self-authorization"
  fi
  [ "$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('_migration_history','RUNNER_FIXTURE');")" = "0" ] || \
    fail ".env self-authorization was rejected after database side effects"
  docker exec "$CONTAINER" sh -c '
    set -eu
    root=/tmp/canonical-migration-runner/backend
    mv "$root/.env.before-self-authorization" "$root/.env"
  ' >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "candidate env self-authorization cleanup failed"

  reset_runner_database none
  if run_production_candidate_runner "$(printf '0%.0s' {1..64})" >"${TEMP_DIR}/runner.out" 2>"${TEMP_DIR}/runner.err"; then
    fail "candidate runner accepted wrong external approval"
  fi
  [ "$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('_migration_history','RUNNER_FIXTURE');")" = "0" ] || \
    fail "wrong candidate approval was rejected after database side effects"

  reset_runner_database none
  run_production_candidate_runner "$candidate_manifest_digest" >"${TEMP_DIR}/runner.out" 2>"${TEMP_DIR}/runner.err" || \
    fail "candidate runner rejected exact external approval"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_history WHERE filename='040_candidate.sql' AND sha256='${candidate_digest}';")" = "1" ] || \
    fail "approved candidate was not recorded with exact digest"

  docker exec "$CONTAINER" sh -c '
    set -eu
    migrations=/tmp/canonical-migration-runner/backend/migrations
    printf "\n-- coordinated drift\n" >> "$migrations/040_candidate.sql"
    digest=$(sha256sum "$migrations/040_candidate.sql" | cut -d " " -f 1)
    printf "%s  040_candidate.sql\n" "$digest" > "$migrations/testdata/canonical_identity_candidate_lineage.sha256"
  ' >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "candidate coordinated-drift setup failed"
  reset_runner_database none
  if run_production_candidate_runner "$candidate_manifest_digest" >"${TEMP_DIR}/runner.out" 2>"${TEMP_DIR}/runner.err"; then
    fail "candidate runner accepted coordinated manifest and payload drift"
  fi
  [ "$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('_migration_history','RUNNER_FIXTURE');")" = "0" ] || \
    fail "candidate coordinated drift was rejected after database side effects"

  docker exec "$CONTAINER" sh -c '
    set -eu
    migrations=/tmp/canonical-migration-runner/backend/migrations
    cp "$migrations/001_fixture.sql" "$migrations/040_candidate.sql"
    digest=$(sha256sum "$migrations/040_candidate.sql" | cut -d " " -f 1)
    printf "%s  040_candidate.sql\n" "$digest" > "$migrations/testdata/canonical_identity_candidate_lineage.sha256"
    cp "$migrations/040_candidate.sql" "$migrations/041_unapproved.sql"
  ' >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "candidate extra-source setup failed"
  candidate_manifest_digest="$(docker exec "$CONTAINER" sh -c \
    'sha256sum /tmp/canonical-migration-runner/backend/migrations/testdata/canonical_identity_candidate_lineage.sha256 | cut -d " " -f 1')"
  reset_runner_database none
  if run_production_candidate_runner "$candidate_manifest_digest" >"${TEMP_DIR}/runner.out" 2>"${TEMP_DIR}/runner.err"; then
    fail "candidate runner accepted an unapproved extra source"
  fi
  [ "$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('_migration_history','RUNNER_FIXTURE');")" = "0" ] || \
    fail "unapproved extra source was rejected after database side effects"
  docker exec "$CONTAINER" rm -rf \
    /tmp/canonical-migration-runner/backend/migrations/040_candidate.sql \
    /tmp/canonical-migration-runner/backend/migrations/041_unapproved.sql \
    /tmp/canonical-migration-runner/backend/migrations/testdata \
    >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "candidate approval fixture cleanup failed"

  cleanup_resources || exit 125
  trap - EXIT
  printf 'RUNNER_HISTORY_AUTHORITY=PASS cases=fresh,reconciled,started_failure,schema_present_history_absent,filename_only,wrong_shape,prefix_primary_key,extra_column,extra_index,malformed_digest,source_collision\n'
  printf 'RUNNER_NUMBER_COLLISION=PASS modes=default,no_backslash_escapes paths=normal,seed\n'
  printf 'RUNNER_CANDIDATE_APPROVAL=PASS cases=missing_approval,env_self_authorization,wrong_approval,approved,co_drift,extra_source\n'
}

run_mixed_engine_probe() {
  start_container "mixed-engine"
  load_fixture "${TESTDATA_DIR}/canonical_identity_fresh_fixture.sql"
  apply_historical_migrations
  local result
  result="$(cat <<'SQL' | mysql_database 2>"${TEMP_DIR}/mysql.err" | tr -d '[:space:]'
CREATE TABLE HARNESS_IDENTITY_CHILD (
    USR_SEQ INT NOT NULL PRIMARY KEY
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
START TRANSACTION;
INSERT INTO WEO_MEMBER (
    USR_SEQ, USR_ID, USR_EMAIL, USR_PWD, USR_NAME, USR_PHONE, USR_FN, USR_DEPT, USR_STATUS
) VALUES (700001, 'mixed-engine-fixture', NULL, NULL, 'Fixture', '01000000000', '0', 'Fixture', 'AAA');
INSERT INTO HARNESS_IDENTITY_CHILD (USR_SEQ) VALUES (700001);
ROLLBACK;
SELECT CONCAT(
    (SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_SEQ=700001),
    ':',
    (SELECT COUNT(*) FROM HARNESS_IDENTITY_CHILD WHERE USR_SEQ=700001)
);
SQL
)"
  [ "$result" = "1:0" ] || fail "mixed-engine probe did not reproduce expected defect"
  printf 'MIXED_ENGINE_PARTIAL_COMMIT=REPRODUCED member=1 child=0\n'
}

transaction_rollback_result() {
  cat <<'SQL' | mysql_database 2>"${TEMP_DIR}/mysql.err" | tr -d '[:space:]'
CREATE TABLE IF NOT EXISTS HARNESS_IDENTITY_CHILD (
    USR_SEQ INT NOT NULL PRIMARY KEY
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
DELETE FROM HARNESS_IDENTITY_CHILD WHERE USR_SEQ=700001;
DELETE FROM WEO_MEMBER WHERE USR_SEQ=700001;
START TRANSACTION;
INSERT INTO WEO_MEMBER (
    USR_SEQ, USR_ID, USR_EMAIL, USR_PWD, USR_NAME, USR_PHONE, USR_FN, USR_DEPT, USR_STATUS
) VALUES (700001, 't03-transaction-fixture', NULL, NULL, 'Fixture', '01000000000', '0', 'Fixture', 'AAA');
INSERT INTO HARNESS_IDENTITY_CHILD (USR_SEQ) VALUES (700001);
ROLLBACK;
SELECT CONCAT(
    (SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_SEQ=700001),
    ':',
    (SELECT COUNT(*) FROM HARNESS_IDENTITY_CHILD WHERE USR_SEQ=700001)
);
SQL
}

run_t03_transaction_boundary_probe() {
  start_container "t03-transaction-boundary"
  load_fixture "${TESTDATA_DIR}/canonical_identity_fresh_fixture.sql"
  apply_historical_migrations

  local migration="${MIGRATIONS_DIR}/040_convert_auth_transaction_boundary_to_innodb.sql"
  local before after engine
  before="$(transaction_rollback_result)"
  [ "$before" = "1:0" ] || fail "T03 pre-conversion defect was not reproduced"
  printf 'T03_TRANSACTION_BOUNDARY=EXPECTED_RED before=1:0\n'

  [ -f "$migration" ] || fail "T03 migration missing"
  apply_migration "$migration" || fail "T03 migration failed"
  after="$(transaction_rollback_result)"
  engine="$(query_scalar "SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER';")"
  [ "$after" = "0:0" ] || fail "T03 rollback remained non-atomic"
  [ "$engine" = "InnoDB" ] || fail "T03 member engine mismatch"

  cleanup_resources || exit 125
  trap - EXIT
  printf 'T03_TRANSACTION_BOUNDARY=PASS before=1:0 after=0:0 engine=InnoDB\n'
}

run_t03_unexpected_engine_probe() {
  start_container "t03-unexpected-engine"
  load_fixture "${TESTDATA_DIR}/canonical_identity_fresh_fixture.sql"
  apply_historical_migrations
  query_scalar "ALTER TABLE WEO_MEMBER ENGINE=MEMORY; SELECT 1;" >/dev/null

  local migration="${MIGRATIONS_DIR}/040_convert_auth_transaction_boundary_to_innodb.sql"
  if apply_migration "$migration"; then
    fail "T03 migration accepted an unexpected source engine"
  fi
  [ "$(query_scalar "SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER';")" = "MEMORY" ] || \
    fail "T03 unexpected-engine rejection changed the table engine"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_history WHERE filename='040_convert_auth_transaction_boundary_to_innodb.sql';")" = "0" ] || \
    fail "T03 unexpected-engine rejection wrote migration history"
  [ "$(query_scalar "SELECT COUNT(*) FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA=DATABASE() AND ROUTINE_TYPE='PROCEDURE' AND LEFT(ROUTINE_NAME, 5)='_040_';")" = "0" ] || \
    fail "T03 unexpected-engine rejection left procedure residue"

  cleanup_resources || exit 125
  trap - EXIT
  printf 'T03_UNEXPECTED_ENGINE=PASS source=MEMORY history=0 procedures=0\n'
}

run_t03_target_resume_probe() {
  start_container "t03-target-resume"
  load_fixture "${TESTDATA_DIR}/canonical_identity_fresh_fixture.sql"
  apply_historical_migrations

  local runner_root="/tmp/t03-started-runner"
  local filename="040_convert_auth_transaction_boundary_to_innodb.sql"
  local candidate="${runner_root}/backend/migrations/${filename}"
  local manifest="${runner_root}/backend/migrations/testdata/canonical_identity_candidate_lineage.sha256"
  docker exec "$CONTAINER" mkdir -p "${runner_root}/backend/migrations/testdata" >/dev/null 2>"${TEMP_DIR}/t03-resume.err" || \
    fail "T03 STARTED runner root setup failed"
  docker cp "${BACKEND_DIR}/../migrate.sh" "${CONTAINER}:${runner_root}/migrate.sh" >/dev/null 2>"${TEMP_DIR}/t03-resume.err" || \
    fail "T03 STARTED runner copy failed"
  docker cp "${MIGRATIONS_DIR}/${filename}" "${CONTAINER}:${candidate}" >/dev/null 2>"${TEMP_DIR}/t03-resume.err" || \
    fail "T03 STARTED candidate copy failed"
  docker exec "$CONTAINER" mkdir -p "${runner_root}/backend/scripts" >/dev/null 2>"${TEMP_DIR}/t03-resume.err" || \
    fail "T03 bound preflight directory setup failed"
  docker cp "${BACKEND_DIR}/scripts/preflight-auth-transaction-boundary.sh" \
    "${CONTAINER}:${runner_root}/backend/scripts/preflight-auth-transaction-boundary.sh" >/dev/null 2>"${TEMP_DIR}/t03-resume.err" || \
    fail "T03 bound preflight copy failed"
  docker exec "$CONTAINER" sh -c '
    set -eu
    root=$1
    candidate=$2
    manifest=$3
    umask 077
    printf "\nCREATE TABLE T03_STARTED_DDL_WITNESS (ID INT NOT NULL PRIMARY KEY) ENGINE=InnoDB;\nSELECT * FROM information_schema._040_injected_runner_interruption;\n" >> "$candidate"
    digest=$(sha256sum "$candidate" | cut -d" " -f1)
    printf "%s  %s\n" "$digest" "$(basename "$candidate")" > "$manifest"
    printf "DB_HOST=127.0.0.1\nDB_PORT=3306\nDB_USER=root\nDB_PASSWORD=%s\nDB_NAME=canonical_identity_test\nDB_SOCKET=/var/run/mysqld/mysqld.sock\n" "$MYSQL_ROOT_PASSWORD" > "$root/backend/.env"
    printf "[client]\nhost=localhost\nsocket=/var/run/mysqld/mysqld.sock\nuser=root\npassword=\"%s\"\ndatabase=canonical_identity_test\nskip-ssl\n" "$MYSQL_ROOT_PASSWORD" > "$root/recovery.cnf"
    chmod 0700 "$root/migrate.sh"
    chmod 0700 "$root/backend/scripts/preflight-auth-transaction-boundary.sh"
    chmod 0600 "$root/backend/.env" "$root/recovery.cnf" "$manifest"
    printf "%s\n" "#!/bin/sh" "if [ \"\$1\" = show ] && [ \"\$2\" = --property=LoadState ] && [ \"\$#\" -eq 3 ]; then printf \"LoadState=masked\\n\"; exit 0; fi" "if [ \"\$1\" = is-active ] && [ \"\$2\" = --quiet ]; then exit 3; fi" "exit 4" > /bin/systemctl
    chmod 0700 /bin/systemctl
    mkdir -p /run/systemd/system
    for service in alumni-backend.service httpd.service crond.service; do ln -sf /dev/null "/run/systemd/system/$service"; done
  ' sh "$runner_root" "$candidate" "$manifest" >/dev/null 2>"${TEMP_DIR}/t03-resume.err" || \
    fail "T03 STARTED runner fixture setup failed"

  local candidate_digest manifest_digest
  candidate_digest="$(docker exec "$CONTAINER" sha256sum "$candidate" | cut -d ' ' -f1)"
  manifest_digest="$(docker exec "$CONTAINER" sha256sum "$manifest" | cut -d ' ' -f1)"
  docker exec "$CONTAINER" sh -c '
    set -eu
    env_file=$1
    cp "$env_file" "${env_file}.valid"
    : > "${env_file}.bad"
    while IFS= read -r line || [ -n "$line" ]; do
      case "$line" in DB_SOCKET=*) printf "DB_SOCKET=/run/t03-missing.sock\n" ;; *) printf "%s\n" "$line" ;; esac
    done < "$env_file" > "${env_file}.bad"
    chmod 0600 "${env_file}.bad" "${env_file}.valid"
    mv "${env_file}.bad" "$env_file"
  ' sh "${runner_root}/backend/.env" >/dev/null 2>"${TEMP_DIR}/t03-resume.err" || fail "T03 invalid socket fixture setup failed"
  if docker exec -e CANONICAL_CANDIDATE_MANIFEST_SHA256="$manifest_digest" -e T03_AUTH_ENGINE_BOUND_APPLY=1 "$CONTAINER" \
    "${runner_root}/migrate.sh" "$candidate" >"${TEMP_DIR}/t03-resume.out" 2>"${TEMP_DIR}/t03-resume.err"; then
    fail "T03 invalid socket runner unexpectedly succeeded"
  fi
  grep -Fq 'T03 bound apply DB_SOCKET is unavailable' "${TEMP_DIR}/t03-resume.err" || \
    fail "T03 invalid socket runner returned an unexpected failure"
  docker exec "$CONTAINER" mv "${runner_root}/backend/.env.valid" "${runner_root}/backend/.env" \
    >/dev/null 2>"${TEMP_DIR}/t03-resume.err" || fail "T03 valid socket fixture restore failed"
  [ "$(query_scalar "SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER';")" = "MyISAM" ] || \
    fail "T03 invalid socket runner changed the member engine"

  if docker exec -e CANONICAL_CANDIDATE_MANIFEST_SHA256="$manifest_digest" "$CONTAINER" \
    "${runner_root}/migrate.sh" "$candidate" >"${TEMP_DIR}/t03-resume.out" 2>"${TEMP_DIR}/t03-resume.err"; then
    fail "T03 unbound runner unexpectedly succeeded"
  fi
  grep -Fq 'bound operational preflight required' "${TEMP_DIR}/t03-resume.err" || \
    fail "T03 unbound runner returned an unexpected failure"
  [ "$(query_scalar "SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER';")" = "MyISAM" ] || \
    fail "T03 unbound runner changed the member engine"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_history WHERE filename='${filename}';")" = "0" ] || fail "T03 unbound runner changed history"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_journal WHERE filename='${filename}';")" = "0" ] || fail "T03 unbound runner changed journal"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_runner_lock;")" = "0" ] || fail "T03 unbound runner left a global lock"

  if docker exec -e CANONICAL_CANDIDATE_MANIFEST_SHA256="$manifest_digest" -e T03_AUTH_ENGINE_BOUND_APPLY=1 "$CONTAINER" \
    "${runner_root}/migrate.sh" "$candidate" >"${TEMP_DIR}/t03-resume.out" 2>"${TEMP_DIR}/t03-resume.err"; then
    fail "T03 failure-injected runner unexpectedly succeeded"
  fi
  if ! grep -Fq 'T03_BOUND_PREFLIGHT=PASS connection=unix-socket masks=verified runner_lock=held' "${TEMP_DIR}/t03-resume.out"; then
    local bound_failure="unclassified"
    grep -Fq 'bound operational preflight failed' "${TEMP_DIR}/t03-resume.err" && bound_failure="preflight-failed"
    grep -Fq 'bound operational preflight malformed' "${TEMP_DIR}/t03-resume.err" && bound_failure="preflight-malformed"
    grep -Fq 'bound operational preflight unavailable' "${TEMP_DIR}/t03-resume.err" && bound_failure="preflight-unavailable"
    if [ "$bound_failure" = "unclassified" ]; then
      grep -Fq 'T03_BOUND_STAGE=connection-configured' "${TEMP_DIR}/t03-resume.out" && bound_failure="after-connection-config"
      grep -Fq 'T03_BOUND_STAGE=runner-locked' "${TEMP_DIR}/t03-resume.out" && bound_failure="after-runner-lock"
      grep -Fq 'T03_BOUND_STAGE=preflight-started' "${TEMP_DIR}/t03-resume.out" && bound_failure="after-preflight-start"
    fi
    fail "T03 runner did not bind operational preflight (${bound_failure})"
  fi
  grep -Fq '_040_injected_runner_interruption' "${TEMP_DIR}/t03-resume.out" || \
    fail "T03 injected interruption was not the causal runner failure"
  [ "$(query_scalar "SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER';")" = "InnoDB" ] || \
    fail "T03 STARTED runner did not complete the engine DDL"
  [ "$(query_scalar "SELECT COUNT(*) FROM T03_STARTED_DDL_WITNESS;")" = "0" ] || fail "T03 STARTED DDL witness mismatch"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_journal WHERE filename='${filename}' AND sha256='${candidate_digest}' AND state='STARTED';")" = "1" ] || \
    fail "T03 STARTED journal evidence missing"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_runner_lock WHERE lock_name='global';")" = "1" ] || \
    fail "T03 STARTED global runner lock missing"

  if docker exec -e CANONICAL_CANDIDATE_MANIFEST_SHA256="$manifest_digest" -e T03_AUTH_ENGINE_BOUND_APPLY=1 "$CONTAINER" \
    "${runner_root}/migrate.sh" "$candidate" >"${TEMP_DIR}/t03-resume.out" 2>"${TEMP_DIR}/t03-resume.err"; then
    fail "T03 STARTED runner replay unexpectedly succeeded"
  fi
  [ "$(query_scalar "SELECT COUNT(*) FROM T03_STARTED_DDL_WITNESS;")" = "0" ] || fail "T03 STARTED runner replayed SQL"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_history WHERE filename='${filename}';")" = "0" ] || fail "T03 STARTED failure entered history"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_journal WHERE filename='${filename}' AND sha256='${candidate_digest}' AND state='STARTED';")" = "1" ] || \
    fail "T03 STARTED evidence changed during replay refusal"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_runner_lock WHERE lock_name='global';")" = "1" ] || \
    fail "T03 STARTED lock was released without restore"

  cleanup_resources || exit 125
  trap - EXIT
  printf 'T03_STARTED_FAIL_CLOSED=PASS preflight=bound socket=reject unbound=reject cause=injected_interruption engine=InnoDB history=0 journal=STARTED lock=1 replay=refused recovery=verified-backup-restore\n'
}

run_t03_bound_apply_probe() {
  start_container "t03-bound-apply"
  load_fixture "${TESTDATA_DIR}/canonical_identity_fresh_fixture.sql"
  apply_historical_migrations

  local runner_root="/tmp/t03-bound-apply-runner"
  local filename="040_convert_auth_transaction_boundary_to_innodb.sql"
  local candidate="${runner_root}/backend/migrations/${filename}"
  local manifest="${runner_root}/backend/migrations/testdata/canonical_identity_candidate_lineage.sha256"
  docker exec "$CONTAINER" mkdir -p "${runner_root}/backend/migrations/testdata" "${runner_root}/backend/scripts" \
    >/dev/null 2>"${TEMP_DIR}/t03-bound.err" || fail "T03 bound runner root setup failed"
  docker cp "${BACKEND_DIR}/../migrate.sh" "${CONTAINER}:${runner_root}/migrate.sh" >/dev/null 2>"${TEMP_DIR}/t03-bound.err" || \
    fail "T03 bound runner copy failed"
  docker cp "${MIGRATIONS_DIR}/${filename}" "${CONTAINER}:${candidate}" >/dev/null 2>"${TEMP_DIR}/t03-bound.err" || \
    fail "T03 bound candidate copy failed"
  docker cp "${BACKEND_DIR}/scripts/preflight-auth-transaction-boundary.sh" \
    "${CONTAINER}:${runner_root}/backend/scripts/preflight-auth-transaction-boundary.sh" >/dev/null 2>"${TEMP_DIR}/t03-bound.err" || \
    fail "T03 bound preflight copy failed"
  docker exec "$CONTAINER" sh -c '
    set -eu
    root=$1
    candidate=$2
    manifest=$3
    umask 077
    digest=$(sha256sum "$candidate" | cut -d" " -f1)
    printf "%s  %s\n" "$digest" "$(basename "$candidate")" > "$manifest"
    printf "DB_HOST=127.0.0.1\nDB_PORT=3306\nDB_USER=root\nDB_PASSWORD=%s\nDB_NAME=canonical_identity_test\nDB_SOCKET=/var/run/mysqld/mysqld.sock\n" "$MYSQL_ROOT_PASSWORD" > "$root/backend/.env"
    chmod 0700 "$root/migrate.sh" "$root/backend/scripts/preflight-auth-transaction-boundary.sh"
    chmod 0600 "$root/backend/.env" "$manifest"
    printf "%s\n" "#!/bin/sh" "if [ \"\$1\" = show ] && [ \"\$2\" = --property=LoadState ] && [ \"\$#\" -eq 3 ]; then printf \"LoadState=masked\\n\"; exit 0; fi" "if [ \"\$1\" = is-active ] && [ \"\$2\" = --quiet ]; then exit 3; fi" "exit 4" > /bin/systemctl
    chmod 0700 /bin/systemctl
    mkdir -p /run/systemd/system
    for service in alumni-backend.service httpd.service crond.service; do ln -sf /dev/null "/run/systemd/system/$service"; done
  ' sh "$runner_root" "$candidate" "$manifest" >/dev/null 2>"${TEMP_DIR}/t03-bound.err" || \
    fail "T03 bound runner fixture setup failed"

  local candidate_digest manifest_digest engine history journal lock rollback procedures
  candidate_digest="$(docker exec "$CONTAINER" sha256sum "$candidate" | cut -d ' ' -f1)"
  manifest_digest="$(docker exec "$CONTAINER" sha256sum "$manifest" | cut -d ' ' -f1)"
  docker exec -e CANONICAL_CANDIDATE_MANIFEST_SHA256="$manifest_digest" -e T03_AUTH_ENGINE_BOUND_APPLY=1 "$CONTAINER" \
    "${runner_root}/migrate.sh" "$candidate" >"${TEMP_DIR}/t03-bound.out" 2>"${TEMP_DIR}/t03-bound.err" || \
    fail "T03 bound runner apply failed"
  grep -Fq 'T03_BOUND_PREFLIGHT=PASS connection=unix-socket masks=verified runner_lock=held' "${TEMP_DIR}/t03-bound.out" || \
    fail "T03 bound runner omitted preflight proof"

  engine="$(query_scalar "SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER';")"
  history="$(query_scalar "SELECT COUNT(*) FROM _migration_history WHERE filename='${filename}' AND sha256='${candidate_digest}';")"
  journal="$(query_scalar "SELECT state FROM _migration_journal WHERE filename='${filename}' AND sha256='${candidate_digest}';")"
  lock="$(query_scalar "SELECT COUNT(*) FROM _migration_runner_lock;")"
  rollback="$(transaction_rollback_result)"
  procedures="$(query_scalar "SELECT COUNT(*) FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA=DATABASE() AND ROUTINE_TYPE='PROCEDURE' AND LEFT(ROUTINE_NAME, 5)='_040_';")"
  [ "$engine:$history:$journal:$lock:$rollback:$procedures" = "InnoDB:1:APPLIED:0:0:0:0" ] || fail "T03 bound runner postcondition mismatch"

  cleanup_resources || exit 125
  trap - EXIT
  printf 'T03_BOUND_APPLY=PASS preflight=bound engine=InnoDB history=1 journal=APPLIED lock=0 rollback=0:0 procedures=0\n'
}

run_t03_preflight_probe() {
  start_container "t03-preflight"
  load_fixture "${TESTDATA_DIR}/canonical_identity_fresh_fixture.sql"
  apply_historical_migrations
  load_database_fixture "${TESTDATA_DIR}/canonical_identity_conflict_duplicate_normalized_email.sql"

  local root="/tmp/t03-preflight"
  docker exec "$CONTAINER" mkdir -p "$root" >/dev/null 2>"${TEMP_DIR}/preflight.err" || \
    fail "T03 preflight root setup failed"
  docker cp "${BACKEND_DIR}/scripts/preflight-auth-transaction-boundary.sh" \
    "${CONTAINER}:${root}/preflight-auth-transaction-boundary.sh" \
    >/dev/null 2>"${TEMP_DIR}/preflight.err" || fail "T03 preflight helper missing"
  if ! docker exec "$CONTAINER" sh -c '
    set -eu
    umask 077
    printf "[client]\nhost=localhost\nsocket=/var/run/mysqld/mysqld.sock\nuser=root\npassword=\"%s\"\n" "$MYSQL_ROOT_PASSWORD" > /run/t03-preflight.cnf
    chmod 0600 /run/t03-preflight.cnf
    printf "%s\n" "#!/bin/sh" "if [ \"\$1\" = show ] && [ \"\$2\" = --property=LoadState ] && [ \"\$#\" -eq 3 ]; then printf \"LoadState=masked\\n\"; exit 0; fi" "if [ \"\$1\" = is-active ] && [ \"\$2\" = --quiet ]; then exit 3; fi" "exit 4" > /bin/systemctl
    chmod 0700 /bin/systemctl
    mkdir -p /run/systemd/system
    for service in alumni-backend.service httpd.service crond.service; do ln -s /dev/null "/run/systemd/system/$service"; done
    chmod 0700 /tmp/t03-preflight/preflight-auth-transaction-boundary.sh
    /tmp/t03-preflight/preflight-auth-transaction-boundary.sh /run/t03-preflight.cnf canonical_identity_test
    rm -f /run/t03-preflight.cnf
  ' >"${TEMP_DIR}/preflight.out" 2>"${TEMP_DIR}/preflight.err"; then
    local preflight_failure
    preflight_failure="$(tr '\n' ' ' < "${TEMP_DIR}/preflight.err")"
    case "$preflight_failure" in T03_AUTH_ENGINE_PREFLIGHT=FAIL\ \[REDACTED\]*) ;; *) preflight_failure="unclassified" ;; esac
    fail "T03 preflight helper failed: ${preflight_failure}"
  fi

  local output
  output="$(tr -d '\r' < "${TEMP_DIR}/preflight.out")"
  case "$output" in *password*|*MYSQL_ROOT_PASSWORD*) fail "T03 preflight leaked credential metadata" ;; esac
  cleanup_resources || exit 125
  trap - EXIT
  printf '%s\n' "$output"
}

expect_t03_preflight_failure() {
  local options_file="$1"
  local scenario="$2"
  if docker exec "$CONTAINER" /tmp/t03-preflight-negative/preflight-auth-transaction-boundary.sh \
    "$options_file" canonical_identity_test >"${TEMP_DIR}/preflight-negative.out" 2>"${TEMP_DIR}/preflight-negative.err"; then
    fail "T03 negative preflight unexpectedly passed"
  fi
  grep -Eq '^T03_AUTH_ENGINE_PREFLIGHT=FAIL \[REDACTED\] ' "${TEMP_DIR}/preflight-negative.err" || \
    fail "T03 negative preflight did not return a redacted failure (${scenario})"
  if grep -Eq 'credential-canary|MYSQL_ROOT_PASSWORD|password=' "${TEMP_DIR}/preflight-negative.out" "${TEMP_DIR}/preflight-negative.err"; then
    fail "T03 negative preflight leaked sensitive stderr"
  fi
}

run_t03_preflight_negative_controls() {
  start_container "t03-preflight-negative"
  load_fixture "${TESTDATA_DIR}/canonical_identity_fresh_fixture.sql"
  apply_historical_migrations

  docker exec "$CONTAINER" mkdir -p /tmp/t03-preflight-negative /run/t03-preflight-real /run/systemd/system \
    >/dev/null 2>"${TEMP_DIR}/preflight-negative.err" || fail "T03 negative preflight root setup failed"
  docker cp "${BACKEND_DIR}/scripts/preflight-auth-transaction-boundary.sh" \
    "${CONTAINER}:/tmp/t03-preflight-negative/preflight-auth-transaction-boundary.sh" \
    >/dev/null 2>"${TEMP_DIR}/preflight-negative.err" || fail "T03 negative preflight helper missing"
  docker exec "$CONTAINER" sh -c '
    set -eu
    umask 077
    printf "[client]\nhost=localhost\nsocket=/var/run/mysqld/mysqld.sock\nuser=root\npassword=\"%s\"\n" "$MYSQL_ROOT_PASSWORD" > /run/t03-preflight.cnf
    cp /run/t03-preflight.cnf /run/t03-preflight-real/options.cnf
    chmod 0600 /run/t03-preflight.cnf /run/t03-preflight-real/options.cnf
    chmod 0700 /tmp/t03-preflight-negative/preflight-auth-transaction-boundary.sh
    printf "%s\n" "#!/bin/sh" "if [ \"\$1\" = show ] && [ \"\$2\" = --property=LoadState ] && [ \"\$#\" -eq 3 ]; then printf \"LoadState=masked\\n\"; exit 0; fi" "if [ \"\$1\" = is-active ] && [ \"\$2\" = --quiet ]; then exit 3; fi" "exit 4" > /bin/systemctl
    chmod 0700 /bin/systemctl
    for service in alumni-backend.service httpd.service crond.service; do ln -s /dev/null "/run/systemd/system/$service"; done
    ln -s /run/t03-preflight-real /run/t03-preflight-link
  ' >/dev/null 2>"${TEMP_DIR}/preflight-negative.err" || fail "T03 negative preflight fixture setup failed"

  expect_t03_preflight_failure /run/t03-preflight-link/options.cnf ancestry

  docker exec "$CONTAINER" rm -f /run/systemd/system/crond.service >/dev/null 2>"${TEMP_DIR}/preflight-negative.err" || \
    fail "T03 writer mask negative setup failed"
  expect_t03_preflight_failure /run/t03-preflight.cnf mask
  docker exec "$CONTAINER" ln -s /dev/null /run/systemd/system/crond.service >/dev/null 2>"${TEMP_DIR}/preflight-negative.err" || \
    fail "T03 writer mask restore failed"

  docker exec "$CONTAINER" sh -c 'printf "%s\n" "#!/bin/sh" "if [ \"\$1\" = show ] && [ \"\$2\" = --property=LoadState ] && [ \"\$#\" -eq 3 ]; then printf \"LoadState=masked\\n\"; exit 0; fi" "if [ \"\$1\" = is-active ]; then printf \"credential-canary\\n\" >&2; exit 0; fi" "exit 4" > /bin/systemctl; chmod 0700 /bin/systemctl' \
    >/dev/null 2>"${TEMP_DIR}/preflight-negative.err" || fail "T03 active-writer negative setup failed"
  expect_t03_preflight_failure /run/t03-preflight.cnf active

  docker exec "$CONTAINER" sh -c '
    set -eu
    printf "%s\n" "#!/bin/sh" "if [ \"\$1\" = show ] && [ \"\$2\" = --property=LoadState ] && [ \"\$#\" -eq 3 ]; then printf \"LoadState=masked\\n\"; exit 0; fi" "if [ \"\$1\" = is-active ] && [ \"\$2\" = --quiet ]; then exit 3; fi" "exit 4" > /bin/systemctl
    mv /usr/bin/mysql /usr/bin/mysql.real
    printf "%s\n" "#!/bin/sh" "set -eu" "output=\$(/usr/bin/mysql.real \"\$@\")" "printf \"%s\\n\" \"\$output\" | /usr/bin/awk '\''BEGIN { FS=OFS=\"\\t\" } { \$8=\"/run/nonlocal-mysql.sock\"; print }'\''" > /usr/bin/mysql
    chmod 0700 /bin/systemctl /usr/bin/mysql
  ' >/dev/null 2>"${TEMP_DIR}/preflight-negative.err" || fail "T03 locality negative setup failed"
  expect_t03_preflight_failure /run/t03-preflight.cnf locality
  docker exec "$CONTAINER" mv /usr/bin/mysql.real /usr/bin/mysql >/dev/null 2>"${TEMP_DIR}/preflight-negative.err" || \
    fail "T03 mysql locality wrapper restore failed"

  docker exec "$CONTAINER" sh -c '
    set -eu
    mv /usr/bin/mysql /usr/bin/mysql.real
    printf "%s\n" "#!/bin/sh" "printf \"MyISAM\\t2\\t1000000000000001\\t0\\t1\\t2\\t0\\t/var/run/mysqld/mysqld.sock\\t/var/lib/mysql/\\n\"" > /usr/bin/mysql
    chmod 0700 /usr/bin/mysql
  ' >/dev/null 2>"${TEMP_DIR}/preflight-negative.err" || fail "T03 overflow negative setup failed"
  expect_t03_preflight_failure /run/t03-preflight.cnf overflow
  docker exec "$CONTAINER" mv /usr/bin/mysql.real /usr/bin/mysql >/dev/null 2>"${TEMP_DIR}/preflight-negative.err" || \
    fail "T03 mysql restore failed"

  cleanup_resources || exit 125
  trap - EXIT
  printf 'T03_PREFLIGHT_NEGATIVE_CONTROLS=PASS ancestry=reject mask=reject active=reject locality=reject overflow=reject stderr=redacted\n'
}

member_checksum() {
  printf 'CHECKSUM TABLE WEO_MEMBER EXTENDED;\n' | mysql_database 2>"${TEMP_DIR}/mysql.err" | awk '{print $2}' | tr -d '[:space:]'
}

member_column_signature() {
  query_scalar "SELECT MD5(GROUP_CONCAT(CONCAT_WS(':',ORDINAL_POSITION,COLUMN_NAME,COLUMN_TYPE,IS_NULLABLE,COALESCE(COLUMN_DEFAULT,'NULL'),EXTRA,COALESCE(CHARACTER_SET_NAME,''),COALESCE(COLLATION_NAME,'')) ORDER BY ORDINAL_POSITION SEPARATOR '|')) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER';"
}

member_index_signature() {
  query_scalar "SELECT MD5(GROUP_CONCAT(CONCAT_WS(':',INDEX_NAME,NON_UNIQUE,SEQ_IN_INDEX,COLUMN_NAME,COALESCE(SUB_PART,0),INDEX_TYPE) ORDER BY INDEX_NAME,SEQ_IN_INDEX SEPARATOR '|')) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER';"
}

member_trigger_signature() {
  query_scalar "SELECT MD5(COALESCE(GROUP_CONCAT(CONCAT_WS(':',TRIGGER_NAME,ACTION_TIMING,EVENT_MANIPULATION,ACTION_STATEMENT) ORDER BY TRIGGER_NAME SEPARATOR '|'),'')) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=DATABASE() AND EVENT_OBJECT_TABLE='WEO_MEMBER';"
}

run_t03_preservation_probe() {
  start_container "t03-preservation"
  load_fixture "${TESTDATA_DIR}/canonical_identity_fresh_fixture.sql"
  apply_historical_migrations
  load_database_fixture "${TESTDATA_DIR}/canonical_identity_conflict_duplicate_normalized_email.sql"

  local migration="${MIGRATIONS_DIR}/040_convert_auth_transaction_boundary_to_innodb.sql"
  local before_rows before_columns before_indexes before_triggers before_auto before_checksum
  local after_rows after_columns after_indexes after_triggers after_auto after_checksum drift_checksum
  before_rows="$(query_scalar 'SELECT COUNT(*) FROM WEO_MEMBER;')"
  before_columns="$(member_column_signature)"
  before_indexes="$(member_index_signature)"
  before_triggers="$(member_trigger_signature)"
  before_auto="$(query_scalar "SELECT AUTO_INCREMENT FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER';")"
  before_checksum="$(member_checksum)"

  apply_migration "$migration" || fail "T03 preservation migration failed"
  after_rows="$(query_scalar 'SELECT COUNT(*) FROM WEO_MEMBER;')"
  after_columns="$(member_column_signature)"
  after_indexes="$(member_index_signature)"
  after_triggers="$(member_trigger_signature)"
  after_auto="$(query_scalar "SELECT AUTO_INCREMENT FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER';")"
  after_checksum="$(member_checksum)"

  [ "$before_rows" = "2" ] && [ "$after_rows" = "$before_rows" ] || fail "T03 row preservation mismatch"
  [ "$after_columns" = "$before_columns" ] || fail "T03 column preservation mismatch"
  [ "$after_indexes" = "$before_indexes" ] || fail "T03 index preservation mismatch"
  [ "$after_triggers" = "$before_triggers" ] || fail "T03 trigger preservation mismatch"
  [ "$after_auto" = "$before_auto" ] || fail "T03 auto-increment preservation mismatch"
  [ "$after_checksum" = "$before_checksum" ] || fail "T03 extended checksum preservation mismatch"

  mysql_database < "$migration" >"${TEMP_DIR}/migration.out" 2>"${TEMP_DIR}/migration.err" || \
    fail "T03 target-state migration rerun failed"
  [ "$(query_scalar 'SELECT COUNT(*) FROM WEO_MEMBER;')" = "$after_rows" ] || fail "T03 target rerun changed rows"
  [ "$(member_column_signature)" = "$after_columns" ] || fail "T03 target rerun changed columns"
  [ "$(member_index_signature)" = "$after_indexes" ] || fail "T03 target rerun changed indexes"
  [ "$(member_trigger_signature)" = "$after_triggers" ] || fail "T03 target rerun changed triggers"
  [ "$(query_scalar "SELECT AUTO_INCREMENT FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER';")" = "$after_auto" ] || \
    fail "T03 target rerun changed auto-increment"
  [ "$(member_checksum)" = "$after_checksum" ] || fail "T03 target rerun changed checksum"

  query_scalar "UPDATE WEO_MEMBER SET USR_NAME='Checksum Drift' WHERE USR_SEQ=810001; SELECT 1;" >/dev/null
  drift_checksum="$(member_checksum)"
  [ "$drift_checksum" != "$after_checksum" ] || fail "T03 checksum negative control was not causal"

  cleanup_resources || exit 125
  trap - EXIT
  printf 'T03_CONVERSION_PRESERVATION=PASS rows=2 columns=same indexes=same triggers=same auto_increment=same checksum=same rerun=pass negative_control=detected\n'
}

assert_positive_count() {
  local sql="$1"
  [ "$(query_scalar "$sql")" -gt 0 ] || fail "conflict fixture category missing"
}

run_provider_subject_conflict_probe() {
  start_container "provider-subject-conflict"
  create_test_database
  load_database_fixture "${TESTDATA_DIR}/authoritative-164788c/kakao_auth_028_035_fixture.sql"
  apply_authoritative_predecessors
  apply_migration "${MIGRATIONS_DIR}/036_extend_mobile_refresh_token_for_rotation.sql" || \
    fail "authoritative migration 036 setup failed"
  load_database_fixture "${TESTDATA_DIR}/canonical_identity_conflict_duplicate_provider_subject.sql"

  local row_count code engine added_columns history_count residual_procedures
  row_count="$(query_scalar "SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL;")"
  if apply_migration "${MIGRATIONS_DIR}/037_harden_member_social_links.sql"; then
    fail "duplicate provider-subject preflight did not fail closed"
  fi
  code="$(mysql_error_code "${TEMP_DIR}/migration.err")"
  engine="$(query_scalar "SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL';")"
  added_columns="$(query_scalar "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL' AND COLUMN_NAME='NMS_EMAIL_ENABLED';")"
  history_count="$(query_scalar "SELECT COUNT(*) FROM _migration_history WHERE filename='037_harden_member_social_links.sql';")"
  residual_procedures="$(query_scalar "SELECT COUNT(*) FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA=DATABASE() AND ROUTINE_TYPE='PROCEDURE' AND LEFT(ROUTINE_NAME, 5)='_037_';")"

  [ "$code" = "ERROR_1644" ] || fail "provider-subject conflict returned an unexpected error class"
  [ "$(query_scalar "SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL;")" = "$row_count" ] || fail "provider-subject conflict changed source rows"
  [ "$engine" = "MyISAM" ] || fail "provider-subject conflict changed table engine"
  [ "$added_columns" = "0" ] || fail "provider-subject conflict altered table columns"
  [ "$history_count" = "0" ] || fail "failed provider-subject migration was recorded"
  [ "$residual_procedures" = "3" ] || fail "provider-subject residual procedure count mismatch"

  printf 'DUPLICATE_PROVIDER_SUBJECT=EXPECTED_RED rejected_before_table_alter residual_procedures=3\n'
}

run_missing_preflight_conflict_probes() {
  start_container "missing-preflight-conflicts"

  load_fixture "${TESTDATA_DIR}/canonical_identity_fresh_fixture.sql"
  load_database_fixture "${TESTDATA_DIR}/canonical_identity_conflict_duplicate_normalized_email.sql"
  assert_positive_count "SELECT COUNT(*) FROM (SELECT LOWER(TRIM(USR_EMAIL)) AS E FROM WEO_MEMBER WHERE USR_EMAIL IS NOT NULL GROUP BY LOWER(TRIM(USR_EMAIL)) HAVING COUNT(*) > 1) AS D;"
  printf 'DUPLICATE_NORMALIZED_EMAIL=EXPECTED_RED preflight_consumer_missing\n'

  load_fixture "${TESTDATA_DIR}/canonical_identity_fresh_fixture.sql"
  load_database_fixture "${TESTDATA_DIR}/canonical_identity_conflict_orphan_social_row.sql"
  assert_positive_count "SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL S LEFT JOIN WEO_MEMBER M ON M.USR_SEQ=S.USR_SEQ WHERE M.USR_SEQ IS NULL;"
  printf 'ORPHAN_SOCIAL_ROW=EXPECTED_RED preflight_consumer_missing\n'

  load_fixture "${TESTDATA_DIR}/canonical_identity_fresh_fixture.sql"
  load_database_fixture "${TESTDATA_DIR}/canonical_identity_conflict_malformed_identity.sql"
  assert_positive_count "SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL WHERE TRIM(NMS_GATE)='' OR TRIM(NMS_ID)='';"
  printf 'MALFORMED_IDENTITY=EXPECTED_RED preflight_consumer_missing\n'

  load_fixture "${TESTDATA_DIR}/canonical_identity_fresh_fixture.sql"
  load_database_fixture "${TESTDATA_DIR}/canonical_identity_conflict_unreadable_algorithm.sql"
  assert_positive_count "SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_PWD LIKE 'UNREADABLE_ALGORITHM_TAG:%';"
  printf 'UNREADABLE_ALGORITHM_TAG=EXPECTED_RED preflight_consumer_missing\n'
}

prepare_authoritative_history_baseline() {
  create_test_database
  load_database_fixture "${TESTDATA_DIR}/authoritative-164788c/kakao_auth_028_035_fixture.sql"
  apply_authoritative_predecessors
  apply_historical_migrations
}

verify_authoritative_history() {
  local manifest="${TESTDATA_DIR}/canonical_identity_authoritative_lineage.sha256"
  local expected_digest filename number row_count actual_filename actual_digest
  local artifact_035

  row_count="$(query_scalar "SELECT COUNT(*) FROM _migration_history WHERE LEFT(filename, 4)='035_';")"
  artifact_035="$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_PUSH_PREFERENCE';")"
  if [ "$row_count" = "0" ]; then
    if [ "$artifact_035" = "1" ]; then return 14; fi
    return 12
  fi
  [ "$row_count" = "1" ] || return 10
  actual_filename="$(query_scalar "SELECT filename FROM _migration_history WHERE LEFT(filename, 4)='035_' LIMIT 1;")"
  [ "$actual_filename" = "035_create_push_preference.sql" ] || return 10
  [ "$artifact_035" = "1" ] || return 13

  while read -r expected_digest filename; do
    case "$expected_digest" in
      \#*|'') continue ;;
    esac
    number="${filename%%_*}"
    row_count="$(query_scalar "SELECT COUNT(*) FROM _migration_history WHERE LEFT(filename, 4)='${number}_';")"
    [ "$row_count" = "1" ] || return 12
    actual_filename="$(query_scalar "SELECT filename FROM _migration_history WHERE LEFT(filename, 4)='${number}_' LIMIT 1;")"
    [ "$actual_filename" = "$filename" ] || return 10
    actual_digest="$(query_scalar "SELECT sha256 FROM _migration_history WHERE filename='${filename}' LIMIT 1;")"
    [ "$actual_digest" = "$expected_digest" ] || return 11
  done < "$manifest"
}

expect_history_verifier_code() {
  local expected_code="$1"
  local actual_code=0
  if verify_authoritative_history; then
    fail "history integrity scenario unexpectedly passed"
  else
    actual_code=$?
  fi
  [ "$actual_code" = "$expected_code" ] || fail "history integrity category mismatch"
}

run_history_integrity_probes() {
  start_container "history-integrity"

  prepare_authoritative_history_baseline
  query_scalar "UPDATE _migration_history SET filename='035_conflicting_history.sql' WHERE filename='035_create_push_preference.sql'; SELECT 1;" >/dev/null
  expect_history_verifier_code 10
  printf 'HISTORY_NUMBER_COLLISION=EXPECTED_RED\n'

  prepare_authoritative_history_baseline
  query_scalar "UPDATE _migration_history SET sha256=REPEAT('0', 64) WHERE filename='035_create_push_preference.sql'; SELECT 1;" >/dev/null
  expect_history_verifier_code 11
  printf 'HISTORY_DIGEST_DRIFT=EXPECTED_RED\n'

  prepare_authoritative_history_baseline
  query_scalar "DELETE FROM _migration_history WHERE filename='035_create_push_preference.sql'; DROP TABLE ALUMNI_PUSH_PREFERENCE; SELECT 1;" >/dev/null
  expect_history_verifier_code 12
  printf 'HISTORY_MISSING_035=EXPECTED_RED\n'

  prepare_authoritative_history_baseline
  query_scalar "DROP TABLE ALUMNI_PUSH_PREFERENCE; SELECT 1;" >/dev/null
  expect_history_verifier_code 13
  printf 'HISTORY_ROW_SCHEMA_ABSENT=EXPECTED_RED\n'

  prepare_authoritative_history_baseline
  query_scalar "DELETE FROM _migration_history WHERE filename='035_create_push_preference.sql'; SELECT 1;" >/dev/null
  expect_history_verifier_code 14
  printf 'SCHEMA_PRESENT_HISTORY_ABSENT=EXPECTED_RED\n'
}

run_production_lineage_reconciliation_probe() {
  start_container "production-lineage-reconciliation"
  local manifest="${TESTDATA_DIR}/canonical_identity_production_lineage_001_039.sha256"
  local manifest_digest filename expected_digest extra runner_root first_filename first_digest
  runner_root=/tmp/production-lineage-reconciliation

  reset_legacy_production_history() {
    mysql_without_database >"${TEMP_DIR}/mysql.out" 2>"${TEMP_DIR}/mysql.err" <<'SQL' || \
      fail "legacy production history database setup failed"
DROP DATABASE IF EXISTS canonical_identity_test;
CREATE DATABASE canonical_identity_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE canonical_identity_test;
CREATE TABLE _migration_history (
  filename VARCHAR(255) CHARACTER SET utf8 COLLATE utf8_general_ci NOT NULL PRIMARY KEY,
  applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
SQL
    first_filename=""
    first_digest=""
    while read -r expected_digest filename extra; do
      case "$expected_digest" in \#*|'') continue ;; esac
      [ -z "${extra:-}" ] || fail "production lineage manifest malformed"
      if [ -z "$first_filename" ]; then
        first_filename="$filename"
        first_digest="$expected_digest"
      fi
      printf "INSERT INTO _migration_history (filename) VALUES ('%s');\n" "$filename" | \
        mysql_database >/dev/null 2>"${TEMP_DIR}/mysql.err" || fail "legacy production history row setup failed"
    done < "$manifest"
  }

  reset_legacy_production_history

  docker exec "$CONTAINER" mkdir -p "$runner_root/backend/scripts" "$runner_root/backend/migrations" \
    >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "lineage reconciliation root setup failed"
  docker cp "${BACKEND_DIR}/../migrate.sh" "${CONTAINER}:${runner_root}/migrate.sh" \
    >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "lineage migration runner copy failed"
  docker cp "${BACKEND_DIR}/scripts/reconcile-production-migration-lineage.sh" \
    "${CONTAINER}:${runner_root}/backend/scripts/reconcile-production-migration-lineage.sh" \
    >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "lineage reconciler copy failed"
  docker cp "${MIGRATIONS_DIR}/." "${CONTAINER}:${runner_root}/backend/migrations/" \
    >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "lineage source packet copy failed"
  docker exec "$CONTAINER" rm -f \
    "${runner_root}/backend/migrations/040_convert_auth_transaction_boundary_to_innodb.sql" \
    "${runner_root}/backend/migrations/testdata/canonical_identity_candidate_lineage.sha256" \
    >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "lineage fixture candidate isolation failed"
  docker exec "$CONTAINER" sh -c '
    set -eu
    root=/tmp/production-lineage-reconciliation
    umask 077
    printf "[client]\nuser=root\npassword=\"%s\"\n" "$MYSQL_ROOT_PASSWORD" > /run/reconcile.cnf
    printf "DB_HOST=127.0.0.1\nDB_PORT=3306\nDB_USER=root\nDB_PASSWORD=%s\nDB_NAME=canonical_identity_test\n" "$MYSQL_ROOT_PASSWORD" > "$root/backend/.env"
    chmod 0600 /run/reconcile.cnf "$root/backend/.env"
    chmod 0700 "$root/migrate.sh" "$root/backend/scripts/reconcile-production-migration-lineage.sh"
  ' >/dev/null 2>"${TEMP_DIR}/runner-setup.err" || fail "lineage credential fixture setup failed"
  manifest_digest="$(shasum -a 256 "$manifest" | cut -d ' ' -f 1)"

  if docker exec "$CONTAINER" "$runner_root/backend/scripts/reconcile-production-migration-lineage.sh" \
      /run/reconcile.cnf canonical_identity_test >"${TEMP_DIR}/missing-approval.out" 2>"${TEMP_DIR}/missing-approval.err"; then
    fail "lineage reconciliation accepted missing approval"
  fi
  [ "$(query_scalar "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history';")" = "2" ] || \
    fail "missing approval reached a database side effect"

  if docker exec -e CANONICAL_PRODUCTION_LINEAGE_MANIFEST_SHA256="$(printf '0%.0s' {1..64})" "$CONTAINER" \
      "$runner_root/backend/scripts/reconcile-production-migration-lineage.sh" /run/reconcile.cnf canonical_identity_test \
      >"${TEMP_DIR}/wrong-approval.out" 2>"${TEMP_DIR}/wrong-approval.err"; then
    fail "lineage reconciliation accepted wrong approval"
  fi
  [ "$(query_scalar "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history';")" = "2" ] || \
    fail "wrong approval reached a database side effect"

  query_scalar "UPDATE _migration_history SET filename='001_unapproved_history.sql' WHERE filename='${first_filename}'; SELECT 1;" >/dev/null
  if docker exec -e CANONICAL_PRODUCTION_LINEAGE_MANIFEST_SHA256="$manifest_digest" "$CONTAINER" \
      "$runner_root/backend/scripts/reconcile-production-migration-lineage.sh" /run/reconcile.cnf canonical_identity_test \
      >"${TEMP_DIR}/wrong-filename.out" 2>"${TEMP_DIR}/wrong-filename.err"; then
    fail "lineage reconciliation accepted a wrong history filename"
  fi
  [ "$(query_scalar "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='_migration_history';")" = "2" ] || \
    fail "wrong history filename reached a database side effect"

  reset_legacy_production_history
  query_scalar "ALTER TABLE _migration_history ADD COLUMN sha256 CHAR(64) CHARACTER SET ascii COLLATE ascii_general_ci NULL AFTER filename; UPDATE _migration_history SET sha256='${first_digest}' WHERE filename='${first_filename}'; SELECT 1;" >/dev/null
  docker exec -e CANONICAL_PRODUCTION_LINEAGE_MANIFEST_SHA256="$manifest_digest" "$CONTAINER" \
    "$runner_root/backend/scripts/reconcile-production-migration-lineage.sh" /run/reconcile.cnf canonical_identity_test \
    >"${TEMP_DIR}/partial-resume.out" 2>"${TEMP_DIR}/partial-resume.err" || fail "approved partial lineage state did not resume"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_history WHERE sha256 IS NOT NULL;")" = "39" ] || \
    fail "partial lineage resume did not populate all digests"

  reset_legacy_production_history
  query_scalar "ALTER TABLE _migration_history ADD COLUMN sha256 CHAR(64) CHARACTER SET ascii COLLATE ascii_general_ci NULL AFTER filename; UPDATE _migration_history SET sha256=REPEAT('0',64) WHERE filename='${first_filename}'; SELECT 1;" >/dev/null
  if docker exec -e CANONICAL_PRODUCTION_LINEAGE_MANIFEST_SHA256="$manifest_digest" "$CONTAINER" \
      "$runner_root/backend/scripts/reconcile-production-migration-lineage.sh" /run/reconcile.cnf canonical_identity_test \
      >"${TEMP_DIR}/partial-wrong-digest.out" 2>"${TEMP_DIR}/partial-wrong-digest.err"; then
    fail "lineage reconciliation accepted a wrong partial digest"
  fi
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_history WHERE filename='${first_filename}' AND sha256=REPEAT('0',64);")" = "1" ] || \
    fail "wrong partial digest was overwritten"

  reset_legacy_production_history
  docker exec -e CANONICAL_PRODUCTION_LINEAGE_MANIFEST_SHA256="$manifest_digest" "$CONTAINER" \
    "$runner_root/backend/scripts/reconcile-production-migration-lineage.sh" /run/reconcile.cnf canonical_identity_test \
    >"${TEMP_DIR}/reconcile.out" 2>"${TEMP_DIR}/reconcile.err" || fail "legacy production lineage reconciliation failed"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_history WHERE sha256 IS NOT NULL AND CHAR_LENGTH(sha256)=64;")" = "39" ] || \
    fail "reconciled production history digest count mismatch"

  docker exec "$CONTAINER" "$runner_root/migrate.sh" >"${TEMP_DIR}/runner.out" 2>"${TEMP_DIR}/runner.err" || \
    fail "real migration runner rejected reconciled production history"
  [ "$(grep -Ec '^  SKIP  0(0[1-9]|[12][0-9]|3[0-9])_' "${TEMP_DIR}/runner.out" || true)" = "39" ] || \
    fail "real migration runner was not a 39-row no-op"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_history;")" = "39" ] || fail "reconciled history cardinality changed"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_journal WHERE state='APPLIED';")" = "39" ] || fail "reconciled journal backfill mismatch"
  [ "$(query_scalar "SELECT COUNT(*) FROM _migration_runner_lock;")" = "0" ] || fail "migration runner lock residue remained"

  docker exec -e CANONICAL_PRODUCTION_LINEAGE_MANIFEST_SHA256="$manifest_digest" "$CONTAINER" \
    "$runner_root/backend/scripts/reconcile-production-migration-lineage.sh" /run/reconcile.cnf canonical_identity_test \
    >"${TEMP_DIR}/reconcile-rerun.out" 2>"${TEMP_DIR}/reconcile-rerun.err" || fail "target-state lineage reconciliation rerun failed"

  cleanup_resources || exit 125
  trap - EXIT
  printf 'PRODUCTION_LINEAGE_RECONCILIATION=PASS cases=missing_approval,wrong_approval,wrong_filename,partial_resume,partial_wrong_digest,legacy,target history=39 journal=39 runner_noop=39\n'
}

run_t04_identity_cardinality_probe() {
  local migration="${MIGRATIONS_DIR}/041_create_canonical_identity_schema.sql"
  local global_subject_code verified_email_code same_account_provider nullable_email engine account_provider_unique

  [ -f "$migration" ] && [ ! -L "$migration" ] || fail "T04 migration 041 missing"

  start_container "t04-identity-cardinality"
  load_fixture "${TESTDATA_DIR}/canonical_identity_fresh_fixture.sql"
  apply_migration "${MIGRATIONS_DIR}/040_convert_auth_transaction_boundary_to_innodb.sql" || \
    fail "T04 prerequisite migration 040 failed"
  apply_migration "$migration" || fail "T04 migration 041 failed"

  if ! mysql_database >/dev/null 2>"${TEMP_DIR}/mysql.err" <<'SQL'
INSERT INTO WEO_MEMBER (USR_SEQ, USR_ID, USR_STATUS) VALUES
    (1001, 't04-account-1', 'CCC'),
    (1002, 't04-account-2', 'CCC');
INSERT INTO AUTH_ACCOUNT_STATE (ACCOUNT_ID, STATUS, CREATED_AT, UPDATED_AT) VALUES
    (1001, 'ACTIVE', NOW(), NOW()),
    (1002, 'ACTIVE', NOW(), NOW());
INSERT INTO AUTH_IDENTITY (
    ACCOUNT_ID, PROVIDER, SUBJECT_KEY, STATUS, VERIFIED_AT, CREATED_AT, UPDATED_AT
) VALUES
    (1001, 'KAKAO', 'kakao-subject-1', 'ACTIVE', NOW(), NOW(), NOW()),
    (1001, 'KAKAO', 'kakao-subject-2', 'ACTIVE', NOW(), NOW(), NOW()),
    (1001, 'APPLE', 'apple-subject-1', 'ACTIVE', NOW(), NOW(), NOW()),
    (1002, 'APPLE', 'apple-subject-2', 'ACTIVE', NOW(), NOW(), NOW());
SQL
  then
    fail "T04 identity positive-control inserts failed code=$(mysql_error_code "${TEMP_DIR}/mysql.err")"
  fi

  if mysql_database >/dev/null 2>"${TEMP_DIR}/global-subject.err" <<'SQL'
INSERT INTO AUTH_IDENTITY (
    ACCOUNT_ID, PROVIDER, SUBJECT_KEY, STATUS, VERIFIED_AT, CREATED_AT, UPDATED_AT
) VALUES (1002, 'KAKAO', 'kakao-subject-1', 'ACTIVE', NOW(), NOW(), NOW());
SQL
  then
    fail "T04 duplicate provider subject was accepted"
  fi
  global_subject_code="$(mysql_error_code "${TEMP_DIR}/global-subject.err")"
  [ "$global_subject_code" = "ERROR_1062" ] || fail "T04 duplicate provider subject returned an unexpected error class"

  if ! mysql_database >/dev/null 2>"${TEMP_DIR}/mysql.err" <<'SQL'
INSERT INTO AUTH_IDENTITY (
    ACCOUNT_ID, PROVIDER, SUBJECT_KEY, NORMALIZED_EMAIL, STATUS, VERIFIED_AT, CREATED_AT, UPDATED_AT
) VALUES (1001, 'EMAIL', 'email-subject-1', 'verified@example.test', 'ACTIVE', NOW(), NOW(), NOW());
SQL
  then
    fail "T04 EMAIL identity positive control failed code=$(mysql_error_code "${TEMP_DIR}/mysql.err")"
  fi
  if mysql_database >/dev/null 2>"${TEMP_DIR}/verified-email.err" <<'SQL'
INSERT INTO AUTH_IDENTITY (
    ACCOUNT_ID, PROVIDER, SUBJECT_KEY, NORMALIZED_EMAIL, STATUS, VERIFIED_AT, CREATED_AT, UPDATED_AT
) VALUES (1002, 'EMAIL', 'email-subject-2', 'verified@example.test', 'ACTIVE', NOW(), NOW(), NOW());
SQL
  then
    fail "T04 duplicate verified normalized EMAIL was accepted"
  fi
  verified_email_code="$(mysql_error_code "${TEMP_DIR}/verified-email.err")"
  [ "$verified_email_code" = "ERROR_1062" ] || fail "T04 duplicate verified normalized EMAIL returned an unexpected error class"

  same_account_provider="$(query_scalar "SELECT COUNT(*) FROM AUTH_IDENTITY WHERE ACCOUNT_ID=1001 AND PROVIDER='KAKAO';")"
  nullable_email="$(query_scalar "SELECT COUNT(*) FROM AUTH_IDENTITY WHERE PROVIDER='APPLE' AND NORMALIZED_EMAIL IS NULL;")"
  engine="$(query_scalar "SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='AUTH_IDENTITY';")"
  account_provider_unique="$(query_scalar "SELECT COUNT(*) FROM (SELECT INDEX_NAME FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='AUTH_IDENTITY' AND NON_UNIQUE=0 GROUP BY INDEX_NAME HAVING GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX)='ACCOUNT_ID,PROVIDER') AS account_provider_indexes;")"

  [ "$same_account_provider" = "2" ] || fail "T04 same-account provider multiplicity was blocked"
  [ "$nullable_email" = "2" ] || fail "T04 nullable normalized-email multiplicity mismatch"
  [ "$engine" = "InnoDB" ] || fail "T04 identity table is not InnoDB"
  [ "$account_provider_unique" = "0" ] || fail "T04 forbidden account/provider unique key exists"

  printf 'T04_IDENTITY_CARDINALITY=PASS same_account_provider=2 global_subject=reject nullable_email=2 verified_email=reject engine=InnoDB account_provider_unique=absent\n'
}

run_t04_credential_token_boundary_probe() {
  local migration="${MIGRATIONS_DIR}/041_create_canonical_identity_schema.sql"
  local table_count social_without_credentials hash_columns raw_token_columns continuation_secret_columns engine_count invalid_acceptances

  [ -f "$migration" ] && [ ! -L "$migration" ] || fail "T04 migration 041 missing"

  start_container "t04-credential-token"
  load_fixture "${TESTDATA_DIR}/canonical_identity_fresh_fixture.sql"
  apply_migration "${MIGRATIONS_DIR}/040_convert_auth_transaction_boundary_to_innodb.sql" || \
    fail "T04 prerequisite migration 040 failed"
  apply_migration "$migration" || fail "T04 migration 041 failed"

  table_count="$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('AUTH_PASSWORD_CREDENTIAL','AUTH_PROVIDER_CREDENTIAL','AUTH_EMAIL_VERIFICATION','AUTH_SIGNUP_CONTINUATION');")"
  [ "$table_count" = "4" ] || fail "T04 credential/token tables missing"

  if ! mysql_database >/dev/null 2>"${TEMP_DIR}/mysql.err" <<'SQL'
INSERT INTO WEO_MEMBER (USR_SEQ, USR_ID, USR_STATUS) VALUES (1101, 't04-boundary-account', 'CCC');
INSERT INTO AUTH_ACCOUNT_STATE (ACCOUNT_ID, STATUS, CREATED_AT, UPDATED_AT)
VALUES (1101, 'ACTIVE', NOW(), NOW());
INSERT INTO AUTH_IDENTITY (
    IDENTITY_ID, ACCOUNT_ID, PROVIDER, SUBJECT_KEY, STATUS, VERIFIED_AT, CREATED_AT, UPDATED_AT
) VALUES
    (2101, 1101, 'KAKAO', 'kakao-boundary-subject', 'ACTIVE', NOW(), NOW(), NOW()),
    (2102, 1101, 'EMAIL', 'email-boundary-subject', 'ACTIVE', NOW(), NOW(), NOW()),
    (2103, 1101, 'KAKAO', 'kakao-boundary-subject-2', 'ACTIVE', NOW(), NOW(), NOW());
INSERT INTO AUTH_PASSWORD_CREDENTIAL (
    IDENTITY_ID, PROVIDER, ALGORITHM, PARAMETERS_TEXT, PASSWORD_HASH, STATUS, CREATED_AT, UPDATED_AT
) VALUES (2102, 'EMAIL', 'ARGON2ID', 'm=19456,t=2,p=1', '$argon2id$redacted-test-value', 'ACTIVE', NOW(), NOW());
INSERT INTO AUTH_PROVIDER_CREDENTIAL (
    IDENTITY_ID, PROVIDER, KEY_ID, NONCE_BYTES, ALGORITHM, CIPHERTEXT, CREATED_AT, UPDATED_AT
) VALUES (2101, 'KAKAO', 'test-key-v1', X'00112233445566778899AABB', 'AES-256-GCM', X'CAFE', NOW(), NOW());
INSERT INTO AUTH_EMAIL_VERIFICATION (
    TOKEN_HASH, NORMALIZED_EMAIL, STATUS, EXPIRES_AT, CREATED_AT
) VALUES (REPEAT('a', 64), 'new@example.test', 'READY', DATE_ADD(NOW(), INTERVAL 30 MINUTE), NOW());
INSERT INTO AUTH_SIGNUP_CONTINUATION (
    TOKEN_HASH, PROVIDER, SUBJECT_KEY, STATUS, PREFILL_TEXT, EXPIRES_AT, CREATED_AT
) VALUES (REPEAT('b', 64), 'KAKAO', 'new-kakao-subject', 'READY', '{"name":"prefill"}', DATE_ADD(NOW(), INTERVAL 30 MINUTE), NOW());
SQL
  then
    fail "T04 credential/token positive controls failed code=$(mysql_error_code "${TEMP_DIR}/mysql.err")"
  fi

  invalid_acceptances=""
  if mysql_database >/dev/null 2>"${TEMP_DIR}/password-provider.err" <<'SQL'
INSERT INTO AUTH_PASSWORD_CREDENTIAL (
    IDENTITY_ID, PROVIDER, ALGORITHM, PARAMETERS_TEXT, PASSWORD_HASH, STATUS, CREATED_AT, UPDATED_AT
) VALUES (2101, 'KAKAO', 'ARGON2ID', 'm=19456,t=2,p=1', '$argon2id$invalid-provider', 'ACTIVE', NOW(), NOW());
SQL
  then
    invalid_acceptances="${invalid_acceptances},password_provider"
    query_scalar "DELETE FROM AUTH_PASSWORD_CREDENTIAL WHERE IDENTITY_ID=2101; SELECT 1;" >/dev/null
  fi
  if mysql_database >/dev/null 2>"${TEMP_DIR}/provider-ownerless.err" <<'SQL'
INSERT INTO AUTH_PROVIDER_CREDENTIAL (
    PROVIDER, KEY_ID, NONCE_BYTES, ALGORITHM, CIPHERTEXT, CREATED_AT, UPDATED_AT
) VALUES ('KAKAO', 'test-key-v1', X'00112233445566778899AABB', 'AES-256-GCM', X'CAFE', NOW(), NOW());
SQL
  then
    invalid_acceptances="${invalid_acceptances},provider_ownerless"
    query_scalar "DELETE FROM AUTH_PROVIDER_CREDENTIAL WHERE IDENTITY_ID IS NULL AND CONTINUATION_TOKEN_HASH IS NULL; SELECT 1;" >/dev/null
  fi
  if mysql_database >/dev/null 2>"${TEMP_DIR}/provider-dual-owner.err" <<'SQL'
INSERT INTO AUTH_PROVIDER_CREDENTIAL (
    IDENTITY_ID, CONTINUATION_TOKEN_HASH, PROVIDER, KEY_ID, NONCE_BYTES, ALGORITHM, CIPHERTEXT, CREATED_AT, UPDATED_AT
) VALUES (2103, REPEAT('b',64), 'KAKAO', 'test-key-v1', X'00112233445566778899AABB', 'AES-256-GCM', X'CAFE', NOW(), NOW());
SQL
  then
    invalid_acceptances="${invalid_acceptances},provider_dual_owner"
    query_scalar "DELETE FROM AUTH_PROVIDER_CREDENTIAL WHERE IDENTITY_ID=2103; SELECT 1;" >/dev/null
  fi
  [ -z "$invalid_acceptances" ] || fail "T04 credential ownership/provider integrity accepted cases=${invalid_acceptances#,}"

  social_without_credentials="$(query_scalar "SELECT COUNT(*) FROM AUTH_IDENTITY i LEFT JOIN AUTH_PASSWORD_CREDENTIAL p ON p.IDENTITY_ID=i.IDENTITY_ID LEFT JOIN AUTH_PROVIDER_CREDENTIAL c ON c.IDENTITY_ID=i.IDENTITY_ID WHERE i.IDENTITY_ID=2101 AND p.IDENTITY_ID IS NULL AND c.IDENTITY_ID IS NOT NULL;")"
  hash_columns="$(query_scalar "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('AUTH_EMAIL_VERIFICATION','AUTH_SIGNUP_CONTINUATION') AND COLUMN_NAME='TOKEN_HASH' AND DATA_TYPE='char' AND CHARACTER_MAXIMUM_LENGTH=64;")"
  raw_token_columns="$(query_scalar "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('AUTH_EMAIL_VERIFICATION','AUTH_SIGNUP_CONTINUATION') AND (COLUMN_NAME IN ('TOKEN','RAW_TOKEN','TOKEN_VALUE') OR COLUMN_NAME LIKE 'RAW\\_%');")"
  continuation_secret_columns="$(query_scalar "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='AUTH_SIGNUP_CONTINUATION' AND COLUMN_NAME IN ('KEY_ID','NONCE_BYTES','ALGORITHM','CIPHERTEXT','PROVIDER_CREDENTIAL');")"
  engine_count="$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('AUTH_PASSWORD_CREDENTIAL','AUTH_PROVIDER_CREDENTIAL','AUTH_EMAIL_VERIFICATION','AUTH_SIGNUP_CONTINUATION') AND ENGINE='InnoDB';")"

  [ "$social_without_credentials" = "1" ] || fail "T04 social credential optionality mismatch"
  [ "$hash_columns" = "2" ] || fail "T04 hashed-token columns mismatch"
  [ "$raw_token_columns" = "0" ] || fail "T04 raw-token column detected"
  [ "$continuation_secret_columns" = "0" ] || fail "T04 provider credential leaked into continuation schema"
  [ "$engine_count" = "4" ] || fail "T04 credential/token engine mismatch"

  printf 'T04_CREDENTIAL_TOKEN_BOUNDARIES=PASS social_without_credentials=1 password=provider-bound provider_secret=exactly-one-provider-bound-owner verification=hash-only continuation=hash-only provider_secret_in_continuation=absent engines=InnoDB\n'
}

run_t04_consent_session_outbox_probe() {
  local migration="${MIGRATIONS_DIR}/041_create_canonical_identity_schema.sql"
  local table_count consent_versions explicit_optional_false session_generation mismatch_code duplicate_code engine_count invalid_acceptances

  [ -f "$migration" ] && [ ! -L "$migration" ] || fail "T04 migration 041 missing"

  start_container "t04-consent-session-outbox"
  load_fixture "${TESTDATA_DIR}/canonical_identity_fresh_fixture.sql"
  apply_migration "${MIGRATIONS_DIR}/040_convert_auth_transaction_boundary_to_innodb.sql" || \
    fail "T04 prerequisite migration 040 failed"
  apply_migration "$migration" || fail "T04 migration 041 failed"

  table_count="$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('AUTH_CONSENT','AUTH_SESSION_FAMILY','AUTH_PROVIDER_REVOKE_OUTBOX');")"
  [ "$table_count" = "3" ] || fail "T04 consent/session/outbox tables missing"

  if ! mysql_database >/dev/null 2>"${TEMP_DIR}/mysql.err" <<'SQL'
INSERT INTO WEO_MEMBER (USR_SEQ, USR_ID, USR_STATUS) VALUES
    (1201, 't04-session-account-1', 'CCC'),
    (1202, 't04-session-account-2', 'CCC');
INSERT INTO AUTH_ACCOUNT_STATE (ACCOUNT_ID, STATUS, CREATED_AT, UPDATED_AT) VALUES
    (1201, 'ACTIVE', NOW(), NOW()),
    (1202, 'ACTIVE', NOW(), NOW());
INSERT INTO AUTH_IDENTITY (
    IDENTITY_ID, ACCOUNT_ID, PROVIDER, SUBJECT_KEY, STATUS, VERIFIED_AT, CREATED_AT, UPDATED_AT
) VALUES
    (2201, 1201, 'KAKAO', 'kakao-session-subject', 'ACTIVE', NOW(), NOW(), NOW()),
    (2202, 1202, 'APPLE', 'apple-session-subject', 'ACTIVE', NOW(), NOW(), NOW());
INSERT INTO AUTH_CONSENT (
    ACCOUNT_ID, CONSENT_TYPE, CONSENT_VERSION, IS_REQUIRED, IS_ACCEPTED,
    ACCEPTED_AT, CREATED_AT
) VALUES
    (1201, 'TERMS', 'terms-v1', 1, 1, NOW(), NOW()),
    (1201, 'TERMS', 'terms-v2', 1, 1, NOW(), NOW()),
    (1201, 'MARKETING', 'marketing-v1', 0, 0, NULL, NOW());
INSERT INTO AUTH_SESSION_FAMILY (
    FAMILY_ID, ACCOUNT_ID, IDENTITY_ID, AUTH_METHOD, GENERATION, STATUS,
    EXPIRES_AT, CREATED_AT, UPDATED_AT
) VALUES (
    REPEAT('d', 64),
    1201, 2201, 'KAKAO', 1, 'ACTIVE', DATE_ADD(NOW(), INTERVAL 30 DAY), NOW(), NOW()
);
INSERT INTO AUTH_PROVIDER_REVOKE_OUTBOX (
    IDEMPOTENCY_KEY, ACCOUNT_ID, IDENTITY_ID, PROVIDER, STATUS,
    ATTEMPT_COUNT, NEXT_ATTEMPT_AT, CREATED_AT, UPDATED_AT
) VALUES (REPEAT('c', 64), 1201, 2201, 'KAKAO', 'PENDING', 0, NOW(), NOW(), NOW());
INSERT INTO AUTH_PROVIDER_CREDENTIAL (
    IDENTITY_ID, PROVIDER, KEY_ID, NONCE_BYTES, ALGORITHM, CIPHERTEXT, CREATED_AT, UPDATED_AT
) VALUES (2202, 'APPLE', 'test-key-v1', X'00112233445566778899AABB', 'AES-256-GCM', X'CAFE', NOW(), NOW());
SQL
  then
    fail "T04 consent/session/outbox positive controls failed code=$(mysql_error_code "${TEMP_DIR}/mysql.err")"
  fi

  if mysql_database >/dev/null 2>"${TEMP_DIR}/session-mismatch.err" <<'SQL'
INSERT INTO AUTH_SESSION_FAMILY (
    FAMILY_ID, ACCOUNT_ID, IDENTITY_ID, AUTH_METHOD, GENERATION, STATUS,
    EXPIRES_AT, CREATED_AT, UPDATED_AT
) VALUES (
    REPEAT('e', 64),
    1202, 2201, 'KAKAO', 1, 'ACTIVE', DATE_ADD(NOW(), INTERVAL 30 DAY), NOW(), NOW()
);
SQL
  then
    fail "T04 mismatched session account/identity was accepted"
  fi
  mismatch_code="$(mysql_error_code "${TEMP_DIR}/session-mismatch.err")"
  [ "$mismatch_code" = "ERROR_1452" ] || fail "T04 mismatched session account/identity returned an unexpected error class"

  invalid_acceptances=""
  if mysql_database >/dev/null 2>"${TEMP_DIR}/session-provider.err" <<'SQL'
INSERT INTO AUTH_SESSION_FAMILY (
    FAMILY_ID, ACCOUNT_ID, IDENTITY_ID, AUTH_METHOD, GENERATION, STATUS,
    EXPIRES_AT, CREATED_AT, UPDATED_AT
) VALUES (REPEAT('a',64), 1201, 2201, 'APPLE', 1, 'ACTIVE', DATE_ADD(NOW(), INTERVAL 30 DAY), NOW(), NOW());
SQL
  then
    invalid_acceptances="${invalid_acceptances},session_provider"
    query_scalar "DELETE FROM AUTH_SESSION_FAMILY WHERE FAMILY_ID=REPEAT('a',64); SELECT 1;" >/dev/null
  fi
  if mysql_database >/dev/null 2>"${TEMP_DIR}/outbox-provider.err" <<'SQL'
INSERT INTO AUTH_PROVIDER_REVOKE_OUTBOX (
    IDEMPOTENCY_KEY, ACCOUNT_ID, IDENTITY_ID, PROVIDER, STATUS,
    ATTEMPT_COUNT, NEXT_ATTEMPT_AT, CREATED_AT, UPDATED_AT
) VALUES (REPEAT('b',64), 1201, 2201, 'APPLE', 'PENDING', 0, NOW(), NOW(), NOW());
SQL
  then
    invalid_acceptances="${invalid_acceptances},outbox_provider"
    query_scalar "DELETE FROM AUTH_PROVIDER_REVOKE_OUTBOX WHERE IDEMPOTENCY_KEY=REPEAT('b',64); SELECT 1;" >/dev/null
  fi
  if mysql_database >/dev/null 2>"${TEMP_DIR}/outbox-credential.err" <<'SQL'
INSERT INTO AUTH_PROVIDER_REVOKE_OUTBOX (
    IDEMPOTENCY_KEY, ACCOUNT_ID, IDENTITY_ID, CREDENTIAL_ID, PROVIDER, STATUS,
    ATTEMPT_COUNT, NEXT_ATTEMPT_AT, CREATED_AT, UPDATED_AT
) SELECT REPEAT('9',64), 1201, 2201, CREDENTIAL_ID, 'KAKAO', 'PENDING', 0, NOW(), NOW(), NOW()
  FROM AUTH_PROVIDER_CREDENTIAL WHERE IDENTITY_ID=2202;
SQL
  then
    invalid_acceptances="${invalid_acceptances},outbox_credential_owner"
    query_scalar "DELETE FROM AUTH_PROVIDER_REVOKE_OUTBOX WHERE IDEMPOTENCY_KEY=REPEAT('9',64); SELECT 1;" >/dev/null
  fi
  [ -z "$invalid_acceptances" ] || fail "T04 session/outbox provider integrity accepted cases=${invalid_acceptances#,}"

  if mysql_database >/dev/null 2>"${TEMP_DIR}/outbox-duplicate.err" <<'SQL'
INSERT INTO AUTH_PROVIDER_REVOKE_OUTBOX (
    IDEMPOTENCY_KEY, ACCOUNT_ID, IDENTITY_ID, PROVIDER, STATUS,
    ATTEMPT_COUNT, NEXT_ATTEMPT_AT, CREATED_AT, UPDATED_AT
) VALUES (REPEAT('c', 64), 1201, 2201, 'KAKAO', 'PENDING', 0, NOW(), NOW(), NOW());
SQL
  then
    fail "T04 duplicate provider-revoke idempotency key was accepted"
  fi
  duplicate_code="$(mysql_error_code "${TEMP_DIR}/outbox-duplicate.err")"
  [ "$duplicate_code" = "ERROR_1062" ] || fail "T04 duplicate provider-revoke key returned an unexpected error class"

  consent_versions="$(query_scalar "SELECT COUNT(*) FROM AUTH_CONSENT WHERE ACCOUNT_ID=1201;")"
  explicit_optional_false="$(query_scalar "SELECT COUNT(*) FROM AUTH_CONSENT WHERE ACCOUNT_ID=1201 AND CONSENT_TYPE='MARKETING' AND IS_REQUIRED=0 AND IS_ACCEPTED=0 AND ACCEPTED_AT IS NULL;")"
  session_generation="$(query_scalar "SELECT GENERATION FROM AUTH_SESSION_FAMILY WHERE ACCOUNT_ID=1201 AND IDENTITY_ID=2201;")"
  engine_count="$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('AUTH_CONSENT','AUTH_SESSION_FAMILY','AUTH_PROVIDER_REVOKE_OUTBOX') AND ENGINE='InnoDB';")"

  [ "$consent_versions" = "3" ] || fail "T04 consent snapshot cardinality mismatch"
  [ "$explicit_optional_false" = "1" ] || fail "T04 explicit optional-consent false snapshot missing"
  [ "$session_generation" = "1" ] || fail "T04 session generation mismatch"
  [ "$engine_count" = "3" ] || fail "T04 consent/session/outbox engine mismatch"

  printf 'T04_CONSENT_SESSION_OUTBOX=PASS consent_versions=3 explicit_optional_false=1 session_identity_account_provider=bound session_generation=1 outbox_identity_provider_credential=bound outbox_idempotency=reject engines=InnoDB\n'
}

run_t04_additive_preparation_probe() {
  local migration="${MIGRATIONS_DIR}/042_prepare_canonical_auth_cutover.sql"
  local table_count duplicate_code engine_count trigger_count member_rows history_count

  [ -f "$migration" ] && [ ! -L "$migration" ] || fail "T04 migration 042 missing"
  if grep -Eiq '^[[:space:]]*(DROP|DELETE|UPDATE|ALTER|TRUNCATE)[[:space:]]' "$migration"; then
    fail "T04 migration 042 contains destructive SQL"
  fi

  start_container "t04-additive-preparation"
  create_test_database
  load_database_fixture "${TESTDATA_DIR}/authoritative-164788c/kakao_auth_028_035_fixture.sql"
  apply_authoritative_predecessors
  apply_historical_migrations
  apply_migration "${MIGRATIONS_DIR}/040_convert_auth_transaction_boundary_to_innodb.sql" || \
    fail "T04 prerequisite migration 040 failed"
  apply_migration "${MIGRATIONS_DIR}/041_create_canonical_identity_schema.sql" || \
    fail "T04 prerequisite migration 041 failed"
  apply_migration "$migration" || fail "T04 migration 042 failed"

  table_count="$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('AUTH_IDENTITY_MIGRATION_RUN','AUTH_IDENTITY_MIGRATION_JOURNAL');")"
  [ "$table_count" = "2" ] || fail "T04 migration run/journal tables missing"

  if ! mysql_database >/dev/null 2>"${TEMP_DIR}/mysql.err" <<'SQL'
INSERT INTO AUTH_IDENTITY_MIGRATION_RUN (
    RUN_ID, STATUS, SOURCE_FINGERPRINT, CONFLICT_COUNT, STARTED_AT, UPDATED_AT
) VALUES (
    '11111111-1111-1111-1111-111111111111',
    'PREFLIGHT_PASSED', REPEAT('a', 64), 0, NOW(), NOW()
);
INSERT INTO AUTH_IDENTITY_MIGRATION_JOURNAL (
    RUN_ID, STEP_KEY, STATUS, STARTED_AT, UPDATED_AT
) VALUES (
    '11111111-1111-1111-1111-111111111111',
    'backfill-identities', 'STARTED', NOW(), NOW()
);
SQL
  then
    fail "T04 migration run/journal positive controls failed code=$(mysql_error_code "${TEMP_DIR}/mysql.err")"
  fi

  if mysql_database >/dev/null 2>"${TEMP_DIR}/journal-duplicate.err" <<'SQL'
INSERT INTO AUTH_IDENTITY_MIGRATION_JOURNAL (
    RUN_ID, STEP_KEY, STATUS, STARTED_AT, UPDATED_AT
) VALUES (
    '11111111-1111-1111-1111-111111111111',
    'backfill-identities', 'STARTED', NOW(), NOW()
);
SQL
  then
    fail "T04 duplicate migration journal step was accepted"
  fi
  duplicate_code="$(mysql_error_code "${TEMP_DIR}/journal-duplicate.err")"
  [ "$duplicate_code" = "ERROR_1062" ] || fail "T04 duplicate migration journal step returned an unexpected error class"

  engine_count="$(query_scalar "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('AUTH_IDENTITY_MIGRATION_RUN','AUTH_IDENTITY_MIGRATION_JOURNAL') AND ENGINE='InnoDB';")"
  trigger_count="$(query_scalar "SELECT COUNT(*) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=DATABASE() AND TRIGGER_NAME IN ('TRG_WEO_MEMBER_AUTH_PRINCIPAL_INSERT','TRG_WEO_MEMBER_AUTH_PRINCIPAL_UPDATE');")"
  member_rows="$(query_scalar "SELECT COUNT(*) FROM WEO_MEMBER;")"
  history_count="$(query_scalar "SELECT COUNT(*) FROM _migration_history;")"

  [ "$engine_count" = "2" ] || fail "T04 migration run/journal engine mismatch"
  [ "$trigger_count" = "2" ] || fail "T04 additive preparation changed legacy authority triggers"
  [ "$member_rows" = "5" ] || fail "T04 additive preparation changed legacy member rows"
  [ "$history_count" = "15" ] || fail "T04 additive preparation history mismatch"

  printf 'T04_ADDITIVE_PREPARATION=PASS migration_tables=2 journal_step_unique=reject engines=InnoDB legacy_triggers=2 legacy_member_rows=5 history=15 destructive_sql=absent\n'
}

run_t04_maintenance_finalization_probe() {
  local migration="${MIGRATIONS_DIR}/043_finalize_identity_authority.sql"
  local trigger_count refresh_active family_active trigger_final member_rows refresh_revoked family_revoked history_count rerun_procedures failure_procedures

  [ -f "$migration" ] && [ ! -L "$migration" ] || fail "T04 migration 043 missing"

  start_container "t04-maintenance-finalization"
  create_test_database
  load_database_fixture "${TESTDATA_DIR}/authoritative-164788c/kakao_auth_028_035_fixture.sql"
  apply_authoritative_predecessors
  apply_historical_migrations
  apply_migration "${MIGRATIONS_DIR}/040_convert_auth_transaction_boundary_to_innodb.sql" || \
    fail "T04 prerequisite migration 040 failed"
  apply_migration "${MIGRATIONS_DIR}/041_create_canonical_identity_schema.sql" || \
    fail "T04 prerequisite migration 041 failed"
  apply_migration "${MIGRATIONS_DIR}/042_prepare_canonical_auth_cutover.sql" || \
    fail "T04 prerequisite migration 042 failed"

  if ! mysql_database >/dev/null 2>"${TEMP_DIR}/mysql.err" <<'SQL'
INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN (
    MRT_JTI, USR_SEQ, MRT_SID, EXPIRES_AT, CREATED_AT
) VALUES ('t04-refresh-jti', 103, 't04-refresh-family', DATE_ADD(NOW(), INTERVAL 30 DAY), NOW());
INSERT INTO AUTH_ACCOUNT_STATE (ACCOUNT_ID, STATUS, CREATED_AT, UPDATED_AT)
VALUES (103, 'ACTIVE', NOW(), NOW());
INSERT INTO AUTH_IDENTITY (
    IDENTITY_ID, ACCOUNT_ID, PROVIDER, SUBJECT_KEY, STATUS, VERIFIED_AT, CREATED_AT, UPDATED_AT
) VALUES (2301, 103, 'KAKAO', 't04-finalize-subject', 'ACTIVE', NOW(), NOW(), NOW());
INSERT INTO AUTH_SESSION_FAMILY (
    FAMILY_ID, ACCOUNT_ID, IDENTITY_ID, AUTH_METHOD, GENERATION, STATUS,
    EXPIRES_AT, CREATED_AT, UPDATED_AT
) VALUES (REPEAT('f', 64), 103, 2301, 'KAKAO', 1, 'ACTIVE', DATE_ADD(NOW(), INTERVAL 30 DAY), NOW(), NOW());
SQL
  then
    fail "T04 maintenance-finalization fixture setup failed code=$(mysql_error_code "${TEMP_DIR}/mysql.err")"
  fi

  if apply_migration "$migration"; then
    fail "T04 maintenance finalization accepted missing verified backfill"
  fi
  trigger_count="$(query_scalar "SELECT COUNT(*) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=DATABASE() AND TRIGGER_NAME IN ('TRG_WEO_MEMBER_AUTH_PRINCIPAL_INSERT','TRG_WEO_MEMBER_AUTH_PRINCIPAL_UPDATE');")"
  refresh_active="$(query_scalar "SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE MRT_JTI='t04-refresh-jti' AND REVOKED_AT IS NULL AND MRT_REVOKED_AT IS NULL;")"
  family_active="$(query_scalar "SELECT COUNT(*) FROM AUTH_SESSION_FAMILY WHERE FAMILY_ID=REPEAT('f',64) AND STATUS='ACTIVE' AND REVOKED_AT IS NULL;")"
  [ "$trigger_count" = "2" ] || fail "T04 unverified finalization changed legacy triggers"
  [ "$refresh_active" = "1" ] || fail "T04 unverified finalization revoked mobile refresh state"
  [ "$family_active" = "1" ] || fail "T04 unverified finalization revoked canonical session state"
  failure_procedures="$(query_scalar "SELECT COUNT(*) FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA=DATABASE() AND ROUTINE_TYPE='PROCEDURE' AND LEFT(ROUTINE_NAME,5)='_043_';")"
  [ "$failure_procedures" = "0" ] || fail "T04 unverified finalization left stored procedures"

  if ! mysql_database >/dev/null 2>"${TEMP_DIR}/mysql.err" <<'SQL'
INSERT INTO AUTH_IDENTITY_MIGRATION_RUN (
    RUN_ID, STATUS, SOURCE_FINGERPRINT, CONFLICT_COUNT,
    STARTED_AT, COMPLETED_AT, UPDATED_AT
) VALUES
    (
        '22222222-2222-2222-2222-222222222222',
        'APPLIED', REPEAT('b',64), 0,
        DATE_SUB(NOW(), INTERVAL 2 MINUTE), DATE_SUB(NOW(), INTERVAL 2 MINUTE), DATE_SUB(NOW(), INTERVAL 2 MINUTE)
    ),
    (
        '33333333-3333-3333-3333-333333333333',
        'FAILED', REPEAT('c',64), 1,
        DATE_SUB(NOW(), INTERVAL 1 MINUTE), DATE_SUB(NOW(), INTERVAL 1 MINUTE), DATE_SUB(NOW(), INTERVAL 1 MINUTE)
    );
SQL
  then
    fail "T04 stale verified-backfill fixture setup failed code=$(mysql_error_code "${TEMP_DIR}/mysql.err")"
  fi

  if apply_migration "$migration"; then
    fail "T04 maintenance finalization accepted stale verified backfill"
  fi
  trigger_count="$(query_scalar "SELECT COUNT(*) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=DATABASE() AND TRIGGER_NAME IN ('TRG_WEO_MEMBER_AUTH_PRINCIPAL_INSERT','TRG_WEO_MEMBER_AUTH_PRINCIPAL_UPDATE');")"
  refresh_active="$(query_scalar "SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE MRT_JTI='t04-refresh-jti' AND REVOKED_AT IS NULL AND MRT_REVOKED_AT IS NULL;")"
  family_active="$(query_scalar "SELECT COUNT(*) FROM AUTH_SESSION_FAMILY WHERE FAMILY_ID=REPEAT('f',64) AND STATUS='ACTIVE' AND REVOKED_AT IS NULL;")"
  [ "$trigger_count" = "2" ] || fail "T04 stale verified finalization changed legacy triggers"
  [ "$refresh_active" = "1" ] || fail "T04 stale verified finalization revoked mobile refresh state"
  [ "$family_active" = "1" ] || fail "T04 stale verified finalization revoked canonical session state"
  failure_procedures="$(query_scalar "SELECT COUNT(*) FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA=DATABASE() AND ROUTINE_TYPE='PROCEDURE' AND LEFT(ROUTINE_NAME,5)='_043_';")"
  [ "$failure_procedures" = "0" ] || fail "T04 stale verified finalization left stored procedures"

  if ! mysql_database >/dev/null 2>"${TEMP_DIR}/mysql.err" <<'SQL'
INSERT INTO AUTH_IDENTITY_MIGRATION_RUN (
    RUN_ID, STATUS, SOURCE_FINGERPRINT, CONFLICT_COUNT,
    STARTED_AT, COMPLETED_AT, UPDATED_AT
) VALUES (
    '44444444-4444-4444-4444-444444444444',
    'APPLIED', REPEAT('d',64), 0, NOW(), NOW(), NOW()
);
SQL
  then
    fail "T04 latest verified-backfill witness setup failed code=$(mysql_error_code "${TEMP_DIR}/mysql.err")"
  fi

  if apply_migration "$migration"; then
    fail "T04 maintenance finalization accepted unjournaled backfill"
  fi
  trigger_count="$(query_scalar "SELECT COUNT(*) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=DATABASE() AND TRIGGER_NAME IN ('TRG_WEO_MEMBER_AUTH_PRINCIPAL_INSERT','TRG_WEO_MEMBER_AUTH_PRINCIPAL_UPDATE');")"
  refresh_active="$(query_scalar "SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE MRT_JTI='t04-refresh-jti' AND REVOKED_AT IS NULL AND MRT_REVOKED_AT IS NULL;")"
  family_active="$(query_scalar "SELECT COUNT(*) FROM AUTH_SESSION_FAMILY WHERE FAMILY_ID=REPEAT('f',64) AND STATUS='ACTIVE' AND REVOKED_AT IS NULL;")"
  [ "$trigger_count" = "2" ] || fail "T04 unjournaled finalization changed legacy triggers"
  [ "$refresh_active" = "1" ] || fail "T04 unjournaled finalization revoked mobile refresh state"
  [ "$family_active" = "1" ] || fail "T04 unjournaled finalization revoked canonical session state"
  failure_procedures="$(query_scalar "SELECT COUNT(*) FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA=DATABASE() AND ROUTINE_TYPE='PROCEDURE' AND LEFT(ROUTINE_NAME,5)='_043_';")"
  [ "$failure_procedures" = "0" ] || fail "T04 unjournaled finalization left stored procedures"

  if ! mysql_database >/dev/null 2>"${TEMP_DIR}/mysql.err" <<'SQL'
INSERT INTO AUTH_IDENTITY_MIGRATION_JOURNAL (
    RUN_ID, STEP_KEY, STATUS, STARTED_AT, APPLIED_AT, UPDATED_AT
) VALUES (
    '44444444-4444-4444-4444-444444444444',
    'backfill-identities', 'APPLIED', NOW(), NOW(), NOW()
);
SQL
  then
    fail "T04 latest backfill journal witness setup failed code=$(mysql_error_code "${TEMP_DIR}/mysql.err")"
  fi

  apply_migration "$migration" || fail "T04 maintenance finalization failed"

  trigger_final="$(query_scalar "SELECT COUNT(*) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=DATABASE() AND TRIGGER_NAME IN ('TRG_WEO_MEMBER_AUTH_PRINCIPAL_INSERT','TRG_WEO_MEMBER_AUTH_PRINCIPAL_UPDATE');")"
  member_rows="$(query_scalar "SELECT COUNT(*) FROM WEO_MEMBER;")"
  refresh_revoked="$(query_scalar "SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE MRT_JTI='t04-refresh-jti' AND REVOKED_AT IS NOT NULL AND MRT_REVOKED_AT IS NOT NULL;")"
  family_revoked="$(query_scalar "SELECT COUNT(*) FROM AUTH_SESSION_FAMILY WHERE FAMILY_ID=REPEAT('f',64) AND STATUS='REVOKED' AND REVOKED_AT IS NOT NULL;")"
  history_count="$(query_scalar "SELECT COUNT(*) FROM _migration_history;")"
  [ "$trigger_final" = "0" ] || fail "T04 maintenance finalization left legacy triggers"
  [ "$member_rows" = "5" ] || fail "T04 maintenance finalization changed legacy member rows"
  [ "$refresh_revoked" = "1" ] || fail "T04 maintenance finalization did not revoke mobile refresh state"
  [ "$family_revoked" = "1" ] || fail "T04 maintenance finalization did not revoke canonical session state"
  [ "$history_count" = "16" ] || fail "T04 maintenance finalization history mismatch"

  if ! mysql_database < "$migration" >"${TEMP_DIR}/rerun.out" 2>"${TEMP_DIR}/rerun.err"; then
    fail "T04 maintenance finalization target-state rerun failed"
  fi
  rerun_procedures="$(query_scalar "SELECT COUNT(*) FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA=DATABASE() AND ROUTINE_TYPE='PROCEDURE' AND LEFT(ROUTINE_NAME,5)='_043_';")"
  [ "$rerun_procedures" = "0" ] || fail "T04 maintenance finalization left stored procedures"

  printf 'T04_MAINTENANCE_FINALIZATION=PASS unverified=reject stale_verified=reject unjournaled=reject failure_procedures=0 triggers=0 legacy_member_rows=5 mobile_refresh=revoked canonical_family=revoked history=16 rerun=pass procedures=0\n'
}

verify_harness_fixture_lineage
if [ "$MODE" = "--check-fixture-lineage" ]; then
  trap - EXIT
  printf 'HARNESS_FIXTURE_LINEAGE=PASS entries=12\n'
  exit 0
fi

if [ "$MODE" = "--self-test-postconditions" ]; then
  run_postcondition_negative_control
  exit 0
fi

if [ "$MODE" = "--self-test-runner-sql-mode" ]; then
  run_runner_sql_mode_probe
  exit 0
fi

if [ "$MODE" = "--self-test-migration-runner" ]; then
  run_exact_migration_runner_probe
  exit 0
fi

if [ "$MODE" = "--self-test-production-lineage-reconciliation" ]; then
  run_production_lineage_reconciliation_probe
  exit 0
fi

if [ "$MODE" = "--self-test-t03-transaction-boundary" ]; then
  run_t03_transaction_boundary_probe
  exit 0
fi

if [ "$MODE" = "--self-test-t03-unexpected-engine" ]; then
  run_t03_unexpected_engine_probe
  exit 0
fi

if [ "$MODE" = "--self-test-t03-target-resume" ]; then
  run_t03_target_resume_probe
  exit 0
fi

if [ "$MODE" = "--self-test-t03-bound-apply" ]; then
  run_t03_bound_apply_probe
  exit 0
fi

if [ "$MODE" = "--self-test-t03-preflight" ]; then
  run_t03_preflight_probe
  exit 0
fi

if [ "$MODE" = "--self-test-t03-preflight-negative-controls" ]; then
  run_t03_preflight_negative_controls
  exit 0
fi

if [ "$MODE" = "--self-test-t03-preservation" ]; then
  run_t03_preservation_probe
  exit 0
fi

if [ "$MODE" = "--self-test-t04-identity-cardinality" ]; then
  run_t04_identity_cardinality_probe
  exit 0
fi

if [ "$MODE" = "--self-test-t04-credential-token-boundaries" ]; then
  run_t04_credential_token_boundary_probe
  exit 0
fi

if [ "$MODE" = "--self-test-t04-consent-session-outbox" ]; then
  run_t04_consent_session_outbox_probe
  exit 0
fi

if [ "$MODE" = "--self-test-t04-additive-preparation" ]; then
  run_t04_additive_preparation_probe
  exit 0
fi

if [ "$MODE" = "--self-test-t04-maintenance-finalization" ]; then
  run_t04_maintenance_finalization_probe
  exit 0
fi

if [ "$MODE" = "--check-candidate-range" ]; then
  check_candidate_range
  trap - EXIT
  exit 0
fi

verify_authoritative_testdata_lineage
check_source_lineage
if [ "$MODE" = "--check-source-lineage" ]; then
  trap - EXIT
  exit 0
fi

check_candidate_range

if [ "$MODE" = "--expect-current-red" ]; then
  run_current_branch_lineage_probe
fi
run_authoritative_upgrade_probe

if [ "$MODE" = "--expect-current-red" ]; then
  run_mixed_engine_probe
  run_provider_subject_conflict_probe
  run_missing_preflight_conflict_probes
  run_history_integrity_probes
  cleanup_resources || exit 125
  trap - EXIT
  printf 'HARNESS_SELF_TEST=PASS expected_red=14 conflict_categories=5 history_categories=5\n'
else
  cleanup_resources || exit 125
  trap - EXIT
  printf 'HARNESS_VERIFY=PASS\n'
fi
