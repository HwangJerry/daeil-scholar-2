#!/usr/bin/env bash
# maintenance-enter.sh — Activate the production write freeze before DB mutation.
set -euo pipefail

SENTINEL=${MAINTENANCE_SENTINEL_PATH:-/run/alumni/maintenance}
CANONICAL_SENTINEL=/run/alumni/maintenance
SERVICE=${BACKEND_SERVICE:-alumni-backend}
HTTPD_SERVICE=${HTTPD_SERVICE:-httpd}
HTTPD_CONFIG=${HTTPD_CONFIG_PATH:-/etc/httpd/conf.d/alumni.conf}
LEGACY_PROBE_URL=${MAINTENANCE_LEGACY_PROBE_URL:-https://daeilfoundation.or.kr/old/index.php}
LEGACY_PROBE_RESOLVE=${MAINTENANCE_LEGACY_PROBE_RESOLVE:-daeilfoundation.or.kr:443:127.0.0.1}
activated=0
backend_stopped=0
httpd_stopped=0
backend_was_active=0
httpd_was_active=0

fail() {
  if [[ $activated == 1 ]]; then
    systemctl stop "$SERVICE" >/dev/null 2>&1 || true
  else
    if [[ $httpd_stopped == 1 && $httpd_was_active == 1 ]]; then
      systemctl start "$HTTPD_SERVICE" >/dev/null 2>&1 || true
    fi
    if [[ $backend_stopped == 1 && $backend_was_active == 1 ]]; then
      systemctl start "$SERVICE" >/dev/null 2>&1 || true
    fi
  fi
  printf 'ERROR maintenance_enter state=blocked reason=%s\n' "$1" >&2
  exit 1
}

systemctl_property() {
  local property=$1 service=$2 output
  output=$(systemctl show --property "$property" "$service") || return 1
  [[ $(grep -c "^${property}=" <<< "$output") == 1 ]] || return 1
  [[ $output == "${property}="* && $output != *$'\n'* ]] || return 1
  printf '%s' "${output#*=}"
}

[[ ${MAINTENANCE_ENTER_APPROVED:-0} == 1 ]] || fail approval_required
[[ ${MAINTENANCE_HTTPD_RESTART_APPROVED:-0} == 1 ]] || fail httpd_restart_approval_required
[[ $SENTINEL == /* ]] || fail sentinel_path_must_be_absolute
if [[ $SENTINEL == "$CANONICAL_SENTINEL" && $(id -u) != 0 ]]; then
  fail canonical_sentinel_requires_root
fi
[[ $LEGACY_PROBE_URL =~ ^https://([A-Za-z0-9][A-Za-z0-9.-]*)/[A-Za-z0-9._~/-]+$ ]] ||
  fail legacy_probe_url_invalid
legacy_probe_host=${BASH_REMATCH[1]}
[[ $LEGACY_PROBE_RESOLVE == "${legacy_probe_host}:443:127.0.0.1" ]] ||
  fail legacy_probe_resolve_invalid
[[ -r $HTTPD_CONFIG ]] || fail httpd_config_unreadable
grep -Fq 'ALUMNI_MAINTENANCE_GATE=1' "$HTTPD_CONFIG" || fail httpd_gate_not_installed
httpd -t >/dev/null || fail httpd_config_invalid

if systemctl is-active --quiet "$SERVICE"; then
  backend_was_active=1
fi
if systemctl is-active --quiet "$HTTPD_SERVICE"; then
  httpd_was_active=1
fi

systemctl stop "$SERVICE"
backend_stopped=1
if systemctl is-active --quiet "$SERVICE"; then
  fail backend_service_still_active
fi
main_pid=$(systemctl_property MainPID "$SERVICE") || fail backend_main_pid_unavailable
[[ $main_pid =~ ^[0-9]+$ && $main_pid == 0 ]] || fail backend_process_not_drained

systemctl stop "$HTTPD_SERVICE" || fail httpd_stop_failed
httpd_stopped=1
httpd_main_pid=$(systemctl_property MainPID "$HTTPD_SERVICE") || fail httpd_main_pid_unavailable
[[ $httpd_main_pid =~ ^[0-9]+$ && $httpd_main_pid == 0 ]] || fail legacy_php_process_not_drained

install -d -m 0755 "$(dirname "$SENTINEL")"
generation=$(openssl rand -hex 16) || fail generation_create_failed
[[ $generation =~ ^[a-f0-9]{32}$ ]] || fail generation_format_invalid
tmp=$(mktemp "${SENTINEL}.tmp.XXXXXX")
trap 'rm -f "$tmp"' EXIT
printf 'state=active\ngeneration=%s\n' "$generation" > "$tmp"
chmod 0644 "$tmp"
mv -f "$tmp" "$SENTINEL"
trap - EXIT
[[ -f $SENTINEL ]] || fail sentinel_activation_failed
activated=1

systemctl start "$HTTPD_SERVICE" || fail httpd_start_failed
httpd_stopped=0
systemctl is-active --quiet "$HTTPD_SERVICE" || fail httpd_not_active_after_drain

legacy_status=$(curl --disable --noproxy '*' --silent --show-error --max-time 10 --output /dev/null \
  --write-out '%{http_code}' --resolve "$LEGACY_PROBE_RESOLVE" "$LEGACY_PROBE_URL") ||
  fail legacy_gate_probe_failed
[[ $legacy_status == 503 ]] || fail legacy_gate_not_blocked

activated=0

printf 'PASS maintenance_enter sentinel=active backend=stopped external_writes=blocked\n'
