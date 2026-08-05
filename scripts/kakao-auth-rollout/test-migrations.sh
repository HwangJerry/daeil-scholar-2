#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
MIGRATIONS_DIR="${ROOT_DIR}/backend/migrations"
IMAGE_FILE="${MIGRATIONS_DIR}/testdata/mariadb-10.1.38.image"
FIXTURE_FILE="${MIGRATIONS_DIR}/testdata/kakao_auth_028_035_fixture.sql"
EDGE_FIXTURE_FILE="${MIGRATIONS_DIR}/testdata/kakao_auth_edge_cases.sql"
PREFLIGHT_FILE="${ROOT_DIR}/scripts/kakao-auth-rollout/preflight.sql"
POSTCHECK_FILE="${ROOT_DIR}/scripts/kakao-auth-rollout/postcheck.sql"
CASE="${1:-037-production-shape}"
CONTAINER="hermes-kakao-migrations-${$}"
DATABASE="kakao_auth_test"

cleanup() {
  local status=$?
  if [[ ${status} -ne 0 ]]; then
    docker logs --tail 80 "${CONTAINER}" >&2 2>/dev/null || true
  fi
  docker rm -f "${CONTAINER}" >/dev/null 2>&1 || true
  return "${status}"
}
trap cleanup EXIT INT TERM
trap 'printf "FAIL line=%s command=%s\n" "${LINENO}" "${BASH_COMMAND}" >&2' ERR

fail() {
  printf 'FAIL %s\n' "$1" >&2
  exit 1
}

[[ -s "${IMAGE_FILE}" ]] || fail "immutable image file is missing"
IMAGE="$(tr -d '[:space:]' < "${IMAGE_FILE}")"
[[ "${IMAGE}" =~ ^mariadb:10\.1\.38@sha256:[0-9a-f]{64}$ ]] || fail "immutable image reference is invalid"
[[ -s "${FIXTURE_FILE}" ]] || fail "fixture file is missing"

docker run --rm -d \
  --name "${CONTAINER}" \
  --platform linux/amd64 \
  --network none \
  -e MYSQL_ALLOW_EMPTY_PASSWORD=yes \
  "${IMAGE}" >/dev/null

for _ in $(seq 1 90); do
  logs="$(docker logs "${CONTAINER}" 2>&1 || true)"
  if [[ "${logs}" == *"MySQL init process done. Ready for start up."* ]] && \
     docker exec "${CONTAINER}" mysqladmin ping -uroot --silent >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
docker exec "${CONTAINER}" mysqladmin ping -uroot --silent >/dev/null 2>&1 || fail "MariaDB did not become ready"

mysql_exec() {
  docker exec "${CONTAINER}" mysql --default-character-set=utf8mb4 -uroot -BN "$@"
}

mysql_file() {
  local database=$1
  local file=$2
  docker exec -i "${CONTAINER}" mysql --default-character-set=utf8mb4 -uroot "${database}" < "${file}"
}

assert_no_warnings_file() {
  local database=$1
  local file=$2
  local output
  output="$(docker exec -i "${CONTAINER}" mysql --show-warnings --default-character-set=utf8mb4 -uroot "${database}" < "${file}")"
  [[ "${output}" != *$'Warning\t'* ]] || fail "migration emitted a warning: $(basename "${file}")"
}

query_file() {
  local database=$1
  local file=$2
  docker exec -i "${CONTAINER}" mysql --default-character-set=utf8mb4 -uroot -BN "${database}" < "${file}"
}

assert_query() {
  local expected=$1
  local query=$2
  local actual
  actual="$(mysql_exec "${DATABASE}" -e "${query}")"
  [[ "${actual}" == "${expected}" ]] || fail "expected '${expected}', got '${actual}' for ${query}"
}

assert_zero_metrics() {
  local file=$1
  local output
  output="$(query_file "${DATABASE}" "${file}")"
  while IFS=$'\t' read -r metric violations; do
    [[ -n "${metric}" ]] || continue
    [[ "${violations}" == "0" ]] || fail "metric ${metric} has ${violations} violation(s)"
  done <<< "${output}"
}

setup_production_shape() {
  mysql_exec -e "CREATE DATABASE ${DATABASE} CHARACTER SET utf8mb4"
  mysql_file "${DATABASE}" "${FIXTURE_FILE}"
  for migration in \
    028_create_mobile_device_token_table.sql \
    029_extend_mobile_device_token_invalid_state.sql \
    030_create_mobile_refresh_token_table.sql \
    031_extend_mobile_device_token_apns_metadata.sql \
    032_allow_android_push_tokens_without_apns_metadata.sql \
    033_backfill_android_push_token_metadata_and_length.sql \
    034_create_push_outbox.sql \
    035_create_push_preference.sql; do
    mysql_file "${DATABASE}" "${MIGRATIONS_DIR}/${migration}"
  done
}

test_037_production_shape() {
  setup_production_shape
  mysql_file "${DATABASE}" "${MIGRATIONS_DIR}/037_harden_member_social_links.sql"

  assert_query $'varchar(20)\tNO\tACTIVE' "SELECT COLUMN_TYPE, IS_NULLABLE, COLUMN_DEFAULT FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL' AND COLUMN_NAME='NMS_STATUS'"
  assert_query $'varchar(255)\tYES' "SELECT COLUMN_TYPE, IS_NULLABLE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL' AND COLUMN_NAME='NMS_EMAIL'"
  assert_query "0" "SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL WHERE NMS_STATUS='Y'"
  assert_query "2" "SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL WHERE NMS_STATUS='ACTIVE'"
  assert_query "1" "SELECT COUNT(DISTINCT INDEX_NAME) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL' AND INDEX_NAME='UK_USR_PROVIDER'"
  assert_query "1" "SELECT COUNT(DISTINCT INDEX_NAME) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL' AND INDEX_NAME='UK_PROVIDER_SUBJECT'"
  assert_query "InnoDB" "SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL'"
  assert_query "2" "SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL"

  printf 'PASS case=037-production-shape mariadb=%s\n' "$(mysql_exec -e 'SELECT VERSION()')"
}

test_038_reapply_warnings() {
  setup_production_shape
  mysql_file "${DATABASE}" "${MIGRATIONS_DIR}/036_extend_mobile_refresh_token_for_rotation.sql"
  mysql_file "${DATABASE}" "${MIGRATIONS_DIR}/037_harden_member_social_links.sql"
  mysql_file "${DATABASE}" "${MIGRATIONS_DIR}/038_create_auth_principal_tables.sql"
  assert_no_warnings_file "${DATABASE}" "${MIGRATIONS_DIR}/038_create_auth_principal_tables.sql"
  printf 'PASS case=038-reapply-warnings warnings=zero\n'
}

test_all() {
  setup_production_shape
  assert_query "10.1.38-MariaDB-1~bionic" "SELECT VERSION()"
  assert_zero_metrics "${PREFLIGHT_FILE}"

  mysql_exec "${DATABASE}" -e "INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN (MRT_JTI,USR_SEQ,MRT_SID,EXPIRES_AT,CREATED_AT,MRT_REVOKED_AT) VALUES ('fixture-jti',103,'fixture-sid',DATE_ADD(NOW(),INTERVAL 1 DAY),NOW(),NOW())"
  mysql_file "${DATABASE}" "${MIGRATIONS_DIR}/036_extend_mobile_refresh_token_for_rotation.sql"
  assert_query "3" "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_MOBILE_REFRESH_TOKEN' AND COLUMN_NAME IN ('CONSUMED_AT','REVOKED_AT','ROTATED_TO_JTI')"
  assert_query "1" "SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_MOBILE_REFRESH_TOKEN' AND INDEX_NAME='IDX_MRT_SID'"
  assert_query "1" "SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE MRT_JTI='fixture-jti' AND REVOKED_AT=MRT_REVOKED_AT"

  mysql_file "${DATABASE}" "${MIGRATIONS_DIR}/037_harden_member_social_links.sql"
  assert_query "InnoDB" "SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL'"
  assert_query "2" "SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL"

  mysql_file "${DATABASE}" "${MIGRATIONS_DIR}/038_create_auth_principal_tables.sql"
  assert_query "4" "SELECT COUNT(*) FROM ALUMNI_VERIFICATION"
  assert_query "1" "SELECT COUNT(*) FROM ALUMNI_ADMIN_ROLE WHERE USR_SEQ=104 AND ADMIN_ROLE='root'"
  assert_query "1" "SELECT COUNT(*) FROM ALUMNI_VERIFICATION WHERE USR_SEQ=101 AND STATUS='rejected' AND REJECTION_REASON IS NOT NULL"
  mysql_exec "${DATABASE}" -e "UPDATE WEO_MEMBER SET USR_STATUS='AAA' WHERE USR_SEQ=103"
  assert_query "approved" "SELECT STATUS FROM ALUMNI_VERIFICATION WHERE USR_SEQ=103"
  mysql_exec "${DATABASE}" -e "UPDATE WEO_MEMBER SET USR_STATUS='CCC' WHERE USR_SEQ=104"
  assert_query "0" "SELECT COUNT(*) FROM ALUMNI_ADMIN_ROLE WHERE USR_SEQ=104 AND ADMIN_ROLE='root'"
  mysql_exec "${DATABASE}" -e "UPDATE WEO_MEMBER SET USR_STATUS='CCC' WHERE USR_SEQ=103; UPDATE WEO_MEMBER SET USR_STATUS='BBB' WHERE USR_SEQ=103"
  assert_query "pending" "SELECT STATUS FROM ALUMNI_VERIFICATION WHERE USR_SEQ=103"
  mysql_exec "${DATABASE}" -e "UPDATE WEO_MEMBER SET USR_STATUS='CCC' WHERE USR_SEQ=103; UPDATE WEO_MEMBER SET USR_STATUS='BAA' WHERE USR_SEQ=103"
  assert_query "1" "SELECT COUNT(*) FROM ALUMNI_VERIFICATION WHERE USR_SEQ=103 AND STATUS='rejected' AND REJECTION_REASON IS NOT NULL"

  mysql_file "${DATABASE}" "${MIGRATIONS_DIR}/039_create_social_link_continuation.sql"
  assert_query "2" "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('ALUMNI_SOCIAL_LINK_REAUTH_GUARD','ALUMNI_SOCIAL_LINK_CONTINUATION') AND ENGINE='InnoDB'"
  mysql_exec "${DATABASE}" -e "INSERT INTO ALUMNI_SOCIAL_LINK_CONTINUATION (SLC_TOKEN_HASH,SLC_PROVIDER,SLC_SUBJECT,SLC_STATUS,SLC_EXPIRES_AT,SLC_CREATED_AT) VALUES (REPEAT('a',64),'KT','rollback-subject','READY',DATE_ADD(NOW(),INTERVAL 10 MINUTE),NOW())"
  mysql_exec "${DATABASE}" -e "START TRANSACTION; INSERT INTO WEO_MEMBER_SOCIAL (USR_SEQ,NMS_GATE,NMS_ID,NMS_EMAIL,NMS_STATUS,NMS_EMAIL_ENABLED,REG_DATE) VALUES (105,'KT','rollback-subject',NULL,'ACTIVE','Y',NOW()); UPDATE ALUMNI_SOCIAL_LINK_CONTINUATION SET SLC_STATUS='CONSUMED',SLC_CONSUMED_AT=NOW() WHERE SLC_TOKEN_HASH=REPEAT('a',64); ROLLBACK"
  assert_query "0" "SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL WHERE NMS_ID='rollback-subject'"
  assert_query "READY" "SELECT SLC_STATUS FROM ALUMNI_SOCIAL_LINK_CONTINUATION WHERE SLC_TOKEN_HASH=REPEAT('a',64)"

  for migration in \
    036_extend_mobile_refresh_token_for_rotation.sql \
    037_harden_member_social_links.sql \
    038_create_auth_principal_tables.sql \
    039_create_social_link_continuation.sql; do
    assert_no_warnings_file "${DATABASE}" "${MIGRATIONS_DIR}/${migration}"
  done
  assert_zero_metrics "${POSTCHECK_FILE}"
  assert_query "2" "SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL"
  assert_query "4" "SELECT COUNT(*) FROM ALUMNI_VERIFICATION"

  mysql_exec -e "DROP DATABASE ${DATABASE}; CREATE DATABASE ${DATABASE} CHARACTER SET utf8mb4"
  mysql_file "${DATABASE}" "${FIXTURE_FILE}"
  mysql_file "${DATABASE}" "${EDGE_FIXTURE_FILE}"
  assert_query "1" "SELECT COUNT(*) FROM (SELECT 1 FROM WEO_MEMBER_SOCIAL GROUP BY USR_SEQ,NMS_GATE HAVING COUNT(*)>1) AS duplicate_user_provider"
  if mysql_file "${DATABASE}" "${MIGRATIONS_DIR}/037_harden_member_social_links.sql" >/dev/null 2>&1; then
    fail "duplicate fixture unexpectedly applied migration 037"
  fi
  assert_query "0" "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL' AND COLUMN_NAME='NMS_EMAIL_ENABLED'"
  assert_query "MyISAM" "SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL'"
  assert_query "enum('Y','N')" "SELECT COLUMN_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL' AND COLUMN_NAME='NMS_STATUS'"
  assert_query "0" "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_ADMIN_ROLE'"

  printf 'PASS case=all clean=036-039 reapply=036-039 rollback=atomic fail_closed=037 postcheck=zero\n'
}

case "${CASE}" in
  037-production-shape) test_037_production_shape ;;
  038-reapply-warnings) test_038_reapply_warnings ;;
  all) test_all ;;
  *) fail "unknown case: ${CASE}" ;;
esac
