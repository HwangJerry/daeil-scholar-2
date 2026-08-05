#!/usr/bin/env bash
# maintenance-db-drain-contract_test.sh — Verify read-only database drain evidence.
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
CHECKER="$ROOT/scripts/kakao-auth-rollout/maintenance-db-drain.sh"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

[[ -x $CHECKER ]] || fail "database drain checker is missing or not executable"
TMP=$(mktemp -d "${TMPDIR:-/tmp}/maintenance-db-drain.XXXXXX")
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$TMP/bin"

cat > "$TMP/bin/mysql" <<'FAKE_MYSQL'
#!/usr/bin/env bash
sql=
while (($#)); do
  if [[ $1 == -e ]]; then
    sql=$2
    break
  fi
  shift
done
case "$sql" in
  'SELECT VERSION()') printf '10.1.38-MariaDB-fixture\n' ;;
  *'information_schema.INNODB_TRX'*) printf '%s\n' "${FAKE_OPEN_TRANSACTIONS:-0}" ;;
  *'information_schema.PROCESSLIST'*)
    if [[ $sql == *"COMMAND<>'Sleep'"* ]]; then
      printf '%s\n' "${FAKE_ACTIVE_QUERIES:-0}"
    else
      if [[ ${FAKE_ACTIVE_QUERIES:-0} != 0 || ${FAKE_OTHER_CONNECTIONS:-0} != 0 ]]; then
        printf '1\n'
      else
        printf '0\n'
      fi
    fi
    ;;
  *) exit 2 ;;
esac
FAKE_MYSQL
chmod +x "$TMP/bin/mysql"

SENTINEL="$TMP/maintenance"
DEFAULTS="$TMP/mysql.cnf"
EVIDENCE="$TMP/db-drain.pass"
GENERATION=abcdef0123456789abcdef0123456789
printf 'state=active\ngeneration=%s\n' "$GENERATION" > "$SENTINEL"
printf '[client]\nuser=fixture\npassword=fixture\n' > "$DEFAULTS"
chmod 0600 "$DEFAULTS"

set +e
PATH="$TMP/bin:$PATH" FAKE_OPEN_TRANSACTIONS=1 MAINTENANCE_DB_DRAIN_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" MYSQL_DEFAULTS_FILE="$DEFAULTS" DB_NAME=fixture_db \
  MAINTENANCE_DB_DRAIN_EVIDENCE_OUTPUT="$EVIDENCE" DB_DRAIN_SAMPLE_INTERVAL_SECONDS=0 \
  "$CHECKER" >/dev/null 2>&1
ACTIVE_TRANSACTION_STATUS=$?
set -e
[[ $ACTIVE_TRANSACTION_STATUS -ne 0 && ! -e $EVIDENCE ]] || fail "open transaction produced DB drain evidence"

set +e
PATH="$TMP/bin:$PATH" FAKE_ACTIVE_QUERIES=1 MAINTENANCE_DB_DRAIN_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" MYSQL_DEFAULTS_FILE="$DEFAULTS" DB_NAME=fixture_db \
  MAINTENANCE_DB_DRAIN_EVIDENCE_OUTPUT="$EVIDENCE" DB_DRAIN_SAMPLE_INTERVAL_SECONDS=0 \
  "$CHECKER" >/dev/null 2>&1
ACTIVE_QUERY_STATUS=$?
set -e
[[ $ACTIVE_QUERY_STATUS -ne 0 && ! -e $EVIDENCE ]] || fail "active query produced DB drain evidence"

set +e
PATH="$TMP/bin:$PATH" FAKE_OTHER_CONNECTIONS=1 MAINTENANCE_DB_DRAIN_APPROVED=1 \
  MAINTENANCE_SENTINEL_PATH="$SENTINEL" MYSQL_DEFAULTS_FILE="$DEFAULTS" DB_NAME=fixture_db \
  MAINTENANCE_DB_DRAIN_EVIDENCE_OUTPUT="$EVIDENCE" DB_DRAIN_SAMPLE_INTERVAL_SECONDS=0 \
  "$CHECKER" >/dev/null 2>&1
SLEEPING_CONNECTION_STATUS=$?
set -e
[[ $SLEEPING_CONNECTION_STATUS -ne 0 && ! -e $EVIDENCE ]] || fail "other target DB connection produced drain evidence"

PATH="$TMP/bin:$PATH" MAINTENANCE_DB_DRAIN_APPROVED=1 MAINTENANCE_SENTINEL_PATH="$SENTINEL" \
  MYSQL_DEFAULTS_FILE="$DEFAULTS" DB_NAME=fixture_db MAINTENANCE_DB_DRAIN_EVIDENCE_OUTPUT="$EVIDENCE" \
  DB_DRAIN_SAMPLE_INTERVAL_SECONDS=0 "$CHECKER" >/dev/null

grep -Fxq 'state=PASS' "$EVIDENCE" || fail "DB drain evidence state is missing"
grep -Fxq 'kind=db-drain' "$EVIDENCE" || fail "DB drain evidence kind is missing"
grep -Fxq "generation=$GENERATION" "$EVIDENCE" || fail "DB drain evidence generation is missing"
grep -Fxq 'open_transactions=0' "$EVIDENCE" || fail "DB drain transaction metric is missing"
grep -Fxq 'other_connections=0' "$EVIDENCE" || fail "DB drain connection metric is missing"
MODE=$(stat -f '%Lp' "$EVIDENCE" 2>/dev/null || stat -c '%a' "$EVIDENCE")
[[ $MODE == 600 ]] || fail "DB drain evidence mode is not 0600"

printf 'PASS: generation-bound read-only database drain evidence\n'
