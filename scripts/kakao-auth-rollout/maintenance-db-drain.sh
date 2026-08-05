#!/usr/bin/env bash
# maintenance-db-drain.sh — Record generation-bound proof that database writers are drained.
set -euo pipefail

SENTINEL=${MAINTENANCE_SENTINEL_PATH:-/run/alumni/maintenance}
DEFAULTS_FILE=${MYSQL_DEFAULTS_FILE:-}
DB_NAME=${DB_NAME:-}
EVIDENCE_OUTPUT=${MAINTENANCE_DB_DRAIN_EVIDENCE_OUTPUT:-/run/alumni/db-drain.pass}
SAMPLE_INTERVAL=${DB_DRAIN_SAMPLE_INTERVAL_SECONDS:-1}
SAMPLE_COUNT=3

fail() {
  printf 'ERROR maintenance_db_drain state=blocked reason=%s\n' "$1" >&2
  exit 1
}

file_mode() {
  stat -c '%a' "$1" 2>/dev/null || stat -f '%Lp' "$1"
}

file_owner_uid() {
  stat -c '%u' "$1" 2>/dev/null || stat -f '%u' "$1"
}

[[ ${MAINTENANCE_DB_DRAIN_APPROVED:-0} == 1 ]] || fail approval_required
[[ $SENTINEL == /* && $DEFAULTS_FILE == /* && $EVIDENCE_OUTPUT == /* ]] || fail paths_must_be_absolute
[[ -f $SENTINEL && ! -L $SENTINEL ]] || fail sentinel_not_active
[[ -f $DEFAULTS_FILE && ! -L $DEFAULTS_FILE && -r $DEFAULTS_FILE ]] || fail mysql_defaults_file_invalid
[[ $(file_mode "$DEFAULTS_FILE") == 600 ]] || fail mysql_defaults_file_mode_not_0600
[[ $(file_owner_uid "$DEFAULTS_FILE") == $(id -u) ]] || fail mysql_defaults_file_owner_invalid
[[ $DB_NAME =~ ^[A-Za-z0-9_]+$ ]] || fail database_name_invalid
[[ $SAMPLE_INTERVAL =~ ^[0-9]+$ ]] || fail sample_interval_invalid
[[ $EVIDENCE_OUTPUT != "$SENTINEL" && $EVIDENCE_OUTPUT != "$DEFAULTS_FILE" ]] || fail evidence_output_conflicts_with_input

evidence_dir=$(dirname "$EVIDENCE_OUTPUT")
[[ -d $evidence_dir && ! -L $evidence_dir ]] || fail evidence_output_directory_invalid

generation_count=$(grep -c '^generation=' "$SENTINEL" || true)
generation=$(sed -n 's/^generation=//p' "$SENTINEL")
[[ $generation_count == 1 && $generation =~ ^[a-f0-9]{32}$ ]] || fail sentinel_generation_invalid

MYSQL=(mysql --defaults-extra-file="$DEFAULTS_FILE" --skip-ssl --batch --skip-column-names "$DB_NAME")
query_scalar() {
  local sql=$1 result
  result=$("${MYSQL[@]}" -e "$sql") || fail database_query_failed
  [[ $result =~ ^[0-9]+$ ]] || fail database_metric_invalid
  printf '%s' "$result"
}

version=$("${MYSQL[@]}" -e 'SELECT VERSION()') || fail database_version_query_failed
[[ $version == 10.1.38-MariaDB* ]] || fail mariadb_version_mismatch

open_transactions=0
other_connections=0
for ((sample = 1; sample <= SAMPLE_COUNT; sample++)); do
  open_transactions=$(query_scalar "SELECT COUNT(*) FROM information_schema.INNODB_TRX t JOIN information_schema.PROCESSLIST p ON p.ID=t.trx_mysql_thread_id WHERE p.DB=DATABASE()")
  other_connections=$(query_scalar "SELECT COUNT(*) FROM information_schema.PROCESSLIST WHERE ID<>CONNECTION_ID()")
  [[ $open_transactions == 0 ]] || fail open_transactions_present
  [[ $other_connections == 0 ]] || fail other_connections_present
  if ((sample < SAMPLE_COUNT)) && ((SAMPLE_INTERVAL > 0)); then
    sleep "$SAMPLE_INTERVAL"
  fi
done

recorded_at=$(date -u '+%Y-%m-%dT%H:%M:%SZ') || fail evidence_clock_failed
evidence_tmp=$(mktemp "${EVIDENCE_OUTPUT}.tmp.XXXXXX") || fail evidence_create_failed
trap 'rm -f "$evidence_tmp"' EXIT
printf 'state=PASS\nkind=db-drain\ngeneration=%s\nopen_transactions=0\nother_connections=0\nsamples=%s\nrecorded_at=%s\n' \
  "$generation" "$SAMPLE_COUNT" "$recorded_at" > "$evidence_tmp"
chmod 0600 "$evidence_tmp"
mv -f "$evidence_tmp" "$EVIDENCE_OUTPUT"
trap - EXIT

printf 'PASS maintenance_db_drain generation=current transactions=0 other_connections=0 samples=%s\n' "$SAMPLE_COUNT"
