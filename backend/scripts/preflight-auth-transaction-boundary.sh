#!/usr/bin/env bash
# preflight-auth-transaction-boundary.sh — Production-safe WEO_MEMBER conversion preflight.
set -euo pipefail

exec 3>&2
exec 2>/dev/null

unexpected_failure() {
  local status=$?
  trap - ERR
  printf 'T03_AUTH_ENGINE_PREFLIGHT=FAIL [REDACTED] unclassified failure\n' >&3
  exit "$status"
}
trap unexpected_failure ERR

fail() {
  trap - ERR
  printf 'T03_AUTH_ENGINE_PREFLIGHT=FAIL [REDACTED] %s\n' "$1" >&3
  exit 1
}

[ "$#" -eq 2 ] || fail "usage"
[ "$(id -u)" -eq 0 ] || fail "root authority required"
SOURCE_OPTIONS_FILE="$1"
DATABASE="$2"
case "$DATABASE" in ''|*[!A-Za-z0-9_]*) fail "unsafe database name" ;; esac
case "$SOURCE_OPTIONS_FILE" in /*) ;; *) fail "option file must be absolute" ;; esac
case "$SOURCE_OPTIONS_FILE" in *'/../'*|*'/./'*|*'//'*) fail "unsafe option file path" ;; esac

exec {OPTIONS_FD}<"$SOURCE_OPTIONS_FILE" 2>/dev/null || fail "option file open failed"
OPEN_OPTIONS_FILE="/proc/$$/fd/${OPTIONS_FD}"
[ -f "$OPEN_OPTIONS_FILE" ] || fail "unsafe opened option file"
opened_identity="$(/usr/bin/stat -Lc '%d:%i:%u:%a' "$OPEN_OPTIONS_FILE" 2>/dev/null)" || fail "opened option file unavailable"
case "$opened_identity" in *:0:400|*:0:600) ;; *) fail "unsafe opened option file authority" ;; esac

validate_path_ancestry() {
  local path="$1"
  local cursor="$path"
  local owner mode group_digit other_digit
  while :; do
    [ ! -L "$cursor" ] || return 1
    owner="$(/usr/bin/stat -c '%u' "$cursor" 2>/dev/null)" || return 1
    mode="$(/usr/bin/stat -c '%a' "$cursor" 2>/dev/null)" || return 1
    [ "$owner" = "0" ] || return 1
    if [ "$cursor" = "$path" ]; then
      [ -f "$cursor" ] || return 1
      case "$mode" in 400|600) ;; *) return 1 ;; esac
    else
      [ -d "$cursor" ] || return 1
      group_digit=$(( (10#$mode / 10) % 10 ))
      other_digit=$(( 10#$mode % 10 ))
      [ $((group_digit & 2)) -eq 0 ] && [ $((other_digit & 2)) -eq 0 ] || return 1
    fi
    [ "$cursor" = "/" ] && break
    cursor="${cursor%/*}"
    [ -n "$cursor" ] || cursor="/"
  done
}
validate_path_ancestry "$SOURCE_OPTIONS_FILE" || fail "unsafe option file ancestry"
path_identity="$(/usr/bin/stat -Lc '%d:%i:%u:%a' "$SOURCE_OPTIONS_FILE" 2>/dev/null)" || fail "option file identity unavailable"
[ "$path_identity" = "$opened_identity" ] || fail "option file identity changed"

PRIVATE_OPTIONS_FILE="$(/bin/mktemp /run/t03-auth-preflight.cnf.XXXXXX)"
ERROR_FILE="$(/bin/mktemp /run/t03-auth-preflight.err.XXXXXX)"
cleanup() {
  local status=$?
  trap - EXIT
  rm -f -- "$PRIVATE_OPTIONS_FILE" "$ERROR_FILE"
  [ ! -e "$PRIVATE_OPTIONS_FILE" ] && [ ! -e "$ERROR_FILE" ] || exit 125
  exit "$status"
}
trap cleanup EXIT
chmod 0600 "$PRIVATE_OPTIONS_FILE" "$ERROR_FILE"
/bin/cp -- "$OPEN_OPTIONS_FILE" "$PRIVATE_OPTIONS_FILE" 2>"$ERROR_FILE" || fail "option file private copy failed"
exec {OPTIONS_FD}<&-
chmod 0600 "$PRIVATE_OPTIONS_FILE"
expected_socket="$(/usr/bin/awk -F= '$1 == "socket" { print substr($0, index($0, "=") + 1) }' "$PRIVATE_OPTIONS_FILE" 2>"$ERROR_FILE")" || fail "database socket option unavailable"
case "$expected_socket" in
  \"*\") expected_socket="${expected_socket#\"}"; expected_socket="${expected_socket%\"}" ;;
esac
[[ $expected_socket =~ ^/[A-Za-z0-9._/-]+$ ]] && [ -S "$expected_socket" ] || fail "database socket option invalid"

[ -x /bin/systemctl ] || fail "systemd authority unavailable"
for service in alumni-backend.service httpd.service crond.service; do
  mask="/run/systemd/system/${service}"
  [ -L "$mask" ] && [ "$(/bin/readlink "$mask" 2>"$ERROR_FILE")" = "/dev/null" ] || fail "writer runtime mask missing"
  load_state="$(/bin/systemctl show --property=LoadState "$service" 2>"$ERROR_FILE")" || fail "writer service lookup failed"
  case "$load_state" in LoadState=loaded|LoadState=masked) ;; *) fail "writer service unavailable" ;; esac
  if /bin/systemctl is-active --quiet "$service" 2>"$ERROR_FILE"; then
    fail "writer service active"
  else
    service_status=$?
    [ "$service_status" -eq 3 ] || fail "writer service state indeterminate"
  fi
done

snapshot_sql="LOCK TABLES WEO_MEMBER READ;
SELECT CONCAT_WS(CHAR(9),
  (SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER' AND TABLE_TYPE='BASE TABLE'),
  (SELECT COUNT(*) FROM WEO_MEMBER),
  (SELECT COALESCE(DATA_LENGTH,0) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER'),
  (SELECT COALESCE(INDEX_LENGTH,0) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER'),
  (SELECT COUNT(DISTINCT INDEX_NAME) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER'),
  (SELECT COUNT(*) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=DATABASE() AND EVENT_OBJECT_TABLE='WEO_MEMBER'),
  (SELECT COUNT(DISTINCT INDEX_NAME) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER' AND INDEX_TYPE='SPATIAL'),
  @@socket,
  @@datadir
);
UNLOCK TABLES;"
if ! snapshot="$(/usr/bin/mysql --defaults-extra-file="$PRIVATE_OPTIONS_FILE" --host=localhost --socket="$expected_socket" --batch --skip-column-names --raw "$DATABASE" -e "$snapshot_sql" 2>"$ERROR_FILE")"; then
  fail "database snapshot failed"
fi

IFS=$'\t' read -r engine rows data_bytes index_bytes indexes triggers spatial_indexes server_socket datadir extra <<< "$snapshot"
[ -z "${extra:-}" ] || fail "database snapshot malformed"
case "$engine" in MyISAM) state=source ;; InnoDB) state=target ;; '') fail "WEO_MEMBER missing" ;; *) fail "unexpected WEO_MEMBER engine" ;; esac
[ "$spatial_indexes" = "0" ] || fail "unsupported spatial index"

[ -S "$server_socket" ] || fail "database Unix socket unavailable"
[ "$server_socket" = "$expected_socket" ] || fail "database Unix socket mismatch"
[ -d "$datadir" ] || fail "local database datadir unavailable"
free_kib="$(/bin/df -Pk "$datadir" 2>"$ERROR_FILE" | /usr/bin/awk 'NR==2 {print $4}')"

MAX_METRIC=1000000000000000
for value in "$rows" "$data_bytes" "$index_bytes" "$indexes" "$triggers" "$spatial_indexes" "$free_kib"; do
  case "$value" in ''|*[!0-9]*) fail "numeric metric malformed" ;; esac
  [ "$value" -le "$MAX_METRIC" ] || fail "numeric metric exceeds bound"
done
copy_bytes=$((data_bytes + index_bytes))
required_kib=$(((copy_bytes * 2 + 1023) / 1024 + 262144))
[ "$free_kib" -ge "$required_kib" ] || fail "insufficient disk headroom"
estimated_lock_seconds=$(((copy_bytes + 5242879) / 5242880))
[ "$estimated_lock_seconds" -gt 0 ] || estimated_lock_seconds=1

printf 'T03_AUTH_ENGINE_PREFLIGHT=PASS engine=%s state=%s rows=%s data_bytes=%s index_bytes=%s indexes=%s triggers=%s unsupported_spatial_indexes=%s free_kib=%s required_kib=%s estimated_lock_seconds=%s headroom=pass writer_freeze=verified writer_masks=verified database_locality=unix-socket snapshot=locked arithmetic=bounded stderr=redacted options=stable-fd-copy execution_binding=pending-controller\n' \
  "$engine" "$state" "$rows" "$data_bytes" "$index_bytes" "$indexes" "$triggers" "$spatial_indexes" \
  "$free_kib" "$required_kib" "$estimated_lock_seconds"
