#!/usr/bin/env bash
# maintenance-release.sh — Release writes only after deployment and smoke evidence pass.
set -euo pipefail

SENTINEL=${MAINTENANCE_SENTINEL_PATH:-/run/alumni/maintenance}
BRIDGE=${MAINTENANCE_RELEASE_BRIDGE_PATH:-/run/alumni/maintenance-release-bridge}
RELEASE_PROOF_FILE=${MAINTENANCE_RELEASE_PROOF_FILE:-/run/alumni/maintenance-release-proof}
RELEASE_ATTEMPT=${MAINTENANCE_RELEASE_APPROVAL_ATTEMPT_ID:-}
RELEASE_OWNER_UID=${MAINTENANCE_RELEASE_OWNER_UID:-0}
RELEASE_CONTROL_BASE_URL=${MAINTENANCE_RELEASE_CONTROL_BASE_URL:-http://127.0.0.1:8080}
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

write_bridge_create() {
	local content=$1 parent payload
	parent=$(dirname "$BRIDGE")
	payload=$(mktemp "$parent/.maintenance-release-bridge.payload.XXXXXX") || return 1
	chmod 0600 "$payload" || { rm -f "$payload"; return 1; }
	printf '%s' "$content" > "$payload" || { rm -f "$payload"; return 1; }
	python3 - "$BRIDGE" "$payload" "$RELEASE_OWNER_UID" <<'PY'
import os
import stat
import sys

path, payload, expected_uid = sys.argv[1], sys.argv[2], int(sys.argv[3])
with open(payload, "rb") as source:
    content = source.read()
fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o644)
try:
    os.fchmod(fd, 0o644)
    os.write(fd, content)
    os.fsync(fd)
finally:
    os.close(fd)
info = os.lstat(path)
if not stat.S_ISREG(info.st_mode) or info.st_uid != expected_uid or stat.S_IMODE(info.st_mode) != 0o644:
    raise SystemExit(1)
parent_fd = os.open(os.path.dirname(path), os.O_RDONLY)
try:
    os.fsync(parent_fd)
finally:
    os.close(parent_fd)
PY
	local status=$?
	rm -f "$payload"
	return "$status"
}

write_bridge_replace() {
	local expected=$1 replacement=$2 parent expected_file replacement_file
	parent=$(dirname "$BRIDGE")
	expected_file=$(mktemp "$parent/.maintenance-release-bridge.expected.XXXXXX") || return 1
	replacement_file=$(mktemp "$parent/.maintenance-release-bridge.replacement.XXXXXX") || {
		rm -f "$expected_file"
		return 1
	}
	chmod 0600 "$expected_file" "$replacement_file" || {
		rm -f "$expected_file" "$replacement_file"
		return 1
	}
	printf '%s' "$expected" > "$expected_file"
	printf '%s' "$replacement" > "$replacement_file"
	python3 - "$BRIDGE" "$expected_file" "$replacement_file" "$RELEASE_OWNER_UID" <<'PY'
import os
import stat
import sys
import tempfile

path, expected_path, replacement_path, expected_uid = sys.argv[1], sys.argv[2], sys.argv[3], int(sys.argv[4])
with open(expected_path, "rb") as source:
    expected = source.read()
with open(replacement_path, "rb") as source:
    replacement = source.read()
info = os.lstat(path)
if not stat.S_ISREG(info.st_mode) or info.st_uid != expected_uid or stat.S_IMODE(info.st_mode) != 0o644:
    raise SystemExit(1)
with open(path, "rb") as source:
    if source.read() != expected:
        raise SystemExit(1)
parent = os.path.dirname(path)
fd, temporary = tempfile.mkstemp(prefix=".maintenance-release-bridge.install.", dir=parent)
try:
    os.fchmod(fd, 0o644)
    os.write(fd, replacement)
    os.fsync(fd)
    os.close(fd)
    fd = -1
    os.replace(temporary, path)
    parent_fd = os.open(parent, os.O_RDONLY)
    try:
        os.fsync(parent_fd)
    finally:
        os.close(parent_fd)
finally:
    if fd >= 0:
        os.close(fd)
    try:
        os.unlink(temporary)
    except FileNotFoundError:
        pass
PY
	local status=$?
	rm -f "$expected_file" "$replacement_file"
	return "$status"
}

call_release_control() {
	local path=$1 expected_state=$2 response
	response=$(curl --disable --config "$release_curl_config" --silent --show-error --fail \
		--request POST --header 'Content-Type: application/json' --data-binary "@$release_request_file" \
		"$RELEASE_CONTROL_BASE_URL$path") || return 1
	[[ $response == "{\"state\":\"$expected_state\"}" ]]
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

systemctl_property() {
	local property=$1 service=$2 output
	output=$(systemctl show --property "$property" "$service") || return 1
	[[ $(grep -c "^${property}=" <<< "$output") == 1 ]] || return 1
	[[ $output == "${property}="* && $output != *$'\n'* ]] || return 1
	printf '%s' "${output#*=}"
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
	current_pid=$(systemctl_property MainPID "$SERVICE") || return 1
	[[ $current_pid == "$recorded_pid" ]] || return 1
	exec_start=$(systemctl_property ExecStart "$SERVICE") || return 1
	[[ " $exec_start " == *" path=$BINARY ;"* ]] || return 1
	running_executable="$PROC_ROOT/$current_pid/exe"
	[[ -e $running_executable ]] || return 1
	[[ $(file_identity "$running_executable") == "$(file_identity "$BINARY")" ]] || return 1
	[[ $(sha256_file "$running_executable") == "$artifact_sha" ]]
}

[[ ${MAINTENANCE_RELEASE_APPROVED:-0} == 1 ]] || fail approval_required
[[ $SENTINEL == /* ]] || fail sentinel_path_must_be_absolute
[[ $BRIDGE == /* ]] || fail bridge_path_must_be_absolute
[[ $RELEASE_PROOF_FILE == /* ]] || fail release_proof_path_must_be_absolute
[[ $RELEASE_OWNER_UID =~ ^[0-9]+$ ]] || fail release_owner_uid_invalid
[[ $RELEASE_CONTROL_BASE_URL =~ ^http://127\.0\.0\.1:[1-9][0-9]{0,4}$ ]] || fail release_control_url_invalid
[[ -n $OBSERVATION_OUTPUT ]] || fail observation_output_path_required
[[ $OBSERVATION_OUTPUT == /* ]] || fail observation_output_path_must_be_absolute
if [[ ! -e $SENTINEL && ! -L $SENTINEL ]]; then
	[[ ! -e $BRIDGE && ! -L $BRIDGE ]] || fail release_bridge_still_active
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
if [[ $SENTINEL == /run/alumni/maintenance || $BRIDGE == /run/alumni/maintenance-release-bridge ||
      $RELEASE_PROOF_FILE == /run/alumni/maintenance-release-proof ]]; then
	[[ $SENTINEL == /run/alumni/maintenance && $BRIDGE == /run/alumni/maintenance-release-bridge &&
       $RELEASE_PROOF_FILE == /run/alumni/maintenance-release-proof && $RELEASE_OWNER_UID == 0 ]] ||
		fail canonical_release_paths_must_be_bound
	[[ $(id -u) == 0 ]] || fail canonical_release_requires_root
fi
[[ $RELEASE_ATTEMPT =~ ^[a-f0-9]{64}$ ]] || fail release_attempt_invalid
[[ -f $RELEASE_PROOF_FILE && ! -L $RELEASE_PROOF_FILE ]] || fail release_proof_file_invalid
[[ $(file_mode "$RELEASE_PROOF_FILE") == 600 ]] || fail release_proof_mode_invalid
[[ $(file_owner_uid "$RELEASE_PROOF_FILE") == "$RELEASE_OWNER_UID" ]] || fail release_proof_owner_invalid
[[ $(LC_ALL=C wc -c < "$RELEASE_PROOF_FILE") -eq 65 ]] || fail release_proof_size_invalid
release_proof=$(<"$RELEASE_PROOF_FILE") || fail release_proof_read_failed
[[ $release_proof =~ ^[a-f0-9]{64}$ ]] || fail release_proof_format_invalid
printf '%s\n' "$release_proof" | cmp -s - "$RELEASE_PROOF_FILE" || fail release_proof_bytes_invalid
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
[[ ! -e $BRIDGE && ! -L $BRIDGE ]] || fail release_bridge_already_exists
[[ $(dirname "$BRIDGE") == $(dirname "$SENTINEL") ]] || fail release_bridge_directory_mismatch

release_curl_config=$(mktemp "${BRIDGE}.curl.XXXXXX") || fail release_curl_config_create_failed
release_request_file=$(mktemp "${BRIDGE}.request.XXXXXX") || {
	rm -f "$release_curl_config"
	fail release_request_create_failed
}
release_tmp=
cleanup_release_temp() {
	rm -f "${release_curl_config:-}" "${release_request_file:-}" "${release_tmp:-}"
}
trap cleanup_release_temp EXIT
chmod 0600 "$release_curl_config" "$release_request_file" || fail release_control_temp_mode_failed
{
	printf 'header = "X-Maintenance-Release-Proof: %s"\n' "$release_proof"
	printf 'connect-timeout = 3\n'
	printf 'max-time = 95\n'
	printf 'noproxy = "*"\n'
} > "$release_curl_config" || fail release_curl_config_write_failed
printf '{"generation":"%s","approval_attempt_id":"%s"}\n' "$generation" "$RELEASE_ATTEMPT" > "$release_request_file" ||
	fail release_request_write_failed
unset release_proof

prepared_bridge=$(printf 'state=prepared\ngeneration=%s\napproval_attempt_id=%s\n' "$generation" "$RELEASE_ATTEMPT")
prepared_bridge=${prepared_bridge}$'\n'
drained_bridge=$(printf 'state=drained\ngeneration=%s\napproval_attempt_id=%s\n' "$generation" "$RELEASE_ATTEMPT")
drained_bridge=${drained_bridge}$'\n'
# PREPARED_RELEASE_BRIDGE_INSTALL
write_bridge_create "$prepared_bridge" || fail release_bridge_create_failed
# APPLICATION_DRAIN_CONTROL_CALL
call_release_control '/internal/maintenance/drain' DRAINED || fail application_drain_failed
# DRAINED_RELEASE_BRIDGE_PERSIST
write_bridge_replace "$prepared_bridge" "$drained_bridge" || fail release_bridge_drain_proof_failed

release_tmp=$(mktemp "${SENTINEL}.release.XXXXXX") || fail release_record_create_failed
printf 'state=STARTED\ngeneration=%s\nstarted_at=%s\n' "$generation" "$release_started_at" > "$release_tmp" ||
  fail release_record_write_failed
chmod 0600 "$release_tmp" || fail release_record_mode_failed
systemctl is-active --quiet "$SERVICE" || fail backend_service_not_active_before_release
runtime_matches_deployment_evidence "$DEPLOY_EVIDENCE" || fail backend_runtime_changed_before_release
mv -f "$release_tmp" "$SENTINEL" || fail release_record_prepare_failed
release_tmp=
# CANONICAL_SENTINEL_TO_OBSERVATION
python3 - "$SENTINEL" "$OBSERVATION_OUTPUT" <<'PY' || fail sentinel_release_failed
import os
import sys

source, destination = sys.argv[1:]
if os.path.isdir(destination):
    raise SystemExit(1)
os.replace(source, destination)
PY

call_release_control '/internal/maintenance/arm-open' ARMED || fail application_arm_open_failed
rm -f "$release_curl_config" "$release_request_file" || fail release_control_temp_cleanup_failed
release_curl_config=
release_request_file=
# FINAL_RELEASE_BRIDGE_UNLINK
python3 - "$BRIDGE" <<'PY' || fail release_bridge_remove_failed
import os
import sys

path = sys.argv[1]
os.unlink(path)
parent_fd = os.open(os.path.dirname(path), os.O_RDONLY)
try:
    os.fsync(parent_fd)
finally:
    os.close(parent_fd)
PY
trap - EXIT

printf 'PASS maintenance_release sentinel=inactive backend=healthy observation=started\n'
