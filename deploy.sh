#!/bin/bash
set -euo pipefail

# Deploy script — cross-compile, upload, and restart services
# Usage: ./deploy.sh [user@host] [ssh_port]
#   ssh_port is optional; omit to use ~/.ssh/config or the default 22.

TARGET=${1:-"daeil-prod"}
SSH_PORT=${2:-}

if [[ -n "${SSH_PORT}" ]]; then
  SSH_OPTS=(-p "${SSH_PORT}")
  SCP_OPTS=(-P "${SSH_PORT}")
else
  SSH_OPTS=()
  SCP_OPTS=()
fi

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

if [[ "${SKIP_ENV_CHECK:-0}" == "1" ]]; then
  echo "=== Skipping env var validation (SKIP_ENV_CHECK=1) ==="
else
  echo "=== Validating production env vars on ${TARGET} ==="
  if ! UNIT_CONTENT=$(ssh "${SSH_OPTS[@]}" "${TARGET}" "cat ${SERVICE_PATH}" 2>&1); then
    echo "✗ Failed to read ${SERVICE_PATH}:" >&2
    echo "${UNIT_CONTENT}" | sed 's/^/    /' >&2
    echo "  Hint: ensure the service file exists and is world-readable (chmod 644)." >&2
    exit 1
  fi

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

echo "=== Building Go backend (linux/amd64) ==="
cd backend
GOOS=linux GOARCH=amd64 go build -o ../dist/server ./cmd/server
GOOS=linux GOARCH=amd64 go build -o ../dist/backfill ./cmd/backfill
cd ..

echo "=== Building User SPA ==="
cd frontend
npm ci
npm run build
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

echo "=== Restarting backend ==="
ssh "${SSH_OPTS[@]}" "${TARGET}" 'sudo systemctl restart alumni-backend'

echo "=== Reloading Apache httpd ==="
ssh "${SSH_OPTS[@]}" "${TARGET}" 'sudo systemctl reload httpd'

echo "=== Deploy complete ==="
