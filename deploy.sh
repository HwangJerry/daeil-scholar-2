#!/bin/bash
set -euo pipefail

# Deploy script — cross-compile, upload, and restart services
# Usage: ./deploy.sh [user@host] [ssh_port] --patch-mode=true|false
#   ssh_port is optional; omit to use ~/.ssh/config or the default 22.
#   --patch-mode is required (no default). Omit to be prompted interactively.
#     true  → user SPA bundles VITE_WIP_ADMIN_CODE from frontend/.env (cover page ON)
#     false → user SPA built with VITE_WIP_ADMIN_CODE="" forced (cover page OFF)

POSITIONAL=()
PATCH_MODE=""
for arg in "$@"; do
  case "$arg" in
    --patch-mode=true)  PATCH_MODE="true" ;;
    --patch-mode=false) PATCH_MODE="false" ;;
    --patch-mode=*)
      echo "✗ Invalid --patch-mode value: '${arg#--patch-mode=}' (must be true or false)" >&2
      exit 1
      ;;
    --patch-mode)
      echo "✗ --patch-mode requires =true or =false (e.g. --patch-mode=true)" >&2
      exit 1
      ;;
    *) POSITIONAL+=("$arg") ;;
  esac
done

TARGET=${POSITIONAL[0]:-"daeil-prod"}
SSH_PORT=${POSITIONAL[1]:-}

if [[ -n "${SSH_PORT}" ]]; then
  SSH_OPTS=(-p "${SSH_PORT}")
  SCP_OPTS=(-P "${SSH_PORT}")
else
  SSH_OPTS=()
  SCP_OPTS=()
fi

# Resolve --patch-mode interactively if it wasn't provided on the command line.
# No default — the user must explicitly answer true or false (mirrors the
# /dev/tty prompt pattern used by the migration drift check below).
if [[ -z "${PATCH_MODE}" ]]; then
  if [[ ! -e /dev/tty ]]; then
    echo "✗ Non-interactive shell — pass --patch-mode=true or --patch-mode=false." >&2
    exit 1
  fi
  while true; do
    read -r -p "Enable WIP gate (admin-code cover page) on the deployed build? [true/false] " ANSWER < /dev/tty
    case "${ANSWER}" in
      true|TRUE|True)    PATCH_MODE="true";  break ;;
      false|FALSE|False) PATCH_MODE="false"; break ;;
      *) echo "  Please type exactly 'true' or 'false'." ;;
    esac
  done
fi
echo "=== patch-mode = ${PATCH_MODE} ==="

# =============================================================================
# DATABASE MIGRATIONS — MANUAL STEP REQUIRED
# =============================================================================
# Migrations are NOT auto-applied by this script.
# Before or after deploying, apply any new migrations manually in order:
#
#   mysql -u USER -p DB_NAME < backend/migrations/NNN_name.sql
#
# Current migration range: 001 through 018
#   001_alter_existing_tables.sql
#   002_create_new_tables.sql
#   003_seed_donation_config.sql
#   004_add_content_format_columns.sql
#   005_add_ad_image_column.sql
#   006_add_usr_photo_column.sql
#   007_create_member_social_table.sql
#   008_alumni_profile_extensions.sql
#   009_create_message_table.sql
#   010_create_subscription_table.sql
#   011 through 018 — see backend/migrations/ for details
#
# Migrations MUST be applied sequentially (in numbered order).
# =============================================================================

# =============================================================================
# ENV VAR VALIDATION — fail fast if the production systemd unit is missing keys
# the application requires, or if a known placeholder value is still in place.
# Source of truth for keys: backend/internal/config/config.go::Load(). Update
# REQUIRED_KEYS / placeholder_value() below when adding/renaming env vars there.
# Skip with: SKIP_ENV_CHECK=1 ./deploy.sh ...
# =============================================================================
SERVICE_PATH="/etc/systemd/system/alumni-backend.service"

REQUIRED_KEYS=(
  ALLOWED_ORIGIN
  SITE_BASE_URL
  DB_USER
  DB_PASSWORD
  DB_NAME
  KAKAO_CLIENT_ID
  KAKAO_CLIENT_SECRET
  KAKAO_REDIRECT_URI
  JWT_SECRET
  UPLOAD_LEGACY_PATH
  EASYPAY_IMMEDIATELY_MALL_ID
  EASYPAY_PROFILE_MALL_ID
  EASYPAY_GW_URL
  EASYPAY_BIN_BASE
  EASYPAY_RETURN_BASE_URL
  SMTP_HOST
  SMTP_USER
  SMTP_PASSWORD
  VISIT_IP_SALT
  ENV
)

# Returns the placeholder value that should NEVER reach production for a key,
# or empty if the key has no known placeholder. Add new entries when introducing
# defaults in config.go that are explicitly insecure or test-only.
placeholder_value() {
  case "$1" in
    JWT_SECRET) echo "change-me-in-production" ;;
    EASYPAY_GW_URL) echo "testgw.easypay.co.kr" ;;
    ENV) echo "dev" ;;
    *) echo "" ;;
  esac
}

# Read systemd unit once; reused by env validation AND migration drift check below.
UNIT_CONTENT=""
if [[ "${SKIP_ENV_CHECK:-0}" != "1" || "${SKIP_MIGRATION_CHECK:-0}" != "1" ]]; then
  if ! UNIT_CONTENT=$(ssh "${SSH_OPTS[@]}" "${TARGET}" "cat ${SERVICE_PATH}" 2>&1); then
    echo "✗ Failed to read ${SERVICE_PATH}:" >&2
    echo "${UNIT_CONTENT}" | sed 's/^/    /' >&2
    echo "  Hint: ensure the service file exists and is world-readable (chmod 644)." >&2
    exit 1
  fi
fi

if [[ "${SKIP_ENV_CHECK:-0}" == "1" ]]; then
  echo "=== Skipping env var validation (SKIP_ENV_CHECK=1) ==="
else
  echo "=== Validating production env vars on ${TARGET} ==="

  MISSING=()
  PLACEHOLDERS=()
  for key in "${REQUIRED_KEYS[@]}"; do
    # Match Environment="KEY=VALUE" and capture VALUE. Empty value counts as missing
    # because config.go's getEnv() falls back to defaults when the var is empty.
    value=$(printf '%s\n' "${UNIT_CONTENT}" | sed -nE "s/^Environment=\"${key}=(.*)\"$/\1/p" | head -n 1)
    if [[ -z "${value}" ]]; then
      MISSING+=("${key}")
      continue
    fi
    expected_placeholder=$(placeholder_value "${key}")
    if [[ -n "${expected_placeholder}" && "${value}" == "${expected_placeholder}" ]]; then
      PLACEHOLDERS+=("${key}=${value}")
    fi
  done

  if [[ ${#MISSING[@]} -gt 0 || ${#PLACEHOLDERS[@]} -gt 0 ]]; then
    echo "" >&2
    echo "✗ Env var validation failed." >&2
    if [[ ${#MISSING[@]} -gt 0 ]]; then
      echo "  Missing or empty (${#MISSING[@]}):" >&2
      for k in "${MISSING[@]}"; do
        echo "    - ${k}" >&2
      done
    fi
    if [[ ${#PLACEHOLDERS[@]} -gt 0 ]]; then
      echo "  Placeholder values still in place (${#PLACEHOLDERS[@]}):" >&2
      for kv in "${PLACEHOLDERS[@]}"; do
        echo "    - ${kv}" >&2
      done
    fi
    echo "" >&2
    echo "Edit ${SERVICE_PATH} on ${TARGET}, then run on the server:" >&2
    echo "    sudo systemctl daemon-reload" >&2
    echo "    sudo systemctl restart alumni-backend" >&2
    echo "Re-run ./deploy.sh once the unit is fixed (or set SKIP_ENV_CHECK=1 to bypass)." >&2
    exit 1
  fi

  echo "✓ ${#REQUIRED_KEYS[@]} required env vars present and non-placeholder"
fi

# =============================================================================
# DEBUG AGENT ENV VALIDATION — all 4 vars must be set together, or all empty.
# Empty = reporter disabled (no-op in observability.NewHook). Partial config is
# rejected because it is almost always a mistake (e.g. forgot to copy SECRET
# into the new unit file). Placeholder secrets/envs are also rejected so a
# half-configured staging value never reaches production.
# Skip with: SKIP_DEBUG_AGENT_CHECK=1 ./deploy.sh ...
# =============================================================================
if [[ "${SKIP_DEBUG_AGENT_CHECK:-0}" == "1" ]]; then
  echo "=== Skipping debug agent env validation (SKIP_DEBUG_AGENT_CHECK=1) ==="
else
  echo "=== Validating debug agent env vars on ${TARGET} ==="

  DA_KEYS=(DEBUG_AGENT_ENDPOINT DEBUG_AGENT_PROJECT DEBUG_AGENT_SECRET DEBUG_AGENT_ENVIRONMENT)

  da_set_count=0
  da_empty_count=0
  da_bad_placeholder=()
  da_present=()
  da_missing=()
  for key in "${DA_KEYS[@]}"; do
    value=$(printf '%s\n' "${UNIT_CONTENT}" | sed -nE "s/^Environment=\"${key}=(.*)\"$/\1/p" | head -n 1)
    if [[ -z "${value}" ]]; then
      da_empty_count=$((da_empty_count + 1))
      da_missing+=("${key}")
    else
      da_set_count=$((da_set_count + 1))
      da_present+=("${key}")
      # Reject known-bad placeholder values. Add new entries here if the team
      # introduces other defaults that must never reach production.
      case "${key}:${value}" in
        DEBUG_AGENT_SECRET:change-me|DEBUG_AGENT_SECRET:test-secret|DEBUG_AGENT_ENVIRONMENT:dev)
          da_bad_placeholder+=("${key}=${value}") ;;
      esac
    fi
  done

  if [[ ${da_set_count} -gt 0 && ${da_empty_count} -gt 0 ]]; then
    echo "" >&2
    echo "✗ Debug agent env vars are partially configured (${da_set_count} set, ${da_empty_count} empty)." >&2
    echo "  Either set all 4, or leave all 4 unset/empty (reporter then runs in no-op mode)." >&2
    echo "  Present:" >&2
    for k in "${da_present[@]}"; do echo "    - ${k}" >&2; done
    echo "  Missing:" >&2
    for k in "${da_missing[@]}"; do echo "    - ${k}" >&2; done
    echo "  Bypass with SKIP_DEBUG_AGENT_CHECK=1 if intentional." >&2
    exit 1
  fi

  if [[ ${#da_bad_placeholder[@]} -gt 0 ]]; then
    echo "" >&2
    echo "✗ Debug agent placeholder values still in production unit:" >&2
    for kv in "${da_bad_placeholder[@]}"; do echo "    - ${kv}" >&2; done
    echo "  Replace with the real values from the Debug Agent dashboard." >&2
    exit 1
  fi

  if [[ ${da_set_count} -eq 4 ]]; then
    echo "✓ Debug agent enabled (all 4 env vars set)"
  else
    echo "✓ Debug agent disabled (no env vars set — reporter will no-op)"
  fi
fi

# =============================================================================
# MIGRATION DRIFT CHECK — fail if local backend/migrations/ has unapplied files
# on the production DB. Migration history is tracked in the _migration_history
# table created by migrate.sh. DB credentials are extracted from the systemd
# unit content read above.
# Skip with: SKIP_MIGRATION_CHECK=1 ./deploy.sh ...
# =============================================================================
if [[ "${SKIP_MIGRATION_CHECK:-0}" == "1" ]]; then
  echo "=== Skipping migration drift check (SKIP_MIGRATION_CHECK=1) ==="
else
  echo "=== Checking migration drift on ${TARGET} ==="

  DB_USER_VAL=$(printf '%s\n' "${UNIT_CONTENT}" | sed -nE 's/^Environment="DB_USER=(.*)"$/\1/p' | head -n 1)
  DB_PASS_VAL=$(printf '%s\n' "${UNIT_CONTENT}" | sed -nE 's/^Environment="DB_PASSWORD=(.*)"$/\1/p' | head -n 1)
  DB_NAME_VAL=$(printf '%s\n' "${UNIT_CONTENT}" | sed -nE 's/^Environment="DB_NAME=(.*)"$/\1/p' | head -n 1)

  if [[ -z "${DB_USER_VAL}" || -z "${DB_NAME_VAL}" ]]; then
    echo "✗ Could not extract DB_USER/DB_NAME from systemd unit." >&2
    echo "  Set SKIP_MIGRATION_CHECK=1 to bypass." >&2
    exit 1
  fi

  if ! REMOTE_LIST=$(ssh "${SSH_OPTS[@]}" "${TARGET}" "MYSQL_PWD='${DB_PASS_VAL}' mysql --skip-ssl -u'${DB_USER_VAL}' -BN '${DB_NAME_VAL}' -e 'SELECT filename FROM _migration_history ORDER BY filename'" 2>&1); then
    echo "✗ Failed to query _migration_history on ${TARGET}:" >&2
    echo "${REMOTE_LIST}" | sed 's/^/    /' >&2
    echo "" >&2
    echo "If _migration_history doesn't exist (fresh DB or pre-migrate.sh era):" >&2
    echo "    1. On the server, run: ./migrate.sh --seed NNN  (NNN = highest already-applied migration number)" >&2
    echo "    2. Re-run ./deploy.sh" >&2
    echo "  Or set SKIP_MIGRATION_CHECK=1 to bypass (NOT recommended)." >&2
    exit 1
  fi

  LOCAL_LIST=$(cd backend/migrations && ls *.sql 2>/dev/null | grep -E '^[0-9]{3}_.*\.sql$' | sort)

  PENDING=()
  while IFS= read -r f; do
    [[ -z "$f" ]] && continue
    if ! grep -Fxq "$f" <<< "${REMOTE_LIST}"; then
      PENDING+=("$f")
    fi
  done <<< "${LOCAL_LIST}"

  if [[ ${#PENDING[@]} -gt 0 ]]; then
    echo "" >&2
    echo "⚠ Migration drift detected — ${#PENDING[@]} unapplied migration(s) on ${TARGET}:" >&2
    for m in "${PENDING[@]}"; do
      echo "    - ${m}" >&2
    done
    echo "" >&2

    if [[ "${APPLY_MIGRATIONS:-0}" == "1" ]]; then
      echo "APPLY_MIGRATIONS=1 set — applying without prompt." >&2
    else
      # Interactive prompt. Read from /dev/tty so this works even if stdin
      # is redirected. In non-interactive shells (no TTY), abort with guidance.
      if [[ ! -e /dev/tty ]]; then
        echo "✗ Non-interactive shell — cannot prompt." >&2
        echo "  Re-run with APPLY_MIGRATIONS=1 to apply, or SKIP_MIGRATION_CHECK=1 to bypass." >&2
        exit 1
      fi
      read -r -p "Apply these ${#PENDING[@]} migration(s) to ${TARGET} now? [y/N] " ANSWER < /dev/tty
      if [[ ! "${ANSWER}" =~ ^[Yy]([Ee][Ss])?$ ]]; then
        echo "Aborted by user. No changes made." >&2
        echo "  Tip: set APPLY_MIGRATIONS=1 to skip this prompt next time." >&2
        exit 1
      fi
    fi

    echo "=== Applying ${#PENDING[@]} pending migration(s) on ${TARGET} ==="
    for m in "${PENDING[@]}"; do
      REMOTE_TMP="/tmp/_mig_$$_${m}"
      echo "  APPLY ${m} ..."
      if ! scp "${SCP_OPTS[@]}" "backend/migrations/${m}" "${TARGET}:${REMOTE_TMP}" >/dev/null; then
        echo "✗ Failed to upload ${m} to ${TARGET}:${REMOTE_TMP}" >&2
        echo "  Backend NOT restarted. Investigate before retrying." >&2
        exit 1
      fi
      if ! ssh "${SSH_OPTS[@]}" "${TARGET}" \
        "MYSQL_PWD='${DB_PASS_VAL}' mysql --skip-ssl -u'${DB_USER_VAL}' '${DB_NAME_VAL}' < ${REMOTE_TMP} \
         && MYSQL_PWD='${DB_PASS_VAL}' mysql --skip-ssl -u'${DB_USER_VAL}' -BN '${DB_NAME_VAL}' \
              -e \"INSERT INTO _migration_history (filename) VALUES ('${m}')\" \
         && rm -f ${REMOTE_TMP}"; then
        echo "✗ Migration failed: ${m}" >&2
        echo "  DB may be in a partial state. Backend NOT restarted. Investigate before retrying." >&2
        ssh "${SSH_OPTS[@]}" "${TARGET}" "rm -f ${REMOTE_TMP}" || true
        exit 1
      fi
      echo "  OK    ${m}"
    done
    echo "✓ Applied ${#PENDING[@]} migration(s)"
  fi

  TOTAL=$(printf '%s\n' "${LOCAL_LIST}" | grep -c .)
  echo "✓ All ${TOTAL} migrations applied on remote"
fi

echo "=== Building Go backend (linux/amd64) ==="
cd backend
GOOS=linux GOARCH=amd64 go build -o ../dist/server ./cmd/server
GOOS=linux GOARCH=amd64 go build -o ../dist/backfill ./cmd/backfill
cd ..

echo "=== Building User SPA (patch-mode=${PATCH_MODE}) ==="
cd frontend
npm ci
if [[ "${PATCH_MODE}" == "true" ]]; then
  # patch-mode=true requires a non-empty VITE_WIP_ADMIN_CODE in frontend/.env so
  # the WipGate component (frontend/src/components/common/WipGate.tsx) is active.
  WIP_CODE_VAL=$(sed -nE 's/^VITE_WIP_ADMIN_CODE=("?)([^"]*)\1[[:space:]]*$/\2/p' .env | head -n 1)
  if [[ -z "${WIP_CODE_VAL}" ]]; then
    echo "✗ patch-mode=true requires VITE_WIP_ADMIN_CODE to be non-empty in frontend/.env" >&2
    echo "  Either set the code in frontend/.env or re-run with --patch-mode=false." >&2
    exit 1
  fi
  npm run build
else
  # Force-empty so .env's value is overridden at build time. The repo .env stays untouched.
  # Vite's loadEnv merges process.env over .env for VITE_*-prefixed vars, so this wins.
  VITE_WIP_ADMIN_CODE="" npm run build
fi
cd ..

echo "=== Building Admin SPA ==="
cd admin
npm ci
npm run build
cd ..

echo "=== Uploading Go binary ==="
scp "${SCP_OPTS[@]}" dist/server "${TARGET}:/app/backend/server.new"
scp "${SCP_OPTS[@]}" dist/backfill "${TARGET}:/app/backend/backfill"
ssh "${SSH_OPTS[@]}" "${TARGET}" 'mv /app/backend/server.new /app/backend/server'

echo "=== Uploading User SPA ==="
rsync -avz --delete -e "ssh ${SSH_OPTS[*]}" frontend/dist/ "${TARGET}:/var/www/app/"

echo "=== Uploading Admin SPA ==="
rsync -avz --delete -e "ssh ${SSH_OPTS[*]}" admin/dist/ "${TARGET}:/var/www/admin/"

echo "=== Reloading systemd and restarting backend ==="
ssh "${SSH_OPTS[@]}" "${TARGET}" 'sudo systemctl daemon-reload && sudo systemctl restart alumni-backend'

echo "=== Uploading Apache httpd config ==="
scp "${SCP_OPTS[@]}" deploy/httpd-alumni.conf "${TARGET}:/tmp/alumni.conf.new"
ssh "${SSH_OPTS[@]}" "${TARGET}" 'sudo mv /tmp/alumni.conf.new /etc/httpd/conf.d/alumni.conf && sudo httpd -t'

echo "=== Uploading legacy PHP compat shims ==="
for shim in _set_docroot.php _legacy_docroot.php _legacy_url_rewriter.php; do
  scp "${SCP_OPTS[@]}" "deploy/${shim}" "${TARGET}:/tmp/${shim}.new"
  ssh "${SSH_OPTS[@]}" "${TARGET}" "sudo mv /tmp/${shim}.new /var/www/html/${shim} && sudo chmod 644 /var/www/html/${shim}"
done

echo "=== Reloading Apache httpd ==="
ssh "${SSH_OPTS[@]}" "${TARGET}" 'sudo systemctl reload httpd'

echo "=== Verifying /old/ legacy routing ==="
SMOKE_HOST="daeilfoundation.or.kr"
# 1) On-server check via loopback with --resolve so hairpin-NAT/DNS issues
#    do not mask Apache config problems. Falls back gracefully if it fails.
ssh "${SSH_OPTS[@]}" "${TARGET}" \
  "curl -s -o /dev/null -w '[server-loopback] HTTP %{http_code} for /old/index.php\n' \
    --resolve ${SMOKE_HOST}:443:127.0.0.1 https://${SMOKE_HOST}/old/index.php" \
  || echo "  ⚠ server-loopback /old/index.php failed (non-fatal)"
ssh "${SSH_OPTS[@]}" "${TARGET}" \
  "curl -s -o /dev/null -w '[server-loopback] HTTP %{http_code} for /old/_sys/css/_common.css\n' \
    --resolve ${SMOKE_HOST}:443:127.0.0.1 https://${SMOKE_HOST}/old/_sys/css/_common.css" \
  || echo "  ⚠ server-loopback /old/_sys/css/_common.css failed (non-fatal)"
# 2) External check from the deploy machine — verifies public reachability.
curl -s -o /dev/null -w "[external]       HTTP %{http_code} for /old/index.php\n" \
  "https://${SMOKE_HOST}/old/index.php" \
  || echo "  ⚠ external /old/index.php failed (non-fatal)"
curl -s -o /dev/null -w "[external]       HTTP %{http_code} for /old/_sys/css/_common.css\n" \
  "https://${SMOKE_HOST}/old/_sys/css/_common.css" \
  || echo "  ⚠ external /old/_sys/css/_common.css failed (non-fatal)"

echo "=== Deploy complete ==="
