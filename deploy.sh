#!/bin/bash
set -euo pipefail

# Deploy script — cross-compile, upload, and restart services
# Usage: ./deploy.sh [user@host]

TARGET=${1:-"user@gabia"}

# =============================================================================
# DATABASE MIGRATIONS — MANUAL STEP REQUIRED
# =============================================================================
# Migrations are NOT auto-applied by this script.
# Before or after deploying, apply any new migrations manually in order:
#
#   mysql -u USER -p DB_NAME < backend/migrations/NNN_name.sql
#
# Current migration range: 001 through 010
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
npm run build
cd ..

echo "=== Building Admin SPA ==="
cd admin
npm run build
cd ..

echo "=== Uploading Go binary ==="
scp dist/server "${TARGET}:/app/backend/server.new"
scp dist/backfill "${TARGET}:/app/backend/backfill"
ssh "${TARGET}" 'mv /app/backend/server.new /app/backend/server'

echo "=== Uploading User SPA ==="
scp -r frontend/dist/ "${TARGET}:/var/www/app/"

echo "=== Uploading Admin SPA ==="
scp -r admin/dist/ "${TARGET}:/var/www/admin/"

echo "=== Restarting backend ==="
ssh "${TARGET}" 'sudo systemctl restart alumni-backend'

echo "=== Reloading Nginx ==="
ssh "${TARGET}" 'sudo systemctl reload nginx'

echo "=== Deploy complete ==="
