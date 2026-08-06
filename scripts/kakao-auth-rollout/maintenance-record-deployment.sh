#!/usr/bin/env bash
# maintenance-record-deployment.sh — Record generation-bound backend deployment evidence.
set -euo pipefail

SENTINEL=${MAINTENANCE_SENTINEL_PATH:-/run/alumni/maintenance}
CANONICAL_SENTINEL=/run/alumni/maintenance
BINARY=${BACKEND_BINARY_PATH:-/app/backend/server}
EXPECTED_SHA256=${BACKEND_EXPECTED_SHA256:-}
ROLLBACK_PATH=${BACKEND_ROLLBACK_PATH:-}
ROLLBACK_EXPECTED_SHA256=${BACKEND_ROLLBACK_EXPECTED_SHA256:-}
EVIDENCE_OUTPUT=${BACKEND_DEPLOY_EVIDENCE_OUTPUT:-/run/alumni/backend-deployment.pass}
SERVICE=${BACKEND_SERVICE:-alumni-backend}
HEALTH_URL=${BACKEND_HEALTH_URL:-http://127.0.0.1:8080/api/health}
PROC_ROOT=${BACKEND_PROC_ROOT:-/proc}

fail() {
  printf 'ERROR maintenance_deployment_evidence state=blocked reason=%s\n' "$1" >&2
  exit 1
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d ' ' -f 1
  else
    shasum -a 256 "$1" | cut -d ' ' -f 1
  fi
}

file_identity() {
  stat -L -c '%d:%i' "$1" 2>/dev/null || stat -L -f '%d:%i' "$1"
}

file_mode() {
  stat -c '%a' "$1" 2>/dev/null || stat -f '%Lp' "$1"
}

file_owner_uid() {
  stat -c '%u' "$1" 2>/dev/null || stat -f '%u' "$1"
}

systemctl_property() {
  local property=$1 service=$2 output
  output=$(systemctl show --property "$property" "$service") || return 1
  [[ $(grep -c "^${property}=" <<< "$output") == 1 ]] || return 1
  [[ $output == "${property}="* && $output != *$'\n'* ]] || return 1
  printf '%s' "${output#*=}"
}

[[ ${MAINTENANCE_DEPLOY_EVIDENCE_APPROVED:-0} == 1 ]] || fail approval_required
[[ $SENTINEL == /* && $BINARY == /* && $ROLLBACK_PATH == /* && $EVIDENCE_OUTPUT == /* ]] || fail paths_must_be_absolute
expected_owner_uid=$(id -u)
if [[ $SENTINEL == "$CANONICAL_SENTINEL" ]]; then
  expected_owner_uid=0
fi
[[ -f $SENTINEL && ! -L $SENTINEL ]] || fail sentinel_not_active
[[ $(file_mode "$SENTINEL") == 644 ]] || fail sentinel_mode_invalid
[[ $(file_owner_uid "$SENTINEL") == "$expected_owner_uid" ]] || fail sentinel_owner_invalid
[[ $(grep -Fxc 'state=active' "$SENTINEL" || true) == 1 ]] || fail sentinel_state_invalid
[[ -f $BINARY && ! -L $BINARY && -x $BINARY ]] || fail backend_binary_invalid
[[ -f $ROLLBACK_PATH && ! -L $ROLLBACK_PATH && -x $ROLLBACK_PATH ]] || fail rollback_binary_invalid
[[ $EXPECTED_SHA256 =~ ^[a-f0-9]{64}$ ]] || fail expected_artifact_digest_invalid
[[ $ROLLBACK_EXPECTED_SHA256 =~ ^[a-f0-9]{64}$ ]] || fail expected_rollback_digest_invalid
[[ $HEALTH_URL =~ ^http://(127\.0\.0\.1|localhost):[0-9]{1,5}/[A-Za-z0-9._~/-]+$ ]] || fail loopback_health_url_required
[[ $EVIDENCE_OUTPUT != "$SENTINEL" && $EVIDENCE_OUTPUT != "$BINARY" &&
   $EVIDENCE_OUTPUT != "$ROLLBACK_PATH" && $ROLLBACK_PATH != "$BINARY" ]] || fail evidence_output_conflicts_with_input

evidence_dir=$(dirname "$EVIDENCE_OUTPUT")
[[ -d $evidence_dir && ! -L $evidence_dir ]] || fail evidence_output_directory_invalid

generation_count=$(grep -c '^generation=' "$SENTINEL" || true)
generation=$(sed -n 's/^generation=//p' "$SENTINEL")
[[ $generation_count == 1 && $generation =~ ^[a-f0-9]{32}$ ]] || fail sentinel_generation_invalid

actual_sha256=$(sha256_file "$BINARY") || fail backend_artifact_digest_failed
[[ $actual_sha256 == "$EXPECTED_SHA256" ]] || fail backend_artifact_digest_mismatch
rollback_sha256=$(sha256_file "$ROLLBACK_PATH") || fail rollback_artifact_digest_failed
[[ $rollback_sha256 =~ ^[a-f0-9]{64}$ ]] || fail rollback_artifact_digest_invalid
[[ $rollback_sha256 == "$ROLLBACK_EXPECTED_SHA256" ]] || fail rollback_artifact_digest_mismatch
systemctl is-active --quiet "$SERVICE" || fail backend_service_not_active
main_pid=$(systemctl_property MainPID "$SERVICE") || fail backend_main_pid_unavailable
[[ $main_pid =~ ^[0-9]+$ && $main_pid -gt 0 ]] || fail backend_main_pid_invalid
exec_start=$(systemctl_property ExecStart "$SERVICE") || fail backend_exec_start_unavailable
[[ " $exec_start " == *" path=$BINARY ;"* ]] || fail backend_exec_start_mismatch
running_executable="$PROC_ROOT/$main_pid/exe"
[[ -e $running_executable ]] || fail backend_running_executable_unavailable
[[ $(file_identity "$running_executable") == "$(file_identity "$BINARY")" ]] || fail backend_running_executable_mismatch
running_sha256=$(sha256_file "$running_executable") || fail backend_running_executable_digest_failed
[[ $running_sha256 == "$EXPECTED_SHA256" ]] || fail backend_running_executable_digest_mismatch
curl --disable --noproxy '*' --silent --show-error --fail --max-time 10 "$HEALTH_URL" >/dev/null || fail backend_health_failed

recorded_at=$(date -u '+%Y-%m-%dT%H:%M:%SZ') || fail evidence_clock_failed
evidence_tmp=$(mktemp "${EVIDENCE_OUTPUT}.tmp.XXXXXX") || fail evidence_create_failed
trap 'rm -f "$evidence_tmp"' EXIT
printf 'state=PASS\nkind=deployment\ngeneration=%s\nartifact_sha256=%s\nrollback_path=%s\nrollback_sha256=%s\nmain_pid=%s\nrecorded_at=%s\n' \
  "$generation" "$actual_sha256" "$ROLLBACK_PATH" "$rollback_sha256" "$main_pid" "$recorded_at" > "$evidence_tmp"
chmod 0600 "$evidence_tmp"
mv -f "$evidence_tmp" "$EVIDENCE_OUTPUT"
trap - EXIT

printf 'PASS maintenance_deployment_evidence generation=current artifact=verified service=healthy\n'
