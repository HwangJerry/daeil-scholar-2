#!/bin/bash
set -euo pipefail

# Deploy script — cross-compile, upload, and restart services
# Usage: ./deploy.sh [user@host]

TARGET=${1:-"user@gabia"}

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
