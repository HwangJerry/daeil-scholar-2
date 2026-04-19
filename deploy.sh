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
