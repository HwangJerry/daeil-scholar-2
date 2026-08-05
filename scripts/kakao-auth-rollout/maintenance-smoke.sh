#!/usr/bin/env bash
# maintenance-smoke.sh — Run one allowlisted loopback write with proof-file authorization.
set -euo pipefail

SENTINEL=${MAINTENANCE_SENTINEL_PATH:-/run/alumni/maintenance}
PROOF_FILE=${MAINTENANCE_SMOKE_PROOF_FILE:-}
SMOKE_URL=${MAINTENANCE_SMOKE_URL:-}
SMOKE_METHOD=${MAINTENANCE_SMOKE_METHOD:-POST}
SMOKE_BODY_FILE=${MAINTENANCE_SMOKE_BODY_FILE:-}
EXPECTED_STATUS=${MAINTENANCE_SMOKE_EXPECT_STATUS:-200}
EVIDENCE_OUTPUT=${MAINTENANCE_SMOKE_EVIDENCE_OUTPUT:-}

fail() {
  printf 'ERROR maintenance_smoke state=failed reason=%s\n' "$1" >&2
  exit 1
}

file_identity() {
  stat -c '%d:%i' "$1" 2>/dev/null || stat -f '%d:%i' "$1"
}

fd_metadata() {
  python3 - "$1" <<'PY'
import os
import stat
import sys

descriptor = int(sys.argv[1])
metadata = os.fstat(descriptor)
print(
    f"{stat.S_IMODE(metadata.st_mode)}:{metadata.st_uid}:"
    f"{metadata.st_dev}:{metadata.st_ino}:{int(stat.S_ISREG(metadata.st_mode))}"
)
PY
}

[[ -f $SENTINEL ]] || fail sentinel_not_active
generation_count=$(grep -c '^generation=' "$SENTINEL" || true)
generation=$(sed -n 's/^generation=//p' "$SENTINEL")
[[ $generation_count == 1 && $generation =~ ^[a-f0-9]{32}$ ]] || fail sentinel_generation_invalid
command -v python3 >/dev/null 2>&1 || fail descriptor_validation_helper_unavailable
[[ -n $PROOF_FILE && $PROOF_FILE == /* ]] || fail proof_file_required
exec 8< "$PROOF_FILE" || fail proof_file_required
[[ -f /dev/fd/8 && ! -L $PROOF_FILE && -f $PROOF_FILE ]] || fail proof_file_required
IFS=: read -r proof_mode proof_owner proof_device proof_inode proof_regular <<< "$(fd_metadata 8)"
[[ $proof_regular == 1 && $proof_mode == 384 ]] || fail proof_file_mode_not_0600
[[ $proof_owner == $(id -u) ]] || fail proof_file_owner_invalid
[[ $(file_identity "$PROOF_FILE") == "${proof_device}:${proof_inode}" ]] || fail proof_file_identity_changed
[[ $SMOKE_URL =~ ^http://(127\.0\.0\.1|localhost):[0-9]{1,5}/[A-Za-z0-9._~/-]+$ ]] || fail loopback_url_required
[[ $SMOKE_METHOD =~ ^(POST|PUT|PATCH|DELETE)$ ]] || fail smoke_method_invalid
[[ $EXPECTED_STATUS =~ ^2[0-9]{2}$ ]] || fail expected_status_must_be_success
[[ -n $EVIDENCE_OUTPUT ]] || fail evidence_output_required
[[ $EVIDENCE_OUTPUT == /* ]] || fail evidence_output_path_must_be_absolute
[[ -z $SMOKE_BODY_FILE || $SMOKE_BODY_FILE == /* ]] || fail body_file_invalid
[[ $SMOKE_BODY_FILE != *$'\n'* && $SMOKE_BODY_FILE != *$'\r'* && $SMOKE_BODY_FILE != *'"'* ]] || fail body_file_path_invalid
if [[ -n $SMOKE_BODY_FILE ]]; then
  exec 9< "$SMOKE_BODY_FILE" || fail body_file_invalid
  [[ -f /dev/fd/9 && ! -L $SMOKE_BODY_FILE && -f $SMOKE_BODY_FILE ]] || fail body_file_invalid
  IFS=: read -r body_mode body_owner body_device body_inode body_regular <<< "$(fd_metadata 9)"
  [[ $body_regular == 1 && $body_mode == 384 ]] || fail body_file_mode_not_0600
  [[ $body_owner == $(id -u) ]] || fail body_file_owner_invalid
  [[ $(file_identity "$SMOKE_BODY_FILE") == "${body_device}:${body_inode}" ]] || fail body_file_identity_changed
fi

proof=$(tr -d '\r\n' <&8)
exec 8<&-
[[ $proof =~ ^[A-Za-z0-9._~-]+$ ]] || fail proof_format_invalid
response=$(mktemp "${TMPDIR:-/tmp}/maintenance-smoke.XXXXXX")
trap 'rm -f "$response"' EXIT

body_copy=''
if [[ -n $SMOKE_BODY_FILE ]]; then
  body_copy=$(mktemp "${TMPDIR:-/tmp}/maintenance-smoke-body.XXXXXX") || fail body_copy_create_failed
  chmod 0600 "$body_copy"
  cat <&9 > "$body_copy" || fail body_copy_failed
  exec 9<&-
fi

curl_config=$(mktemp "${TMPDIR:-/tmp}/maintenance-smoke-curl.XXXXXX")
trap 'rm -f "$response" "$curl_config" "$body_copy"' EXIT
chmod 0600 "$curl_config"
{
  printf 'silent\n'
  printf 'show-error\n'
  printf 'max-time = 30\n'
  printf 'noproxy = "*"\n'
  printf 'url = "%s"\n' "$SMOKE_URL"
  printf 'request = "%s"\n' "$SMOKE_METHOD"
  printf 'header = "X-Maintenance-Smoke-Proof: %s"\n' "$proof"
  printf 'header = "Content-Type: application/json"\n'
  [[ -z $body_copy ]] || printf 'data-binary = "@%s"\n' "$body_copy"
  printf 'output = "%s"\nwrite-out = "%%{http_code}"\n' "$response"
} > "$curl_config"
unset proof

status=$(curl --disable --config "$curl_config") || fail request_failed
[[ $status == "$EXPECTED_STATUS" ]] || fail unexpected_http_status
evidence_dir=$(dirname "$EVIDENCE_OUTPUT")
[[ -d $evidence_dir && ! -L $evidence_dir ]] || fail evidence_output_directory_invalid
evidence_tmp=$(mktemp "${EVIDENCE_OUTPUT}.tmp.XXXXXX") || fail evidence_record_create_failed
trap 'rm -f "$response" "$curl_config" "$body_copy" "$evidence_tmp"' EXIT
recorded_at=$(date -u '+%Y-%m-%dT%H:%M:%SZ') || fail evidence_clock_failed
printf 'state=PASS\nkind=smoke\ngeneration=%s\nstatus=%s\nrecorded_at=%s\n' \
  "$generation" "$status" "$recorded_at" > "$evidence_tmp"
chmod 0600 "$evidence_tmp"
mv -f "$evidence_tmp" "$EVIDENCE_OUTPUT"
trap 'rm -f "$response" "$curl_config" "$body_copy"' EXIT
printf 'PASS maintenance_smoke status=%s evidence=recorded\n' "$status"
