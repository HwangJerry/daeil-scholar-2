#!/usr/bin/env bash
# maintenance-release.sh — Release writes only after deployment and smoke evidence pass.
set -euo pipefail

SENTINEL=${MAINTENANCE_SENTINEL_PATH:-/run/alumni/maintenance}
SERVICE=${BACKEND_SERVICE:-alumni-backend}
HEALTH_URL=${BACKEND_HEALTH_URL:-http://127.0.0.1:8080/api/health}
BLOCK_PROBE_URL=${MAINTENANCE_BLOCK_PROBE_URL:-http://127.0.0.1:8080/api/auth/login}
DEPLOY_EVIDENCE=${BACKEND_DEPLOY_EVIDENCE:-}
SMOKE_EVIDENCE=${BACKEND_SMOKE_EVIDENCE:-}
MIGRATION_EVIDENCE=${BACKEND_MIGRATION_EVIDENCE:-}
OBSERVATION_OUTPUT=${PUSH_OBSERVATION_START_OUTPUT:-}
BINARY=${BACKEND_BINARY_PATH:-/app/backend/server}
PROC_ROOT=${BACKEND_PROC_ROOT:-/proc}

fail() {
  printf 'ERROR maintenance_release state=blocked reason=%s\n' "$1" >&2
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

filesystem_device() {
  stat -c '%d' "$1" 2>/dev/null || stat -f '%d' "$1"
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

has_exact_line_once() {
  local path=$1 expected=$2 count
  count=$(grep -Fxc "$expected" "$path" || true)
  [[ $count == 1 ]]
}

observation_started() {
	local path=$1
	[[ $path == /* && -f $path && ! -L $path ]] || return 1
	[[ $(file_mode "$path") == 600 ]] || return 1
	[[ $(file_owner_uid "$path") == $(id -u) ]] || return 1
	has_exact_line_once "$path" 'state=STARTED' || return 1
	[[ $(grep -Ec '^generation=[a-f0-9]{32}$' "$path" || true) == 1 ]] || return 1
	[[ $(grep -Ec '^started_at=[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$' "$path" || true) == 1 ]]
}

evidence_passed() {
	local path=$1 kind=$2 generation=$3 main_pid rollback_path rollback_sha256
	[[ $path == /* ]] || return 1
	[[ -f $path && ! -L $path ]] || return 1
	[[ $(file_mode "$path") == 600 ]] || return 1
	[[ $(file_owner_uid "$path") == $(id -u) ]] || return 1
	if [[ -e $SENTINEL || -L $SENTINEL ]]; then
		[[ $(file_mtime "$path") -ge $(file_mtime "$SENTINEL") ]] || return 1
	fi
	has_exact_line_once "$path" 'state=PASS' || return 1
	has_exact_line_once "$path" "kind=$kind" || return 1
	has_exact_line_once "$path" "generation=$generation" || return 1
	[[ $(grep -Ec '^recorded_at=[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$' "$path" || true) == 1 ]] || return 1
	case "$kind" in
		deployment)
			[[ $(grep -Ec '^artifact_sha256=[a-f0-9]{64}$' "$path" || true) == 1 ]] || return 1
			[[ $(grep -Ec '^rollback_path=/[^[:space:]]+$' "$path" || true) == 1 ]] || return 1
			[[ $(grep -Ec '^rollback_sha256=[a-f0-9]{64}$' "$path" || true) == 1 ]] || return 1
			[[ $(grep -Ec '^main_pid=[1-9][0-9]*$' "$path" || true) == 1 ]] || return 1
			main_pid=$(sed -n 's/^main_pid=//p' "$path")
			rollback_path=$(sed -n 's/^rollback_path=//p' "$path")
			rollback_sha256=$(sed -n 's/^rollback_sha256=//p' "$path")
			[[ $main_pid -gt 0 && -f $rollback_path && ! -L $rollback_path && -x $rollback_path ]] || return 1
			[[ $(sha256_file "$rollback_path") == "$rollback_sha256" ]]
			;;
		smoke)
			[[ $(grep -Ec '^status=2[0-9]{2}$' "$path" || true) == 1 ]]
			;;
		migration-postcheck)
			has_exact_line_once "$path" 'range=036-039' || return 1
			has_exact_line_once "$path" 'postcheck_metrics=15'
			;;
		*) return 1 ;;
	esac
}

runtime_matches_deployment_evidence() {
	local path=$1 recorded_pid artifact_sha current_pid exec_start running_executable
	recorded_pid=$(sed -n 's/^main_pid=//p' "$path")
	artifact_sha=$(sed -n 's/^artifact_sha256=//p' "$path")
	[[ $recorded_pid =~ ^[1-9][0-9]*$ && $artifact_sha =~ ^[a-f0-9]{64}$ ]] || return 1
	[[ $BINARY == /* && -f $BINARY && ! -L $BINARY && -x $BINARY ]] || return 1
	current_pid=$(systemctl show --property MainPID --value "$SERVICE") || return 1
	[[ $current_pid == "$recorded_pid" ]] || return 1
	exec_start=$(systemctl show --property ExecStart --value "$SERVICE") || return 1
	[[ " $exec_start " == *" path=$BINARY ;"* ]] || return 1
	running_executable="$PROC_ROOT/$current_pid/exe"
	[[ -e $running_executable ]] || return 1
	[[ $(file_identity "$running_executable") == "$(file_identity "$BINARY")" ]] || return 1
	[[ $(sha256_file "$running_executable") == "$artifact_sha" ]]
}

[[ ${MAINTENANCE_RELEASE_APPROVED:-0} == 1 ]] || fail approval_required
[[ $SENTINEL == /* ]] || fail sentinel_path_must_be_absolute
[[ -n $OBSERVATION_OUTPUT ]] || fail observation_output_path_required
[[ $OBSERVATION_OUTPUT == /* ]] || fail observation_output_path_must_be_absolute
if [[ ! -e $SENTINEL && ! -L $SENTINEL ]]; then
	observation_started "$OBSERVATION_OUTPUT" || fail sentinel_not_active
	observation_generation_count=$(grep -c '^generation=' "$OBSERVATION_OUTPUT" || true)
	observation_generation=$(sed -n 's/^generation=//p' "$OBSERVATION_OUTPUT")
	[[ $observation_generation_count == 1 && $observation_generation =~ ^[a-f0-9]{32}$ ]] ||
		fail observation_generation_invalid
	[[ -n $DEPLOY_EVIDENCE ]] || fail deployment_evidence_path_required
	[[ -n $SMOKE_EVIDENCE ]] || fail smoke_evidence_path_required
	[[ -n $MIGRATION_EVIDENCE ]] || fail migration_evidence_path_required
	evidence_passed "$DEPLOY_EVIDENCE" deployment "$observation_generation" || fail deployment_evidence_not_pass
	evidence_passed "$SMOKE_EVIDENCE" smoke "$observation_generation" || fail smoke_evidence_not_pass
	evidence_passed "$MIGRATION_EVIDENCE" migration-postcheck "$observation_generation" || fail migration_evidence_not_pass
	printf 'PASS maintenance_release sentinel=inactive backend=previously_verified observation=started idempotent=true\n'
	exit 0
fi
[[ -f $SENTINEL && ! -L $SENTINEL ]] || fail sentinel_not_active
generation_count=$(grep -c '^generation=' "$SENTINEL" || true)
generation=$(sed -n 's/^generation=//p' "$SENTINEL")
[[ $generation_count == 1 && $generation =~ ^[a-f0-9]{32}$ ]] || fail sentinel_generation_invalid
[[ -n $DEPLOY_EVIDENCE ]] || fail deployment_evidence_path_required
[[ -n $SMOKE_EVIDENCE ]] || fail smoke_evidence_path_required
[[ -n $MIGRATION_EVIDENCE ]] || fail migration_evidence_path_required
evidence_passed "$DEPLOY_EVIDENCE" deployment "$generation" || fail deployment_evidence_not_pass
evidence_passed "$SMOKE_EVIDENCE" smoke "$generation" || fail smoke_evidence_not_pass
evidence_passed "$MIGRATION_EVIDENCE" migration-postcheck "$generation" || fail migration_evidence_not_pass
systemctl is-active --quiet "$SERVICE" || fail backend_service_not_active
runtime_matches_deployment_evidence "$DEPLOY_EVIDENCE" || fail backend_runtime_changed_after_deployment_evidence
curl --disable --noproxy '*' --silent --show-error --fail --max-time 10 "$HEALTH_URL" >/dev/null || fail health_precheck_failed
block_status=$(curl --disable --noproxy '*' --silent --show-error --max-time 10 --output /dev/null \
  --write-out '%{http_code}' --request POST "$BLOCK_PROBE_URL") || fail block_probe_failed
[[ $block_status == 503 ]] || fail no_proof_write_not_blocked

release_started_at=$(date -u '+%Y-%m-%dT%H:%M:%SZ') || fail observation_start_clock_failed
observation_dir=$(dirname "$OBSERVATION_OUTPUT")
[[ -d $observation_dir && ! -L $observation_dir ]] || fail observation_output_directory_invalid
[[ $observation_dir == $(dirname "$SENTINEL") ]] || fail observation_output_directory_mismatch
[[ ! -e $OBSERVATION_OUTPUT && ! -L $OBSERVATION_OUTPUT ]] || fail observation_output_already_exists
[[ $(filesystem_device "$SENTINEL") == $(filesystem_device "$observation_dir") ]] || fail observation_output_cross_filesystem
command -v python3 >/dev/null 2>&1 || fail atomic_release_helper_unavailable
release_tmp=$(mktemp "${SENTINEL}.release.XXXXXX") || fail release_record_create_failed
trap 'rm -f "$release_tmp"' EXIT
printf 'state=STARTED\ngeneration=%s\nstarted_at=%s\n' "$generation" "$release_started_at" > "$release_tmp" ||
  fail release_record_write_failed
chmod 0600 "$release_tmp" || fail release_record_mode_failed
systemctl is-active --quiet "$SERVICE" || fail backend_service_not_active_before_release
runtime_matches_deployment_evidence "$DEPLOY_EVIDENCE" || fail backend_runtime_changed_before_release
mv -f "$release_tmp" "$SENTINEL" || fail release_record_prepare_failed
trap - EXIT
python3 - "$SENTINEL" "$OBSERVATION_OUTPUT" <<'PY' || fail sentinel_release_failed
import os
import sys

source, destination = sys.argv[1:]
if os.path.isdir(destination):
    raise SystemExit(1)
os.replace(source, destination)
PY

printf 'PASS maintenance_release sentinel=inactive backend=healthy observation=started\n'
