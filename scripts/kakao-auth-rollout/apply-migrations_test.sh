#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
RUNNER="${ROOT}/scripts/kakao-auth-rollout/apply-migrations.sh"
MANIFEST="${ROOT}/backend/migrations/kakao-auth-036-039.sha256"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[[ -f "${RUNNER}" ]] || fail "dedicated migration runner is missing"
if grep -Eq -- '(^|[[:space:]])-p(\$|\{|[^[:space:]]+)|MYSQL_PWD' "${RUNNER}"; then
  fail "runner must not expose DB password in argv or environment"
fi
for gate in KAKAO_AUTH_MIGRATION_APPROVED MAINTENANCE_WRITES_FROZEN MAINTENANCE_DB_DRAIN_EVIDENCE VERIFIED_BACKUP_RESTORE; do
  grep -Fq "${gate}" "${RUNNER}" || fail "runner is missing ${gate} gate"
done
for bound_input in \
  scripts/kakao-auth-rollout/preflight.sql \
  scripts/kakao-auth-rollout/postcheck.sql \
  scripts/kakao-auth-rollout/apply-migrations.sh; do
  grep -Fq "  ${bound_input}" "$MANIFEST" || fail "manifest does not bind ${bound_input}"
done

TMP="$(mktemp -d "${TMPDIR:-/tmp}/kakao-migration-runner.XXXXXX")"
trap 'rm -rf "${TMP}"' EXIT
mkdir -p "${TMP}/bin"
printf '[client]\nuser=fixture\n' > "${TMP}/client.cnf"
chmod 0600 "${TMP}/client.cnf"

cat > "${TMP}/bin/mysql" <<'FAKE_MYSQL'
#!/usr/bin/env bash
set -euo pipefail
query=''
while [[ $# -gt 0 ]]; do
  if [[ "$1" == '-e' ]]; then
    query=$2
    break
  fi
  shift
done
case "${query}" in
  'SELECT VERSION()') printf '10.1.38-MariaDB-1~bionic\n'; exit 0 ;;
  *'information_schema.INNODB_TRX'*) printf '%s\n' "${FAKE_LIVE_TRANSACTIONS:-0}"; exit 0 ;;
  *'information_schema.PROCESSLIST'*) printf '%s\n' "${FAKE_LIVE_OTHER_CONNECTIONS:-0}"; exit 0 ;;
  *"TABLE_NAME='_migration_history'"*) printf '1\n'; exit 0 ;;
  *"filename='035_create_push_preference.sql'"*) printf '1\n'; exit 0 ;;
  *"filename IN ('036_"*) printf '0\n'; exit 0 ;;
  *"COLUMN_NAME IN ('CONSUMED_AT'"*) printf '3\n'; exit 0 ;;
  *'MRT_REVOKED_AT IS NOT NULL'*) printf '0\n'; exit 0 ;;
  *'SELECT ENGINE'*"TABLE_NAME='WEO_MEMBER_SOCIAL'"*) printf 'InnoDB\n'; exit 0 ;;
  *"NMS_STATUS='Y'"*) printf '0\n'; exit 0 ;;
  *'COUNT(DISTINCT INDEX_NAME)'*"INDEX_NAME='UK_USR_PROVIDER'"*) printf '1\n'; exit 0 ;;
  *'COUNT(DISTINCT INDEX_NAME)'*"INDEX_NAME='UK_PROVIDER_SUBJECT'"*) printf '1\n'; exit 0 ;;
  *"INDEX_NAME='UK_USR_PROVIDER'"*) printf '2\n'; exit 0 ;;
  *"INDEX_NAME='UK_PROVIDER_SUBJECT'"*) printf '2\n'; exit 0 ;;
  *"TABLE_NAME IN ('ALUMNI_ADMIN_ROLE','ALUMNI_VERIFICATION')"*) printf '2\n'; exit 0 ;;
  *'m.USR_STATUS IN'*) printf '0\n'; exit 0 ;;
  *"TABLE_NAME IN ('ALUMNI_SOCIAL_LINK_REAUTH_GUARD'"*) printf '2\n'; exit 0 ;;
  'INSERT INTO _migration_history'*) exit 0 ;;
esac
payload="$(cat)"
if [[ ${FAKE_EMPTY_METRICS:-0} == 1 &&
      ("${payload}" == *'duplicate_user_provider_groups'* || "${payload}" == *'social_engine_not_innodb'*) ]]; then
  exit 0
fi
if [[ "${payload}" == *'missing_required_tables'* ]]; then
  printf '%s\t0\n' \
    missing_required_tables \
    duplicate_user_provider_groups \
    conflicting_provider_subject_groups \
    unsupported_social_status_rows \
    cohort_source_overflow_rows \
    department_source_overflow_rows
  exit 0
fi
if [[ "${payload}" == *'social_engine_not_innodb'* ]]; then
  printf '%s\t0\n' \
    social_engine_not_innodb \
    missing_canonical_social_columns \
    invalid_social_status_column_shape \
    invalid_social_email_column_shape \
    legacy_active_status_rows \
    duplicate_user_provider_groups \
    conflicting_provider_subject_groups \
    missing_unique_user_provider_index \
    missing_unique_provider_subject_index \
    missing_refresh_rotation_columns \
    missing_refresh_sid_index \
    missing_auth_tables \
    invalid_auth_table_engines \
    legacy_verification_projection_mismatch \
    legacy_root_projection_missing
  exit 0
fi
if [[ "${payload}" == *'Migration 03'* ]]; then
  exit 0
fi
printf 'unexpected fake mysql invocation query=%s\n' "${query}" >&2
exit 2
FAKE_MYSQL
chmod +x "${TMP}/bin/mysql"

set +e
EMPTY_METRICS_OUTPUT=$(
  PATH="${TMP}/bin:${PATH}" FAKE_EMPTY_METRICS=1 \
  MYSQL_DEFAULTS_FILE="${TMP}/client.cnf" DB_NAME=fixture \
  bash "${RUNNER}" --preflight-only 2>&1
)
EMPTY_METRICS_STATUS=$?
set -e
[[ ${EMPTY_METRICS_STATUS} -eq 1 ]] || fail "empty preflight metrics produced false PASS"
[[ "${EMPTY_METRICS_OUTPUT}" == *'validation metrics are incomplete'* ]] ||
  fail "empty preflight metric failure was not explicit"

OUTPUT=$(
  PATH="${TMP}/bin:${PATH}" \
  MYSQL_DEFAULTS_FILE="${TMP}/client.cnf" \
  DB_NAME=fixture \
  bash "${RUNNER}" --preflight-only
)
[[ "${OUTPUT}" == *'PREFLIGHT PASS'* ]] || fail "preflight-only did not pass"
[[ "${OUTPUT}" == *'no migration was applied'* ]] || fail "preflight-only did not report non-mutation"
printf 'PASS: checksummed read-only preflight\n'

set +e
APPLY_OUTPUT=$(
  PATH="${TMP}/bin:${PATH}" \
  MYSQL_DEFAULTS_FILE="${TMP}/client.cnf" \
  DB_NAME=fixture \
  bash "${RUNNER}" --apply 2>&1
)
APPLY_STATUS=$?
set -e
[[ ${APPLY_STATUS} -eq 1 ]] || fail "apply without approval gates must fail"
[[ "${APPLY_OUTPUT}" == *'approval gates are incomplete'* ]] || fail "missing gate failure was not explicit"
printf 'PASS: apply approval gates fail closed\n'

set +e
MISSING_DRAIN_OUTPUT=$(
  PATH="${TMP}/bin:${PATH}" \
  MYSQL_DEFAULTS_FILE="${TMP}/client.cnf" \
  DB_NAME=fixture \
  KAKAO_AUTH_MIGRATION_APPROVED=1 \
  MAINTENANCE_WRITES_FROZEN=1 \
  VERIFIED_BACKUP_RESTORE=1 \
  bash "${RUNNER}" --apply 2>&1
)
MISSING_DRAIN_STATUS=$?
set -e
[[ ${MISSING_DRAIN_STATUS} -eq 1 ]] || fail "apply without DB drain evidence must fail"
[[ "${MISSING_DRAIN_OUTPUT}" == *'database drain evidence is invalid'* ]] ||
  fail "missing DB drain evidence failure was not explicit"

SENTINEL="${TMP}/maintenance"
DB_DRAIN_EVIDENCE="${TMP}/db-drain.pass"
MIGRATION_EVIDENCE="${TMP}/migration-postcheck.pass"
GENERATION=0123456789abcdef0123456789abcdef
printf 'state=active\ngeneration=%s\n' "${GENERATION}" > "${SENTINEL}"
printf 'state=PASS\nkind=db-drain\ngeneration=%s\nopen_transactions=0\nother_connections=0\nsamples=3\nrecorded_at=2026-08-05T00:00:00Z\n' \
  "${GENERATION}" > "${DB_DRAIN_EVIDENCE}"
chmod 0600 "${DB_DRAIN_EVIDENCE}"

set +e
LIVE_WRITER_OUTPUT=$(
  PATH="${TMP}/bin:${PATH}" \
  FAKE_LIVE_OTHER_CONNECTIONS=1 \
  MYSQL_DEFAULTS_FILE="${TMP}/client.cnf" DB_NAME=fixture \
  KAKAO_AUTH_MIGRATION_APPROVED=1 MAINTENANCE_WRITES_FROZEN=1 VERIFIED_BACKUP_RESTORE=1 \
  MAINTENANCE_SENTINEL_PATH="${SENTINEL}" MAINTENANCE_DB_DRAIN_EVIDENCE="${DB_DRAIN_EVIDENCE}" \
  MAINTENANCE_MIGRATION_EVIDENCE_OUTPUT="${MIGRATION_EVIDENCE}" \
  bash "${RUNNER}" --apply 2>&1
)
LIVE_WRITER_STATUS=$?
set -e
[[ ${LIVE_WRITER_STATUS} -eq 1 ]] || fail "live database writer did not block migration apply"
[[ "${LIVE_WRITER_OUTPUT}" == *'live database writer drain check failed'* ]] ||
  fail "live database writer failure was not explicit"

set +e
GATED_OUTPUT=$(
  PATH="${TMP}/bin:${PATH}" \
  MYSQL_DEFAULTS_FILE="${TMP}/client.cnf" \
  DB_NAME=fixture \
  KAKAO_AUTH_MIGRATION_APPROVED=1 \
  MAINTENANCE_WRITES_FROZEN=1 \
  VERIFIED_BACKUP_RESTORE=1 \
  MAINTENANCE_SENTINEL_PATH="${SENTINEL}" \
  MAINTENANCE_DB_DRAIN_EVIDENCE="${DB_DRAIN_EVIDENCE}" \
  MAINTENANCE_MIGRATION_EVIDENCE_OUTPUT="${MIGRATION_EVIDENCE}" \
  bash "${RUNNER}" --apply 2>&1
)
GATED_STATUS=$?
set -e
[[ ${GATED_STATUS} -eq 0 ]] || fail "approved apply path failed: ${GATED_OUTPUT}"
[[ "${GATED_OUTPUT}" == *'MIGRATION PASS range=036-039 postcheck=zero'* ]] ||
  fail "approved apply did not complete in exact order"
[[ -f ${MIGRATION_EVIDENCE} && ! -L ${MIGRATION_EVIDENCE} ]] ||
  fail "migration postcheck evidence was not produced"
grep -Fxq 'state=PASS' "${MIGRATION_EVIDENCE}" || fail "migration evidence state is missing"
grep -Fxq 'kind=migration-postcheck' "${MIGRATION_EVIDENCE}" || fail "migration evidence kind is missing"
grep -Fxq "generation=${GENERATION}" "${MIGRATION_EVIDENCE}" || fail "migration evidence generation is missing"
grep -Fxq 'range=036-039' "${MIGRATION_EVIDENCE}" || fail "migration evidence range is missing"
grep -Fxq 'postcheck_metrics=15' "${MIGRATION_EVIDENCE}" || fail "migration evidence metric count is missing"
printf 'PASS: approved apply validates and records 036-039\n'
